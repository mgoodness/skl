package skl_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mgoodness/skl/internal/skl"
)

// fakeHome points $HOME and $XDG_CONFIG_HOME at fresh temp directories and
// creates detection markers for the given adapter names ("claude-code"
// and/or "kit"), giving each test full control over what DetectedAdapters
// reports, independent of the actual machine running the test. Shared by
// every internal/skl test that needs deterministic adapter detection.
func fakeHome(t *testing.T, detected ...string) {
	t.Helper()
	home := t.TempDir()
	configHome := t.TempDir()
	for _, name := range detected {
		switch name {
		case "claude-code":
			if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
				t.Fatalf("creating fake ~/.claude: %v", err)
			}
		case "kit":
			if err := os.MkdirAll(filepath.Join(configHome, "kit"), 0o755); err != nil {
				t.Fatalf("creating fake $XDG_CONFIG_HOME/kit: %v", err)
			}
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)
}

func TestAdd_DefaultAdapters_ZeroDetected_InstallsToUniversalOnly(t *testing.T) {
	fakeHome(t) // neither claude-code nor kit detected
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:      "testdata/fixtures/simple-skill",
		ProjectRoot: projectRoot,
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if len(result.Adapters) != 1 {
		t.Fatalf("result.Adapters = %v, want exactly 1 entry", result.Adapters)
	}
	if _, ok := result.Adapters["universal"]; !ok {
		t.Errorf("result.Adapters missing \"universal\"")
	}
	assertNoDestination(t, projectRoot, ".claude/skills/simple-skill")
	assertNoDestination(t, projectRoot, ".kit/skills/simple-skill")
}

func TestAdd_DefaultAdapters_OneDetected_InstallsToUniversalPlusThatOne(t *testing.T) {
	fakeHome(t, "claude-code")
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:      "testdata/fixtures/simple-skill",
		ProjectRoot: projectRoot,
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if len(result.Adapters) != 2 {
		t.Fatalf("result.Adapters = %v, want exactly 2 entries", result.Adapters)
	}
	for _, want := range []string{"universal", "claude-code"} {
		if _, ok := result.Adapters[want]; !ok {
			t.Errorf("result.Adapters missing %q", want)
		}
	}
	assertNoDestination(t, projectRoot, ".kit/skills/simple-skill")
}

func TestAdd_DefaultAdapters_TwoOrMoreDetected_Errors(t *testing.T) {
	fakeHome(t, "claude-code", "kit")
	projectRoot := t.TempDir()

	_, err := skl.Add(skl.AddOptions{
		Source:      "testdata/fixtures/simple-skill",
		ProjectRoot: projectRoot,
	})
	if err == nil {
		t.Fatalf("Add() error = nil, want error when 2+ adapters detected and no --agent given")
	}
	for _, want := range []string{"claude-code", "kit"} {
		if !contains(err.Error(), want) {
			t.Errorf("error %q does not list detected adapter %q as a pick-list entry", err.Error(), want)
		}
	}
	assertNoDestination(t, projectRoot, ".claude/skills/simple-skill")
	assertNoDestination(t, projectRoot, ".kit/skills/simple-skill")
	assertNoDestination(t, projectRoot, ".agents/skills/simple-skill")
}

func TestAdd_ExplicitAdapters_UnknownName_Errors(t *testing.T) {
	fakeHome(t, "claude-code")
	projectRoot := t.TempDir()

	_, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"not-a-real-adapter"},
	})
	if err == nil {
		t.Fatalf("Add() error = nil, want error for unknown adapter name")
	}
	if !contains(err.Error(), "universal") || !contains(err.Error(), "claude-code") {
		t.Errorf("error %q does not list detected adapters as suggestions", err.Error())
	}
}

func TestAdd_ExplicitAdapters_UndetectedName_Errors(t *testing.T) {
	fakeHome(t) // kit not detected
	projectRoot := t.TempDir()

	_, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"kit"},
	})
	if err == nil {
		t.Fatalf("Add() error = nil, want error for known-but-undetected adapter")
	}
	if !contains(err.Error(), "universal") {
		t.Errorf("error %q does not list detected adapters as suggestions", err.Error())
	}
	assertNoDestination(t, projectRoot, ".kit/skills/simple-skill")
}

func TestAdd_ExplicitAdapters_CommaSeparatedAndRepeatedAreEquivalent(t *testing.T) {
	fakeHome(t, "claude-code", "kit")

	commaRoot := t.TempDir()
	commaResult, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       commaRoot,
		RequestedAdapters: []string{"claude-code,kit"},
	})
	if err != nil {
		t.Fatalf("Add() comma-separated error = %v", err)
	}

	repeatedRoot := t.TempDir()
	repeatedResult, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       repeatedRoot,
		RequestedAdapters: []string{"claude-code", "kit"},
	})
	if err != nil {
		t.Fatalf("Add() repeated-flag error = %v", err)
	}

	if len(commaResult.Adapters) != len(repeatedResult.Adapters) {
		t.Fatalf("comma-separated installed %v, repeated-flag installed %v", commaResult.Adapters, repeatedResult.Adapters)
	}
	for name := range commaResult.Adapters {
		if _, ok := repeatedResult.Adapters[name]; !ok {
			t.Errorf("adapter %q installed via comma-separated but not via repeated flags", name)
		}
	}
	if _, ok := commaResult.Adapters["universal"]; ok {
		t.Errorf("explicit --agent claude-code,kit should not implicitly include universal, got %v", commaResult.Adapters)
	}
}

func TestAdd_WildcardAdapters_InstallsToUniversalPlusAllDetected_NeverErrors(t *testing.T) {
	fakeHome(t, "claude-code", "kit")
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v, want no error for --agent '*' even with 2+ adapters detected", err)
	}

	for _, want := range []string{"universal", "claude-code", "kit"} {
		if _, ok := result.Adapters[want]; !ok {
			t.Errorf("result.Adapters missing %q", want)
		}
	}
}

func TestAdd_WildcardAdapters_NeverTargetsUndetectedAdapter(t *testing.T) {
	fakeHome(t, "claude-code") // kit not detected
	projectRoot := t.TempDir()

	result, err := skl.Add(skl.AddOptions{
		Source:            "testdata/fixtures/simple-skill",
		ProjectRoot:       projectRoot,
		RequestedAdapters: []string{"*"},
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if _, ok := result.Adapters["kit"]; ok {
		t.Errorf("result.Adapters = %v, should never include undetected adapter %q", result.Adapters, "kit")
	}
	assertNoDestination(t, projectRoot, ".kit/skills/simple-skill")
}

func assertNoDestination(t *testing.T, projectRoot, relPath string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(relPath))); !os.IsNotExist(err) {
		t.Errorf("expected %s to not exist, stat err = %v", relPath, err)
	}
}
