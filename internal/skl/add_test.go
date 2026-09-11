package skl_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mgoodness/skl/internal/skl"
)

func TestAdd_LocalSingleSkill_CopiesIntoAllThreeAdapterDestinations(t *testing.T) {
	fakeHome(t, "claude-code", "kit")
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if result.Name != "simple-skill" {
		t.Errorf("result.Name = %q, want %q", result.Name, "simple-skill")
	}

	wantDestinations := map[string]string{
		"claude-code": filepath.Join(".claude", "skills", "simple-skill"),
		"kit":         filepath.Join(".kit", "skills", "simple-skill"),
		"universal":   filepath.Join(".agents", "skills", "simple-skill"),
	}

	for adapter, relDest := range wantDestinations {
		entry, ok := result.Adapters[adapter]
		if !ok {
			t.Errorf("result.Adapters missing entry for %q", adapter)
			continue
		}
		if filepath.FromSlash(entry.Path) != relDest {
			t.Errorf("result.Adapters[%q].Path = %q, want %q", adapter, entry.Path, relDest)
		}

		installed := filepath.Join(projectRoot, relDest, "SKILL.md")
		data, err := os.ReadFile(installed)
		if err != nil {
			t.Fatalf("reading installed SKILL.md for %q: %v", adapter, err)
		}
		want, err := os.ReadFile("testdata/fixtures/simple-skill/SKILL.md")
		if err != nil {
			t.Fatalf("reading fixture SKILL.md: %v", err)
		}
		if string(data) != string(want) {
			t.Errorf("installed SKILL.md content for %q does not match fixture", adapter)
		}
	}
}

func TestAdd_LocalSingleSkill_WritesLockfileEntry(t *testing.T) {
	fakeHome(t, "claude-code", "kit")
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	lockPath := filepath.Join(projectRoot, ".skl-lock.json")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("reading %s: %v", lockPath, err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parsing lockfile: %v", err)
	}

	entryData, ok := raw["simple-skill"]
	if !ok {
		t.Fatalf("lockfile missing entry for simple-skill; got keys %v", raw)
	}

	var entry skl.LockEntry
	if err := json.Unmarshal(entryData, &entry); err != nil {
		t.Fatalf("parsing entry: %v", err)
	}

	if entry.Source != "testdata/fixtures/simple-skill" {
		t.Errorf("entry.Source = %q, want %q", entry.Source, "testdata/fixtures/simple-skill")
	}
	if entry.SourceType != "local" {
		t.Errorf("entry.SourceType = %q, want %q", entry.SourceType, "local")
	}
	if entry.SkillPath == "" {
		t.Errorf("entry.SkillPath is empty, want non-empty")
	}
	if entry.ContentHash == "" {
		t.Errorf("entry.ContentHash is empty, want non-empty")
	}

	wantAdapterPaths := map[string]string{
		"claude-code": filepath.ToSlash(filepath.Join(".claude", "skills", "simple-skill")),
		"kit":         filepath.ToSlash(filepath.Join(".kit", "skills", "simple-skill")),
		"universal":   filepath.ToSlash(filepath.Join(".agents", "skills", "simple-skill")),
	}
	if len(entry.Adapters) != len(wantAdapterPaths) {
		t.Fatalf("entry.Adapters = %v, want 3 entries", entry.Adapters)
	}
	for adapter, wantPath := range wantAdapterPaths {
		got, ok := entry.Adapters[adapter]
		if !ok {
			t.Errorf("entry.Adapters missing %q", adapter)
			continue
		}
		if got.Path != wantPath {
			t.Errorf("entry.Adapters[%q].Path = %q, want %q", adapter, got.Path, wantPath)
		}
	}

	// No timestamps: the raw entry object must not contain installedAt/updatedAt keys.
	var entryMap map[string]json.RawMessage
	if err := json.Unmarshal(entryData, &entryMap); err != nil {
		t.Fatalf("parsing entry as map: %v", err)
	}
	for _, forbidden := range []string{"installedAt", "updatedAt"} {
		if _, present := entryMap[forbidden]; present {
			t.Errorf("lockfile entry unexpectedly contains %q", forbidden)
		}
	}
}

