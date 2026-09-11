package skl

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

// fixtureRootFetcher is another in-package Fetcher fake: it returns a
// fresh copy of a fixture directory representing a whole repository's
// contents, so tree-path resolution can be exercised against a directory
// shaped like a real fetched repo (nested skill directories included) and
// still assert Cleanup removes the *entire* fetched root, not just the
// tree-path's subdirectory.
type fixtureRootFetcher struct {
	fixtureDir string
	gotRef     string
}

func (f *fixtureRootFetcher) Fetch(_ GitHubSource, ref string) (string, error) {
	f.gotRef = ref
	dir, err := os.MkdirTemp("", "skl-fixture-root-*")
	if err != nil {
		return "", err
	}
	if err := filepath.WalkDir(f.fixtureDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(f.fixtureDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}); err != nil {
		return "", err
	}
	return dir, nil
}

func TestResolveSource_GitHubTreePath_PointsDirAtSkillSubdirectory(t *testing.T) {
	fetcher := &fixtureRootFetcher{fixtureDir: "testdata/fixtures/nested-repo"}

	rs, err := resolveSource(AddOptions{Source: "owner/nested-repo/tree/main/skills/tdd", Fetcher: fetcher})
	if err != nil {
		t.Fatalf("resolveSource() error = %v", err)
	}
	defer rs.Cleanup()

	if fetcher.gotRef != "main" {
		t.Errorf("fetcher was called with ref %q, want %q", fetcher.gotRef, "main")
	}
	if _, err := os.Stat(filepath.Join(rs.Dir, "SKILL.md")); err != nil {
		t.Errorf("rs.Dir = %q does not contain SKILL.md directly, want it pointed at the skill subdirectory: %v", rs.Dir, err)
	}
	if rs.SourceType != sourceTypeGitHub {
		t.Errorf("rs.SourceType = %q, want %q", rs.SourceType, sourceTypeGitHub)
	}
	if rs.Source != "owner/nested-repo/tree/main/skills/tdd" {
		t.Errorf("rs.Source = %q, want canonical %q", rs.Source, "owner/nested-repo/tree/main/skills/tdd")
	}
	if rs.SourceURL != "https://github.com/owner/nested-repo/tree/main/skills/tdd" {
		t.Errorf("rs.SourceURL = %q, want %q", rs.SourceURL, "https://github.com/owner/nested-repo/tree/main/skills/tdd")
	}
	if rs.Ref != "main" {
		t.Errorf("rs.Ref = %q, want %q", rs.Ref, "main")
	}
	if rs.SkillPath != "skills/tdd" {
		t.Errorf("rs.SkillPath = %q, want %q", rs.SkillPath, "skills/tdd")
	}
	if rs.SuggestedName != "" {
		t.Errorf("rs.SuggestedName = %q, want empty (basename of rs.Dir is already meaningful)", rs.SuggestedName)
	}
	if filepath.Base(rs.Dir) != "tdd" {
		t.Errorf("filepath.Base(rs.Dir) = %q, want %q", filepath.Base(rs.Dir), "tdd")
	}
}

func TestResolveSource_GitHubTreePath_CleanupRemovesEntireFetchedRoot(t *testing.T) {
	fetcher := &fixtureRootFetcher{fixtureDir: "testdata/fixtures/nested-repo"}

	rs, err := resolveSource(AddOptions{Source: "owner/nested-repo/tree/main/skills/tdd", Fetcher: fetcher})
	if err != nil {
		t.Fatalf("resolveSource() error = %v", err)
	}

	// rs.Dir is a subdirectory of the fetched root; capture the root two
	// levels up (skills/tdd -> skills -> root) to assert Cleanup removes
	// the whole fetched tree, not just the subdirectory rs.Dir points at.
	fetchedRoot := filepath.Dir(filepath.Dir(rs.Dir))

	rs.Cleanup()

	if _, err := os.Stat(fetchedRoot); !os.IsNotExist(err) {
		t.Errorf("expected fetched root %s to be removed by Cleanup, stat err = %v", fetchedRoot, err)
	}
}

func TestResolveSource_GitHubTreePath_PathNotFound_ErrorsClearly(t *testing.T) {
	fetcher := &fixtureRootFetcher{fixtureDir: "testdata/fixtures/nested-repo"}

	_, err := resolveSource(AddOptions{Source: "owner/nested-repo/tree/main/skills/does-not-exist", Fetcher: fetcher})
	if err == nil {
		t.Fatal("resolveSource() error = nil, want error for a tree-path naming a path that doesn't exist in the fetched repository")
	}
	if !strings.Contains(err.Error(), "skills/does-not-exist") {
		t.Errorf("error %q does not name the missing path", err.Error())
	}
}

func TestResolveSource_GitHubTreePath_PathExistsButNotASkill_ErrorsFromDiscovery(t *testing.T) {
	fetcher := &fixtureRootFetcher{fixtureDir: "testdata/fixtures/nested-repo"}

	rs, err := resolveSource(AddOptions{Source: "owner/nested-repo/tree/main/docs", Fetcher: fetcher})
	if err != nil {
		t.Fatalf("resolveSource() error = %v, want resolveSource to succeed (the path exists) and let discoverSkillDir report the missing SKILL.md", err)
	}
	defer rs.Cleanup()

	if _, err := discoverSkillDir(rs.Dir); err == nil {
		t.Error("discoverSkillDir() error = nil, want error: docs/ has no SKILL.md")
	}
}
