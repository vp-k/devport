package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export port registrations to JSON or CSV",
	RunE:  runExport,
}

var (
	exportFlagOutput string
	exportFlagFormat string
)

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVar(&exportFlagOutput, "output", "", "Output file path (default: stdout)")
	exportCmd.Flags().StringVar(&exportFlagFormat, "format", "json", "Output format: json or csv")
}

// exportEntry is the serialised form of a single registry entry.
type exportEntry struct {
	Key         string `json:"key"`
	Port        int    `json:"port"`
	DisplayName string `json:"displayName"`
	Framework   string `json:"framework,omitempty"`
	ProjectPath string `json:"projectPath"`
	KeySource   string `json:"keySource,omitempty"`
	AllocatedAt string `json:"allocatedAt"`
}

func runExport(cmd *cobra.Command, _ []string) error {
	home, err := cmdUserHomeDir()
	if err != nil {
		return fmt.Errorf("homedir: %w", err)
	}

	reg, err := cmdRegistryLoad(home)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	// Build sorted slice for deterministic output.
	keys := make([]string, 0, len(reg.Entries))
	for k := range reg.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	entries := make([]exportEntry, 0, len(keys))
	for _, k := range keys {
		e := reg.Entries[k]
		entries = append(entries, exportEntry{
			Key:         k,
			Port:        e.Port,
			DisplayName: e.DisplayName,
			Framework:   e.Framework,
			ProjectPath: e.ProjectPath,
			KeySource:   string(e.KeySource),
			AllocatedAt: formatRegistryTime(e.AllocatedAt),
		})
	}

	// Determine the output writer.
	out := cmd.OutOrStdout()
	if exportFlagOutput != "" {
		f, createErr := os.Create(exportFlagOutput)
		if createErr != nil {
			return fmt.Errorf("create output file: %w", createErr)
		}
		defer f.Close()
		out = f
	}

	switch exportFlagFormat {
	case "csv":
		w := csv.NewWriter(out)
		// csv.Writer buffers writes; errors accumulate in w.Error() after Flush.
		_ = w.Write([]string{"key", "port", "displayName", "framework", "projectPath", "allocatedAt"})
		for _, e := range entries {
			_ = w.Write([]string{
				e.Key,
				fmt.Sprintf("%d", e.Port),
				e.DisplayName,
				e.Framework,
				e.ProjectPath,
				e.AllocatedAt,
			})
		}
		w.Flush()
		return w.Error()
	default: // json
		if len(entries) == 0 {
			_, err = fmt.Fprintln(out, "[]")
			return err
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}
}
