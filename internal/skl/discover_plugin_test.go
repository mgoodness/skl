package skl

import (
	"strings"
	"testing"
)

// pluginSourceSkills is discoverSkills' expected output shape for the
// plugin-source fixture: four skills, three of them plugin members.
// standalone lives outside the walked skills/ tree and is only
// discoverable through the manifest's declared path.
func pluginSourceSkills() []discoveredSkill {
	return []discoveredSkill{
		{Name: "standalone", Dir: "testdata/fixtures/plugin-source/extra/standalone", Group: "extra", PluginName: "engineering-plugin"},
		{Name: "code-review", Dir: "testdata/fixtures/plugin-source/skills/engineering/code-review", Group: "skills/engineering", PluginName: "engineering-plugin"},
		{Name: "tdd", Dir: "testdata/fixtures/plugin-source/skills/engineering/tdd", Group: "skills/engineering", PluginName: "engineering-plugin"},
		{Name: "blog", Dir: "testdata/fixtures/plugin-source/skills/writing/blog", Group: "skills/writing"},
	}
}

// TestDiscoverSkills_PluginManifest_AssignsPluginName is AC #1: a
// .claude-plugin/plugin.json at the source root is detected, and its
// declared skills[] paths contribute a plugin group -- recorded per skill
// as PluginName. The declared paths cover both forms: a scan root whose
// nested skill directories all join the group (./skills/engineering) and
// a single skill directory (extra/standalone).
func TestDiscoverSkills_PluginManifest_AssignsPluginName(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/plugin-source", SkillPath: "."}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}

	want := pluginSourceSkills()
	if len(got) != len(want) {
		t.Fatalf("discoverSkills() = %v, want %d skills", got, len(want))
	}
	for i := range want {
		if got[i].Name != want[i].Name || got[i].Group != want[i].Group || got[i].PluginName != want[i].PluginName {
			t.Errorf("discoverSkills()[%d] = {%s %s %s}, want {%s %s %s}",
				i, got[i].Name, got[i].Group, got[i].PluginName,
				want[i].Name, want[i].Group, want[i].PluginName)
		}
	}
}

// TestDiscoverSkills_PluginManifest_ExtendsDiscovery asserts that a
// declared skills[] path outside the walked skills/ tree (extra/standalone)
// is discovered *because* the manifest declares it: the depth-3 skills/
// walk succeeds, so the unbounded fallback never runs, and without the
// manifest extension the skill would be invisible to every selection mode
// including --skill <plugin-name> and --skill '*'.
func TestDiscoverSkills_PluginManifest_ExtendsDiscovery(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/plugin-source", SkillPath: "."}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}

	for _, d := range got {
		if d.Name == "standalone" && d.PluginName != "engineering-plugin" {
			t.Errorf("manifest-declared skill standalone PluginName = %q, want %q", d.PluginName, "engineering-plugin")
		}
	}
}

// TestDiscoverSkills_PluginManifest_DeclaredPathMissingFromDisk_Ignored
// asserts that a declared skills[] path with no skill on disk (skills/ghost)
// is tolerated rather than an error: discovery still succeeds and the
// group simply contains only the skills that actually exist.
func TestDiscoverSkills_PluginManifest_DeclaredPathMissingFromDisk_Ignored(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/plugin-source", SkillPath: "."}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v, want a missing declared path to be ignored", err)
	}
	for _, d := range got {
		if d.Name == "ghost" {
			t.Errorf("discovered unexpected skill %q from a nonexistent declared path", d.Name)
		}
	}
}

// TestDiscoverSkills_MarketplaceManifest_AssignsPluginNames is AC #1 for
// the marketplace.json form: each plugins[] entry declaring skills[]
// contributes its own plugin group, and skill paths are resolved relative
// to the entry's source directory. An entry without skills[] (bare-pack)
// contributes no group.
func TestDiscoverSkills_MarketplaceManifest_AssignsPluginNames(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/marketplace-source", SkillPath: "."}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}

	wantPlugins := map[string]string{
		"react": "frontend-pack",
		"api":   "backend-pack",
	}
	if len(got) != len(wantPlugins) {
		t.Fatalf("discoverSkills() = %v, want %d skills", got, len(wantPlugins))
	}
	for i := range got {
		wantName, ok := wantPlugins[got[i].Name]
		if !ok {
			t.Errorf("discoverSkills() unexpectedly discovered %q", got[i].Name)
			continue
		}
		if got[i].PluginName != wantName {
			t.Errorf("skill %q PluginName = %q, want %q", got[i].Name, got[i].PluginName, wantName)
		}
	}
}

// TestDiscoverSkills_MalformedManifest_Errors asserts a manifest that
// exists but can't be parsed is a hard error, not a silent skip: the
// manifest is load-bearing for selection (--skill <plugin-name>) and for
// the lockfile's pluginName, so proceeding without it would mis-record
// installs.
func TestDiscoverSkills_MalformedManifest_Errors(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/plugin-manifest-malformed-source", SkillPath: "."}

	_, err := discoverSkills(rs)
	if err == nil {
		t.Fatal("discoverSkills() error = nil, want error for a malformed plugin manifest")
	}
	if !strings.Contains(err.Error(), "plugin.json") {
		t.Errorf("error %q does not name the malformed manifest", err.Error())
	}
}

// TestDiscoverSkills_NoManifest_NoPluginName asserts a source without a
// .claude-plugin directory behaves exactly as before: every skill's
// PluginName is empty and discovery succeeds.
func TestDiscoverSkills_NoManifest_NoPluginName(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/grouped-skill-source", SkillPath: "."}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}
	for _, d := range got {
		if d.PluginName != "" {
			t.Errorf("skill %q PluginName = %q, want empty (no manifest)", d.Name, d.PluginName)
		}
	}
}

// TestDiscoverSkills_TreePathSource_NoPluginGroups asserts tier 1's
// authority: a tree-path source's Dir *is* the skill, so no manifest
// discovery happens even when the enclosing fixture root has one -- a
// tree-path selection installs exactly the named skill.
func TestDiscoverSkills_TreePathSource_NoPluginGroups(t *testing.T) {
	rs := resolvedSource{Dir: "testdata/fixtures/plugin-source/skills/engineering/tdd", SkillPath: "skills/engineering/tdd"}

	got, err := discoverSkills(rs)
	if err != nil {
		t.Fatalf("discoverSkills() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "tdd" {
		t.Fatalf("discoverSkills() = %v, want exactly the tdd skill", got)
	}
	if got[0].PluginName != "" {
		t.Errorf("tree-path skill PluginName = %q, want empty (no manifest discovery for tree-path sources)", got[0].PluginName)
	}
}
