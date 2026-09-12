package skl_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/mgoodness/skl/internal/skl"
)

// installedSkillNames returns the skill names Add reports installed, in
// result order.
func installedSkillNames(t *testing.T, result *skl.AddResult) []string {
	t.Helper()
	var names []string
	for _, sk := range result.Skills {
		names = append(names, sk.Name)
	}
	return names
}

// TestAdd_DirectoryGroup_InstallsEverySkillNestedUnderIt is AC #1: a
// directory group path like "skills/engineering" installs all discovered
// skills under that path, including ones nested one group deeper
// (skills/engineering/tools/git-hooks).
func TestAdd_DirectoryGroup_InstallsEverySkillNestedUnderIt(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/grouped-skill-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"skills/engineering"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	names := installedSkillNames(t, result)
	sort.Strings(names)
	want := []string{"code-review", "git-hooks", "tdd"}
	if len(names) != len(want) {
		t.Fatalf("installed skills = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("installed skill %d = %q, want %q (installed all: %v)", i, names[i], want[i], names)
		}
	}

	// Skills outside the group must not be installed.
	for _, name := range []string{"blog", "scaffolding"} {
		if _, err := os.Stat(filepath.Join(projectRoot, ".agents", "skills", name)); !os.IsNotExist(err) {
			t.Errorf("expected skill outside group %q not to be installed, stat err = %v", name, err)
		}
	}
	for _, name := range want {
		installed := filepath.Join(projectRoot, ".agents", "skills", name, "SKILL.md")
		if _, err := os.Stat(installed); err != nil {
			t.Errorf("expected %s to exist: %v", installed, err)
		}
	}
}

// TestAdd_DirectoryGroup_DeepPathSelectsOnlySkillsUnderIt asserts AC #1
// scoped by the full group path: "skills/engineering/tools" selects only
// the skills under that path, not the shallower engineering siblings.
func TestAdd_DirectoryGroup_DeepPathSelectsOnlySkillsUnderIt(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/grouped-skill-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"skills/engineering/tools"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	names := installedSkillNames(t, result)
	if len(names) != 1 || names[0] != "git-hooks" {
		t.Fatalf("installed skills = %v, want exactly git-hooks", names)
	}
}

// TestAdd_DirectoryGroupsAndIndividualSkills_Combine is AC #3: a group
// path, another group path, and an individual skill name all mix in the
// same --skill value list.
func TestAdd_DirectoryGroupsAndIndividualSkills_Combine(t *testing.T) {
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/grouped-skill-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"skills/writing,skills/engineering,scaffolding"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	names := installedSkillNames(t, result)
	sort.Strings(names)
	want := []string{"blog", "code-review", "git-hooks", "scaffolding", "tdd"}
	if len(names) != len(want) {
		t.Fatalf("installed skills = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("installed skill %d = %q, want %q", i, names[i], want[i])
		}
	}

	// Lockfile carries every selected skill, each with its own entry and
	// the correct source-relative SkillPath.
	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	wantSkillPaths := map[string]string{
		"code-review": "skills/engineering/code-review",
		"tdd":         "skills/engineering/tdd",
		"git-hooks":   "skills/engineering/tools/git-hooks",
		"blog":        "skills/writing/blog",
		"scaffolding": "skills/scaffolding",
	}
	for name, skillPath := range wantSkillPaths {
		entry, ok := lf[name]
		if !ok {
			t.Errorf("lockfile missing entry for %q", name)
			continue
		}
		if entry.SkillPath != skillPath {
			t.Errorf("entry.SkillPath for %q = %q, want %q", name, entry.SkillPath, skillPath)
		}
	}
}

// TestAdd_DirectoryGroup_ErrorWithoutSkillFlag_NestsNamesUnderGroups is
// AC #2: the multi-skill-without---skill error lists every discovered
// skill nested under its directory group, doubling as a picker.
func TestAdd_DirectoryGroup_ErrorWithoutSkillFlag_NestsNamesUnderGroups(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	_, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/grouped-skill-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	})
	if err == nil {
		t.Fatal("Add() error = nil, want error when --skill is omitted for a group-containing multi-skill source")
	}
	for _, want := range []string{
		"skills/engineering:",
		"skills/writing:",
		"skills/engineering/tools:",
		"code-review", "tdd", "git-hooks", "blog", "scaffolding",
	} {
		if !contains(err.Error(), want) {
			t.Errorf("error %q does not list %q", err.Error(), want)
		}
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".skl-lock.json")); !os.IsNotExist(err) {
		t.Errorf("expected no lockfile to be written when the group selection is refused")
	}
}

// TestAdd_DirectoryGroup_UnknownGroup_Errors asserts a bogus group path
// is refused (no silent guessing), with the discovered names + groups as
// suggestions.
func TestAdd_DirectoryGroup_UnknownGroup_Errors(t *testing.T) {
	projectRoot := t.TempDir()

	_, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/grouped-skill-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"skills/nope"},
	})
	if err == nil {
		t.Fatal("Add() error = nil, want error for an unknown directory group")
	}
	if !contains(err.Error(), "skills/nope") {
		t.Errorf("error %q does not name the unknown group", err.Error())
	}
	if !contains(err.Error(), "code-review") {
		t.Errorf("error %q does not list discovered skills as suggestions", err.Error())
	}
}

// TestAdd_DirectoryGroup_EachSkillGetsItsOwnFlattenedDestination asserts
// that group-installed skills land flattened in the adapter destination
// (no "skills/engineering/tools/" path pollution on disk).
func TestAdd_DirectoryGroup_EachSkillGetsItsOwnFlattenedDestination(t *testing.T) {
	projectRoot := t.TempDir()

	if _, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/grouped-skill-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"skills/engineering"},
	}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// git-hooks must land at .agents/skills/git-hooks, not at a nested
	// mirror of its source path.
	installed := filepath.Join(projectRoot, ".agents", "skills", "git-hooks", "SKILL.md")
	if _, err := os.Stat(installed); err != nil {
		t.Errorf("expected flattened destination %s to exist: %v", installed, err)
	}
}
