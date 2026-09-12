package skl_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/mgoodness/skl/internal/skl"
)

// pluginFixture is the local plugin-manifest fixture source's expected
// shape: four skills, three stamped with the manifest's plugin name.
var pluginFixture = map[string]string{
	"standalone":  "engineering-plugin",
	"code-review": "engineering-plugin",
	"tdd":         "engineering-plugin",
	"blog":        "",
}

// lockfilePluginNames returns each lockfile entry's recorded pluginName.
func lockfilePluginNames(t *testing.T, projectRoot string) map[string]string {
	t.Helper()
	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	got := make(map[string]string, len(lf))
	for name, entry := range lf {
		got[name] = entry.PluginName
	}
	return got
}

// TestAdd_PluginGroup_InstallsEveryDeclaredSkill is AC #1 and #2:
// --skill <plugin-name> installs every skill the manifest declares,
// including one discovered only through the manifest (standalone), and
// not the skills it doesn't cover (blog).
func TestAdd_PluginGroup_InstallsEveryDeclaredSkill(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/plugin-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"engineering-plugin"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	names := installedSkillNames(t, result)
	sort.Strings(names)
	want := []string{"code-review", "standalone", "tdd"}
	if len(names) != len(want) {
		t.Fatalf("installed skills = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("installed skill %d = %q, want %q", i, names[i], want[i])
		}
	}

	// blog isn't declared by the plugin and must not be installed.
	if _, err := os.Stat(filepath.Join(projectRoot, ".agents", "skills", "blog")); !os.IsNotExist(err) {
		t.Errorf("expected undeclared skill %q not to be installed, stat err = %v", "blog", err)
	}

	// AC #4: each entry records pluginName, selected via the plugin group.
	gotPlugins := lockfilePluginNames(t, projectRoot)
	for name, want := range pluginFixture {
		if name == "blog" {
			continue
		}
		if gotPlugins[name] != want {
			t.Errorf("lockfile pluginName for %q = %q, want %q", name, gotPlugins[name], want)
		}
	}
	if _, ok := gotPlugins["blog"]; ok {
		t.Errorf("lockfile unexpectedly has an entry for undeclared skill %q", "blog")
	}
}

// TestAdd_PluginGroup_IndividualAndDirectoryAndWildcardAllRecordPluginName
// is AC #4: the lockfile's pluginName is recorded identically whichever
// way the skill was selected -- individually, via its directory group, or
// via --skill '*'.
func TestAdd_PluginGroup_IndividualAndDirectoryAndWildcardAllRecordPluginName(t *testing.T) {
	cases := []struct {
		name       string
		requested  []string
		wantSkills []string
	}{
		{"individually", []string{"tdd"}, []string{"tdd"}},
		{"via directory group", []string{"skills/engineering"}, []string{"code-review", "tdd"}},
		{"via wildcard", []string{"*"}, []string{"blog", "code-review", "standalone", "tdd"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeHome(t) // universal only
			projectRoot := t.TempDir()

			if _, err := skl.Add(skl.AddOptions{
				Source:            "testdata/fixtures/plugin-source",
				ProjectRoot:       projectRoot,
				RequestedAdapters: []string{"*"},
				RequestedSkills:   tc.requested,
			}); err != nil {
				t.Fatalf("Add() error = %v", err)
			}

			gotPlugins := lockfilePluginNames(t, projectRoot)
			if len(gotPlugins) != len(tc.wantSkills) {
				t.Fatalf("lockfile entries = %v, want exactly %v", gotPlugins, tc.wantSkills)
			}
			for _, name := range tc.wantSkills {
				want := pluginFixture[name]
				if gotPlugins[name] != want {
					t.Errorf("lockfile pluginName for %q = %q, want %q", name, gotPlugins[name], want)
				}
			}
		})
	}
}

