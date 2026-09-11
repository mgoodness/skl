package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mgoodness/skl/cmd"
)

// setUpFakeProject fakes $HOME/$XDG_CONFIG_HOME with markers for every
// adapter, chdirs into a fresh project root, and writes a minimal
// single-skill fixture at ./my-skill, returning the project root plus the
// fake home and XDG config home directories (needed by tests that also
// assert against global-scope destinations). This keeps every test's
// expectations independent of what's actually installed on the machine
// running it.
func setUpFakeProject(t *testing.T) (projectRoot, home, configHome string) {
	t.Helper()

	home = t.TempDir()
	configHome = t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatalf("creating fake ~/.claude: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(configHome, "kit"), 0o755); err != nil {
		t.Fatalf("creating fake $XDG_CONFIG_HOME/kit: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	projectRoot = t.TempDir()
	t.Chdir(projectRoot)

	fixture := filepath.Join(projectRoot, "my-skill")
	if err := os.MkdirAll(fixture, 0o755); err != nil {
		t.Fatalf("creating fixture dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "SKILL.md"), []byte("# My Skill\n"), 0o644); err != nil {
		t.Fatalf("writing fixture SKILL.md: %v", err)
	}

	return projectRoot, home, configHome
}

func TestAddCommand_InstallsLocalSkillIntoProjectRoot(t *testing.T) {
	projectRoot, _, _ := setUpFakeProject(t)

	buf := new(bytes.Buffer)
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"add", "./my-skill", "--agent", "*"})

	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("Execute() error = %v, output: %s", err, buf.String())
	}

	if !strings.Contains(buf.String(), "my-skill") {
		t.Errorf("output = %q, want it to mention the installed skill name", buf.String())
	}

	for _, dir := range []string{
		filepath.Join(".claude", "skills", "my-skill"),
		filepath.Join(".kit", "skills", "my-skill"),
		filepath.Join(".agents", "skills", "my-skill"),
	} {
		if _, err := os.Stat(filepath.Join(projectRoot, dir, "SKILL.md")); err != nil {
			t.Errorf("expected %s/SKILL.md to exist: %v", dir, err)
		}
	}

	if _, err := os.Stat(filepath.Join(projectRoot, ".skl-lock.json")); err != nil {
		t.Errorf("expected .skl-lock.json to exist: %v", err)
	}
}

func TestAddCommand_MissingSourceArgument_Errors(t *testing.T) {
	buf := new(bytes.Buffer)
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"add"})

	if err := root.ExecuteContext(t.Context()); err == nil {
		t.Fatal("Execute() error = nil, want error for missing <source> argument")
	}
}

func TestAddCommand_UnknownAgentFlag_Errors(t *testing.T) {
	setUpFakeProject(t)

	buf := new(bytes.Buffer)
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"add", "./my-skill", "--agent", "not-a-real-adapter"})

	if err := root.ExecuteContext(t.Context()); err == nil {
		t.Fatal("Execute() error = nil, want error for unknown --agent value")
	}
}

func TestAddCommand_GlobalFlag_InstallsIntoGlobalDestinationsAndGlobalLockfile(t *testing.T) {
	projectRoot, home, configHome := setUpFakeProject(t)
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	buf := new(bytes.Buffer)
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"add", "./my-skill", "--global", "--agent", "*"})

	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("Execute() error = %v, output: %s", err, buf.String())
	}

	for _, dir := range []string{
		filepath.Join(home, ".claude", "skills", "my-skill"),
		filepath.Join(configHome, "kit", "skills", "my-skill"),
		filepath.Join(home, ".agents", "skills", "my-skill"),
	} {
		if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
			t.Errorf("expected %s/SKILL.md to exist: %v", dir, err)
		}
	}

	if _, err := os.Stat(filepath.Join(projectRoot, ".skl-lock.json")); !os.IsNotExist(err) {
		t.Errorf("expected no project-scoped .skl-lock.json with --global, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataHome, "skl", "lock.json")); err != nil {
		t.Errorf("expected global lock.json to exist: %v", err)
	}
}
