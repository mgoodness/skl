package skl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSource_LocalPath_Nonexistent_Errors(t *testing.T) {
	_, err := resolveSource(AddOptions{Source: filepath.Join(t.TempDir(), "does-not-exist")})
	if err == nil {
		t.Fatal("resolveSource() error = nil, want error for a nonexistent local path")
	}
}

func TestResolveSource_LocalPath_File_Errors(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("writing fixture file: %v", err)
	}

	_, err := resolveSource(AddOptions{Source: file})
	if err == nil {
		t.Fatal("resolveSource() error = nil, want error for a source path that's a file, not a directory")
	}
}

func TestResolveSource_LocalPath_ExistingDirectory_ResolvesWithLocalIdentity(t *testing.T) {
	dir := t.TempDir()

	rs, err := resolveSource(AddOptions{Source: dir})
	if err != nil {
		t.Fatalf("resolveSource() error = %v", err)
	}
	if rs.SourceType != sourceTypeLocal {
		t.Errorf("rs.SourceType = %q, want %q", rs.SourceType, sourceTypeLocal)
	}
	if rs.Source != dir {
		t.Errorf("rs.Source = %q, want %q", rs.Source, dir)
	}
	if rs.SuggestedName != "" {
		t.Errorf("rs.SuggestedName = %q, want empty for a local source", rs.SuggestedName)
	}
	if rs.Cleanup != nil {
		t.Error("rs.Cleanup is non-nil, want nil for a local source")
	}
}

func TestResolveSource_GitHubSource_UsesInjectedFetcher(t *testing.T) {
	fetchedDir := t.TempDir()
	fetcher := &recordingOnlyFetcher{dir: fetchedDir}

	rs, err := resolveSource(AddOptions{Source: "owner/repo", Fetcher: fetcher})
	if err != nil {
		t.Fatalf("resolveSource() error = %v", err)
	}
	if rs.Dir != fetchedDir {
		t.Errorf("rs.Dir = %q, want the fetcher's returned directory %q", rs.Dir, fetchedDir)
	}
	if rs.SourceType != sourceTypeGitHub {
		t.Errorf("rs.SourceType = %q, want %q", rs.SourceType, sourceTypeGitHub)
	}
	if rs.Source != "owner/repo" {
		t.Errorf("rs.Source = %q, want canonical %q", rs.Source, "owner/repo")
	}
	if rs.SourceURL != "https://github.com/owner/repo" {
		t.Errorf("rs.SourceURL = %q, want %q", rs.SourceURL, "https://github.com/owner/repo")
	}
	if rs.SuggestedName != "repo" {
		t.Errorf("rs.SuggestedName = %q, want %q", rs.SuggestedName, "repo")
	}
	if rs.Cleanup == nil {
		t.Error("rs.Cleanup is nil, want non-nil for a GitHub source")
	}
}

// recordingOnlyFetcher is a minimal in-package Fetcher fake (distinct from
// add_github_test.go's fakeFetcher, which lives in package skl_test and
// isn't reachable from these white-box resolveSource tests).
type recordingOnlyFetcher struct {
	dir string
}

func (f *recordingOnlyFetcher) Fetch(GitHubSource, string) (string, error) {
	return f.dir, nil
}
