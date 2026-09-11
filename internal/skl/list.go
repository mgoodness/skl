package skl

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Scope labels which lockfile a ListEntry came from.
const (
	ProjectScope = "project"
	GlobalScope  = "global"
)

// ListOptions configures a call to List.
type ListOptions struct {
	// ProjectRoot is the project root List reads .skl-lock.json from.
	// Defaults to the current working directory when empty. Unlike Add,
	// there is no Global toggle here: List always reads both the project
	// and global lockfiles (see resolveLockPath), each independently
	// per ADR-0003.
	ProjectRoot string
	// Agent, when non-empty, filters the listing down to that one
	// adapter: a skill not installed for it is omitted entirely, and
	// every other adapter is omitted from a skill that is. Must name a
	// known adapter (see Adapters).
	Agent string
}

// ListAdapterEntry is one adapter's install state within a ListEntry,
// annotated with any drift List detects relative to the current machine
// and disk state.
type ListAdapterEntry struct {
	// Name is the adapter's identifier, e.g. "claude-code".
	Name string
	// Path is the destination path recorded in the lockfile: project-
	// root-relative in project scope, absolute in global scope (see
	// AdapterEntry).
	Path string
	// NotDetected is true when the lockfile records this adapter but the
	// current machine doesn't have it detected (see DetectedAdapters).
	NotDetected bool
	// Missing is true when Path no longer exists on disk.
	Missing bool
}

// ListEntry is one skill's listing: which scope it came from, its name,
// and its per-adapter install state.
type ListEntry struct {
	// Scope is either ProjectScope or GlobalScope.
	Scope string
	// Name is the skill's name, matching its lockfile key.
	Name string
	// Adapters holds this skill's per-adapter entries, in Adapters order,
	// filtered by ListOptions.Agent when set.
	Adapters []ListAdapterEntry
}

// List reads both the project-scoped and global lockfiles and returns
// every installed skill's entry, labeled by scope and annotated with
// (not detected)/(missing) drift per adapter. Project-scope entries come
// before global-scope entries; within a scope, entries are sorted
// alphabetically by name (Lockfile's map form has no inherent order).
func List(opts ListOptions) ([]ListEntry, error) {
	if opts.Agent != "" {
		if _, ok := adapterByName(opts.Agent); !ok {
			known := make([]string, 0, len(Adapters))
			for _, a := range Adapters {
				known = append(known, a.Name)
			}
			return nil, fmt.Errorf("unknown adapter %q; known adapters: %s", opts.Agent, strings.Join(known, ", "))
		}
	}

	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("determining project root: %w", err)
		}
		projectRoot = wd
	}

	projectLockPath, err := resolveLockPath(projectRoot, false)
	if err != nil {
		return nil, fmt.Errorf("resolving project lockfile path: %w", err)
	}
	projectLF, err := ReadLockfile(projectLockPath)
	if err != nil {
		return nil, fmt.Errorf("reading project lockfile: %w", err)
	}

	globalLockPath, err := resolveLockPath(projectRoot, true)
	if err != nil {
		return nil, fmt.Errorf("resolving global lockfile path: %w", err)
	}
	globalLF, err := ReadLockfile(globalLockPath)
	if err != nil {
		return nil, fmt.Errorf("reading global lockfile: %w", err)
	}

	detectedSet := make(map[string]bool, len(Adapters))
	for _, name := range DetectedAdapters() {
		detectedSet[name] = true
	}

	entries := listScope(ProjectScope, projectLF, projectRoot, opts.Agent, detectedSet)
	entries = append(entries, listScope(GlobalScope, globalLF, projectRoot, opts.Agent, detectedSet)...)
	return entries, nil
}

// listScope builds the sorted ListEntry slice for a single lockfile.
// projectRoot is only consulted for scope == ProjectScope, to resolve
// each entry's project-root-relative path down to an absolute one for the
// on-disk existence check; global-scope paths are already absolute.
func listScope(scope string, lf Lockfile, projectRoot, agentFilter string, detectedSet map[string]bool) []ListEntry {
	names := make([]string, 0, len(lf))
	for name := range lf {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]ListEntry, 0, len(names))
	for _, name := range names {
		lockEntry := lf[name]

		adapters := make([]ListAdapterEntry, 0, len(lockEntry.Adapters))
		for _, a := range Adapters {
			entry, ok := lockEntry.Adapters[a.Name]
			if !ok {
				continue
			}
			if agentFilter != "" && a.Name != agentFilter {
				continue
			}
			adapters = append(adapters, ListAdapterEntry{
				Name:        a.Name,
				Path:        entry.Path,
				NotDetected: !detectedSet[a.Name],
				Missing:     !destinationExists(scope, projectRoot, entry.Path),
			})
		}
		if len(adapters) == 0 {
			continue
		}

		entries = append(entries, ListEntry{Scope: scope, Name: name, Adapters: adapters})
	}
	return entries
}

// destinationExists reports whether entryPath still exists on disk.
// entryPath is project-root-relative in project scope (resolved against
// projectRoot) and already absolute in global scope, matching
// resolveDestination's AdapterEntry.Path convention.
func destinationExists(scope, projectRoot, entryPath string) bool {
	full := entryPath
	if scope == ProjectScope {
		full = filepath.Join(projectRoot, filepath.FromSlash(entryPath))
	}
	_, err := os.Stat(full)
	return err == nil
}