// TestAdd_DualMembership_NotInstalledTwice is AC #3 end-to-end: a skill
// reachable through both its directory group and the plugin group is
// installed exactly once when both addressings are requested together.
func TestAdd_DualMembership_NotInstalledTwice(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/plugin-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"skills/engineering,engineering-plugin"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	names := installedSkillNames(t, result)
	sort.Strings(names)
	want := []string{"code-review", "standalone", "tdd"}
	if len(names) != len(want) {
		t.Fatalf("installed skills = %v, want %v (no duplicates)", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("installed skill %d = %q, want %q", i, names[i], want[i])
		}
	}

	// And exactly one lockfile entry per skill, not one per addressing.
	lf, err := skl.ReadLockfile(filepath.Join(projectRoot, ".skl-lock.json"))
	if err != nil {
		t.Fatalf("ReadLockfile() error = %v", err)
	}
	if len(lf) != len(want) {
		t.Errorf("lockfile has %d entries, want %d", len(lf), len(want))
	}
}

// TestAdd_MarketplacePluginGroup_InstallsDeclaredSkills asserts the
// marketplace.json form: each plugins[] entry declaring skills[] is its
// own --skill-selectable plugin group, with skill paths resolved relative
// to the entry's source directory.
func TestAdd_MarketplacePluginGroup_InstallsDeclaredSkills(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/marketplace-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"frontend-pack,backend-pack"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	names := installedSkillNames(t, result)
	sort.Strings(names)
	want := []string{"api", "react"}
	if len(names) != len(want) {
		t.Fatalf("installed skills = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("installed skill %d = %q, want %q", i, names[i], want[i])
		}
	}

	gotPlugins := lockfilePluginNames(t, projectRoot)
	if gotPlugins["react"] != "frontend-pack" {
		t.Errorf("lockfile pluginName for react = %q, want %q", gotPlugins["react"], "frontend-pack")
	}
	if gotPlugins["api"] != "backend-pack" {
		t.Errorf("lockfile pluginName for api = %q, want %q", gotPlugins["api"], "backend-pack")
	}
}

// TestAdd_PluginGroup_PickerListsBothGroupings is AC #3: the
// multi-skill-without---skill error lists every skill under its directory
// grouping AND its plugin grouping.
func TestAdd_PluginGroup_PickerListsBothGroupings(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	_, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/plugin-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	})
	if err == nil {
		t.Fatal("Add() error = nil, want the multi-skill-without---skill error")
	}
	for _, want := range []string{
		"skills/engineering:",
		"skills/writing:",
		"extra:",
		"engineering-plugin (plugin):",
		"tdd", "code-review", "blog", "standalone",
	} {
		if !contains(err.Error(), want) {
			t.Errorf("error %q does not list %q", err.Error(), want)
		}
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".skl-lock.json")); !os.IsNotExist(err) {
		t.Errorf("expected no lockfile to be written when the selection is refused")
	}
}

// TestAdd_PluginGroup_UnknownName_Errors asserts a plugin group name no
// manifest declares is refused, with the discovered skills and groups as
// suggestions.
func TestAdd_PluginGroup_UnknownName_Errors(t *testing.T) {
	projectRoot := t.TempDir()

	_, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/plugin-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"no-such-plugin"},
	})
	if err == nil {
		t.Fatal("Add() error = nil, want error for an unknown plugin group name")
	}
	if !contains(err.Error(), "no-such-plugin") {
		t.Errorf("error %q does not name the unknown value", err.Error())
	}
	if !contains(err.Error(), "tdd") {
		t.Errorf("error %q does not list discovered skills as suggestions", err.Error())
	}
}

// TestAdd_PluginNameCollision_IndividualSkillWins is AC #5 end-to-end:
// when an individual skill's name matches the plugin group's name, the
// individual skill wins and the plugin's other members are not swept in.
func TestAdd_PluginNameCollision_IndividualSkillWins(t *testing.T) {
	fakeHome(t) // universal only
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/plugin-name-collision-source",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
		RequestedSkills:   []string{"dup"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	names := installedSkillNames(t, result)
	if len(names) != 1 || names[0] != "dup" {
		t.Fatalf("installed skills = %v, want exactly the individual skill dup", names)
	}

	// The individual skill "dup" is not a plugin member, so its entry
	// records no pluginName; the plugin member "a" was not installed.
	gotPlugins := lockfilePluginNames(t, projectRoot)
	if gotPlugins["dup"] != "" {
		t.Errorf("lockfile pluginName for dup = %q, want empty (individual selection, not a plugin member)", gotPlugins["dup"])
	}
}
