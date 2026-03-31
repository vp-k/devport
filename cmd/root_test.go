package cmd

import (
	"testing"
)

func TestExecute(t *testing.T) {
	// Execute with --version should succeed without calling os.Exit.
	rootCmd.SetArgs([]string{"--version"})
	Execute() // must not panic or exit
}
