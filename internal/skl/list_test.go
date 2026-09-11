package skl_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mgoodness/skl/internal/skl"
)

// TestList_ShowsBothScopesLabeled installs one skill project-scoped and a
// different skill global-scoped, then asserts List returns both, each
// labeled by scope, with no drift annotations (adapter detected, path on
// disk).
func TestList_ShowsBothScopesLabeled(t *testing.T) {
	fakeGlobalHome(t) // detects claude-code + kit, sets XDG_DATA_HOME
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"universal"},
	}); err != nil {
		t.Fatalf("project-scoped Add() error = %v", err)
	}
	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/skill-with-resource",
		Global:            true,
		RequestedAdapters: []string{"universal"},
	}); err != nil {
		t.Fatalf("global-scoped Add() error = %v", err)
	}

	entries, err := skl.List(skl.ListOptions{ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("List() returned %d entries, want 2: %+v", len(entries), entries)
	}

	byName := map[string]skl.ListEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}

	project, ok := byName["simple-skill"]
	if !ok {
		t.Fatalf("List() missing entry for simple-skill: %+v", entries)
	}
	if project.Scope != skl.ProjectScope {
		t.Errorf("simple-skill.Scope = %q, want %q", project.Scope, skl.ProjectScope)
	}
	if len(project.Adapters) != 1 || project.Adapters[0].Name != "universal" {
		t.Errorf("simple-skill.Adapters = %+v, want exactly [universal]", project.Adapters)
	}
	if project.Adapters[0].NotDetected || project.Adapters[0].Missing {
		t.Errorf("simple-skill.Adapters[0] = %+v, want no drift annotations", project.Adapters[0])
	}

	global, ok := byName["skill-with-resource"]
	if !ok {
		t.Fatalf("List() missing entry for skill-with-resource: %+v", entries)
	}
	if global.Scope != skl.GlobalScope {
		t.Errorf("skill-with-resource.Scope = %q, want %q", global.Scope, skl.GlobalScope)
	}
}

// TestList_AnnotatesNotDetected installs a skill for an adapter that is
// detected at install time, then simulates that adapter becoming
// undetected (e.g. the tool was uninstalled) and asserts List annotates
// it, while its on-disk destination is untouched so Missing stays false.
func TestList_AnnotatesNotDetected(t *testing.T) {
	fakeHome(t, "claude-code")
	home := os.Getenv("HOME")
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"claude-code"},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Simulate claude-code no longer being detected on this machine.
	if err := os.RemoveAll(filepath.Join(home, ".claude")); err != nil {
		t.Fatalf("removing fake ~/.claude: %v", err)
	}

	entries, err := skl.List(skl.ListOptions{ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 || len(entries[0].Adapters) != 1 {
		t.Fatalf("List() = %+v, want exactly 1 entry with 1 adapter", entries)
	}

	a := entries[0].Adapters[0]
	if !a.NotDetected {
		t.Errorf("adapter %+v, want NotDetected = true", a)
	}
	if a.Missing {
		t.Errorf("adapter %+v, want Missing = false (destination still on disk)", a)
	}
}

// TestList_AnnotatesMissing installs a skill, then removes its destination
// directory from disk directly (simulating manual deletion) and asserts
// List annotates it, while the adapter is still detected so NotDetected
// stays false.
func TestList_AnnotatesMissing(t *testing.T) {
	fakeHome(t, "claude-code")
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"claude-code"},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	dest := filepath.Join(projectRoot, ".claude", "skills", "simple-skill")
	if err := os.RemoveAll(dest); err != nil {
		t.Fatalf("removing destination %s: %v", dest, err)
	}

	entries, err := skl.List(skl.ListOptions{ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 || len(entries[0].Adapters) != 1 {
		t.Fatalf("List() = %+v, want exactly 1 entry with 1 adapter", entries)
	}

	a := entries[0].Adapters[0]
	if !a.Missing {
		t.Errorf("adapter %+v, want Missing = true", a)
	}
	if a.NotDetected {
		t.Errorf("adapter %+v, want NotDetected = false (adapter still detected)", a)
	}
}

// TestList_AnnotatesBothNotDetectedAndMissing combines the two prior
// scenarios and asserts both annotations show simultaneously rather than
// one masking the other.
func TestList_AnnotatesBothNotDetectedAndMissing(t *testing.T) {
	fakeHome(t, "claude-code")
	home := os.Getenv("HOME")
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"claude-code"},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if err := os.RemoveAll(filepath.Join(home, ".claude")); err != nil {
		t.Fatalf("removing fake ~/.claude: %v", err)
	}
	dest := filepath.Join(projectRoot, ".claude", "skills", "simple-skill")
	if err := os.RemoveAll(dest); err != nil {
		t.Fatalf("removing destination %s: %v", dest, err)
	}

	entries, err := skl.List(skl.ListOptions{ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 || len(entries[0].Adapters) != 1 {
		t.Fatalf("List() = %+v, want exactly 1 entry with 1 adapter", entries)
	}

	a := entries[0].Adapters[0]
	if !a.NotDetected || !a.Missing {
		t.Errorf("adapter %+v, want both NotDetected and Missing true simultaneously", a)
	}
}

// TestList_AgentFilter_FiltersToOnlyThatAdaptersEntries asserts
// -a/--agent filtering both excludes a skill entirely when it lacks the
// named adapter and trims a skill's Adapters slice down to just that
// adapter when it has it among others.
func TestList_AgentFilter_FiltersToOnlyThatAdaptersEntries(t *testing.T) {
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
		RequestedAdapters: []string{"universal"},
	}); err != nil {
		t.Fatalf("Add() skill-with-resource error = %v", err)
	}

	entries, err := skl.List(skl.ListOptions{ProjectRoot: projectRoot, Agent: "claude-code"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("List(Agent=claude-code) = %+v, want exactly 1 entry (skill-with-resource has no claude-code)", entries)
	}
	if entries[0].Name != "simple-skill" {
		t.Errorf("entries[0].Name = %q, want %q", entries[0].Name, "simple-skill")
	}
	if len(entries[0].Adapters) != 1 || entries[0].Adapters[0].Name != "claude-code" {
		t.Errorf("entries[0].Adapters = %+v, want exactly [claude-code]", entries[0].Adapters)
	}
}

// TestList_AgentFilter_UnknownAdapter_Errors asserts an unrecognized
// --agent value is refused rather than silently returning an empty list,
// matching the no-silent-guessing rule applied elsewhere (e.g. --skill,
// add's --agent).
func TestList_AgentFilter_UnknownAdapter_Errors(t *testing.T) {
	fakeHome(t)
	projectRoot := t.TempDir()

	_, err := skl.List(skl.ListOptions{ProjectRoot: projectRoot, Agent: "not-a-real-adapter"})
	if err == nil {
		t.Fatal("List() error = nil, want error for unknown --agent value")
	}
}

// TestList_NoSkillsInstalled_ReturnsEmpty asserts an empty, non-nil
// result rather than an error when neither lockfile has any entries.
func TestList_NoSkillsInstalled_ReturnsEmpty(t *testing.T) {
	fakeHome(t)
	projectRoot := t.TempDir()

	entries, err := skl.List(skl.ListOptions{ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("List() = %+v, want empty", entries)
	}
}
