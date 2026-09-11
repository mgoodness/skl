package skl

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// hashFiles computes a deterministic content hash over files (in the order
// given, which readSkillFiles produces in lexical path order), keyed by
// each file's RelPath. The result is prefixed "sha256:" to match the
// lockfile's contentHash field format.
func hashFiles(files []skillFile) string {
	h := sha256.New()
	for _, f := range files {
		if f.IsDir {
			continue
		}
		fmt.Fprintf(h, "%s\x00", f.RelPath)
		h.Write(f.Data)
		h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
