package skl

import (
	"reflect"
	"strings"
	"testing"
)

func TestSelectSkills_SingleDiscoveredSkill_NoSkillFlagNeeded(t *testing.T) {
	discovered := []discoveredSkill{{Name: "simple-skill", Dir: "dir/simple-skill"}}

	got, err := selectSkills(discovered, nil)
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}
	if !reflect.DeepEqual(got, discovered) {
		t.Errorf("selectSkills() = %v, want %v", got, discovered)
	}
}

func TestSelectSkills_MultipleDiscovered_NoSkillFlag_ErrorsListingEveryName(t *testing.T) {
	discovered := []discoveredSkill{
		{Name: "foo", Dir: "d/foo"},
		{Name: "bar", Dir: "d/bar"},
	}

	_, err := selectSkills(discovered, nil)
	if err == nil {
		t.Fatal("selectSkills() error = nil, want a multi-skill-without---skill error")
	}
	for _, want := range []string{"foo", "bar"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not list discovered skill %q", err.Error(), want)
		}
	}
}

func TestSelectSkills_Wildcard_BypassesAmbiguityAndSelectsAll(t *testing.T) {
	discovered := []discoveredSkill{
		{Name: "foo", Dir: "d/foo"},
		{Name: "bar", Dir: "d/bar"},
	}

	got, err := selectSkills(discovered, []string{"*"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v, want no error for --skill '*'", err)
	}
	if len(got) != 2 {
		t.Fatalf("selectSkills() = %v, want all %d discovered skills", got, len(discovered))
	}
}

func TestSelectSkills_NamedSelection_CommaSeparatedAndRepeatedAreEquivalent(t *testing.T) {
	discovered := []discoveredSkill{
		{Name: "foo", Dir: "d/foo"},
		{Name: "bar", Dir: "d/bar"},
		{Name: "baz", Dir: "d/baz"},
	}

	comma, err := selectSkills(discovered, []string{"foo,bar"})
	if err != nil {
		t.Fatalf("selectSkills() comma-separated error = %v", err)
	}
	repeated, err := selectSkills(discovered, []string{"foo", "bar"})
	if err != nil {
		t.Fatalf("selectSkills() repeated-flag error = %v", err)
	}
	if !reflect.DeepEqual(comma, repeated) {
		t.Errorf("comma-separated result %v != repeated-flag result %v", comma, repeated)
	}

	wantNames := []string{"foo", "bar"}
	for i, want := range wantNames {
		if comma[i].Name != want {
			t.Errorf("comma[%d].Name = %q, want %q", i, comma[i].Name, want)
		}
	}
}

func TestSelectSkills_UnknownName_ErrorsListingDiscoveredNames(t *testing.T) {
	discovered := []discoveredSkill{
		{Name: "foo", Dir: "d/foo"},
		{Name: "bar", Dir: "d/bar"},
	}

	_, err := selectSkills(discovered, []string{"nope"})
	if err == nil {
		t.Fatal("selectSkills() error = nil, want error for an unknown skill name")
	}
	for _, want := range []string{"nope", "foo", "bar"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
}

func TestSelectSkills_RepeatedNamesAreDeduplicated(t *testing.T) {
	discovered := []discoveredSkill{
		{Name: "foo", Dir: "d/foo"},
		{Name: "bar", Dir: "d/bar"},
	}

	got, err := selectSkills(discovered, []string{"foo", "foo"})
	if err != nil {
		t.Fatalf("selectSkills() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("selectSkills() = %v, want exactly 1 entry (deduplicated)", got)
	}
}

// TestSelectSkills_SingleDiscoveredSkill_WrongExplicitNameStillErrors
// guards against silently ignoring an explicit --skill value just because
// there was only one skill to begin with -- the "no --skill needed"
// exception applies only when --skill was *omitted*, not when it names
// something wrong.
func TestSelectSkills_SingleDiscoveredSkill_WrongExplicitNameStillErrors(t *testing.T) {
	discovered := []discoveredSkill{{Name: "simple-skill", Dir: "dir/simple-skill"}}

	_, err := selectSkills(discovered, []string{"not-the-right-name"})
	if err == nil {
		t.Fatal("selectSkills() error = nil, want error for an explicit --skill value that doesn't match the one discovered skill")
	}
	if !strings.Contains(err.Error(), "not-the-right-name") || !strings.Contains(err.Error(), "simple-skill") {
		t.Errorf("error %q does not mention both the unknown name and the discovered name", err.Error())
	}
}
