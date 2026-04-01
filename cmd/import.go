package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vp-k/devport/internal/registry"
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import port registrations from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE:  runImport,
}

var (
	importFlagMerge     bool
	importFlagOverwrite bool
	importFlagDryRun    bool
)

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().BoolVar(&importFlagMerge, "merge", false, "Add only new keys, skip existing (default behaviour)")
	importCmd.Flags().BoolVar(&importFlagOverwrite, "overwrite", false, "Overwrite existing entries on conflict")
	importCmd.Flags().BoolVar(&importFlagDryRun, "dry-run", false, "Show what would be imported without saving")
	_ = importCmd.Flags().MarkDeprecated("merge", "merge is the default; omit this flag")
}

func runImport(cmd *cobra.Command, args []string) error {
	if importFlagMerge && importFlagOverwrite {
		return fmt.Errorf("cannot use --merge with --overwrite")
	}

	filePath := args[0]

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var incoming []exportEntry
	if err := json.Unmarshal(data, &incoming); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}

	home, err := cmdUserHomeDir()
	if err != nil {
		return fmt.Errorf("homedir: %w", err)
	}

	reg, err := cmdRegistryLoad(home)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	usedPorts := make(map[int]string)
	for k, e := range reg.Entries {
		usedPorts[e.Port] = k
	}

	type action struct {
		key         string
		entry       exportEntry
		reason      string // "add", "overwrite", "skip-exists", "skip-port", "skip-key"
		conflictKey string // set for "skip-port": who owns the conflicting port
	}

	mergeMode := importFlagMerge || !importFlagOverwrite

	// pendingPorts tracks ports claimed by earlier entries in this import
	// payload to catch within-payload port duplicates before they reach the registry.
	pendingPorts := make(map[int]string)
	// seenKeys tracks every key encountered in this payload (regardless of
	// whether it was skipped or accepted) so that a second occurrence of the
	// same key is always caught, even if the first was a skip-port/skip-exists.
	seenKeys := make(map[string]bool)

	var actions []action
	for _, ie := range incoming {
		// 1. Port conflict with an existing registry entry.
		if existKey, conflict := usedPorts[ie.Port]; conflict && existKey != ie.Key {
			seenKeys[ie.Key] = true
			actions = append(actions, action{key: ie.Key, entry: ie, reason: "skip-port", conflictKey: existKey})
			continue
		}
		// 2. Port conflict with an earlier entry in the same payload.
		if pendingKey, conflict := pendingPorts[ie.Port]; conflict && pendingKey != ie.Key {
			seenKeys[ie.Key] = true
			actions = append(actions, action{key: ie.Key, entry: ie, reason: "skip-port", conflictKey: pendingKey})
			continue
		}
		// 3. Duplicate key within the same payload.
		if seenKeys[ie.Key] {
			actions = append(actions, action{key: ie.Key, entry: ie, reason: "skip-key"})
			continue
		}
		// 4. Mark key as seen before deciding the action.
		seenKeys[ie.Key] = true
		// 5. Decide: skip-exists / overwrite / add.
		if _, exists := reg.Entries[ie.Key]; exists && mergeMode {
			actions = append(actions, action{key: ie.Key, entry: ie, reason: "skip-exists"})
			continue
		}
		if _, exists := reg.Entries[ie.Key]; exists {
			actions = append(actions, action{key: ie.Key, entry: ie, reason: "overwrite"})
		} else {
			actions = append(actions, action{key: ie.Key, entry: ie, reason: "add"})
		}
		pendingPorts[ie.Port] = ie.Key
	}

	added, overwritten, skipped := 0, 0, 0
	for _, a := range actions {
		switch a.reason {
		case "add":
			fmt.Fprintf(cmd.OutOrStdout(), "  add       %s -> port %d\n", a.key, a.entry.Port)
			added++
		case "overwrite":
			fmt.Fprintf(cmd.OutOrStdout(), "  overwrite %s -> port %d\n", a.key, a.entry.Port)
			overwritten++
		case "skip-exists":
			fmt.Fprintf(cmd.OutOrStdout(), "  skip      %s (already registered)\n", a.key)
			skipped++
		case "skip-port":
			fmt.Fprintf(cmd.OutOrStdout(), "  skip      %s (port %d taken by %q)\n",
				a.key, a.entry.Port, a.conflictKey)
			skipped++
		case "skip-key":
			fmt.Fprintf(cmd.OutOrStdout(), "  skip      %s (duplicate key in payload)\n", a.key)
			skipped++
		}
	}

	if importFlagDryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "Dry-run: %d add, %d overwrite, %d skip.\n", added, overwritten, skipped)
		return nil
	}

	if added+overwritten == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Nothing to import.")
		return nil
	}

	if err := cmdTransaction(home, func(r *registry.Registry) error {
		now := time.Now().UTC()
		for _, a := range actions {
			if a.reason != "add" && a.reason != "overwrite" {
				continue
			}
			allocAt, parseErr := parseRegistryTime(a.entry.AllocatedAt)
			if parseErr != nil {
				allocAt = now
			}
			ks := resolveKeySource(a.entry.KeySource, a.reason, r.Entries[a.key])
			r.Entries[a.key] = &registry.Entry{
				Port:           a.entry.Port,
				KeySource:      ks,
				DisplayName:    a.entry.DisplayName,
				Framework:      a.entry.Framework,
				ProjectPath:    a.entry.ProjectPath,
				AllocatedAt:    allocAt,
				LastAccessedAt: now,
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Imported: %d add, %d overwrite, %d skip.\n", added, overwritten, skipped)
	return nil
}

// resolveKeySource picks the KeySource for an imported entry.
// Priority: value from the incoming payload → existing entry value (overwrite
// only) → path (neutral default for new entries with no source info).
func resolveKeySource(incoming, reason string, existing *registry.Entry) registry.KeySource {
	switch registry.KeySource(incoming) {
	case registry.KeySourcePackageJSON, registry.KeySourceGitRemote, registry.KeySourcePath:
		return registry.KeySource(incoming)
	}
	if reason == "overwrite" && existing != nil {
		return existing.KeySource
	}
	return registry.KeySourcePath
}
