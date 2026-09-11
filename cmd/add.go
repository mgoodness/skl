package cmd

import (
	"fmt"

	"github.com/mgoodness/skl/internal/skl"
	"github.com/spf13/cobra"
)

// newAddCmd wires the "add" command's flags and arguments to
// internal/skl.Add. It contains no business logic of its own.
func newAddCmd() *cobra.Command {
	var agents []string
	var global bool

	cmd := &cobra.Command{
		Use:   "add <source>",
		Short: "Install a skill from a source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := skl.Add(skl.AddOptions{Source: args[0], RequestedAdapters: agents, Global: global})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Installed %s:\n", result.Name)
			for _, adapter := range skl.Adapters {
				entry, ok := result.Adapters[adapter.Name]
				if !ok {
					continue
				}
				_, _ = fmt.Fprintf(out, "  %s: %s\n", adapter.Name, entry.Path)
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&agents, "agent", "a", nil, "adapter(s) to install for: comma-separated and/or repeated, supports \"*\" for all detected adapters (default: universal plus the detected adapter, if exactly one is detected)")
	cmd.Flags().BoolVarP(&global, "global", "g", false, "install into each adapter's global destination and the global lockfile, instead of the project-scoped equivalents")

	return cmd
}
