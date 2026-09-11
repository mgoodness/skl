package cmd

import (
	"fmt"
	"strings"

	"github.com/mgoodness/skl/internal/skl"
	"github.com/spf13/cobra"
)

// newListCmd wires the "list" command's flags to internal/skl.List. It
// contains no business logic of its own.
//
// There is deliberately no --global flag here: List always shows both
// project and global scope, each entry labeled by scope. --agent also
// deliberately differs from add's: it takes a single adapter name (no
// comma-separated list, no repeated flags, no "*") since it filters an
// existing listing rather than choosing install targets.
func newListCmd() *cobra.Command {
	var agent string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed skills across project and global scope",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := skl.List(skl.ListOptions{Agent: agent})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(entries) == 0 {
				_, _ = fmt.Fprintln(out, "No skills installed.")
				return nil
			}

			for _, entry := range entries {
				_, _ = fmt.Fprintf(out, "%s (%s)\n", entry.Name, entry.Scope)
				for _, a := range entry.Adapters {
					_, _ = fmt.Fprintf(out, "  %s: %s%s\n", a.Name, a.Path, annotate(a))
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&agent, "agent", "a", "", "filter the listing to only this adapter's entries")

	return cmd
}

// annotate renders a's drift annotations as a suffix: " (not detected)",
// " (missing)", both together as " (not detected, missing)", or "" when
// neither applies.
func annotate(a skl.ListAdapterEntry) string {
	var notes []string
	if a.NotDetected {
		notes = append(notes, "not detected")
	}
	if a.Missing {
		notes = append(notes, "missing")
	}
	if len(notes) == 0 {
		return ""
	}
	return " (" + strings.Join(notes, ", ") + ")"
}
