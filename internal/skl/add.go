// Package skl implements skl's core business logic: fetching a skill from a
// source, discovering which directory within it is the skill, copying that
// skill into each targeted adapter's destination, and recording the result
// in a lockfile. It has no dependency on Cobra or Viper, so it is directly
// unit-testable via t.TempDir().
package skl

import (
	"fmt"
	"os"
	"path/filepath"
)

// Adapter is a consuming coding-agent client that skl installs skills for.
// Each adapter has its own independent destination directory (see
// ADR-0002), rooted at ProjectDir relative to the project root.
type Adapter struct {
	// Name is the adapter's identifier, e.g. "claude-code".
	Name string
	// ProjectDir is this adapter's project-scope skills directory, relative
	// to the project root, e.g. ".claude/skills".
	ProjectDir string
}

// Adapter names, as named constants to avoid typo-prone string literals
// scattered across detection, resolution, and destination logic.
const (
	ClaudeCodeAdapter = "claude-code"
	KitAdapter        = "kit"
	UniversalAdapter  = "universal"
)

// Adapters is the fixed set of adapters skl installs to in v1.0.
var Adapters = []Adapter{
	{Name: ClaudeCodeAdapter, ProjectDir: filepath.Join(".claude", "skills")},
	{Name: KitAdapter, ProjectDir: filepath.Join(".kit", "skills")},
	{Name: UniversalAdapter, ProjectDir: filepath.Join(".agents", "skills")},
}

// resolveLockPath returns the lockfile path Add should read from and write
// to: the global lockfile (see globalLockfilePath) when global is true,
// else the project-scoped .skl-lock.json at projectRoot's root.
func resolveLockPath(projectRoot string, global bool) (string, error) {
	if global {
		return globalLockfilePath()
	}
	return filepath.Join(projectRoot, ".skl-lock.json"), nil
}

// resolveDestination returns where a skill named name should be installed
// on disk for adapter, and the corresponding AdapterEntry.Path to record
// in the lockfile. In project scope, dest is projectRoot-rooted and Path
// is root-relative (e.g. ".claude/skills/<name>"), matching resolveLockPath's
// project-scoped branch. In global scope, dest is adapter's global
// directory (see globalAdapterDir) and Path is that same absolute path,
// since global installs have no shared root to make it relative to.
func resolveDestination(adapter Adapter, name, projectRoot string, global bool) (dest, entryPath string, err error) {
	if global {
		globalDir, err := globalAdapterDir(adapter.Name)
		if err != nil {
			return "", "", fmt.Errorf("resolving global destination for adapter %q: %w", adapter.Name, err)
		}
		dest = filepath.Join(globalDir, name)
		return dest, dest, nil
	}
	relDest := filepath.Join(adapter.ProjectDir, name)
	return filepath.Join(projectRoot, relDest), relDest, nil
}

// adapterByName looks up an Adapter by name within Adapters.
func adapterByName(name string) (Adapter, bool) {
	for _, a := range Adapters {
		if a.Name == name {
			return a, true
		}
	}
	return Adapter{}, false
}

