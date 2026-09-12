package skl_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mgoodness/skl/internal/skl"
)

func TestAdd_MultiSkillSource_SkillFlagCommaSeparated_InstallsSelected(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/multi-skill-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"foo,bar"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if len(result.Skills) != 2 {
		t.Fatalf("result.Skills = %v, want exactly 2 entries", result.Skills)
	}
	gotNames := map[string]bool{}
	for _, sk := range result.Skills {
		gotNames[sk.Name] = true
	}
	for _, want := range []string{"foo", "bar"} {
		if !gotNames[want] {
			t.Errorf("result.Skills missing %q, got %v", want, result.Skills)
		}
	}
	if gotNames["baz"] {
		t.Errorf("result.Skills unexpectedly includes unselected skill %q", "baz")
	}

	for _, name := range []string{"foo", "bar"} {
		installed := filepath.Join(projectRoot, ".agents", "skills", name, "SKILL.md")
		if _, err := os.Stat(installed); err != nil {
			t.Errorf("expected %s to exist: %v", installed, err)
		}
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".agents", "skills", "baz")); !os.IsNotExist(err) {
		t.Errorf("expected unselected skill %q not to be installed, stat err = %v", "baz", err)
	}
}

func TestAdd_MultiSkillSource_SkillFlagRepeated_EquivalentToCommaSeparated(t *testing.T) {
	fakeHome(t) // universal only

	commaRoot := t.TempDir()
	commaResult, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/multi-skill-source",
		ProjectRoot:       commaRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"foo,bar"},
	})
	if err != nil {
		t.Fatalf("Add() comma-separated error = %v", err)
	}

	repeatedRoot := t.TempDir()
	repeatedResult, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/multi-skill-source",
		ProjectRoot:       repeatedRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"foo", "bar"},
	})
	if err != nil {
		t.Fatalf("Add() repeated-flag error = %v", err)
	}

	if len(commaResult.Skills) != len(repeatedResult.Skills) {
		t.Fatalf("comma-separated installed %v, repeated-flag installed %v", commaResult.Skills, repeatedResult.Skills)
	}
	commaNames := map[string]bool{}
	for _, sk := range commaResult.Skills {
		commaNames[sk.Name] = true
	}
	for _, sk := range repeatedResult.Skills {
		if !commaNames[sk.Name] {
			t.Errorf("skill %q installed via repeated flags but not via comma-separated", sk.Name)
		}
	}
}

func TestAdd_MultiSkillSource_Wildcard_InstallsEveryDiscoveredSkill(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/multi-skill-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v, want no error for --skill '*'", err)
	}

	if len(result.Skills) != 3 {
		t.Fatalf("result.Skills = %v, want all 3 discovered skills", result.Skills)
	}
	for _, name := range []string{"foo", "bar", "baz"} {
		installed := filepath.Join(projectRoot, ".agents", "skills", name, "SKILL.md")
		if _, err := os.Stat(installed); err != nil {
			t.Errorf("expected %s to exist: %v", installed, err)
		}
	}
}

func TestAdd_MultiSkillSource_NoSkillFlag_ErrorsListingEveryDiscoveredName(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	_, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/multi-skill-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	})
	if err == nil {
		t.Fatal("Add() error = nil, want error when --skill is omitted for a multi-skill source")
	}
	for _, want := range []string{"foo", "bar", "baz"} {
		if !contains(err.Error(), want) {
			t.Errorf("error %q does not list discovered skill %q", err.Error(), want)
		}
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".skl-lock.json")); !os.IsNotExist(err) {
		t.Errorf("expected no lockfile to be written when the multi-skill selection is refused")
	}
}

func TestAdd_MultiSkillSource_UnknownSkillName_Errors(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	_, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/multi-skill-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"nope"},
	})
	if err == nil {
		t.Fatal("Add() error = nil, want error for an unknown --skill name")
	}
	if !contains(err.Error(), "nope") {
		t.Errorf("error %q does not name the unknown skill", err.Error())
	}
}

// TestAdd_MultiSkillSource_NestedSkill_DestinationIsFlattened asserts
// destination flattening: "baz" is nested two levels deep
// (skills/group/baz) within the source, but its destination directory
// name is just "baz", not "group/baz" or "group-baz".
func TestAdd_MultiSkillSource_NestedSkill_DestinationIsFlattened(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/multi-skill-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"baz"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if len(result.Skills) != 1 || result.Skills[0].Name != "baz" {
		t.Fatalf("result.Skills = %v, want a single skill named %q", result.Skills, "baz")
	}

	installed := filepath.Join(projectRoot, ".agents", "skills", "baz", "SKILL.md")
	if _, err := os.Stat(installed); err != nil {
		t.Errorf("expected flattened destination %s to exist: %v", installed, err)
	}

	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	entry, ok := lf["baz"]
	if !ok {
		t.Fatalf("lockfile missing entry for %q, got %v", "baz", lf)
	}
	if entry.SkillPath != "skills/group/baz" {
		t.Errorf("entry.SkillPath = %q, want %q", entry.SkillPath, "skills/group/baz")
	}
}

func TestAdd_MultiSkillSource_EachSelectedSkillGetsItsOwnLockfileEntry(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/multi-skill-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"*"},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	if len(lf) != 3 {
		t.Fatalf("len(lockfile) = %d, want 3, got %v", len(lf), lf)
	}
	for _, name := range []string{"foo", "bar", "baz"} {
		entry, ok := lf[name]
		if !ok {
			t.Errorf("lockfile missing entry for %q", name)
			continue
		}
		if entry.Source != "testdata/fixtures/multi-skill-source" {
			t.Errorf("entry.Source for %q = %q, want %q", name, entry.Source, "testdata/fixtures/multi-skill-source")
		}
	}
}
