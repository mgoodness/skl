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
// path information of its own: a tree-path source's path lives on
// parsedSource instead (see parsedSource.Path), since a repo-relative path
// isn't part of "which repository" identity.
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
//
// v1 has no ref-pinning concept at all (see #11, deferred to a future
// version): every GitHub source, tree-path included, is always fetched at
// the repository's default branch. A tree-path source therefore encodes
// only a path, not a ref — unlike a real GitHub tree URL, which always
// includes a branch/tag/commit segment before the path. Pasting a literal
// GitHub tree URL (e.g. copied from a browser) needs that ref segment
// stripped by hand before it's a valid skl tree-path source; that's a
// documented v1 limitation, lifted once #11 lands.
type parsedSource struct {
	Kind   sourceKind
	GitHub GitHubSource
	// Path is the repo-relative path encoded in a GitHub tree-path
	// source, e.g. "skills/tdd": forward-slash form, with no leading,
	// trailing, "." or ".." segments (see validateTreePath). It names the
	// directory within the fetched repository (always the default
	// branch, see parsedSource's own doc comment) that *is* the skill,
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
	// shorthand form: owner/repo/tree/<path>. There is no <ref> segment
	// (see parsedSource's doc comment): everything after "tree/" is the
	// path, installed from the repository's default branch.
	githubTreePathPattern = regexp.MustCompile(`^(` + githubOwnerPart + `)/(` + githubRepoPart + `)/tree/(.+)$`)

	// githubTreePathURLPattern matches the same tree-path grammar as
	// githubTreePathPattern, in full-URL form:
	// http(s)://github.com/owner/repo/tree/<path>.
	githubTreePathURLPattern = regexp.MustCompile(`^https?://github\.com/(` + githubOwnerPart + `)/(` + githubRepoPart + `)/tree/(.+)$`)
)

// parseSource classifies raw as either a GitHub source (shorthand, full
// URL, or tree-path) or a local filesystem path, based on syntax alone —
// no filesystem or network access happens here. Ref pinning of any kind
// (a #ref fragment, or a ref segment within a tree-path) is out of scope
// for v1 (see #11): a source using either syntax doesn't match any GitHub
// pattern below and falls through to local-path handling, whose existing
// stat-based error makes clear the string wasn't found as a directory
// either.
func parseSource(raw string) (parsedSource, error) {
	if m := githubTreePathURLPattern.FindStringSubmatch(raw); m != nil {
		return newGitHubTreePathSource(m[1], m[2], m[3])
	}
	if m := githubTreePathPattern.FindStringSubmatch(raw); m != nil {
		return newGitHubTreePathSource(m[1], m[2], m[3])
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

// newGitHubTreePathSource builds the parsedSource for a matched tree-path
// source, validating rawPath along the way.
func newGitHubTreePathSource(owner, repo, rawPath string) (parsedSource, error) {
	path, err := validateTreePath(rawPath)
	if err != nil {
		return parsedSource{}, err
	}
	return parsedSource{
		Kind:   sourceKindGitHub,
		GitHub: GitHubSource{Owner: owner, Repo: repo},
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