// AddOptions configures a call to Add.
type AddOptions struct {
	// Source is where the skill is fetched from: a local filesystem path,
	// a GitHub shorthand ("owner/repo"), a full GitHub URL
	// ("https://github.com/owner/repo"), a GitHub tree-path URL
	// ("owner/repo/tree/<ref>/<path>", or the equivalent full
	// "https://github.com/owner/repo/tree/<ref>/<path>") naming a ref and
	// a repo-relative path to install the skill from directly, or a
	// shorthand/full-URL GitHub source with an @ref pin ("owner/repo@v2.1.0")
	// or the compact @ref/<path> shorthand ("owner/repo@main/skills/tdd",
	// equivalent in effect to the corresponding tree-path URL); see #11. A
	// pin can't be combined with a tree-path source (which already encodes
	// a ref) or a local-path source (which isn't fetched from a ref at
	// all). An unpinned, non-tree-path GitHub source is fetched at its
	// default branch and must contain a root SKILL.md.
	Source string
	// Fetcher fetches a GitHub source's contents into a local directory. A
	// nil Fetcher (the default) uses GitHubFetcher, which performs a real
	// network fetch. Tests inject a fake implementation so the rest of the
	// test suite runs without network access. Unused for local-path
	// sources, which bypass the Fetcher entirely.
	Fetcher Fetcher
	// ProjectRoot is the project root skl installs into and where it reads
	// and writes .skl-lock.json. Defaults to the current working directory
	// when empty.
	ProjectRoot string
	// RequestedAdapters holds the raw --agent flag values: comma-separated
	// entries, repeated-flag entries, or both, optionally containing the
	// literal "*". Empty means no --agent was given, triggering the
	// detected-adapter default heuristic. See ResolveAdapters.
	RequestedAdapters []string
	// Global selects global scope: each adapter's global destination
	// directory (e.g. ~/.claude/skills/<name>) and the global lockfile
	// (see globalLockfilePath) instead of the project-scoped equivalents.
	// ProjectRoot is ignored when true.
	Global bool
	// Force overrides two refusals that would otherwise stop Add: a
	// conflict (an existing lockfile entry with the same name but a
	// different source) and a destination collision (a targeted
	// destination directory that already exists and is non-empty). In
	// both cases Force replaces what's there rather than merging with it.
	// See CONTEXT.md's "Expand"/"Conflict" definitions.
	Force bool
}

// AddResult describes the outcome of a successful Add call.
type AddResult struct {
	// Name is the installed skill's name (its destination directory name).
	Name string
	// Adapters maps each adapter name Add installed to to its destination:
	// project-root-relative in project scope, or the absolute global
	// destination path in global scope (see resolveDestination). Shares
	// AdapterEntry with LockEntry since both describe the same "adapter
	// name -> destination" fact.
	Adapters map[string]AdapterEntry
}

