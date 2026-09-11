package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mgoodness/skl/cmd"
)

func TestListCommand_ShowsInstalledSkill(t *testing.T) {
	setUpFakeProject(t)

	addCmd := cmd.NewRootCmd()
	addBuf := new(bytes.Buffer)
	addCmd.SetOut(addBuf)
	addCmd.SetErr(addBuf)
	addCmd.SetArgs([]string{"add", "./my-skill", "--agent", "claude-code"})
	if err := addCmd.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("add Execute() error = %v, output: %s", err, addBuf.String())
	}

	listBuf := new(bytes.Buffer)
	listCmd := cmd.NewRootCmd()
	listCmd.SetOut(listBuf)
	listCmd.SetErr(listBuf)
	listCmd.SetArgs([]string{"list"})
	if err := listCmd.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("list Execute() error = %v, output: %s", err, listBuf.String())
	}

	out := listBuf.String()
	for _, want := range []string{"my-skill", "project", "claude-code"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
}

func TestListCommand_NoSkillsInstalled_ReportsNone(t *testing.T) {
	setUpFakeProject(t)

	buf := new(bytes.Buffer)
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"list"})
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("Execute() error = %v, output: %s", err, buf.String())
	}

	if !strings.Contains(buf.String(), "No skills installed") {
		t.Errorf("output = %q, want it to report nothing installed", buf.String())
	}
}

func TestListCommand_UnknownAgentFlag_Errors(t *testing.T) {
	setUpFakeProject(t)

	buf := new(bytes.Buffer)
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"list", "--agent", "not-a-real-adapter"})

	if err := root.ExecuteContext(t.Context()); err == nil {
		t.Fatal("Execute() error = nil, want error for unknown --agent value")
	}
}

func TestListCommand_AgentFlag_FiltersOutput(t *testing.T) {
	setUpFakeProject(t)

	addCmd := cmd.NewRootCmd()
	addBuf := new(bytes.Buffer)
	addCmd.SetOut(addBuf)
	addCmd.SetErr(addBuf)
	addCmd.SetArgs([]string{"add", "./my-skill", "--agent", "*"})
	if err := addCmd.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("add Execute() error = %v, output: %s", err, addBuf.String())
	}

	listBuf := new(bytes.Buffer)
	listCmd := cmd.NewRootCmd()
	listCmd.SetOut(listBuf)
	listCmd.SetErr(listBuf)
	listCmd.SetArgs([]string{"list", "--agent", "kit"})
	if err := listCmd.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("list Execute() error = %v, output: %s", err, listBuf.String())
	}

	out := listBuf.String()
	if !strings.Contains(out, "kit") {
		t.Errorf("output = %q, want it to contain %q", out, "kit")
	}
	if strings.Contains(out, "universal") {
		t.Errorf("output = %q, want it to filter out other adapters when --agent is given", out)
	}
}
