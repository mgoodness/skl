package skl

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// pluginManifestDir is the directory under a source root where
// Claude-plugin manifests live. "Manifest" is reserved for exactly these
// files (see CONTEXT.md) -- skl's own record of what's installed is the
// lockfile.
const pluginManifestDir = ".claude-plugin"

// pluginGroup is one plugin manifest's declared set of skills: the
// manifest's plugin name (the --skill-selectable group value) and the
// source-root-relative, forward-slash paths its skills[] entries resolve
// to. A declared path may name a single skill directory (containing a
// SKILL.md) or a scan root whose nested skill directories all belong to
// the group, mirroring how Claude Code itself treats a plugin manifest's
// skills field.
type pluginGroup struct {
	Name  string
	Paths []string
}

// pluginManifest is the subset of .claude-plugin/plugin.json skl reads.
type pluginManifest struct {
	Name   string   `json:"name"`
	Skills []string `json:"skills"`
}

// marketplacePluginEntry is the subset of a marketplace.json plugins[]
// entry skl reads: the plugin's name, the directory it lives in (Source,
// relative to the marketplace root), and any skill paths it declares
// (relative to that source directory, per Claude's marketplace schema).
type marketplacePluginEntry struct {
	Name   string   `json:"name"`
	Source string   `json:"source"`
	Skills []string `json:"skills"`
}

// marketplaceManifest is the subset of .claude-plugin/marketplace.json
// skl reads.
type marketplaceManifest struct {
	Plugins []marketplacePluginEntry `json:"plugins"`
}

// discoverPluginGroups reads the plugin manifests at sourceRoot's
// .claude-plugin directory, if any, returning the plugin groups they
// declare: plugin.json's own skills[] first, then each marketplace.json
// plugins[] entry that declares skills[] of its own, with the first
// declaration of a given plugin name winning. An absent manifest yields
// nil, nil -- a manifest is optional metadata on top of the discovery
// walk, never required. A manifest that exists but can't be read or
// parsed is an error rather than a silent skip: it is load-bearing for
// selection (--skill <plugin-name>) and for the lockfile's pluginName, so
// proceeding without it would mis-record installs.
func discoverPluginGroups(sourceRoot string) ([]pluginGroup, error) {
	var groups []pluginGroup
	seen := map[string]bool{}
	// add appends a group unless it's unusable (no name, no declared
	// paths) or a group of the same name was already declared.
	add := func(g pluginGroup) {
		if g.Name == "" || len(g.Paths) == 0 || seen[g.Name] {
			return
		}
		seen[g.Name] = true
		groups = append(groups, g)
	}

	manifestAt := func(name string) string {
		return filepath.Join(sourceRoot, pluginManifestDir, name)
	}

	pluginDecl, found, err := readPluginJSON[pluginManifest](manifestAt("plugin.json"))
	if err != nil {
		return nil, err
	}
	if found {
		add(pluginGroup{Name: strings.TrimSpace(pluginDecl.Name), Paths: cleanManifestPaths("", pluginDecl.Skills)})
	}

	marketplace, found, err := readPluginJSON[marketplaceManifest](manifestAt("marketplace.json"))
	if err != nil {
		return nil, err
	}
	if found {
		for _, entry := range marketplace.Plugins {
			add(pluginGroup{Name: strings.TrimSpace(entry.Name), Paths: cleanManifestPaths(entry.Source, entry.Skills)})
		}
	}

	return groups, nil
}

// readPluginJSON reads and unmarshals the JSON file at path into T. A
// missing file is not an error: it returns the zero value and false,
// matching the "no manifest here" state. Any other read error, or a
// parse error, is returned wrapped with the path.
func readPluginJSON[T any](p string) (T, bool, error) {
	var out T
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return out, false, nil
		}
		return out, false, fmt.Errorf("reading %s: %w", p, err)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, false, fmt.Errorf("parsing %s: %w", p, err)
	}
	return out, true, nil
}

// cleanManifestPaths resolves manifest-declared skill paths into
// source-root-relative, forward-slash form: each entry is relative to
// base (the plugin's source directory for a marketplace entry, the
// source root for plugin.json). Entries are cleaned lexically; one that
// cleans to "." or escapes the source root matches no discovered skill
// and is dropped, since declared paths are only ever matched against
// resolved paths, never read from or written to directly.
func cleanManifestPaths(base string, skills []string) []string {
	paths := make([]string, 0, len(skills))
	for _, s := range skills {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		p := path.Join(filepath.ToSlash(base), s)
		if p == "." || p == ".." || strings.HasPrefix(p, "../") {
			continue
		}
		paths = append(paths, p)
	}
	return paths
}
