package skl

import (
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
// ref/path information: this ticket's scope is limited to a source's
// default branch and its root SKILL.md (see #10 and #11 for tree-path and
// #ref support).
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

// parsedSource is parseSource's output: either a GitHub source (shorthand
// or full URL) or a local filesystem path passed through unchanged for
// the existing stat-based handling.
type parsedSource struct {
	Kind   sourceKind
	GitHub GitHubSource
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
	// or fewer falls through to local-path handling instead. A GitHub
	// tree-path source (owner/repo/tree/...) has more segments too, and is
	// out of this ticket's scope (see #10).
	githubShorthandPattern = regexp.MustCompile(`^` + githubOwnerPart + `/` + githubRepoPart + `$`)

	// githubURLPattern matches a full GitHub repository URL:
	// http(s)://github.com/owner/repo, with an optional trailing slash.
	githubURLPattern = regexp.MustCompile(`^https?://github\.com/(` + githubOwnerPart + `)/(` + githubRepoPart + `)/?$`)
)

// parseSource classifies raw as either a GitHub source (shorthand or full
// URL) or a local filesystem path, based on syntax alone — no filesystem
// or network access happens here. GitHub tree-path URLs and #ref fragments
// are out of this ticket's scope (see #10, #11); a source using either
// syntax doesn't match either GitHub pattern below and falls through to
// local-path handling, whose existing stat-based error makes clear the
// string wasn't found as a directory either.
func parseSource(raw string) parsedSource {
	if m := githubURLPattern.FindStringSubmatch(raw); m != nil {
		return parsedSource{Kind: sourceKindGitHub, GitHub: GitHubSource{Owner: m[1], Repo: m[2]}}
	}
	if githubShorthandPattern.MatchString(raw) {
		owner, repo, ok := strings.Cut(raw, "/")
		if ok {
			return parsedSource{Kind: sourceKindGitHub, GitHub: GitHubSource{Owner: owner, Repo: repo}}
		}
	}
	return parsedSource{Kind: sourceKindLocal}
}
