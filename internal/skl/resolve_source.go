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
	// Dir is the local directory discoverSkills should look within: the
	// local source path itself, or a GitHub source's freshly fetched temp
	// directory -- for a tree-path source, the authoritative skill
	// subdirectory within that fetched temp directory (see
	// resolveGitHubSource), so discoverSkills' root-SKILL.md check lands
	// directly on the skill, no further walk needed.
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
	// directory has no meaningful basename of its own. Only ever applied
	// by Add when the whole source resolves to a single skill (see
	// discoverSkills) -- a plain GitHub source containing multiple skills
	// keeps each one's own flattened name instead.
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
	// SkillPath is "." for a local source or a plain GitHub source (Dir
	// itself is where skill discovery starts) or the tree-path's path for
	// a tree-path GitHub source (Dir already *is* the skill). Add derives
	// each individual discovered skill's own LockEntry.SkillPath from
	// this via skillPathFor, joining in the skill's path relative to Dir
	// when SkillPath is "." and discovery found it nested (e.g.
	// "skills/tdd").
	SkillPath string
}

// resolveSource classifies opts.Source and turns it into a resolvedSource:
// a GitHub source is fetched (via opts.Fetcher, defaulting to
// GitHubFetcher) at its default branch, or at the ref a tree-path source
// or an @ref pin (see #11) names; a local source is validated as an
// existing directory. Local-path sources bypass the Fetcher entirely.
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
// parsed.Path is set (a tree-path source, or a shorthand/full-URL source
// pinned via #11's @ref/<path> syntax), Dir is pointed directly at that
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
	cleanup := func() { _ = os.RemoveAll(dir) }

	if parsed.Path == "" {
		return resolvedSource{
			Dir:           dir,
			Cleanup:       cleanup,
			SuggestedName: src.Repo,
			Source:        pinnedIdentity(src.String(), parsed.Ref),
			SourceType:    sourceTypeGitHub,
			SourceURL:     pinnedIdentity(src.URL(), parsed.Ref),
			Ref:           parsed.Ref,
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
		Cleanup:    cleanup,
		Source:     treePathOf(src.String(), parsed.Ref, parsed.Path),
		SourceType: sourceTypeGitHub,
		SourceURL:  treePathOf(src.URL(), parsed.Ref, parsed.Path),
		Ref:        parsed.Ref,
		SkillPath:  parsed.Path,
	}, nil
}

// treePathOf appends a tree-path source's ref and path onto base (either a
// GitHubSource's canonical "owner/repo" form or its full URL), producing
// the canonical "…/tree/<ref>/<path>" identity resolveGitHubSource records
// for both Source and SourceURL.
func treePathOf(base, ref, path string) string {
	return fmt.Sprintf("%s/tree/%s/%s", base, ref, path)
}

// pinnedIdentity appends a bare @ref pin (no path) onto base, producing
// the canonical "owner/repo@<ref>" (or full-URL equivalent) identity
// resolveGitHubSource records for a shorthand/full-URL source pinned via
// #11's @ref syntax — so pinning to a different ref is tracked as a
// different source (see ADR-0003's conflict/expand identity), and re-add
// with the same ref expands the existing entry regardless of whether it
// was reached via shorthand or full URL. Returns base unchanged when ref
// is empty, the plain unpinned case.
func pinnedIdentity(base, ref string) string {
	if ref == "" {
		return base
	}
	return base + "@" + ref
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
