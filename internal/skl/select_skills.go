package skl

import (
	"fmt"
	"sort"
	"strings"
)

// selectSkills computes which of the skills discoverSkills found should
// actually be installed, given the raw --skill flag values (which may be
// comma-separated, repeated, both, or contain the literal "*").
//
// Resolution follows the same no-silent-guessing rule ResolveAdapters
// applies to --agent:
//
//   - --skill '*' (alone or combined with anything else): every
//     discovered skill is installed, bypassing the multi-skill-without-
//     --skill error entirely.
//   - No --skill given, exactly one discovered skill: it's installed
//     regardless (an unambiguous single-skill source is frictionless,
//     per #1).
//   - No --skill given, 2+ discovered skills: a hard error listing every
//     discovered skill nested under its directory group, doubling as a
//     picker (#13).
//   - A --skill value that names a discovered skill: that skill is
//     installed; an unknown value is a hard error listing the discovered
//     names and groups as suggestions, never silently ignored even when
//     there was only one skill to begin with. Repeated names are
//     de-duplicated, first-seen order preserved.
//   - A --skill value that names a directory group (a source-relative
//     path like "skills/engineering", or an ancestor directory of one):
//     every discovered skill whose Group is that path -- or nested under
//     it -- is installed, so a shared parent path selects its whole tree
//     (#13). Group values mix freely with individual names in the same
//     list. On a name collision between an individual skill name and a
//     group value, the individual skill wins (#1).
//   - A --skill value that names a plugin group (the plugin name a
//     source's .claude-plugin manifest declares): every discovered skill
//     stamped with that PluginName is installed (#14), alongside -- never
//     instead of -- the directory groups, and mixed just as freely. A
//     skill belonging to both a directory group and a plugin group is
//     selected exactly once: matched by resolved path, never installed
//     twice.
func selectSkills(discovered []discoveredSkill, requestedSkillValues []string) ([]discoveredSkill, error) {
	requested := normalizeFlagValues(requestedSkillValues)

	for _, r := range requested {
		if r == "*" {
			return discovered, nil
		}
	}

	if len(requested) == 0 {
		if len(discovered) == 1 {
			return discovered, nil
		}
		return nil, fmt.Errorf(
			"multiple skills found in source; specify --skill with a skill name, a directory group (e.g. \"skills/engineering\"), a plugin group name, or \"*\" for all of them:\n%s",
			renderSkillPicker(discovered),
		)
	}

	byName := make(map[string]discoveredSkill, len(discovered))
	for _, d := range discovered {
		byName[d.Name] = d
	}

	result := make([]discoveredSkill, 0, len(requested))
	seen := make(map[string]bool, len(discovered))
	// add appends a selected skill to result unless already selected by an
	// earlier value in this list, so a skill named by name and by group (or
	// twice) is never duplicated in the install set (#1 story 19).
	add := func(d discoveredSkill) {
		if seen[d.Dir] {
			return
		}
		seen[d.Dir] = true
		result = append(result, d)
	}
	for _, r := range requested {
		if d, ok := byName[r]; ok {
			// An individual skill name beats a directory or plugin group of
			// the same name (#1's collision rule): select only this skill,
			// don't sweep the group's other members in.
			add(d)
			continue
		}

		matched := false
		for _, d := range discovered {
			if !groupMatches(d, r) && d.PluginName != r {
				continue
			}
			matched = true
			add(d)
		}
		if !matched {
			return nil, fmt.Errorf(
				"unknown skill, directory group, or plugin group %q; skills found in source:\n%s",
				r, renderSkillPicker(discovered),
			)
		}
	}
	return result, nil
}

// groupMatches reports whether value v selects d as a directory group: v
// equals d's Group path exactly, or names an ancestor directory of it
// (v is a path prefix of Group ending at a "/" boundary).
func groupMatches(d discoveredSkill, v string) bool {
	if d.Group == "" || v == "" || strings.ContainsAny(v, "\\") {
		return false
	}
	if v == d.Group {
		return true
	}
	return strings.HasPrefix(d.Group, v+"/")
}

// renderSkillPicker formats discovered's skills as a picker nested under
// their groups: one group line per directory group and per plugin group,
// skill names indented beneath (sorted for determinism). Each directory
// group line is the source-relative path a user can pass to --skill to
// select the whole group; each plugin group line is the plugin name,
// suffixed "(plugin)" to distinguish it from directory-group paths. A
// dual-member skill (both a directory group and a plugin group, #14) is
// listed under both, since either addressing selects it. Each indented
// name selects that one skill (except skills with no named parent
// directory, shown under a literal "(source root)" placeholder that is
// not itself a selectable value) (#1 stories 11 and 18, #13).
func renderSkillPicker(discovered []discoveredSkill) string {
	namesByGroup := make(map[string][]string)
	for _, d := range discovered {
		g := d.Group
		if g == "" || g == "." {
			g = "(source root)"
		}
		namesByGroup[g] = append(namesByGroup[g], d.Name)
	}

	groups := make([]string, 0, len(namesByGroup))
	for g := range namesByGroup {
		groups = append(groups, g)
	}
	sort.Strings(groups)

	var b strings.Builder
	for _, g := range groups {
		names := namesByGroup[g]
		sort.Strings(names)
		fmt.Fprintf(&b, "  %s:\n", g)
		for _, n := range names {
			fmt.Fprintf(&b, "    %s\n", n)
		}
	}

	namesByPlugin := make(map[string][]string)
	for _, d := range discovered {
		if d.PluginName != "" {
			namesByPlugin[d.PluginName] = append(namesByPlugin[d.PluginName], d.Name)
		}
	}
	pluginNames := make([]string, 0, len(namesByPlugin))
	for name := range namesByPlugin {
		pluginNames = append(pluginNames, name)
	}
	sort.Strings(pluginNames)
	for _, name := range pluginNames {
		names := namesByPlugin[name]
		sort.Strings(names)
		fmt.Fprintf(&b, "  %s (plugin):\n", name)
		for _, n := range names {
			fmt.Fprintf(&b, "    %s\n", n)
		}
	}

	return strings.TrimSuffix(b.String(), "\n")
}

// normalizeFlagValues flattens a raw multi-value flag (which may mix
// comma-separated entries and repeated-flag entries) into individual,
// trimmed values, preserving first-seen order. Shared by --skill
// (selectSkills) and --agent (ResolveAdapters), whose flags follow the
// identical comma-separated/repeated/wildcard grammar.
func normalizeFlagValues(raw []string) []string {
	var out []string
	for _, r := range raw {
		for _, part := range strings.Split(r, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			out = append(out, part)
		}
	}
	return out
}
