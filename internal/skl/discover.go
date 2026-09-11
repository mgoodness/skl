package skl

import (
	"fmt"
	"os"
	"path/filepath"
)

// discoverSkillDir locates the skill within a fetched source. This version
// only supports the single-skill case: a root SKILL.md directly under
// sourceRoot. Multi-skill discovery (skills/ walk, unbounded fallback,
// tree-path sources) is out of scope for this ticket.
func discoverSkillDir(sourceRoot string) (string, error) {
	skillMD := filepath.Join(sourceRoot, "SKILL.md")
	info, err := os.Stat(skillMD)
	switch {
	case os.IsNotExist(err):
		return "", fmt.Errorf("no SKILL.md found at the root of source %q", sourceRoot)
	case err != nil:
		return "", fmt.Errorf("checking for SKILL.md in %q: %w", sourceRoot, err)
	case info.IsDir():
		return "", fmt.Errorf("SKILL.md at the root of source %q is a directory, not a file", sourceRoot)
	}
	return sourceRoot, nil
}
