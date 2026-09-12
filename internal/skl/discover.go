package skl

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// skillsWalkDepth caps how many path segments below a source's top-level
// "skills/" directory findSkillDirs will descend into while looking for
// SKILL.md-containing directories (tier 3 of discoverSkills' priority
// order). A directory containing a SKILL.md deeper than this is only
// found if tier 3 finds nothing at all and tier 4's unbounded fallback
// kicks in.
const skillsWalkDepth = 3

// discoveredSkill pairs a discovered skill's absolute directory with a
// name derived from its own final path segment -- the destination name
// Add installs it under, flattened regardless of how deeply nested the
// directory was within its source -- and with the skill's directory
// group: the source-relative forward-slash path of the directory that
// contains it, whose name a user can pass to --skill to select the whole
// group at once (#13). Group is empty when the skill *is* the source
// root (the whole source is one skill), since there is no deeper parent
// path to name.
//
// PluginName is the skill's plugin group: the name its source's plugin
// manifest (see discoverPluginGroups) declares, when one of that
// manifest's skills[] paths resolves to the skill's directory (or a
// parent of it) -- also a selectable --skill value (#14). Empty when the
// source has no plugin manifest covering the skill. A skill may belong
// to a directory group and a plugin group at once; both address the same
// skill, matched by resolved path.
type discoveredSkill struct {
	Name       string
	Dir        string
	Group      string
	PluginName string
}

// discoverSkills locates every skill within rs, following the priority
// order fixed by #1's spec:
//
//  1. A tree-path source (rs.SkillPath != ".") already names its skill
//     directory authoritatively -- resolveGitHubSource points rs.Dir at
//     it directly (see resolveGitHubSource) -- so rs.Dir *is* the skill:
//     no further discovery walk happens, it's either a valid skill
//     directory or an error.
//  2. Otherwise, if rs.Dir itself has a root SKILL.md, the whole source
//     is a single skill.
//  3. Otherwise, walk a top-level "skills/" directory, up to depth 3,
//     collecting every directory containing a SKILL.md.
//  4. Otherwise (no "skills/" directory, or it contains nothing found by
//     tier 3), fall back to an unbounded recursive search for SKILL.md
//     anywhere under rs.Dir.
//
// On top of the walk, Plugin Manifest Discovery (#14) applies whenever
// rs.Dir has a .claude-plugin/plugin.json or marketplace.json: the
// manifest's declared skills[] paths contribute plugin groups, extend
// discovery with skill directories the walk missed (outside "skills/",
// or deeper than its depth cap), and stamp each covered skill's
// PluginName. A tree-path source (tier 1) has no manifest discovery: its
// path is authoritative, naming exactly one skill.
//
// Results are sorted by Dir for deterministic ordering (e.g. in the
// multi-skill-without-a-selection error listing).
func discoverSkills(rs resolvedSource) ([]discoveredSkill, error) {
	if rs.SkillPath != "." {
		return discoverRootSkill(rs.Dir)
	}

	var dirs []string
	if hasSkillMD(rs.Dir) {
		dirs = []string{rs.Dir}
	} else {
		found, err := discoverWalkedSkillDirs(rs.Dir)
		if err != nil {
			return nil, err
		}
		dirs = found
	}

	groups, err := discoverPluginGroups(rs.Dir)
	if err != nil {
		return nil, err
	}
	dirs, err = extendWithDeclaredSkillDirs(dirs, groups, rs.Dir)
	if err != nil {
		return nil, err
	}

	if len(dirs) == 0 {
		return nil, fmt.Errorf("no SKILL.md found in %q", rs.Dir)
	}

	skills := toDiscoveredSkills(dirs, rs.Dir)
	assignPluginNames(skills, groups, rs.Dir)
	return skills, nil
}

// discoverWalkedSkillDirs is the tier 3/4 walk: a depth-capped pass over
// a top-level "skills/" directory if it yields anything, else an
// unbounded search under sourceRoot.
func discoverWalkedSkillDirs(sourceRoot string) ([]string, error) {
	skillsDir := filepath.Join(sourceRoot, "skills")
	if info, err := os.Stat(skillsDir); err == nil && info.IsDir() {
		found, err := findSkillDirs(skillsDir, skillsWalkDepth)
		if err != nil {
			return nil, fmt.Errorf("walking %q: %w", skillsDir, err)
		}
		if len(found) > 0 {
			return found, nil
		}
	}

	return findSkillDirs(sourceRoot, -1)
}

