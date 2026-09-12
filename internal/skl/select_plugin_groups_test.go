package skl

import (
	"reflect"
	"strings"
	"testing"
)

// testPluginSkills mirrors discoverSkills' output shape for a
// manifest-bearing source: tdd and code-review are dual members (both in
// the skills/engineering directory group and the engineering-plugin
// plugin group), standalone is plugin-only, blog is directory-only.
func testPluginSkills() []discoveredSkill {
	return []discoveredSkill{
		{Name: "standalone", Dir: "src/extra/standalone", Group: "extra", PluginName: "engineering-plugin"},
		{Name: "code-review", Dir: "src/skills/engineering/code-review", Group: "skills/engineering", PluginName: "engineering-plugin"},
		{Name: "tdd", Dir: "src/skills/engineering/tdd", Group: "skills/engineering", PluginName: "engineering-plugin"},
		{Name: "blog", Dir: "src/skills/writing/blog", Group: "skills/writing"},
	}
}

func TestSelectSkills_PluginGroup_SelectsEveryDeclaredSkill(t *testing.T) {
	got, err := selectSkills(testPluginSkills(), []string{"engineering-plugin"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}

	var names []string
	for _, s := range got {
		names = append(names, s.Name)
	}
	want := []string{"standalone", "code-review", "tdd"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("selectSkills() = %v, want %v", names, want)
	}
}

// TestSelectSkills_DualMembership_NotDuplicated is AC #3: a skill
// belonging to both a directory group and a plugin group is treated as
// one skill -- selected once even when both addressings are requested in
// the same list.
func TestSelectSkills_DualMembership_NotDuplicated(t *testing.T) {
	got, err := selectSkills(testPluginSkills(), []string{"skills/engineering,engineering-plugin"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}

	var names []string
	for _, s := range got {
		names = append(names, s.Name)
	}
	// standalone belongs to the plugin group too; only the dual-member
	// skills (code-review, tdd) are deduplicated against the directory
	// group's selection.
	want := []string{"code-review", "tdd", "standalone"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("selectSkills() = %v, want %v (each dual-member skill exactly once)", names, want)
	}
}

// TestSelectSkills_DirectoryGroupAndPluginGroup_Combine asserts group
// values of both kinds mix freely in one --skill list.
func TestSelectSkills_DirectoryGroupAndPluginGroup_Combine(t *testing.T) {
	got, err := selectSkills(testPluginSkills(), []string{"skills/writing,engineering-plugin"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}

	var names []string
	for _, s := range got {
		names = append(names, s.Name)
	}
	want := []string{"blog", "standalone", "code-review", "tdd"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("selectSkills() = %v, want %v", names, want)
	}
}

// TestSelectSkills_PluginNameCollision_IndividualSkillWins is AC #5 for
// the plugin kind: when a --skill value matches both an individual skill
// name and a plugin group name, the individual skill wins and the
// plugin's other members are not swept in.
func TestSelectSkills_PluginNameCollision_IndividualSkillWins(t *testing.T) {
	discovered := []discoveredSkill{
		{Name: "dup", Dir: "src/skills/dup", Group: "skills", PluginName: ""},
		{Name: "a", Dir: "src/skills/a", Group: "skills", PluginName: "dup"},
	}

	got, err := selectSkills(discovered, []string{"dup"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "dup" {
		t.Fatalf("selectSkills() = %v, want only the individual skill dup", got)
	}
}

// TestSelectSkills_PluginGroupListedInPicker asserts the multi-skill
// error/picker output lists plugin groups alongside directory groups,
// with dual-membership skills nested under both (AC #3, #18).
func TestSelectSkills_PluginGroupListedInPicker(t *testing.T) {
	_, err := selectSkills(testPluginSkills(), nil)
	if err == nil {
		t.Fatal("selectSkills() error = nil, want the multi-skill-without---skill error")
	}

	for _, want := range []string{
		"skills/engineering:",
		"skills/writing:",
		"extra:",
		"engineering-plugin (plugin):",
		"code-review", "tdd", "blog", "standalone",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not list %q", err.Error(), want)
		}
	}
}

// TestSelectSkills_UnknownPluginGroup_Errors asserts a --skill value
// matching no skill, directory group, or plugin group is refused with
// the discovered names as suggestions.
func TestSelectSkills_UnknownPluginGroup_Errors(t *testing.T) {
	_, err := selectSkills(testPluginSkills(), []string{"no-such-plugin"})
	if err == nil {
		t.Fatal("selectSkills() error = nil, want error for an unknown plugin group")
	}
	if !strings.Contains(err.Error(), "no-such-plugin") {
		t.Errorf("error %q does not name the unknown value", err.Error())
	}
	for _, want := range []string{"tdd", "standalone", "blog"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not list discovered skill %q", err.Error(), want)
		}
	}
}
