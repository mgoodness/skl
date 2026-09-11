package skl_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mgoodness/skl/internal/skl"
)

// Expand: same name and source, new adapter targeted.

func TestAdd_SameNameAndSource_NewAdapter_ExpandsExistingEntry(t *testing.T) {
	fakeHome(t, "claude-code", "kit")
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"universal"},
	}); err != nil {
		t.Fatalf("first Add() error = %v", err)
	}

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"kit"},
	})
	if err != nil {
		t.Fatalf("second Add() error = %v", err)
	}
	if _, ok := result.Adapters["kit"]; !ok {
		t.Errorf("result.Adapters missing newly-targeted %q, got %v", "kit", result.Adapters)
	}

	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	if len(lf) != 1 {
		t.Fatalf("len(lockfile) = %d, want 1 (expanded, not duplicated), got %v", len(lf), lf)
	}

	entry := lf["simple-skill"]
	for _, want := range []string{"universal", "kit"} {
		if _, ok := entry.Adapters[want]; !ok {
			t.Errorf("entry.Adapters missing %q after expand, got %v", want, entry.Adapters)
		}
	}
	if len(entry.Adapters) != 2 {
		t.Errorf("entry.Adapters = %v, want exactly 2 entries after expand", entry.Adapters)
	}
}

// Conflict: same name, different source.

func TestAdd_SameNameDifferentSource_RefusedWithoutForce(t *testing.T) {
	fakeHome(t, "claude-code", "kit")
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/conflict-source-a/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"universal"},
	}); err != nil {
		t.Fatalf("first Add() error = %v", err)
	}

	_, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/conflict-source-b/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"universal"},
	})
	if err == nil {
		t.Fatalf("second Add() error = nil, want conflict error")
	}
	if !contains(err.Error(), "conflict-source-a/simple-skill") {
		t.Errorf("error %q does not identify the conflicting (existing) source", err.Error())
	}

	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	entry, ok := lf["simple-skill"]
	if !ok {
		t.Fatalf("lockfile lost its entry for simple-skill after a refused conflict")
	}
	if !contains(entry.Source, "conflict-source-a") {
		t.Errorf("entry.Source = %q, want unchanged original source", entry.Source)
	}
}

// Conflict + --force: replaces the entry's source/metadata and proceeds.

func TestAdd_SameNameDifferentSource_ForceReplacesEntry(t *testing.T) {
	fakeHome(t, "claude-code", "kit")
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/conflict-source-a/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"universal"},
	}); err != nil {
		t.Fatalf("first Add() error = %v", err)
	}

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/conflict-source-b/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"kit"},
		Force:             true,
	})
	if err != nil {
		t.Fatalf("forced Add() error = %v", err)
	}
	if _, ok := result.Adapters["kit"]; !ok {
		t.Errorf("result.Adapters missing %q, got %v", "kit", result.Adapters)
	}

	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	if len(lf) != 1 {
		t.Fatalf("len(lockfile) = %d, want 1, got %v", len(lf), lf)
	}

	entry := lf["simple-skill"]
	if !contains(entry.Source, "conflict-source-b") {
		t.Errorf("entry.Source = %q, want it replaced with the new (forced) source", entry.Source)
	}
	if _, ok := entry.Adapters["universal"]; ok {
		t.Errorf("entry.Adapters still tracks %q from the replaced source, want it dropped, got %v", "universal", entry.Adapters)
	}

	oldDest := filepath.Join(projectRoot, ".agents", "skills", "simple-skill")
	if _, err := os.Stat(oldDest); !os.IsNotExist(err) {
		t.Errorf("expected orphaned old destination %s to be removed, stat err = %v", oldDest, err)
	}

	installed := filepath.Join(projectRoot, ".kit", "skills", "simple-skill", "SKILL.md")
	data, err := os.ReadFile(installed)
	if err != nil {
		t.Fatalf("reading installed SKILL.md: %v", err)
	}
	want, err := os.ReadFile("testdata/fixtures/conflict-source-b/simple-skill/SKILL.md")
	if err != nil {
		t.Fatalf("reading fixture B SKILL.md: %v", err)
	}
	if string(data) != string(want) {
		t.Errorf("installed content does not match the forced source's content")
	}
}

// Destination collision: target directory already exists and is non-empty.

func TestAdd_NonEmptyDestination_RefusedWithoutForce(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	destDir := filepath.Join(projectRoot, ".agents", "skills", "simple-skill")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("creating stray destination: %v", err)
	}
	strayFile := filepath.Join(destDir, "leftover.txt")
	if err := os.WriteFile(strayFile, []byte("pre-existing, unrelated to skl"), 0o644); err != nil {
		t.Fatalf("writing stray file: %v", err)
	}

	_, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"universal"},
	})
	if err == nil {
		t.Fatalf("Add() error = nil, want destination-collision error")
	}
	if !contains(err.Error(), destDir) {
		t.Errorf("error %q does not identify the colliding destination %q", err.Error(), destDir)
	}

	if _, err := os.Stat(strayFile); err != nil {
		t.Errorf("stray file was removed despite the refusal: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".skl-lock.json")); !os.IsNotExist(err) {
		t.Errorf("expected no lockfile to be written on a refused destination collision")
	}
}

// Destination collision + --force: old contents removed, replaced cleanly.

func TestAdd_NonEmptyDestination_ForceOverwritesCleanly(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	destDir := filepath.Join(projectRoot, ".agents", "skills", "simple-skill")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatalf("creating stray destination: %v", err)
	}
	strayFile := filepath.Join(destDir, "leftover.txt")
	if err := os.WriteFile(strayFile, []byte("pre-existing, unrelated to skl"), 0o644); err != nil {
		t.Fatalf("writing stray file: %v", err)
	}

	_, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"universal"},
		Force:             true,
	})
	if err != nil {
		t.Fatalf("forced Add() error = %v", err)
	}

	if _, err := os.Stat(strayFile); !os.IsNotExist(err) {
		t.Errorf("expected stray leftover file to be removed, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "SKILL.md")); err != nil {
		t.Errorf("expected new skill's SKILL.md to be installed: %v", err)
	}
}
