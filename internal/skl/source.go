package skl

import (
	"fmt"
	"regexp"
	"strings"
)

// sourceKind distinguishes how parseSource classified a raw source string.
type sourceKind int

const (
	sourceKindLocal sourceKind = iota
	sourceKindGitHub
)

// GitHubSource identifies a single GitHub repository to fetch, parsed from
// either shorthand (owner/repo) or full URL source syntax. It carries no
// ref/path information of its own: a tree-path source's ref and path live
// on parsedSource instead (see parsedSource.Ref and parsedSource.Path),
// since Fetcher.Fetch already takes ref as a separate argument and a
// repo-relative path isn't part of "which repository" identity.
type GitHubSource struct {
	Owner string
	Repo  string
}

// String returns src's canonical shorthand form, e.g. "owner/repo". This
// is what's recorded as a LockEntry's Source and compared for
// expand/conflict identity, so a GitHub source typed as a full URL and the
// same source typed as shorthand are recognized as the same source.
func (src GitHubSource) String() string {
	return src.Owner + "/" + src.Repo
}

// URL returns src's canonical full GitHub URL, recorded in a LockEntry's
// SourceURL field regardless of which surface syntax the user typed.
func (src GitHubSource) URL() string {
	return "https://github.com/" + src.Owner + "/" + src.Repo
}

// parsedSource is parseSource's output: either a GitHub source (shorthand,
// full URL, or tree-path) or a local filesystem path passed through
// unchanged for the existing stat-based handling.
type parsedSource struct {
	Kind   sourceKind
	GitHub GitHubSource
	// Ref is the ref (branch, tag, or commit SHA) encoded in a GitHub
	// tree-path source, e.g. "main". Empty for a shorthand/full-URL
	// GitHub source (fetched at its default branch, see resolveGitHubSource)
	// and for a local source.
	Ref string
	// Path is the repo-relative path encoded in a GitHub tree-path
	// source, e.g. "skills/tdd": forward-slash form, with no leading,
	// trailing, "." or ".." segments (see validateTreePath). It names the
	// directory within the fetched repository that *is* the skill,
	// authoritatively — resolveGitHubSource points resolvedSource.Dir at
	// it directly, so no discovery walk is needed. Empty for a
	// shorthand/full-URL GitHub source and for a local source.
	Path string
}

// githubOwnerPattern and githubRepoPattern approximate GitHub's own
// username/repo naming rules closely enough to distinguish real GitHub
// identifiers from local paths, without being a strict validator: owner
// names are alphanumeric with interior hyphens, repo names are
// alphanumeric plus '.', '-', '_'.
const (
	githubOwnerPart = `[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?`
	githubRepoPart  = `[A-Za-z0-9._-]+`
)

var (
	// githubShorthandPattern matches "owner/repo" exactly: two path
	// segments. A local relative path with more segments (this repo's own
	// test fixtures all use three, e.g. "testdata/fixtures/simple-skill")
	// or fewer falls through to local-path handling instead.
	githubShorthandPattern = regexp.MustCompile(`^` + githubOwnerPart + `/` + githubRepoPart + `$`)

	// githubURLPattern matches a full GitHub repository URL:
	// http(s)://github.com/owner/repo, with an optional trailing slash.
	githubURLPattern = regexp.MustCompile(`^https?://github\.com/(` + githubOwnerPart + `)/(` + githubRepoPart + `)/?$`)

	// githubTreePathPattern matches a GitHub tree-path source in
	// shorthand form: owner/repo/tree/<ref>/<path>. <ref> is everything
	// up to the next "/" and <path> is everything after it — the same
	// deterministic, no-ambiguity-resolution grammar #11's "@ref/<path>"
	// shorthand uses, and for the same reason: without querying GitHub's
	// API we have no way to know which branches exist, so a ref
	// containing "/" (e.g. "feature/foo") isn't reachable through this
	// syntax. That's a documented limitation, not a bug.
	githubTreePathPattern = regexp.MustCompile(`^(` + githubOwnerPart + `)/(` + githubRepoPart + `)/tree/([^/]+)/(.+)$`)

	// githubTreePathURLPattern matches the same tree-path grammar as
	// githubTreePathPattern, in full-URL form:
	// http(s)://github.com/owner/repo/tree/<ref>/<path>.
	githubTreePathURLPattern = regexp.MustCompile(`^https?://github\.com/(` + githubOwnerPart + `)/(` + githubRepoPart + `)/tree/([^/]+)/(.+)$`)
)

