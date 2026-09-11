package skl

import (
	"os"
	"path/filepath"
)

// DetectedAdapters returns the names of every adapter present on the
// current machine, in Adapters order. "universal" has no external
// dependency and is therefore always included.
//
// Detection is filesystem-based: claude-code is detected via ~/.claude
// existing, and kit via ~/.config/kit existing (honoring $XDG_CONFIG_HOME
// when set, per the XDG Base Directory spec, on every OS — not just the
// Unix systems Go's os.UserConfigDir applies it to — so detection stays
// testable and consistent across platforms).
func DetectedAdapters() []string {
	detected := make([]string, 0, len(Adapters))
	for _, a := range Adapters {
		if isAdapterDetected(a.Name) {
			detected = append(detected, a.Name)
		}
	}
	return detected
}

// isAdapterDetected reports whether a single named adapter is present on
// this machine.
func isAdapterDetected(name string) bool {
	switch name {
	case UniversalAdapter:
		return true
	case ClaudeCodeAdapter:
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		return isDir(filepath.Join(home, ".claude"))
	case KitAdapter:
		cfgDir, err := xdgConfigHome()
		if err != nil {
			return false
		}
		return isDir(filepath.Join(cfgDir, "kit"))
	default:
		return false
	}
}

// xdgConfigHome resolves the XDG config home directory: $XDG_CONFIG_HOME
// if set, else $HOME/.config. Unlike os.UserConfigDir, this is applied the
// same way on every OS, matching the spec's explicit "~/.config/kit"
// detection path and making it controllable in tests via plain
// t.Setenv("XDG_CONFIG_HOME", ...) / t.Setenv("HOME", ...) regardless of
// the platform running the test.
func xdgConfigHome() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
