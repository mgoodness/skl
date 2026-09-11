package skl_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/mgoodness/skl/internal/skl"
)

// fakeFetcher is a test-only skl.Fetcher: it never touches the network,
// instead copying a registered fixture directory into a fresh temp
// directory it owns, so tests can assert that Add cleans it up afterward.
type fakeFetcher struct {
	// fixtures maps a GitHub source's canonical "owner/repo" form to the
	// local fixture directory Fetch should return a copy of.
	fixtures map[string]string
	// err, when non-nil, is returned by Fetch for every call, simulating a
	// nonexistent/inaccessible repository or any other fetch failure.
	err error

	calls []fakeFetchCall
}

type fakeFetchCall struct {
	Source skl.GitHubSource
	Ref    string
}

func (f *fakeFetcher) Fetch(src skl.GitHubSource, ref string) (string, error) {
	f.calls = append(f.calls, fakeFetchCall{Source: src, Ref: ref})

	if f.err != nil {
		return "", f.err
	}

	key := src.String()
	fixtureDir, ok := f.fixtures[key]
	if !ok {
		return "", fmt.Errorf("fakeFetcher: no fixture registered for %q", key)
	}

	dir, err := os.MkdirTemp("", "skl-fake-fetch-*")
	if err != nil {
		return "", err
	}
	if err := copyTree(fixtureDir, dir); err != nil {
		return "", err
	}
	return dir, nil
}

