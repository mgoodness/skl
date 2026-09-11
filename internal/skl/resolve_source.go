package skl

import (
	"fmt"
	"os"
	"path/filepath"
)

// Source type identifiers, recorded verbatim in LockEntry.SourceType.
// Named as constants (mirroring sourceKind's typed constants in
// source.go) to avoid repeating untyped string literals at each call
// site.
const (
	sourceTypeLocal  = "local"
	sourceTypeGitHub = "github"
)

// resolvedSource is what resolveSource turns an AddOptions.Source into: a
// local directory ready for skill discovery, plus the identity fields Add
// records into the resulting LockEntry.
type resolvedSource struct {
	// Dir is the local directory discoverSkillDir should walk: the local
	// source path itself, or a GitHub source's freshly fetched temp
	// directory — for a tree-path source, the authoritative skill
	// subdirectory within that fetched temp directory (see
	// resolveGitHubSource), so discoverSkillDir's root-SKILL.md check
	// lands directly on the skill, no walk needed.
	Dir string
	// Cleanup removes any temporary state resolving the source created —
	// non-nil only for GitHub sources, whose fetched temp directory Add
	// must remove once it's done copying from it. nil for local sources,
	// which own nothing extra to clean up.
	Cleanup func()
	// SuggestedName, when non-empty, is the skill name Add should use
	// instead of deriving one from the discovered skill directory's own
	// basename. Empty for local sources (whose directory's basename is
	// the name) and for a tree-path GitHub source (whose Dir already
	// points at the meaningful, authoritative skill subdirectory, whose
	// basename is just as good a name); set to the repository name for a
	// plain (non-tree-path) GitHub source, since a fetched source's temp
	// directory has no meaningful basename of its own.
	SuggestedName string
	// Source, SourceType, SourceURL, Ref, and SkillPath are recorded
	// verbatim into the resulting LockEntry, and Source is what
	// conflict/expand identity is matched against (see ADR-0003) — the
	// canonical "owner/repo" form for a plain GitHub source, or
	// "owner/repo/tree/<ref>/<path>" for a tree-path source, regardless of
	// whether the user typed shorthand or a full URL, so both surface
	// syntaxes for the same repository (and, for a tree-path source, the
	// same ref and path) are recognized as the same source.
	Source, SourceType, SourceURL, Ref string
	// SkillPath is the repo-relative path (forward-slash form) that is
	// the skill within its source: "." for a local source or a plain
	// GitHub source (the source's own root is the skill), or the
	// tree-path's path for a tree-path GitHub source.
	SkillPath string
}

// resolveSource classifies opts.Source and turns it into a resolvedSource:
// a GitHub source is fetched (via opts.Fetcher, defaulting to
// GitHubFetcher) at its default branch, or at the ref a tree-path source
// names; a local source is validated as an existing directory. Local-path
// sources bypass the Fetcher entirely.
func resolveSource(opts AddOptions) (resolvedSource, error) {
	parsed, err := parseSource(opts.Source)
	if err != nil {
		return resolvedSource{}, err
	}
	if parsed.Kind == sourceKindGitHub {
		return resolveGitHubSource(opts, parsed)
	}
	return resolveLocalSource(opts)
}

// resolveGitHubSource fetches parsed.GitHub at parsed.Ref (empty meaning
// the source's default branch) via opts.Fetcher (defaulting to
// GitHubFetcher) and describes the result as a resolvedSource. When
// parsed.Path is set (a tree-path source), Dir is pointed directly at that
// subdirectory within the fetched repository — the path is authoritative,
// so no discovery walk happens.
func resolveGitHubSource(opts AddOptions, parsed parsedSource) (resolvedSource, error) {
	fetcher := opts.Fetcher
	if fetcher == nil {
		fetcher = &GitHubFetcher{}
	}

	src := parsed.GitHub
	dir, err := fetcher.Fetch(src, parsed.Ref)
	if err != nil {
		return resolvedSource{}, fmt.Errorf("fetching %s: %w", src, err)
	}

	if parsed.Path == "" {
		return resolvedSource{
			Dir:           dir,
			Cleanup:       func() { _ = os.RemoveAll(dir) },
			SuggestedName: src.Repo,
			Source:        src.String(),
			SourceType:    sourceTypeGitHub,
			SourceURL:     src.URL(),
			SkillPath:     ".",
		}, nil
	}

	skillDir := filepath.Join(dir, filepath.FromSlash(parsed.Path))
	if info, statErr := os.Stat(skillDir); statErr != nil || !info.IsDir() {
		_ = os.RemoveAll(dir)
		return resolvedSource{}, fmt.Errorf("path %q not found in %s at ref %q", parsed.Path, src, parsed.Ref)
	}

	return resolvedSource{
		Dir:        skillDir,
		Cleanup:    func() { _ = os.RemoveAll(dir) },
		Source:     fmt.Sprintf("%s/tree/%s/%s", src.String(), parsed.Ref, parsed.Path),
		SourceType: sourceTypeGitHub,
		SourceURL:  fmt.Sprintf("%s/tree/%s/%s", src.URL(), parsed.Ref, parsed.Path),
		Ref:        parsed.Ref,
		SkillPath:  parsed.Path,
	}, nil
}

// resolveLocalSource validates opts.Source as an existing local directory
// and describes it as a resolvedSource.
func resolveLocalSource(opts AddOptions) (resolvedSource, error) {
	sourceAbs, err := filepath.Abs(opts.Source)
	if err != nil {
		return resolvedSource{}, fmt.Errorf("resolving source %q: %w", opts.Source, err)
	}
	info, err := os.Stat(sourceAbs)
	if err != nil || !info.IsDir() {
		return resolvedSource{}, fmt.Errorf(
			"unsupported source %q: not a local directory, and not a recognized GitHub source (expected \"owner/repo\" shorthand, a full https://github.com/owner/repo URL, or a tree-path owner/repo/tree/<ref>/<path> URL)",
			opts.Source,
		)
	}

	return resolvedSource{
		Dir:        sourceAbs,
		Source:     opts.Source,
		SourceType: sourceTypeLocal,
		SkillPath:  ".",
	}, nil
}