// Add fetches the skill at opts.Source and installs it into each targeted
// adapter's project destination under opts.ProjectRoot, then records the
// install in that project's .skl-lock.json. Which adapters are targeted is
// determined by ResolveAdapters from opts.RequestedAdapters and the
// adapters detected on this machine.
//
// Lockfile identity is matched by name and source together (see ADR-0003
// and CONTEXT.md's "Expand"/"Conflict" definitions):
//
//   - Same name, same source: the existing entry's adapters map is
//     expanded with the newly targeted adapter(s) rather than duplicated.
//   - Same name, different source: a conflict, refused unless opts.Force
//     is set, in which case the entry's source/metadata is replaced
//     entirely, only the adapters targeted by this call are recorded, and
//     any destination the old entry tracked for an adapter this call
//     doesn't target is removed from disk (it would otherwise be an
//     orphaned, untracked install).
//
// Independently, each targeted adapter's destination is checked for a
// collision: an existing, non-empty directory not already tracked by this
// same name+source lockfile entry is refused unless opts.Force is set, in
// which case its contents are removed and replaced cleanly.
func Add(opts AddOptions) (*AddResult, error) {
	if opts.Source == "" {
		return nil, fmt.Errorf("source is required")
	}

	projectRoot := opts.ProjectRoot
	if !opts.Global && projectRoot == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("determining project root: %w", err)
		}
		projectRoot = wd
	}

	rs, err := resolveSource(opts)
	if err != nil {
		return nil, err
	}
	if rs.Cleanup != nil {
		defer rs.Cleanup()
	}

	targetNames, err := ResolveAdapters(opts.RequestedAdapters, DetectedAdapters())
	if err != nil {
		return nil, err
	}

	skillDir, err := discoverSkillDir(rs.Dir)
	if err != nil {
		if rs.SourceType == sourceTypeGitHub {
			return nil, fmt.Errorf("locating a skill in %s: %w", rs.Source, err)
		}
		return nil, err
	}
	name := rs.SuggestedName
	if name == "" {
		name = filepath.Base(skillDir)
	}

	// Read the skill directory once; both the content hash and every
	// adapter's copy are derived from this single snapshot rather than
	// re-walking the filesystem per adapter.
	files, err := readSkillFiles(skillDir)
	if err != nil {
		return nil, fmt.Errorf("reading skill %q: %w", name, err)
	}
	contentHash := hashFiles(files)

	lockPath, err := resolveLockPath(projectRoot, opts.Global)
	if err != nil {
		return nil, fmt.Errorf("resolving lockfile path: %w", err)
	}
	lf, err := ReadLockfile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("reading lockfile: %w", err)
	}

	existing, hasEntry := lf[name]
	conflict := hasEntry && existing.Source != rs.Source
	if conflict && !opts.Force {
		return nil, fmt.Errorf("skill %q is already installed from a different source %q; refusing to overwrite (use --force to replace it)", name, existing.Source)
	}

	if conflict && opts.Force {
		// Fully replacing a conflicting entry: any adapter destination
		// tracked under the old source but not re-targeted by this call
		// would otherwise be left on disk holding stale content the
		// lockfile no longer references. Remove those so the lockfile stays
		// the single accurate record of what's installed.
		targetSet := make(map[string]bool, len(targetNames))
		for _, t := range targetNames {
			targetSet[t] = true
		}
		for adapterName, entry := range existing.Adapters {
			if targetSet[adapterName] {
				continue
			}
			stalePath := entry.Path
			if !opts.Global {
				stalePath = filepath.Join(projectRoot, filepath.FromSlash(entry.Path))
			}
			if err := os.RemoveAll(stalePath); err != nil {
				return nil, fmt.Errorf("removing stale destination %q for adapter %q: %w", stalePath, adapterName, err)
			}
		}
	}

	// alreadyTracked holds the adapters this exact name+source is already
	// recorded as installed for, so reinstalling to one of them is treated
	// as an update, not a destination collision.
	alreadyTracked := map[string]bool{}
	if hasEntry && !conflict {
		for adapterName := range existing.Adapters {
			alreadyTracked[adapterName] = true
		}
	}

	// Resolve and validate every target's destination before writing
	// anything, so a collision discovered on a later adapter doesn't leave
	// an earlier adapter's directory written but the lockfile untouched.
	type installTarget struct {
		adapter   Adapter
		dest      string
		entryPath string
	}
	targets := make([]installTarget, 0, len(targetNames))
	for _, targetName := range targetNames {
		adapter, ok := adapterByName(targetName)
		if !ok {
			return nil, fmt.Errorf("internal error: resolved unknown adapter %q", targetName)
		}

		dest, entryPath, err := resolveDestination(adapter, name, projectRoot, opts.Global)
		if err != nil {
			return nil, err
		}

		if !opts.Force && !alreadyTracked[targetName] {
			nonEmpty, err := dirHasEntries(dest)
			if err != nil {
				return nil, fmt.Errorf("checking destination %q: %w", dest, err)
			}
			if nonEmpty {
				return nil, fmt.Errorf("destination %q already exists and is not empty; refusing to overwrite (use --force to replace it)", dest)
			}
		}

		targets = append(targets, installTarget{adapter: adapter, dest: dest, entryPath: entryPath})
	}

	adapterEntries := make(map[string]AdapterEntry, len(targets))
	for _, t := range targets {
		if err := writeFiles(files, t.dest); err != nil {
			return nil, fmt.Errorf("installing %q for adapter %q: %w", name, t.adapter.Name, err)
		}
		adapterEntries[t.adapter.Name] = AdapterEntry{Path: filepath.ToSlash(t.entryPath)}
	}

	if hasEntry && !conflict {
		// Expand: merge this call's adapters into the existing entry rather
		// than creating a duplicate.
		merged := make(map[string]AdapterEntry, len(existing.Adapters)+len(adapterEntries))
		for adapterName, entry := range existing.Adapters {
			merged[adapterName] = entry
		}
		for adapterName, entry := range adapterEntries {
			merged[adapterName] = entry
		}
		existing.ContentHash = contentHash
		existing.Adapters = merged
		lf[name] = existing
	} else {
		// Brand-new entry, or a forced conflict replacing the old one
		// entirely: only this call's targeted adapters are recorded.
		lf[name] = LockEntry{
			Source:      rs.Source,
			SourceType:  rs.SourceType,
			SourceURL:   rs.SourceURL,
			Ref:         rs.Ref,
			SkillPath:   rs.SkillPath,
			ContentHash: contentHash,
			Adapters:    adapterEntries,
		}
	}

	if err := WriteLockfile(lockPath, lf); err != nil {
		return nil, fmt.Errorf("writing lockfile: %w", err)
	}

	return &AddResult{Name: name, Adapters: adapterEntries}, nil
}