func TestAdd_LockfileEntriesAreAlphabetizedBySkillName(t *testing.T) {
	fakeHome(t, "claude-code", "kit")
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	}); err != nil {
		t.Fatalf("Add() simple-skill error = %v", err)
	}
	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/skill-with-resource",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	}); err != nil {
		t.Fatalf("Add() skill-with-resource error = %v", err)
	}

	lockPath := filepath.Join(projectRoot, ".skl-lock.json")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("reading %s: %v", lockPath, err)
	}

	// "simple-skill" sorts after "skill-with-resource" is false: 'i'<'k' so
	// "simple-skill" < "skill-with-resource" alphabetically.
	idxSimple := indexOf(string(data), `"simple-skill"`)
	idxResource := indexOf(string(data), `"skill-with-resource"`)
	if idxSimple == -1 || idxResource == -1 {
		t.Fatalf("expected both entries present in lockfile, got: %s", data)
	}
	if idxSimple > idxResource {
		t.Errorf("expected simple-skill to appear before skill-with-resource in lockfile, got: %s", data)
	}
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func TestAdd_InvalidSource_Errors(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"source without root SKILL.md", "testdata/fixtures/not-a-skill"},
		{"nonexistent source", "testdata/fixtures/does-not-exist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectRoot := t.TempDir()

			_, err := skl.Add(skl.AddOptions{
				Source:      tt.source,
				ProjectRoot: projectRoot,
			})
			if err == nil {
				t.Fatalf("Add() error = nil, want error for %s", tt.name)
			}
		})
	}
}

func TestAdd_CopiesNestedResourceFiles(t *testing.T) {
	fakeHome(t, "claude-code", "kit")
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/skill-with-resource",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	for adapter, relDest := range result.Adapters {
		installed := filepath.Join(projectRoot, filepath.FromSlash(relDest.Path), "resources", "notes.md")
		data, err := os.ReadFile(installed)
		if err != nil {
			t.Fatalf("reading installed nested resource for %q: %v", adapter, err)
		}
		want, err := os.ReadFile("testdata/fixtures/skill-with-resource/resources/notes.md")
		if err != nil {
			t.Fatalf("reading fixture resource: %v", err)
		}
		if string(data) != string(want) {
			t.Errorf("installed resource content for %q does not match fixture", adapter)
		}
	}
}

func TestAdd_DifferentSkillsHaveDifferentContentHashes(t *testing.T) {
	fakeHome(t, "claude-code", "kit")
	projectRootA := t.TempDir()
	projectRootB := t.TempDir()

	resultA, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRootA,
		RequestedAdapters: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() simple-skill error = %v", err)
	}
	resultB, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/skill-with-resource",
		ProjectRoot:       projectRootB,
		RequestedAdapters: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() skill-with-resource error = %v", err)
	}

	hashA := contentHashFromLockfile(t, projectRootA, resultA.Name)
	hashB := contentHashFromLockfile(t, projectRootB, resultB.Name)

	if hashA == "" || hashB == "" {
		t.Fatalf("expected non-empty hashes, got %q and %q", hashA, hashB)
	}
	if hashA == hashB {
		t.Errorf("expected different content hashes for different skills, both were %q", hashA)
	}
}

func TestAdd_ReAddingSameSkill_UpdatesEntryWithoutDuplicating(t *testing.T) {
	fakeHome(t, "claude-code", "kit")
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	}); err != nil {
		t.Fatalf("first Add() error = %v", err)
	}
	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	}); err != nil {
		t.Fatalf("second Add() error = %v", err)
	}

	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	if len(lf) != 1 {
		t.Fatalf("len(lockfile) = %d, want 1 (no duplicate entry), got %v", len(lf), lf)
	}
}

func contentHashFromLockfile(t *testing.T, projectRoot, name string) string {
	t.Helper()
	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	entry, ok := lf[name]
	if !ok {
		t.Fatalf("lockfile missing entry for %q", name)
	}
	return entry.ContentHash
}
