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
//     discovered skill's name, doubling as a picker.
//   - --skill <name>[,<name>...], any discovered count (including one):
//     every named skill must match a discovered skill's name; an unknown
//     name is a hard error listing the discovered names as suggestions,
//     never silently ignored even when there was only one skill to begin
//     with. Repeated names are de-duplicated, first-seen order preserved.
func selectSkills(discovered []discoveredSkill, requestedSkillValues []string) ([]discoveredSkill, error) {
	requested := normalizeFlagValues(requestedSkillValues)

	for _, r := range requested {
		if r == "*" {
			return discovered, nil
		}
	}

	names := discoveredNames(discovered)
	if len(requested) == 0 {
		if len(discovered) == 1 {
			return discovered, nil
		}
		return nil, fmt.Errorf(
			"multiple skills found in source (%s); specify --skill to choose one or more, or --skill '*' for all of them",
			strings.Join(names, ", "),
		)
	}

	byName := make(map[string]discoveredSkill, len(discovered))
	for _, d := range discovered {
		byName[d.Name] = d
	}

	seen := make(map[string]bool, len(requested))
	result := make([]discoveredSkill, 0, len(requested))
	for _, r := range requested {
		d, ok := byName[r]
		if !ok {
			return nil, fmt.Errorf(
				"unknown skill %q; skills found in source: %s",
				r, strings.Join(names, ", "),
			)
		}
		if !seen[d.Name] {
			seen[d.Name] = true
			result = append(result, d)
		}
	}
	return result, nil
}

// discoveredNames returns discovered's skill names, sorted, for use in
// error messages that double as a picker.
func discoveredNames(discovered []discoveredSkill) []string {
	names := make([]string, len(discovered))
	for i, d := range discovered {
		names[i] = d.Name
	}
	sort.Strings(names)
	return names
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
