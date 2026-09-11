//go:build integration

// Package skl's github_fetcher_integration_test.go exercises GitHubFetcher
// against real network access to catch upstream API/archive-format drift.
// It's excluded from the default `go test ./...` run by the "integration"
// build tag; run it explicitly with `go test -tags=integration ./...`.
package skl

import (
	"os"
	"path/filepath"
	"testing"
)

// octocat/Hello-World is GitHub's own long-standing example repository
// (used in GitHub's own API documentation), making it about as stable a
// real-network fixture as exists: small, public, and extremely unlikely to
// be renamed, deleted, or made private.
var helloWorldSource = GitHubSource{Owner: "octocat", Repo: "Hello-World"}

func TestGitHubFetcher_Fetch_RealRepository(t *testing.T) {
	f := &GitHubFetcher{}

	dir, err := f.Fetch(helloWorldSource, "")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("Fetch() returned %q, want an existing directory (err = %v)", dir, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading fetched directory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("fetched directory %s is empty, want the repository's contents", dir)
	}

	// The tarball's top-level "octocat-Hello-World-<sha>/" wrapper
	// directory should have been stripped, leaving dir itself as the
	// repository root — so a file every version of this repo has always
	// had should sit directly under dir, not one level down.
	if _, err := os.Stat(filepath.Join(dir, "README")); err != nil {
		t.Errorf("expected %s/README to exist (wrapper directory not stripped?): %v", dir, err)
	}
}

func TestGitHubFetcher_Fetch_NonexistentRepository_ReturnsClearError(t *testing.T) {
	f := &GitHubFetcher{}

	_, err := f.Fetch(GitHubSource{Owner: "mgoodness", Repo: "this-repo-should-never-exist-skl-integration-test"}, "")
	if err == nil {
		t.Fatalf("Fetch() error = nil, want an error for a nonexistent repository")
	}
}
