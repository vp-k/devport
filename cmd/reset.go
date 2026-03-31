package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/user01/devport/internal/allocator"
	"github.com/user01/devport/internal/detect"
	"github.com/user01/devport/internal/registry"
)

var resetCmd = &cobra.Command{
	Use:   "reset [key]",
	Short: "Force re-allocation of the port for a project",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runReset,
}

var resetFlagForce bool

func init() {
	rootCmd.AddCommand(resetCmd)
	resetCmd.Flags().BoolVar(&resetFlagForce, "force", false, "Skip confirmation prompt")
}

func runReset(cmd *cobra.Command, args []string) error {
	dir, err := cmdGetwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	home, err := cmdUserHomeDir()
	if err != nil {
		return fmt.Errorf("homedir: %w", err)
	}

	reg, err := cmdRegistryLoad(home)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	var key string
	if len(args) == 1 {
		key = args[0]
	} else {
		res, err := cmdResolve(dir)
		if err != nil {
			return fmt.Errorf("resolve key: %w", err)
		}
		key = res.Key
	}

	oldEntry, exists := reg.Entries[key]
	if exists {
		if !resetFlagForce && !confirmFn(fmt.Sprintf("Reset port %d for %q? [y/N] ", oldEntry.Port, key)) {
			fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}
		delete(reg.Entries, key)
	}

	framework := detect.Detect(dir)
	res, err := cmdResolve(dir)
	if err != nil {
		return fmt.Errorf("resolve key (2): %w", err)
	}
	// Use the resolved key if no args were given.
	if len(args) == 0 {
		key = res.Key
	}

	port, err := cmdAllocate(key, framework, reg, allocator.Options{})
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	allocatedAt := now
	if exists {
		allocatedAt = oldEntry.AllocatedAt
	}

	reg.Entries[key] = &registry.Entry{
		Port:           port,
		KeySource:      registry.KeySource(res.Source),
		DisplayName:    res.Name,
		ProjectPath:    dir,
		Framework:      framework,
		AllocatedAt:    allocatedAt,
		LastAccessedAt: now,
	}

	if err := cmdRegistrySave(home, reg); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Reset: assigned port %d for %q.\n", port, key)

	// Offer to update the env file.
	cfg := detect.EnvConfigFor(framework, detect.EnvOptions{})
	do := resetFlagForce || confirmFn(fmt.Sprintf("Update %s with %s=%d? [y/N] ", cfg.File, cfg.VarName, port))
	if do {
		if err := cmdWriteEnvFile(dir, port, cfg); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Warning: could not update env file: %v\n", err)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Updated %s.\n", cfg.File)
		}
	}

	return nil
}