// copyTree recursively copies src's contents into dst, both of which must
// already exist as directories (dst is created by os.MkdirTemp; src is a
// testdata fixture). Used only to give each fakeFetcher.Fetch call its own
// disposable directory, mirroring what a real Fetcher hands back.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func TestAdd_GitHubShorthand_FetchesAndInstalls(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	fetcher := &fakeFetcher{fixtures: map[string]string{
		"owner/simple-skill": "testdata/fixtures/simple-skill",
	}}

	result, err := skl.Add(skl.AddOptions{
		Source:            "owner/simple-skill",
		Fetcher:           fetcher,
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if result.Name != "simple-skill" {
		t.Errorf("result.Name = %q, want %q", result.Name, "simple-skill")
	}

	installed := filepath.Join(projectRoot, ".agents", "skills", "simple-skill", "SKILL.md")
	data, err := os.ReadFile(installed)
	if err != nil {
		t.Fatalf("reading installed SKILL.md: %v", err)
	}
	want, err := os.ReadFile("testdata/fixtures/simple-skill/SKILL.md")
	if err != nil {
		t.Fatalf("reading fixture SKILL.md: %v", err)
	}
	if string(data) != string(want) {
		t.Errorf("installed SKILL.md content does not match fixture")
	}

	if len(fetcher.calls) != 1 {
		t.Fatalf("fetcher.calls = %v, want exactly 1 call", fetcher.calls)
	}
	call := fetcher.calls[0]
	if call.Source.Owner != "owner" || call.Source.Repo != "simple-skill" {
		t.Errorf("fetcher called with %+v, want Owner=%q Repo=%q", call.Source, "owner", "simple-skill")
	}
	if call.Ref != "" {
		t.Errorf("fetcher called with ref %q, want empty (default branch) — #ref pinning is out of this ticket's scope", call.Ref)
	}
}

func TestAdd_GitHubFullURL_BehavesIdenticallyToShorthand(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	fetcher := &fakeFetcher{fixtures: map[string]string{
		"owner/simple-skill": "testdata/fixtures/simple-skill",
	}}

	result, err := skl.Add(skl.AddOptions{
		Source:            "https://github.com/owner/simple-skill",
		Fetcher:           fetcher,
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if result.Name != "simple-skill" {
		t.Errorf("result.Name = %q, want %q", result.Name, "simple-skill")
	}

	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	entry, ok := lf["simple-skill"]
	if !ok {
		t.Fatalf("lockfile missing entry for simple-skill")
	}
	if entry.Source != "owner/simple-skill" {
		t.Errorf("entry.Source = %q, want canonical shorthand %q even though a full URL was given", entry.Source, "owner/simple-skill")
	}
	if entry.SourceURL != "https://github.com/owner/simple-skill" {
		t.Errorf("entry.SourceURL = %q, want %q", entry.SourceURL, "https://github.com/owner/simple-skill")
	}
	if entry.SourceType != "github" {
		t.Errorf("entry.SourceType = %q, want %q", entry.SourceType, "github")
	}
}

func TestAdd_GitHubSource_ShorthandAndFullURL_ExpandSameEntry(t *testing.T) {
	fakeHome(t, "claude-code")
	projectRoot := t.TempDir()

	fetcher := &fakeFetcher{fixtures: map[string]string{
		"owner/simple-skill": "testdata/fixtures/simple-skill",
	}}

	if _, err := skl.Add(skl.AddOptions{
		Source:            "owner/simple-skill",
		Fetcher:           fetcher,
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"universal"},
	}); err != nil {
		t.Fatalf("first Add() (shorthand) error = %v", err)
	}

	if _, err := skl.Add(skl.AddOptions{
		Source:            "https://github.com/owner/simple-skill",
		Fetcher:           fetcher,
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"claude-code"},
	}); err != nil {
		t.Fatalf("second Add() (full URL) error = %v", err)
	}

	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	if len(lf) != 1 {
		t.Fatalf("len(lockfile) = %d, want 1 (expanded, not duplicated because shorthand and full URL are the same source), got %v", len(lf), lf)
	}
	entry := lf["simple-skill"]
	for _, want := range []string{"universal", "claude-code"} {
		if _, ok := entry.Adapters[want]; !ok {
			t.Errorf("entry.Adapters missing %q after expand, got %v", want, entry.Adapters)
		}
	}
}

func TestAdd_GitHubSource_RootSkillWithResource_CopiesNestedFiles(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	fetcher := &fakeFetcher{fixtures: map[string]string{
		"owner/skill-with-resource": "testdata/fixtures/skill-with-resource",
	}}

	result, err := skl.Add(skl.AddOptions{
		Source:            "owner/skill-with-resource",
		Fetcher:           fetcher,
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	installed := filepath.Join(projectRoot, filepath.FromSlash(result.Adapters["universal"].Path), "resources", "notes.md")
	data, err := os.ReadFile(installed)
	if err != nil {
		t.Fatalf("reading installed nested resource: %v", err)
	}
	want, err := os.ReadFile("testdata/fixtures/skill-with-resource/resources/notes.md")
	if err != nil {
		t.Fatalf("reading fixture resource: %v", err)
	}
	if string(data) != string(want) {
		t.Errorf("installed resource content does not match fixture")
	}
}

func TestAdd_GitHubSource_FetchedTempDirIsCleanedUpAfterInstall(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	var fetchedDir string
	fetcher := &recordingFetcher{fakeFetcher: fakeFetcher{fixtures: map[string]string{
		"owner/simple-skill": "testdata/fixtures/simple-skill",
	}}, gotDir: &fetchedDir}

	if _, err := skl.Add(skl.AddOptions{
		Source:            "owner/simple-skill",
		Fetcher:           fetcher,
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if fetchedDir == "" {
		t.Fatalf("fetcher never recorded a fetched directory")
	}
	if _, err := os.Stat(fetchedDir); !os.IsNotExist(err) {
		t.Errorf("expected fetched temp directory %s to be removed after Add, stat err = %v", fetchedDir, err)
	}
}

// recordingFetcher wraps fakeFetcher to capture the directory it returned,
// so a test can assert Add removed it afterward.
type recordingFetcher struct {
	fakeFetcher
	gotDir *string
}

func (f *recordingFetcher) Fetch(src skl.GitHubSource, ref string) (string, error) {
	dir, err := f.fakeFetcher.Fetch(src, ref)
	if err == nil {
		*f.gotDir = dir
	}
	return dir, err
}

func TestAdd_GitHubSource_FetcherError_IsPropagatedClearly(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	fetcher := &fakeFetcher{err: fmt.Errorf("owner/does-not-exist not found: it may not exist, or it may be a private repository you don't have access to")}

	_, err := skl.Add(skl.AddOptions{
		Source:      "owner/does-not-exist",
		Fetcher:     fetcher,
		ProjectRoot: projectRoot,
	})
	if err == nil {
		t.Fatalf("Add() error = nil, want error for a fetch failure")
	}
	if !contains(err.Error(), "owner/does-not-exist") {
		t.Errorf("error %q does not identify the source that failed to fetch", err.Error())
	}
}

func TestAdd_GitHubSource_NoRootSkillMD_ErrorNamesTheSource(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	fetcher := &fakeFetcher{fixtures: map[string]string{
		"owner/not-a-skill": "testdata/fixtures/not-a-skill",
	}}

	_, err := skl.Add(skl.AddOptions{
		Source:      "owner/not-a-skill",
		Fetcher:     fetcher,
		ProjectRoot: projectRoot,
	})
	if err == nil {
		t.Fatalf("Add() error = nil, want error for a source with no root SKILL.md")
	}
	if !contains(err.Error(), "owner/not-a-skill") {
		t.Errorf("error %q does not name the source %q, want it to (a fetched temp dir's path alone isn't actionable)", err.Error(), "owner/not-a-skill")
	}
}

func TestAdd_MalformedGitHubLikeSource_FallsBackToLocalPathErrorClearly(t *testing.T) {
	projectRoot := t.TempDir()

	// Three path segments: not a valid "owner/repo" shorthand (that
	// pattern requires exactly two), not a tree-path (missing the "tree"
	// literal segment), and not an existing local directory either.
	_, err := skl.Add(skl.AddOptions{
		Source:      "owner/repo/extra-path-segment",
		ProjectRoot: projectRoot,
	})
	if err == nil {
		t.Fatalf("Add() error = nil, want a clear error for an unsupported source")
	}
	if !contains(err.Error(), "owner/repo/extra-path-segment") {
		t.Errorf("error %q does not identify the offending source", err.Error())
	}
}

func TestAdd_GitHubTreePath_Shorthand_FetchesDefaultBranchAndInstallsOnlyThePath(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	fetcher := &fakeFetcher{fixtures: map[string]string{
		"owner/nested-repo": "testdata/fixtures/nested-repo",
	}}

	result, err := skl.Add(skl.AddOptions{
		Source:            "owner/nested-repo/tree/skills/tdd",
		Fetcher:           fetcher,
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if result.Name != "tdd" {
		t.Errorf("result.Name = %q, want %q (the tree-path's own final segment, not the repo name)", result.Name, "tdd")
	}

	installed := filepath.Join(projectRoot, ".agents", "skills", "tdd", "SKILL.md")
	data, err := os.ReadFile(installed)
	if err != nil {
		t.Fatalf("reading installed SKILL.md: %v", err)
	}
	want, err := os.ReadFile("testdata/fixtures/nested-repo/skills/tdd/SKILL.md")
	if err != nil {
		t.Fatalf("reading fixture SKILL.md: %v", err)
	}
	if string(data) != string(want) {
		t.Errorf("installed SKILL.md content does not match fixture")
	}

	// The nested resource file should have come along too, and the
	// sibling skill at skills/other-skill should not have.
	if _, err := os.ReadFile(filepath.Join(projectRoot, ".agents", "skills", "tdd", "resources", "notes.md")); err != nil {
		t.Errorf("reading installed nested resource: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".agents", "skills", "other-skill")); !os.IsNotExist(err) {
		t.Errorf("expected sibling skill other-skill not to be installed, stat err = %v", err)
	}

	if len(fetcher.calls) != 1 {
		t.Fatalf("fetcher.calls = %v, want exactly 1 call", fetcher.calls)
	}
	call := fetcher.calls[0]
	if call.Source.Owner != "owner" || call.Source.Repo != "nested-repo" {
		t.Errorf("fetcher called with %+v, want Owner=%q Repo=%q", call.Source, "owner", "nested-repo")
	}
	if call.Ref != "" {
		t.Errorf("fetcher called with ref %q, want empty (v1 always fetches the default branch, see #11)", call.Ref)
	}

	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	entry, ok := lf["tdd"]
	if !ok {
		t.Fatalf("lockfile missing entry for tdd")
	}
	if entry.Source != "owner/nested-repo/tree/skills/tdd" {
		t.Errorf("entry.Source = %q, want canonical %q", entry.Source, "owner/nested-repo/tree/skills/tdd")
	}
	if entry.SourceURL != "https://github.com/owner/nested-repo/tree/skills/tdd" {
		t.Errorf("entry.SourceURL = %q, want %q", entry.SourceURL, "https://github.com/owner/nested-repo/tree/skills/tdd")
	}
	if entry.Ref != "" {
		t.Errorf("entry.Ref = %q, want empty (v1 has no ref-pinning concept, see #11)", entry.Ref)
	}
	if entry.SkillPath != "skills/tdd" {
		t.Errorf("entry.SkillPath = %q, want %q", entry.SkillPath, "skills/tdd")
	}
}

func TestAdd_GitHubTreePath_FullURL_BehavesIdenticallyToShorthand(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	fetcher := &fakeFetcher{fixtures: map[string]string{
		"owner/nested-repo": "testdata/fixtures/nested-repo",
	}}

	result, err := skl.Add(skl.AddOptions{
		Source:            "https://github.com/owner/nested-repo/tree/skills/tdd",
		Fetcher:           fetcher,
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if result.Name != "tdd" {
		t.Errorf("result.Name = %q, want %q", result.Name, "tdd")
	}

	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	entry := lf["tdd"]
	if entry.Source != "owner/nested-repo/tree/skills/tdd" {
		t.Errorf("entry.Source = %q, want canonical shorthand form %q even though a full URL was given", entry.Source, "owner/nested-repo/tree/skills/tdd")
	}
}

func TestAdd_GitHubTreePath_ShorthandAndFullURL_ExpandSameEntry(t *testing.T) {
	fakeHome(t, "claude-code")
	projectRoot := t.TempDir()

	fetcher := &fakeFetcher{fixtures: map[string]string{
		"owner/nested-repo": "testdata/fixtures/nested-repo",
	}}

	if _, err := skl.Add(skl.AddOptions{
		Source:            "owner/nested-repo/tree/skills/tdd",
		Fetcher:           fetcher,
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"universal"},
	}); err != nil {
		t.Fatalf("first Add() (shorthand) error = %v", err)
	}

	if _, err := skl.Add(skl.AddOptions{
		Source:            "https://github.com/owner/nested-repo/tree/skills/tdd",
		Fetcher:           fetcher,
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"claude-code"},
	}); err != nil {
		t.Fatalf("second Add() (full URL) error = %v", err)
	}

	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	if len(lf) != 1 {
		t.Fatalf("len(lockfile) = %d, want 1 (expanded, not duplicated), got %v", len(lf), lf)
	}
	entry := lf["tdd"]
	for _, want := range []string{"universal", "claude-code"} {
		if _, ok := entry.Adapters[want]; !ok {
			t.Errorf("entry.Adapters missing %q after expand, got %v", want, entry.Adapters)
		}
	}
}

func TestAdd_GitHubTreePath_PathNotFoundInFetchedRepo_ErrorsClearly(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	fetcher := &fakeFetcher{fixtures: map[string]string{
		"owner/nested-repo": "testdata/fixtures/nested-repo",
	}}

	_, err := skl.Add(skl.AddOptions{
		Source:      "owner/nested-repo/tree/skills/does-not-exist",
		Fetcher:     fetcher,
		ProjectRoot: projectRoot,
	})
	if err == nil {
		t.Fatalf("Add() error = nil, want error for a tree-path naming a nonexistent path")
	}
	if !contains(err.Error(), "skills/does-not-exist") {
		t.Errorf("error %q does not name the missing path", err.Error())
	}
}
