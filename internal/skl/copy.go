package skl

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFiles materializes files (as produced by readSkillFiles) at dst. Per
// ADR-0001, this is a real copy, never a symlink.
//
// Any existing contents at dst are removed first. Destination-collision
// handling (refusing when dst is already non-empty, honoring --force) is
// out of scope for this ticket and lands in a later one; until then, Add
// always overwrites cleanly rather than merging with stale contents.
func writeFiles(files []skillFile, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("removing existing %s: %w", dst, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}

	for _, f := range files {
		target := filepath.Join(dst, filepath.FromSlash(f.RelPath))
		if f.IsDir {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("creating %s: %w", target, err)
			}
			continue
		}
		if err := os.WriteFile(target, f.Data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", target, err)
		}
	}
	return nil
}
