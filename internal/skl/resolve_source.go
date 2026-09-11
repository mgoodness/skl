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
	// Source, SourceType, SourceURL, and SkillPath are recorded verbatim
	// into the resulting LockEntry, and Source is what conflict/expand
	// identity is matched against (see ADR-0003) — the canonical
	// "owner/repo" form for a plain GitHub source, or
	// "owner/repo/tree/<path>" for a tree-path source, regardless of
	// whether the user typed shorthand or a full URL, so both surface
	// syntaxes for the same repository (and, for a tree-path source, the
	// same path) are recognized as the same source. There is no Ref field
	// here: v1 has no ref-pinning concept at all (see #11, deferred), so
	// every GitHub source — tree-path included — is always fetched at the
	// repository's default branch, and LockEntry.Ref is simply left unset.
	Source, SourceType, SourceURL string
	// SkillPath is the repo-relative path (forward-slash form) that is
	// the skill within its source: "." for a local source or a plain
	// GitHub source (the source's own root is the skill), or the
	// tree-path's path for a tree-path GitHub source.
	SkillPath string
}

// resolveSource classifies opts.Source and turns it into a resolvedSource:
// a GitHub source is fetched (via opts.Fetcher, defaulting to
// GitHubFetcher) at its default branch — always, even for a tree-path
// source (see parsedSource's doc comment; ref-pinning is deferred to
// #11); a local source is validated as an existing directory. Local-path
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

// resolveGitHubSource fetches parsed.GitHub at its default branch via
// opts.Fetcher (defaulting to GitHubFetcher) and describes the result as a
// resolvedSource. When parsed.Path is set (a tree-path source), Dir is
// pointed directly at that subdirectory within the fetched repository —
// the path is authoritative, so no discovery walk happens.
func resolveGitHubSource(opts AddOptions, parsed parsedSource) (resolvedSource, error) {
	fetcher := opts.Fetcher
	if fetcher == nil {
		fetcher = &GitHubFetcher{}
	}

	src := parsed.GitHub
	dir, err := fetcher.Fetch(src, "")
	if err != nil {
		return resolvedSource{}, fmt.Errorf("fetching %s: %w", src, err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	if parsed.Path == "" {
		return resolvedSource{
			Dir:           dir,
			Cleanup:       cleanup,
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
		return resolvedSource{}, fmt.Errorf("path %q not found in %s's default branch", parsed.Path, src)
	}

	return resolvedSource{
		Dir:        skillDir,
		Cleanup:    cleanup,
		Source:     treePathOf(src.String(), parsed.Path),
		SourceType: sourceTypeGitHub,
		SourceURL:  treePathOf(src.URL(), parsed.Path),
		SkillPath:  parsed.Path,
	}, nil
}

// treePathOf appends a tree-path source's path onto base (either a
// GitHubSource's canonical "owner/repo" form or its full URL), producing
// the canonical "…/tree/<path>" identity resolveGitHubSource records for
// both Source and SourceURL.
func treePathOf(base, path string) string {
	return fmt.Sprintf("%s/tree/%s", base, path)
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
			"unsupported source %q: not a local directory, and not a recognized GitHub source (expected \"owner/repo\" shorthand, a full https://github.com/owner/repo URL, or a tree-path owner/repo/tree/<path> URL)",
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
