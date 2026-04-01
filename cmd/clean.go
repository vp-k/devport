package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vp-k/devport/internal/registry"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove stale or old port registrations",
	RunE:  runClean,
}

var (
	cleanFlagStale    bool
	cleanFlagOlderThan int
	cleanFlagAll      bool
	cleanFlagForce    bool
)

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVar(&cleanFlagStale, "stale", false, "Remove entries with missing project paths")
	cleanCmd.Flags().IntVar(&cleanFlagOlderThan, "older-than", 0, "Remove entries not accessed in N days")
	cleanCmd.Flags().BoolVar(&cleanFlagAll, "all", false, "Remove all entries")
	cleanCmd.Flags().BoolVar(&cleanFlagForce, "force", false, "Skip confirmation prompt")
}

func runClean(cmd *cobra.Command, _ []string) error {
	home, err := cmdUserHomeDir()
	if err != nil {
		return fmt.Errorf("homedir: %w", err)
	}

	reg, err := cmdRegistryLoad(home)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	// Identify keys to remove.
	toRemove := make(map[string]struct{})

	if cleanFlagAll {
		for k := range reg.Entries {
			toRemove[k] = struct{}{}
		}
	}

	if cleanFlagStale {
		for k, e := range reg.Entries {
			if e.ProjectPath != "" {
				if _, statErr := os.Stat(e.ProjectPath); os.IsNotExist(statErr) {
					toRemove[k] = struct{}{}
				}
			}
		}
	}

	if cleanFlagOlderThan > 0 {
		cutoff := time.Now().UTC().AddDate(0, 0, -cleanFlagOlderThan)
		for k, e := range reg.Entries {
			if e.LastAccessedAt.Before(cutoff) {
				toRemove[k] = struct{}{}
			}
		}
	}

	if len(toRemove) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Nothing to clean.")
		return nil
	}

	if !cleanFlagForce && !confirmFn(fmt.Sprintf("Remove %d registration(s)? [y/N] ", len(toRemove))) {
		fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
		return nil
	}

	if err := cmdTransaction(home, func(r *registry.Registry) error {
		for k := range toRemove {
			delete(r.Entries, k)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Removed %d registration(s).\n", len(toRemove))
	return nil
}
