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
	var skills []string
	var global bool
	var force bool

	cmd := &cobra.Command{
		Use:   "add <source>",
		Short: "Install a skill from a source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := skl.Add(skl.AddOptions{Source: args[0], RequestedAdapters: agents, RequestedSkills: skills, Global: global, Force: force})
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			for _, sk := range result.Skills {
				_, _ = fmt.Fprintf(out, "Installed %s:\n", sk.Name)
				for _, adapter := range skl.Adapters {
					entry, ok := sk.Adapters[adapter.Name]
					if !ok {
						continue
					}
					_, _ = fmt.Fprintf(out, "  %s: %s\n", adapter.Name, entry.Path)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&agents, "agent", "a", nil, "adapter(s) to install for: comma-separated and/or repeated, supports \"*\" for all detected adapters (default: universal plus the detected adapter, if exactly one is detected)")
	cmd.Flags().StringSliceVarP(&skills, "skill", "s", nil, "skill(s) to install from a multi-skill source: skill names and/or directory-group paths (e.g. \"skills/engineering\"), comma-separated and/or repeated, supports \"*\" for every skill found (required when the source contains more than one skill)")
	cmd.Flags().BoolVarP(&global, "global", "g", false, "install into each adapter's global destination and the global lockfile, instead of the project-scoped equivalents")
	cmd.Flags().BoolVar(&force, "force", false, "override a conflicting source or a non-empty destination, replacing what's there instead of refusing")

	return cmd
}
