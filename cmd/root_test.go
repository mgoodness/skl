package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mgoodness/skl/cmd"
)

func TestRootCommand_VersionFlag_ReportsBuildInfo(t *testing.T) {
	buf := new(bytes.Buffer)
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--version"})

	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatalf("Execute() error = %v, output: %s", err, buf.String())
	}

	// Defaults set in root.go for un-ldflags'd builds (`go build`, `go run`,
	// and this test binary itself).
	for _, want := range []string{"dev", "commit: none", "built: unknown"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("output = %q, want it to contain %q", buf.String(), want)
		}
	}
}
