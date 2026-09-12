package skl

import (
	"reflect"
	"sort"
	"testing"
)

func TestDiscoverSkills_RootSkillMD_SingleSkill(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/simple-skill", SkillPath: "."}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "simple-skill" {
		t.Fatalf("discoverSkills() = %v, want a single skill named %q", got, "simple-skill")
	}
}

func TestDiscoverSkills_TreePathSource_UsesDirDirectlyNoWalk(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/nested-repo/skills/tdd", SkillPath: "skills/tdd"}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "tdd" {
		t.Fatalf("discoverSkills() = %v, want a single skill named %q", got, "tdd")
	}
}

// TestDiscoverSkills_TreePathSource_NoRootSkillMD_Errors asserts tier 1's
// "no further discovery" rule: docs/ has no SKILL.md of its own, and even
// though it's a subdirectory of a repo whose skills/ directory does have
// skills, a tree-path source never walks past its own named directory.
func TestDiscoverSkills_TreePathSource_NoRootSkillMD_Errors(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/nested-repo/docs", SkillPath: "docs"}

	if _, err := discoverSkills(rs); err == nil {
		t.Fatal("discoverSkills() error = nil, want error: docs/ has no SKILL.md, and a tree-path source doesn't fall back to a walk")
	}
}

func TestDiscoverSkills_DepthThreeWalk_FindsEveryFlattenedSkill(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/multi-skill-source", SkillPath: "."}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}

	names := discoveredNamesSorted(got)
	want := []string{"bar", "baz", "foo"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("discoverSkills() names = %v, want %v", names, want)
	}
}

// TestDiscoverSkills_DepthThreeWalk_StopsAtDepthThree asserts the depth-3
// cap: "here" is 3 segments below skills/ and must be found; "too-deep",
// one segment past that, must not be -- even though tier 3 already
// succeeded (found "here") and so never falls back to the unbounded
// search that would otherwise find "too-deep" too.
func TestDiscoverSkills_DepthThreeWalk_StopsAtDepthThree(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/depth-limit-source", SkillPath: "."}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "here" {
		t.Fatalf("discoverSkills() = %v, want exactly one skill named %q (the depth-4 %q skill must be excluded)", got, "here", "too-deep")
	}
}

func TestDiscoverSkills_UnboundedFallback_FindsArbitrarilyDeepSkill(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/unbounded-fallback-source", SkillPath: "."}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "my-tool" {
		t.Fatalf("discoverSkills() = %v, want exactly one skill named %q", got, "my-tool")
	}
}

func TestDiscoverSkills_NoSkillFoundAnywhere_Errors(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/not-a-skill", SkillPath: "."}

	if _, err := discoverSkills(rs); err == nil {
		t.Fatal("discoverSkills() error = nil, want error")
	}
}

// TestDiscoverSkills_GroupedFixture_GroupsEachSkillUnderItsParentPath
// asserts directory-group assignment: each discovered skill's Group is
// the source-relative path of its parent directory (the deepest path a
// user can name to select that skill, per #1's "skills sharing a parent
// path"). "skills/engineering/tools" contains another directory, so
// git-hooks' group is that deeper path, not "skills/engineering"; a
// skill directly under "skills/" (scaffolding) gets group "skills".
func TestDiscoverSkills_GroupedFixture_GroupsEachSkillUnderItsParentPath(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/grouped-skill-source", SkillPath: "."}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}

	wantGroups := map[string]string{
		"code-review": "skills/engineering",
		"tdd":         "skills/engineering",
		"git-hooks":   "skills/engineering/tools",
		"blog":        "skills/writing",
		"scaffolding": "skills",
	}
	if len(got) != len(wantGroups) {
		t.Fatalf("discoverSkills() = %v, want %d skills", got, len(wantGroups))
	}
	for name, group := range wantGroups {
		var found *discoveredSkill
		for i := range got {
			if got[i].Name == name {
				found = &got[i]
				break
			}
		}
		if found == nil {
			t.Errorf("discoverSkills() missing skill %q, got %v", name, got)
			continue
		}
		if found.Group != group {
			t.Errorf("discoverSkills() skill %q Group = %q, want %q", name, found.Group, group)
		}
	}
}

// TestDiscoverSkills_MultiSkillFixture_GroupsDirectChildrenUnderSkillsDir
// locks in grouping for the flat fixture: foo and bar share the parent
// path "skills", and baz, one level deeper, is grouped under
// "skills/group" rather than the shallower "skills".
func TestDiscoverSkills_MultiSkillFixture_GroupsDirectChildrenUnderSkillsDir(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/multi-skill-source", SkillPath: "."}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}

	wantGroups := map[string]string{
		"foo": "skills",
		"bar": "skills",
		"baz": "skills/group",
	}
	if len(got) != len(wantGroups) {
		t.Fatalf("discoverSkills() = %v, want %d skills", got, len(wantGroups))
	}
	for name, group := range wantGroups {
		for i := range got {
			if got[i].Name == name && got[i].Group != group {
				t.Errorf("skill %q Group = %q, want %q", name, got[i].Group, group)
			}
		}
	}
}

// TestDiscoverSkills_RootSkill_HasNoGroup asserts a skill that *is* the
// source root (tier 1 root-SKILL.md) has an empty Group: there is no
// parent path within the source to name as a directory group.
func TestDiscoverSkills_RootSkill_HasNoGroup(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/simple-skill", SkillPath: "."}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("discoverSkills() = %v, want 1 skill", got)
	}
	if got[0].Group != "" {
		t.Errorf("discoverSkills() Group = %q, want empty", got[0].Group)
	}
}

func discoveredNamesSorted(d []discoveredSkill) []string {
	names := make([]string, len(d))
	for i, s := range d {
		names[i] = s.Name
	}
	sort.Strings(names)
	return names
}
