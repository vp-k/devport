package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vp-k/devport/internal/allocator"
	"github.com/vp-k/devport/internal/detect"
	"github.com/vp-k/devport/internal/registry"
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

	// Determine the target key.
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

	// Load once outside the transaction to display port in the confirmation prompt.
	regSnap, err := cmdRegistryLoad(home)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	if snap := regSnap.Entries[key]; snap != nil {
		if !resetFlagForce && !confirmFn(fmt.Sprintf("Reset port %d for %q? [y/N] ", snap.Port, key)) {
			fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}
	}

	// Capture values set inside the transaction for post-transaction output.
	var port int
	var projectPath = dir
	var framework string

	if err := cmdTransaction(home, func(reg *registry.Registry) error {
		oldE := reg.Entries[key]

		if oldE != nil {
			// Use existing entry's metadata so keyed resets don't inherit cwd.
			framework = oldE.Framework
			projectPath = oldE.ProjectPath

			// Temporarily reserve the old port so the allocator cannot hand it
			// back immediately (P2: reset must produce a different port).
			reg.Reserved = append(reg.Reserved, oldE.Port)
			delete(reg.Entries, key)
		} else {
			// New key: resolve framework and metadata from cwd.
			framework = detect.Detect(dir)
		}

		var allocErr error
		port, allocErr = cmdAllocate(key, framework, reg, allocator.Options{})

		// Always clean up the temporary reservation regardless of outcome.
		if oldE != nil {
			reg.Reserved = removeInt(reg.Reserved, oldE.Port)
		}
		if allocErr != nil {
			return allocErr
		}

		now := time.Now().UTC()
		allocatedAt := now
		var keySource registry.KeySource
		var displayName string

		if oldE != nil {
			allocatedAt = oldE.AllocatedAt
			keySource = oldE.KeySource
			displayName = oldE.DisplayName
		} else {
			res, resolveErr := cmdResolve(dir)
			if resolveErr != nil {
				return fmt.Errorf("resolve key: %w", resolveErr)
			}
			keySource = registry.KeySource(res.Source)
			displayName = res.Name
			projectPath = dir
		}

		reg.Entries[key] = &registry.Entry{
			Port:           port,
			KeySource:      keySource,
			DisplayName:    displayName,
			ProjectPath:    projectPath,
			Framework:      framework,
			AllocatedAt:    allocatedAt,
			LastAccessedAt: now,
		}
		return nil
	}); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Reset: assigned port %d for %q.\n", port, key)

	// Offer to update the env file in the project's own directory.
	cfg := detect.EnvConfigFor(framework, detect.EnvOptions{})
	do := resetFlagForce || confirmFn(fmt.Sprintf("Update %s with %s=%d? [y/N] ", cfg.File, cfg.VarName, port))
	if do {
		if err := cmdWriteEnvFile(projectPath, port, cfg); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Warning: could not update env file: %v\n", err)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Updated %s.\n", cfg.File)
		}
	}

	return nil
}

// removeInt returns a copy of s with the first occurrence of v removed.
func removeInt(s []int, v int) []int {
	for i, x := range s {
		if x == v {
			return append(s[:i:i], s[i+1:]...)
		}
	}
	return s
}
