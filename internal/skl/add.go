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
	// Source is where the skill is fetched from. In this ticket's scope,
	// only a local filesystem path is supported.
	Source string
	// ProjectRoot is the project root skl installs into and where it reads
	// and writes .skl-lock.json. Defaults to the current working directory
	// when empty.
	ProjectRoot string
	// RequestedAdapters holds the raw --agent flag values: comma-separated
	// entries, repeated-flag entries, or both, optionally containing the
	// literal "*". Empty means no --agent was given, triggering the
	// detected-adapter default heuristic. See ResolveAdapters.
	RequestedAdapters []string
}

// AddResult describes the outcome of a successful Add call.
type AddResult struct {
	// Name is the installed skill's name (its destination directory name).
	Name string
	// Adapters maps each adapter name Add installed to to its
	// project-relative destination. Shares AdapterEntry with LockEntry
	// since both describe the same "adapter name -> destination" fact.
	Adapters map[string]AdapterEntry
}

// Add fetches the skill at opts.Source and installs it into each targeted
// adapter's project destination under opts.ProjectRoot, then records the
// install in that project's .skl-lock.json. Which adapters are targeted is
// determined by ResolveAdapters from opts.RequestedAdapters and the
// adapters detected on this machine.
func Add(opts AddOptions) (*AddResult, error) {
	if opts.Source == "" {
		return nil, fmt.Errorf("source is required")
	}

	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("determining project root: %w", err)
		}
		projectRoot = wd
	}

	sourceAbs, err := filepath.Abs(opts.Source)
	if err != nil {
		return nil, fmt.Errorf("resolving source %q: %w", opts.Source, err)
	}
	info, err := os.Stat(sourceAbs)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("unsupported source %q: only local directory paths are supported in this version", opts.Source)
	}

	targetNames, err := ResolveAdapters(opts.RequestedAdapters, DetectedAdapters())
	if err != nil {
		return nil, err
	}

	skillDir, err := discoverSkillDir(sourceAbs)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(skillDir)

	// Read the skill directory once; both the content hash and every
	// adapter's copy are derived from this single snapshot rather than
	// re-walking the filesystem per adapter.
	files, err := readSkillFiles(skillDir)
	if err != nil {
		return nil, fmt.Errorf("reading skill %q: %w", name, err)
	}
	contentHash := hashFiles(files)

	adapterEntries := make(map[string]AdapterEntry, len(targetNames))
	for _, targetName := range targetNames {
		adapter, ok := adapterByName(targetName)
		if !ok {
			return nil, fmt.Errorf("internal error: resolved unknown adapter %q", targetName)
		}
		relDest := filepath.Join(adapter.ProjectDir, name)
		dest := filepath.Join(projectRoot, relDest)
		if err := writeFiles(files, dest); err != nil {
			return nil, fmt.Errorf("installing %q for adapter %q: %w", name, adapter.Name, err)
		}
		adapterEntries[adapter.Name] = AdapterEntry{Path: filepath.ToSlash(relDest)}
	}

	lockPath := filepath.Join(projectRoot, ".skl-lock.json")
	lf, err := ReadLockfile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("reading lockfile: %w", err)
	}

	lf[name] = LockEntry{
		Source:      opts.Source,
		SourceType:  "local",
		SkillPath:   ".",
		ContentHash: contentHash,
		Adapters:    adapterEntries,
	}

	if err := WriteLockfile(lockPath, lf); err != nil {
		return nil, fmt.Errorf("writing lockfile: %w", err)
	}

	return &AddResult{Name: name, Adapters: adapterEntries}, nil
}
