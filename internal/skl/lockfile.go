package skl

import (
	"encoding/json"
	"fmt"
	"os"
)

// AdapterEntry is a lockfile entry's record of one adapter's install
// destination.
type AdapterEntry struct {
	Path string `json:"path"`
}

// LockEntry is a single skill's tracked state in a lockfile. Identity is
// matched by Source, not by name alone (see ADR-0003 and CONTEXT.md's
// "Expand"/"Conflict" definitions).
type LockEntry struct {
	Source      string                  `json:"source"`
	SourceType  string                  `json:"sourceType"`
	SourceURL   string                  `json:"sourceUrl,omitempty"`
	Ref         string                  `json:"ref,omitempty"`
	SkillPath   string                  `json:"skillPath"`
	ContentHash string                  `json:"contentHash"`
	PluginName  string                  `json:"pluginName,omitempty"`
	Adapters    map[string]AdapterEntry `json:"adapters"`
}

// Lockfile is the in-memory form of a .skl-lock.json (project scope) or
// lock.json (global scope) file: skill name to its tracked entry.
//
// Encoding a Go map with string keys via encoding/json sorts keys
// alphabetically. That's what keeps the on-disk file's entries alphabetized
// by skill name with no extra sorting code.
type Lockfile map[string]LockEntry

// ReadLockfile reads the lockfile at path. A missing file is not an error:
// it returns an empty, non-nil Lockfile, matching the "nothing installed
// yet" state.
func ReadLockfile(path string) (Lockfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Lockfile{}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var lf Lockfile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if lf == nil {
		lf = Lockfile{}
	}
	return lf, nil
}

// WriteLockfile writes lf to path as indented JSON with a trailing newline.
func WriteLockfile(path string, lf Lockfile) error {
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding lockfile: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
