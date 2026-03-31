package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/user01/devport/internal/registry"
)

var freeCmd = &cobra.Command{
	Use:   "free [key|port]",
	Short: "Release the port registration for a project",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runFree,
}

var (
	freeFlagAll   bool
	freeFlagForce bool
)

func init() {
	rootCmd.AddCommand(freeCmd)
	freeCmd.Flags().BoolVar(&freeFlagAll, "all", false, "Release all registrations")
	freeCmd.Flags().BoolVar(&freeFlagForce, "force", false, "Skip confirmation prompt")
}

// confirmFn is injectable for testing.
var confirmFn = func(prompt string) bool {
	fmt.Print(prompt)
	var answer string
	fmt.Scanln(&answer)
	return answer == "y" || answer == "Y" || answer == "yes"
}

func runFree(cmd *cobra.Command, args []string) error {
	home, err := cmdUserHomeDir()
	if err != nil {
		return fmt.Errorf("homedir: %w", err)
	}

	reg, err := cmdRegistryLoad(home)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	if freeFlagAll {
		if !freeFlagForce && !confirmFn("Release ALL port registrations? [y/N] ") {
			fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}
		if err := cmdTransaction(home, func(reg *registry.Registry) error {
			reg.Entries = make(map[string]*registry.Entry)
			return nil
		}); err != nil {
			return fmt.Errorf("save registry: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "All registrations released.")
		return nil
	}

	var key string
	if len(args) == 1 {
		// Check if argument is a port number.
		if port, err := strconv.Atoi(args[0]); err == nil {
			key = findKeyByPort(reg, port)
			if key == "" {
				return fmt.Errorf("no registration found for port %d", port)
			}
		} else {
			key = args[0]
		}
	} else {
		// Default: use cwd-based key.
		dir, err := cmdGetwd()
		if err != nil {
			return fmt.Errorf("getwd: %w", err)
		}
		res, err := cmdResolve(dir)
		if err != nil {
			return fmt.Errorf("resolve key: %w", err)
		}
		key = res.Key
	}

	entry, ok := reg.Entries[key]
	if !ok {
		return fmt.Errorf("no registration found for %q", key)
	}

	if !freeFlagForce && !confirmFn(fmt.Sprintf("Release port %d for %q? [y/N] ", entry.Port, key)) {
		fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
		return nil
	}

	portNum := entry.Port
	if err := cmdTransaction(home, func(reg *registry.Registry) error {
		delete(reg.Entries, key)
		return nil
	}); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Released port %d for %q.\n", portNum, key)
	return nil
}

func findKeyByPort(reg *registry.Registry, port int) string {
	for k, e := range reg.Entries {
		if e.Port == port {
			return k
		}
	}
	return ""
}
