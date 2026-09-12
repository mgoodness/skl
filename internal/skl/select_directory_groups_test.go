package skl

import (
	"reflect"
	"strings"
	"testing"
)

// testSkillsFromFixture mirrors discoverSkills' output shape for the
// grouped fixture: names with their parent-directory groups.
func testSkillsFromFixture() []discoveredSkill {
	return []discoveredSkill{
		{Name: "code-review", Dir: "grouped-skill-source/skills/engineering/code-review", Group: "skills/engineering"},
		{Name: "tdd", Dir: "grouped-skill-source/skills/engineering/tdd", Group: "skills/engineering"},
		{Name: "git-hooks", Dir: "grouped-skill-source/skills/engineering/tools/git-hooks", Group: "skills/engineering/tools"},
		{Name: "blog", Dir: "grouped-skill-source/skills/writing/blog", Group: "skills/writing"},
		{Name: "scaffolding", Dir: "grouped-skill-source/skills/scaffolding", Group: "skills"},
	}
}

func TestSelectSkills_DirectoryGroup_SelectsEverySkillUnderIt(t *testing.T) {
	discovered := testSkillsFromFixture()

	got, err := selectSkills(discovered, []string{"skills/engineering"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}

	// git-hooks lives under skills/engineering/tools (one level deeper
	// than its siblings), but "naming a shared parent path" must still
	// select it: a group names a path, and every discovered skill nested
	// under that path is included (#1 story 16).
	var names []string
	for _, s := range got {
		names = append(names, s.Name)
	}
	want := []string{"code-review", "tdd", "git-hooks"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("selectSkills() = %v, want %v", names, want)
	}
}

func TestSelectSkills_DirectoryGroup_DeepGroupPathSelectsOnlySkillsUnderIt(t *testing.T) {
	discovered := testSkillsFromFixture()

	got, err := selectSkills(discovered, []string{"skills/engineering/tools"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "git-hooks" {
		t.Fatalf("selectSkills() = %v, want exactly the git-hooks skill", got)
	}
}

func TestSelectSkills_DirectoryGroup_AndIndividualNames_Combine(t *testing.T) {
	discovered := testSkillsFromFixture()

	got, err := selectSkills(discovered, []string{"skills/writing,blog"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}
	// "blog" is named both by group and individually; it must appear
	// exactly once (deduplicated), not once per selector (AC: "a skill
	// available both individually and as part of a directory group ...
	// treated the same skill ... no duplicate-install surprise").
	var names []string
	for _, s := range got {
		names = append(names, s.Name)
	}
	want := []string{"blog"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("selectSkills() = %v, want %v", names, want)
	}
}

func TestSelectSkills_TwoDirectoryGroups_Combine(t *testing.T) {
	discovered := testSkillsFromFixture()

	got, err := selectSkills(discovered, []string{"skills/engineering", "skills/writing"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}
	var names []string
	for _, s := range got {
		names = append(names, s.Name)
	}
	want := []string{"code-review", "tdd", "git-hooks", "blog"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("selectSkills() = %v, want %v", names, want)
	}
}

func TestSelectSkills_DirectoryGroup_AndIndividualName_Combine(t *testing.T) {
	discovered := testSkillsFromFixture()

	got, err := selectSkills(discovered, []string{"skills/writing,scaffolding"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}
	var names []string
	for _, s := range got {
		names = append(names, s.Name)
	}
	want := []string{"blog", "scaffolding"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("selectSkills() = %v, want %v", names, want)
	}
}

func TestSelectSkills_DirectoryGroup_AmbiguousWithoutSkillFlag_ListingNestsUnderGroups(t *testing.T) {
	discovered := testSkillsFromFixture()

	_, err := selectSkills(discovered, nil)
	if err == nil {
		t.Fatal("selectSkills() error = nil, want the multi-skill-without---skill error")
	}
	for _, want := range []string{"skills/engineering", "skills/writing", "code-review", "tdd", "git-hooks", "blog", "scaffolding"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not list %q", err.Error(), want)
		}
	}
}

func TestSelectSkills_UnknownDirectoryGroup_Errors(t *testing.T) {
	discovered := testSkillsFromFixture()

	_, err := selectSkills(discovered, []string{"skills/nonexistent-group"})
	if err == nil {
		t.Fatal("selectSkills() error = nil, want error for an unknown directory group")
	}
	if !strings.Contains(err.Error(), "skills/nonexistent-group") {
		t.Errorf("error %q does not name the unknown group", err.Error())
	}
	// The discovered names must still be listed as suggestions.
	for _, want := range []string{"code-review", "tdd", "blog"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not list discovered skill %q", err.Error(), want)
		}
	}
}

// TestSelectSkills_GroupNameCollision_IndividualSkillWins asserts the #1
// collision rule: when a --skill value matches both an individual skill
// name and a directory group, the skill is selected individually and the
// group's other members are not silently swept in.
func TestSelectSkills_GroupNameCollision_IndividualSkillWins(t *testing.T) {
	discovered := []discoveredSkill{
		{Name: "skills", Dir: "skills/skills", Group: "skills"},
		{Name: "engineering", Dir: "skills/engineering/tools/engineering", Group: "skills/engineering/tools"},
		{Name: "git-hooks", Dir: "skills/engineering/tools/git-hooks", Group: "skills/engineering/tools"},
	}

	got, err := selectSkills(discovered, []string{"skills/engineering"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}

	// "skills/engineering" is not a skill name here (the names are
	// "skills", "engineering" and "git-hooks"), so it must resolve as the
	// directory group, selecting both members of skills/engineering/tools.
	var names []string
	for _, s := range got {
		names = append(names, s.Name)
	}
	want := []string{"engineering", "git-hooks"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("selectSkills() = %v, want %v", names, want)
	}

	// Conversely "engineering" names the individual skill and *only* it:
	// the group member git-hooks must not be swept in.
	got, err = selectSkills(discovered, []string{"engineering"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "engineering" {
		t.Fatalf("selectSkills() = %v, want only the individual skill engineering", got)
	}
}
