package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/vp-k/devport/internal/allocator"
	"github.com/vp-k/devport/internal/registry"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the devport registry for issues",
	RunE:  runDoctor,
}

var doctorFlagFix bool

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorFlagFix, "fix", false, "Automatically fix issues where possible")
}

type checkResult struct {
	name   string
	status string // "OK", "WARN", "FIXED", "ERROR"
	detail string
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	home, err := cmdUserHomeDir()
	if err != nil {
		return fmt.Errorf("homedir: %w", err)
	}

	out := cmd.OutOrStdout()
	var results []checkResult

	// --- Check 1: registry file exists ---
	regPath := filepath.Join(home, ".devports.json")
	lockPath := regPath + ".lock"

	if _, err := os.Stat(regPath); os.IsNotExist(err) {
		if doctorFlagFix {
			_ = cmdTransaction(home, func(_ *registry.Registry) error { return nil })
			results = append(results, checkResult{"registry file", "FIXED", "created " + regPath})
		} else {
			results = append(results, checkResult{"registry file", "WARN", "not found: " + regPath})
		}
	} else {
		results = append(results, checkResult{"registry file", "OK", regPath})
	}

	// --- Load registry for remaining checks ---
	reg, loadErr := cmdRegistryLoad(home)
	if loadErr != nil {
		results = append(results, checkResult{"JSON validity", "ERROR", loadErr.Error()})
		printDoctorResults(out, results)
		return nil
	}
	results = append(results, checkResult{"JSON validity", "OK", "registry parsed successfully"})

	// --- Check 3: schema version ---
	if reg.Version != 1 {
		if doctorFlagFix {
			_ = cmdTransaction(home, func(r *registry.Registry) error {
				r.Version = 1
				return nil
			})
			results = append(results, checkResult{"schema version", "FIXED", fmt.Sprintf("set version to 1 (was %d)", reg.Version)})
		} else {
			results = append(results, checkResult{"schema version", "WARN", fmt.Sprintf("unexpected version %d (expected 1)", reg.Version)})
		}
	} else {
		results = append(results, checkResult{"schema version", "OK", "version 1"})
	}

	// --- Check 4: port listening status ---
	for key, entry := range reg.Entries {
		if !allocator.ProbePort(entry.Port) {
			results = append(results, checkResult{"port in use", "WARN", fmt.Sprintf("port %d (%s) appears to be in use", entry.Port, key)})
		}
	}
	if len(reg.Entries) > 0 {
		results = append(results, checkResult{"port availability", "OK", fmt.Sprintf("checked %d entries", len(reg.Entries))})
	}

	// --- Check 5: duplicate ports ---
	duplicateKeys := duplicateKeysToRemove(reg)
	if len(duplicateKeys) > 0 {
		if doctorFlagFix {
			_ = cmdTransaction(home, func(r *registry.Registry) error {
				for _, key := range duplicateKeys {
					delete(r.Entries, key)
				}
				return nil
			})
			results = append(results, checkResult{"duplicate ports", "FIXED", fmt.Sprintf("removed %d older duplicate entries", len(duplicateKeys))})
		} else {
			results = append(results, checkResult{"duplicate ports", "WARN", fmt.Sprintf("%d duplicate port assignments", len(duplicateKeys))})
		}
	} else {
		results = append(results, checkResult{"duplicate ports", "OK", "no duplicates"})
	}

	// --- Check 6: stale paths ---
	var stalePaths []string
	for key, entry := range reg.Entries {
		if entry.ProjectPath != "" {
			if _, err := os.Stat(entry.ProjectPath); os.IsNotExist(err) {
				stalePaths = append(stalePaths, key)
			}
		}
	}
	if len(stalePaths) > 0 {
		if doctorFlagFix {
			_ = cmdTransaction(home, func(r *registry.Registry) error {
				for _, key := range stalePaths {
					delete(r.Entries, key)
				}
				return nil
			})
			results = append(results, checkResult{"stale paths", "FIXED", fmt.Sprintf("removed %d stale entries", len(stalePaths))})
		} else {
			results = append(results, checkResult{"stale paths", "WARN", fmt.Sprintf("%d entries with missing project paths", len(stalePaths))})
		}
	} else {
		results = append(results, checkResult{"stale paths", "OK", "all project paths exist"})
	}

	// --- Check 7: lock file path health ---
	lockInfo, lockErr := os.Stat(lockPath)
	switch {
	case os.IsNotExist(lockErr):
		results = append(results, checkResult{"lock file", "OK", "lock file will be created on demand"})
	case lockErr != nil:
		results = append(results, checkResult{"lock file", "ERROR", fmt.Sprintf("could not stat lock file: %v", lockErr)})
	case lockInfo.IsDir():
		if doctorFlagFix {
			if removeErr := os.RemoveAll(lockPath); removeErr != nil {
				results = append(results, checkResult{"lock file", "ERROR", fmt.Sprintf("could not remove directory at lock path: %v", removeErr)})
			} else {
				results = append(results, checkResult{"lock file", "FIXED", "removed directory at lock path: " + lockPath})
			}
		} else {
			results = append(results, checkResult{"lock file", "ERROR", "lock path is a directory: " + lockPath})
		}
	default:
		results = append(results, checkResult{"lock file", "OK", "lock file path is usable"})
	}

	printDoctorResults(out, results)
	return nil
}

func printDoctorResults(out interface{ Write([]byte) (int, error) }, results []checkResult) {
	for _, r := range results {
		fmt.Fprintf(out, "[%s] %s: %s\n", r.status, r.name, r.detail)
	}
}

func duplicateKeysToRemove(reg *registry.Registry) []string {
	portKeys := make(map[int][]string)
	for key, entry := range reg.Entries {
		portKeys[entry.Port] = append(portKeys[entry.Port], key)
	}

	ports := make([]int, 0, len(portKeys))
	for port, keys := range portKeys {
		if len(keys) > 1 {
			ports = append(ports, port)
		}
	}
	sort.Ints(ports)

	var duplicates []string
	for _, port := range ports {
		keys := append([]string(nil), portKeys[port]...)
		sort.Slice(keys, func(i, j int) bool {
			left := reg.Entries[keys[i]]
			right := reg.Entries[keys[j]]
			if !left.AllocatedAt.Equal(right.AllocatedAt) {
				return left.AllocatedAt.After(right.AllocatedAt)
			}
			return keys[i] < keys[j]
		})
		duplicates = append(duplicates, keys[1:]...)
	}

	return duplicates
}
