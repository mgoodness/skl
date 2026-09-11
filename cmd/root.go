package cmd

import (
	"github.com/spf13/cobra"
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
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newAddCmd())

	return root
}
