package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version, commit, and date are set at build time via -ldflags (see
// .goreleaser.yaml). They default to "dev"/"none"/"unknown" for `go build`
// and `go run` invocations that don't inject them.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Execute builds a fresh root command and runs it. It is the single entry
// point main.go calls.
func Execute() error {
	return NewRootCmd().Execute()
}

// NewRootCmd builds a fresh command tree. Building fresh (rather than a
// package-level var) keeps the tree free of shared state, which is what
// lets tests build an isolated tree per test case.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "skl",
		Short:         "Install Agent Skills across multiple coding-agent clients",
		Version:       fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newAddCmd())
	root.AddCommand(newListCmd())

	return root
}
