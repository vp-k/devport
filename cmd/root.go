package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/vp-k/devport/internal/output"
)

var version = "0.2.0"

// Out is the global printer used by all commands. Tests can replace it.
var Out = output.NewPlain(os.Stdout)

var rootCmd = &cobra.Command{
	Use:     "devport",
	Short:   "Conflict-free port allocation for local development",
	Version: version,
}

// Execute runs the CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		cmdOsExit(1)
	}
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
}
