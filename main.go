// Command skl installs Agent Skills (SKILL.md-based capabilities) across
// multiple coding-agent clients.
package main

import (
	"fmt"
	"os"

	"github.com/mgoodness/skl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