// discoverRootSkill is the single-skill case: sourceRoot itself must have
// a root SKILL.md file (not a directory of that name).
func discoverRootSkill(sourceRoot string) ([]discoveredSkill, error) {
	skillMD := filepath.Join(sourceRoot, "SKILL.md")
	info, err := os.Stat(skillMD)
	switch {
	case os.IsNotExist(err):
		return nil, fmt.Errorf("no SKILL.md found at the root of source %q", sourceRoot)
	case err != nil:
		return nil, fmt.Errorf("checking for SKILL.md in %q: %w", sourceRoot, err)
	case info.IsDir():
		return nil, fmt.Errorf("SKILL.md at the root of source %q is a directory, not a file", sourceRoot)
	}
	return []discoveredSkill{{Name: filepath.Base(sourceRoot), Dir: sourceRoot}}, nil
}

// hasSkillMD reports whether dir has a root-level SKILL.md file (not a
// directory of that name).
func hasSkillMD(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && !info.IsDir()
}

// findSkillDirs walks root looking for directories containing a
// SKILL.md, returning their absolute paths in lexical order. maxDepth
// caps how many path segments below root a SKILL.md-containing directory
// may be found at; -1 means unbounded. A ".git" directory, if
// encountered, is never descended into: it's never a source of skills
// and can be large.
func findSkillDirs(root string, maxDepth int) ([]string, error) {
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == ".git" {
			return filepath.SkipDir
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		depth := 0
		if rel != "." {
			depth = len(strings.Split(filepath.ToSlash(rel), "/"))
		}
		if maxDepth >= 0 && depth > maxDepth {
			return filepath.SkipDir
		}

		if hasSkillMD(path) {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(dirs)
	return dirs, nil
}

// toDiscoveredSkills converts a sorted list of skill directories into
// discoveredSkill values, deriving each one's flattened Name from its own
// final path segment and its directory Group from the path of the
// directory that contains it, relative to sourceRoot (see Group's
// comment on discoveredSkill).
func toDiscoveredSkills(dirs []string, sourceRoot string) []discoveredSkill {
	out := make([]discoveredSkill, len(dirs))
	for i, d := range dirs {
		group := ""
		if rel, err := filepath.Rel(sourceRoot, d); err == nil && rel != "." {
			group = filepath.ToSlash(filepath.Dir(rel))
		}
		out[i] = discoveredSkill{Name: filepath.Base(d), Dir: d, Group: group}
	}
	return out
}

// extendWithDeclaredSkillDirs appends skill directories the plugin
// groups declare that the priority-ordered walk didn't already find: a
// manifest can point at skill directories outside the walked "skills/"
// tree, or deeper than its depth cap, and every skill its manifest
// declares must be installable via --skill <plugin-name> and --skill '*'
// (#14). A declared path that is itself a skill directory contributes
// just that directory; one without a SKILL.md is scanned as a group
// root, like Claude Code scans a plugin manifest's skills paths. A
// declared path missing from disk (or naming a file) contributes
// nothing -- tolerated, since the walk already found whatever the source
// holds elsewhere. The returned list is re-sorted for deterministic
// ordering.
func extendWithDeclaredSkillDirs(dirs []string, groups []pluginGroup, sourceRoot string) ([]string, error) {
	known := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		known[d] = true
	}

	var added []string
	for _, g := range groups {
		for _, p := range g.Paths {
			abs := filepath.Join(sourceRoot, filepath.FromSlash(p))
			if known[abs] {
				continue
			}
			if info, err := os.Stat(abs); err != nil || !info.IsDir() {
				continue
			}
			found, err := findSkillDirs(abs, -1)
			if err != nil {
				return nil, fmt.Errorf("scanning plugin-declared skills path %q: %w", p, err)
			}
			for _, d := range found {
				if known[d] {
					continue
				}
				known[d] = true
				added = append(added, d)
			}
		}
	}
	if len(added) == 0 {
		return dirs, nil
	}
	dirs = append(dirs, added...)
	sort.Strings(dirs)
	return dirs, nil
}

// assignPluginNames stamps each skill's PluginName: the name of the
// first plugin group whose declared paths resolve to the skill's
// directory (or contain it, for a scan-root path) -- matched by resolved
// path (#14), so a skill reachable through both a directory group and a
// plugin group is still one skill. Groups and their paths are processed
// in manifest declaration order, and the first matching group wins when
// manifests overlap.
func assignPluginNames(skills []discoveredSkill, groups []pluginGroup, sourceRoot string) {
	for i := range skills {
		rel, err := filepath.Rel(sourceRoot, skills[i].Dir)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		for _, g := range groups {
			for _, p := range g.Paths {
				if rel == p || strings.HasPrefix(rel, p+"/") {
					skills[i].PluginName = g.Name
					break
				}
			}
			if skills[i].PluginName != "" {
				break
			}
		}
	}
}
