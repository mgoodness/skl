package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mgoodness/skl/cmd"
)

func TestAddCommand_InstallsLocalSkillIntoProjectRoot(t *testing.T) {
	projectRoot := t.TempDir()
	t.Chdir(projectRoot)

	fixture := filepath.Join(projectRoot, "my-skill")
	if err := os.MkdirAll(fixture, 0o755); err != nil {
		t.Fatalf("creating fixture dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "SKILL.md"), []byte("# My Skill\n"), 0o644); err != nil {
		t.Fatalf("writing fixture SKILL.md: %v", err)
	}

	buf := new(bytes.Buffer)
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"add", "./my-skill"})

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
