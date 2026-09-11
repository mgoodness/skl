package skl_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mgoodness/skl/internal/skl"
)

// fakeGlobalHome delegates to fakeHome for $HOME/$XDG_CONFIG_HOME setup
// and detection markers (claude-code and kit, so --agent '*' targets all
// three adapters), then layers $XDG_DATA_HOME on top and returns all
// three resolved directories, giving each global-scope test full control
// over where global destinations and the global lockfile land,
// independent of the actual machine running the test.
func fakeGlobalHome(t *testing.T) (home, configHome, dataHome string) {
	t.Helper()
	fakeHome(t, "claude-code", "kit")
	home = os.Getenv("HOME")
	configHome = os.Getenv("XDG_CONFIG_HOME")
	dataHome = t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	return home, configHome, dataHome
}

func TestAdd_Global_InstallsIntoEachAdapterGlobalDestination(t *testing.T) {
	home, configHome, _ := fakeGlobalHome(t)

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		Global:            true,
		RequestedAdapters: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	wantDestinations := map[string]string{
		"claude-code": filepath.Join(home, ".claude", "skills", "simple-skill"),
		"kit":         filepath.Join(configHome, "kit", "skills", "simple-skill"),
		"universal":   filepath.Join(home, ".agents", "skills", "simple-skill"),
	}

	for adapter, wantDest := range wantDestinations {
		entry, ok := result.Adapters[adapter]
		if !ok {
			t.Errorf("result.Adapters missing entry for %q", adapter)
			continue
		}
		if filepath.FromSlash(entry.Path) != wantDest {
			t.Errorf("result.Adapters[%q].Path = %q, want %q", adapter, entry.Path, wantDest)
		}
		if _, err := os.Stat(filepath.Join(wantDest, "SKILL.md")); err != nil {
			t.Errorf("expected %s/SKILL.md to exist: %v", wantDest, err)
		}
	}
}

func TestAdd_Global_WritesToXDGDataHomeLockfileWhenSet(t *testing.T) {
	_, _, dataHome := fakeGlobalHome(t)

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		Global:            true,
		RequestedAdapters: []string{"*"},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	lockPath := filepath.Join(dataHome, "skl", "lock.json")
	lf, err := skl.ReadLockfile(lockPath)
	if err != nil {
		t.Fatalf("ReadLockfile(%s) error = %v", lockPath, err)
	}
	if _, ok := lf["simple-skill"]; !ok {
		t.Errorf("global lockfile at %s missing entry for simple-skill; got %v", lockPath, lf)
	}
}

func TestAdd_Global_FallsBackToHomeLocalShareWhenXDGDataHomeUnset(t *testing.T) {
	fakeHome(t) // neither claude-code nor kit detected; irrelevant to this test
	home := os.Getenv("HOME")
	t.Setenv("XDG_DATA_HOME", "")

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		Global:            true,
		RequestedAdapters: []string{"*"},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	lockPath := filepath.Join(home, ".local", "share", "skl", "lock.json")
	lf, err := skl.ReadLockfile(lockPath)
	if err != nil {
		t.Fatalf("ReadLockfile(%s) error = %v", lockPath, err)
	}
	if _, ok := lf["simple-skill"]; !ok {
		t.Errorf("global lockfile at %s missing entry for simple-skill; got %v", lockPath, lf)
	}
}

func TestAdd_Global_LockfileEntrySchema_NoTimestamps(t *testing.T) {
	_, _, dataHome := fakeGlobalHome(t)

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		Global:            true,
		RequestedAdapters: []string{"*"},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	lockPath := filepath.Join(dataHome, "skl", "lock.json")
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
	if entry.ContentHash == "" {
		t.Errorf("entry.ContentHash is empty, want non-empty")
	}
	if len(entry.Adapters) != 3 {
		t.Errorf("entry.Adapters = %v, want 3 entries", entry.Adapters)
	}

	var entryMap map[string]json.RawMessage
	if err := json.Unmarshal(entryData, &entryMap); err != nil {
		t.Fatalf("parsing entry as map: %v", err)
	}
	for _, forbidden := range []string{"installedAt", "updatedAt"} {
		if _, present := entryMap[forbidden]; present {
			t.Errorf("global lockfile entry unexpectedly contains %q", forbidden)
		}
	}
}

func TestAdd_Global_LockfileEntriesAreAlphabetizedBySkillName(t *testing.T) {
	_, _, dataHome := fakeGlobalHome(t)

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		Global:            true,
		RequestedAdapters: []string{"*"},
	}); err != nil {
		t.Fatalf("Add() simple-skill error = %v", err)
	}
	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/skill-with-resource",
		Global:            true,
		RequestedAdapters: []string{"*"},
	}); err != nil {
		t.Fatalf("Add() skill-with-resource error = %v", err)
	}

	lockPath := filepath.Join(dataHome, "skl", "lock.json")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("reading %s: %v", lockPath, err)
	}

	idxSimple := indexOf(string(data), `"simple-skill"`)
	idxResource := indexOf(string(data), `"skill-with-resource"`)
	if idxSimple == -1 || idxResource == -1 {
		t.Fatalf("expected both entries present in lockfile, got: %s", data)
	}
	if idxSimple > idxResource {
		t.Errorf("expected simple-skill to appear before skill-with-resource in lockfile, got: %s", data)
	}
}

func TestAdd_OmittingGlobal_DefaultsToProjectScope(t *testing.T) {
	fakeHome(t, "claude-code", "kit")
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectRoot, ".skl-lock.json")); err != nil {
		t.Errorf("expected project-scoped .skl-lock.json to exist: %v", err)
	}
}
