package skl

import (
	"fmt"
	"os"
	"path/filepath"
)

// globalAdapterDir returns the named adapter's global-scope skills
// directory: ~/.claude/skills for claude-code, $XDG_CONFIG_HOME/kit/skills
// (falling back to ~/.config/kit/skills) for kit, and ~/.agents/skills for
// universal. This mirrors the $XDG_CONFIG_HOME-honoring behavior
// isAdapterDetected already uses for kit, so detection and installation
// agree on where kit's directory lives.
func globalAdapterDir(name string) (string, error) {
	switch name {
	case ClaudeCodeAdapter:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}
		return filepath.Join(home, ".claude", "skills"), nil
	case KitAdapter:
		cfgDir, err := xdgConfigHome()
		if err != nil {
			return "", fmt.Errorf("resolving XDG config home: %w", err)
		}
		return filepath.Join(cfgDir, "kit", "skills"), nil
	case UniversalAdapter:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}
		return filepath.Join(home, ".agents", "skills"), nil
	default:
		return "", fmt.Errorf("internal error: unknown adapter %q", name)
	}
}

// globalLockfilePath returns the global-scope lockfile's path:
// $XDG_DATA_HOME/skl/lock.json when XDG_DATA_HOME is set, falling back to
// ~/.local/share/skl/lock.json when it is unset, per the XDG Base
// Directory spec.
func globalLockfilePath() (string, error) {
	dataHome, err := xdgDataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataHome, "skl", "lock.json"), nil
}

// xdgDataHome resolves the XDG data home directory: $XDG_DATA_HOME if set,
// else $HOME/.local/share. Applied the same way on every OS so it stays
// testable via plain t.Setenv, matching xdgConfigHome's approach.
func xdgDataHome() (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}
