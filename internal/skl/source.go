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
	// tree-path source, e.g. "main", or in a shorthand/full-URL GitHub
	// source's @ref pin (see #11 and parsePinnedSource). Empty when no ref
	// was named, in which case the source is fetched at its default
	// branch (see resolveGitHubSource). Always empty for a local source.
	Ref string
	// Path is the repo-relative path encoded in a GitHub tree-path
	// source, e.g. "skills/tdd", or in a shorthand/full-URL GitHub
	// source's @ref/<path> pin: forward-slash form, with no leading,
	// trailing, "." or ".." segments (see validateTreePath). It names the
	// directory within the fetched repository that *is* the skill,
	// authoritatively — resolveGitHubSource points resolvedSource.Dir at
	// it directly, so no discovery walk is needed. Empty for a
	// bare/unpinned GitHub source and for a local source.
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
// no filesystem or network access happens here. A raw source containing
// an "@" is routed to parsePinnedSource: an @ref (or @ref/<path>) pin on
// a shorthand/full-URL GitHub source (see #11), or a rejection when that
// pin is combined with a tree-path source (which already encodes a ref)
// or a local-path source (which isn't fetched from a ref at all).
func parseSource(raw string) (parsedSource, error) {
	if base, pin, cut := strings.Cut(raw, "@"); cut {
		return parsePinnedSource(raw, base, pin)
	}

	if m := githubTreePathURLPattern.FindStringSubmatch(raw); m != nil {
		return newGitHubTreePathSource(m[1], m[2], m[3], m[4])
	}
	if m := githubTreePathPattern.FindStringSubmatch(raw); m != nil {
		return newGitHubTreePathSource(m[1], m[2], m[3], m[4])
	}
	if owner, repo, ok := githubOwnerRepo(raw); ok {
		return parsedSource{Kind: sourceKindGitHub, GitHub: GitHubSource{Owner: owner, Repo: repo}}, nil
	}
	return parsedSource{Kind: sourceKindLocal}, nil
}

// isGitHubTreePath reports whether raw matches the GitHub tree-path
// grammar, in either shorthand or full-URL form.
func isGitHubTreePath(raw string) bool {
	return githubTreePathPattern.MatchString(raw) || githubTreePathURLPattern.MatchString(raw)
}

// parsePinnedSource parses raw's "@" pin (base is everything before the
// first "@", pin everything after) into a parsedSource. Grammar note:
// everything up to the first "/" in pin is the ref, and everything after
// that (if any) is the repo-relative path — the same deterministic,
// no-ambiguity-resolution grammar the tree-path source uses (see
// githubTreePathPattern's comment), and for the same reason: without
// querying GitHub's API we have no way to know which branches exist, so a
// ref containing "/" (e.g. "feature/foo") isn't reachable through this
// shorthand. That's a documented limitation, not a bug.
//
// base must resolve to a shorthand or full-URL GitHub source (a
// shorthand/full-URL tree-path source is rejected outright, since it
// already encodes a ref; a local path is rejected too, since a local
// source isn't fetched from a ref at all).
func parsePinnedSource(raw, base, pin string) (parsedSource, error) {
	if isGitHubTreePath(base) {
		return parsedSource{}, fmt.Errorf(
			"source %q combines a GitHub tree-path URL with an @%s ref pin: the tree-path already encodes a ref; remove the \"@%s\" pin",
			raw, pin, pin,
		)
	}

	owner, repo, ok := githubOwnerRepo(base)
	if !ok {
		return parsedSource{}, fmt.Errorf(
			"source %q combines a local-path source with an @%s ref pin: local sources aren't fetched from a ref, so pinning doesn't apply; remove the \"@%s\" pin",
			raw, pin, pin,
		)
	}
	if pin == "" {
		return parsedSource{}, fmt.Errorf("source %q has an empty @ ref pin: name a branch, tag, or commit SHA after \"@\"", raw)
	}

	ref, rawPath, hasPath := strings.Cut(pin, "/")
	if !hasPath {
		return parsedSource{Kind: sourceKindGitHub, GitHub: GitHubSource{Owner: owner, Repo: repo}, Ref: ref}, nil
	}

	path, err := validateTreePath(rawPath)
	if err != nil {
		return parsedSource{}, err
	}
	return parsedSource{Kind: sourceKindGitHub, GitHub: GitHubSource{Owner: owner, Repo: repo}, Ref: ref, Path: path}, nil
}

// githubOwnerRepo reports whether base is a shorthand ("owner/repo") or
// full-URL ("https://github.com/owner/repo") GitHub source — the two
// forms an @ref pin (see parsePinnedSource) can attach to — and if so,
// returns the owner and repo it names.
func githubOwnerRepo(base string) (owner, repo string, ok bool) {
	if m := githubURLPattern.FindStringSubmatch(base); m != nil {
		return m[1], m[2], true
	}
	if githubShorthandPattern.MatchString(base) {
		owner, repo, ok = strings.Cut(base, "/")
		return owner, repo, ok
	}
	return "", "", false
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
