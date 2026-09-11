package skl

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// skillFile is one entry (file or directory) from a skill directory, read
// once so callers needing both a content hash and a copy of the tree (Add
// does both, once per adapter) don't each re-walk the filesystem.
type skillFile struct {
	// RelPath is slash-separated and relative to the skill directory root.
	RelPath string
	IsDir   bool
	Data    []byte
}

// readSkillFiles walks dir once, in lexical order (which filepath.WalkDir
// guarantees), and returns every entry relative to dir.
func readSkillFiles(dir string) ([]skillFile, error) {
	var files []skillFile
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			files = append(files, skillFile{RelPath: rel, IsDir: true})
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		files = append(files, skillFile{RelPath: rel, Data: data})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", dir, err)
	}
	return files, nil
}
