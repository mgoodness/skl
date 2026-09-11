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
	// directory.
	Dir string
	// Cleanup removes any temporary state resolving the source created —
	// non-nil only for GitHub sources, whose fetched temp directory Add
	// must remove once it's done copying from it. nil for local sources,
	// which own nothing extra to clean up.
	Cleanup func()
	// SuggestedName, when non-empty, is the skill name Add should use
	// instead of deriving one from the discovered skill directory's own
	// basename. Empty for local sources (whose directory's basename is
	// the name); set to the repository name for GitHub sources, since a
	// fetched source's temp directory has no meaningful basename of its
	// own.
	SuggestedName string
	// Source, SourceType, SourceURL, and Ref are recorded verbatim into
	// the resulting LockEntry, and Source is what conflict/expand
	// identity is matched against (see ADR-0003) — the canonical
	// "owner/repo" form for GitHub sources, regardless of whether the
	// user typed shorthand or a full URL, so both surface syntaxes for
	// the same repository are recognized as the same source.
	Source, SourceType, SourceURL, Ref string
}

// resolveSource classifies opts.Source and turns it into a resolvedSource:
// a GitHub source is fetched (via opts.Fetcher, defaulting to
// GitHubFetcher) at its default branch; a local source is validated as an
// existing directory. Local-path sources bypass the Fetcher entirely.
func resolveSource(opts AddOptions) (resolvedSource, error) {
	parsed := parseSource(opts.Source)
	if parsed.Kind == sourceKindGitHub {
		return resolveGitHubSource(opts, parsed.GitHub)
	}
	return resolveLocalSource(opts)
}

// resolveGitHubSource fetches src at its default branch via opts.Fetcher
// (defaulting to GitHubFetcher) and describes the result as a
// resolvedSource.
func resolveGitHubSource(opts AddOptions, src GitHubSource) (resolvedSource, error) {
	fetcher := opts.Fetcher
	if fetcher == nil {
		fetcher = &GitHubFetcher{}
	}

	dir, err := fetcher.Fetch(src, "")
	if err != nil {
		return resolvedSource{}, fmt.Errorf("fetching %s: %w", src, err)
	}

	return resolvedSource{
		Dir:           dir,
		Cleanup:       func() { _ = os.RemoveAll(dir) },
		SuggestedName: src.Repo,
		Source:        src.String(),
		SourceType:    sourceTypeGitHub,
		SourceURL:     src.URL(),
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
			"unsupported source %q: not a local directory, and not a recognized GitHub source (expected \"owner/repo\" shorthand or a full https://github.com/owner/repo URL)",
			opts.Source,
		)
	}

	return resolvedSource{
		Dir:        sourceAbs,
		Source:     opts.Source,
		SourceType: sourceTypeLocal,
	}, nil
}