// parseSource classifies raw as either a GitHub source (shorthand, full
// URL, or tree-path) or a local filesystem path, based on syntax alone —
// no filesystem or network access happens here. An @ref pin (see #11) is
// out of this ticket's scope for non-tree-path sources: a source using
// that syntax doesn't match any GitHub pattern below and falls through to
// local-path handling, whose existing stat-based error makes clear the
// string wasn't found as a directory either. Combining a tree-path source
// with an @ref pin is rejected outright, since the tree-path already
// encodes a ref.
func parseSource(raw string) (parsedSource, error) {
	if base, pin, cut := strings.Cut(raw, "@"); cut && isGitHubTreePath(base) {
		return parsedSource{}, fmt.Errorf(
			"source %q combines a GitHub tree-path URL with an @%s ref pin: the tree-path already encodes a ref; remove the \"@%s\" pin",
			raw, pin, pin,
		)
	}

	if m := githubTreePathURLPattern.FindStringSubmatch(raw); m != nil {
		return newGitHubTreePathSource(m[1], m[2], m[3], m[4])
	}
	if m := githubTreePathPattern.FindStringSubmatch(raw); m != nil {
		return newGitHubTreePathSource(m[1], m[2], m[3], m[4])
	}
	if m := githubURLPattern.FindStringSubmatch(raw); m != nil {
		return parsedSource{Kind: sourceKindGitHub, GitHub: GitHubSource{Owner: m[1], Repo: m[2]}}, nil
	}
	if githubShorthandPattern.MatchString(raw) {
		owner, repo, ok := strings.Cut(raw, "/")
		if ok {
			return parsedSource{Kind: sourceKindGitHub, GitHub: GitHubSource{Owner: owner, Repo: repo}}, nil
		}
	}
	return parsedSource{Kind: sourceKindLocal}, nil
}

// isGitHubTreePath reports whether raw matches the GitHub tree-path
// grammar, in either shorthand or full-URL form.
func isGitHubTreePath(raw string) bool {
	return githubTreePathPattern.MatchString(raw) || githubTreePathURLPattern.MatchString(raw)
}

// newGitHubTreePathSource builds the parsedSource for a matched tree-path
// source, validating rawPath along the way.
func newGitHubTreePathSource(owner, repo, ref, rawPath string) (parsedSource, error) {
	path, err := validateTreePath(rawPath)
	if err != nil {
		return parsedSource{}, err
	}
	return parsedSource{
		Kind:   sourceKindGitHub,
		GitHub: GitHubSource{Owner: owner, Repo: repo},
		Ref:    ref,
		Path:   path,
	}, nil
}

// validateTreePath cleans and validates a tree-path source's raw path
// component: a trailing slash (from a URL typed with one) is trimmed, and
// each "/"-separated segment is checked to rule out an empty segment (a
// stray "//"), "." or ".." (which would otherwise let a crafted source
// path escape the fetched repository directory once joined onto it, see
// resolveGitHubSource). The returned path is forward-slash form, suitable
// for both display and filepath.FromSlash.
func validateTreePath(rawPath string) (string, error) {
	path := strings.TrimSuffix(rawPath, "/")
	for _, seg := range strings.Split(path, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", fmt.Errorf("invalid tree-path source path %q", rawPath)
		}
	}
	return path, nil
}
