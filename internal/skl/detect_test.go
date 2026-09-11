package skl_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mgoodness/skl/internal/skl"
)

func TestDetectedAdapters_UniversalAlwaysPresent(t *testing.T) {
	home := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	detected := skl.DetectedAdapters()

	assertContains(t, detected, "universal")
	assertNotContains(t, detected, "claude-code")
	assertNotContains(t, detected, "kit")
}

func TestDetectedAdapters_ClaudeCodeDetectedViaHomeDotClaudeDir(t *testing.T) {
	home := t.TempDir()
	configHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatalf("creating fake ~/.claude: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	detected := skl.DetectedAdapters()

	assertContains(t, detected, "claude-code")
	assertNotContains(t, detected, "kit")
}

func TestDetectedAdapters_KitDetectedViaXDGConfigHomeKitDir(t *testing.T) {
	home := t.TempDir()
	configHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(configHome, "kit"), 0o755); err != nil {
		t.Fatalf("creating fake $XDG_CONFIG_HOME/kit: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	detected := skl.DetectedAdapters()

	assertContains(t, detected, "kit")
	assertNotContains(t, detected, "claude-code")
}

func TestDetectedAdapters_KitDetectedViaHomeDotConfigKitDirWhenXDGConfigHomeUnset(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".config", "kit"), 0o755); err != nil {
		t.Fatalf("creating fake ~/.config/kit: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	detected := skl.DetectedAdapters()

	assertContains(t, detected, "kit")
}

func TestDetectedAdapters_NoneOfClaudeCodeOrKitPresent(t *testing.T) {
	home := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	detected := skl.DetectedAdapters()

	if len(detected) != 1 || detected[0] != "universal" {
		t.Errorf("DetectedAdapters() = %v, want [universal]", detected)
	}
}

func TestDetectedAdapters_BothClaudeCodeAndKitPresent(t *testing.T) {
	home := t.TempDir()
	configHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatalf("creating fake ~/.claude: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(configHome, "kit"), 0o755); err != nil {
		t.Fatalf("creating fake $XDG_CONFIG_HOME/kit: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	detected := skl.DetectedAdapters()

	for _, want := range []string{"universal", "claude-code", "kit"} {
		assertContains(t, detected, want)
	}
	if len(detected) != 3 {
		t.Errorf("DetectedAdapters() = %v, want exactly 3 entries", detected)
	}
}

func assertContains(t *testing.T, s []string, v string) {
	t.Helper()
	for _, e := range s {
		if e == v {
			return
		}
	}
	t.Errorf("expected %v to contain %q", s, v)
}

func assertNotContains(t *testing.T, s []string, v string) {
	t.Helper()
	for _, e := range s {
		if e == v {
			t.Errorf("expected %v to not contain %q", s, v)
			return
		}
	}
}
