package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/user01/devport/internal/output"
	"github.com/user01/devport/internal/registry"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered projects and their ports",
	RunE:  runList,
}

var (
	listFlagJSON    bool
	listFlagVerbose bool
)

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listFlagJSON, "json", false, "Output as JSON")
	listCmd.Flags().BoolVarP(&listFlagVerbose, "verbose", "v", false, "Show additional columns")
}

func runList(cmd *cobra.Command, _ []string) error {
	home, err := cmdUserHomeDir()
	if err != nil {
		return fmt.Errorf("homedir: %w", err)
	}

	reg, err := cmdRegistryLoad(home)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	if len(reg.Entries) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No projects registered.")
		return nil
	}

	// Sort by port ascending.
	type row struct {
		key   string
		entry *registry.Entry
	}
	rows := make([]row, 0, len(reg.Entries))
	for k, e := range reg.Entries {
		rows = append(rows, row{k, e})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].entry.Port < rows[j].entry.Port
	})

	if listFlagJSON {
		type jsonEntry struct {
			Key         string `json:"key"`
			Port        int    `json:"port"`
			DisplayName string `json:"displayName"`
			Framework   string `json:"framework"`
			ProjectPath string `json:"projectPath"`
			KeySource   string `json:"keySource"`
			AllocatedAt string `json:"allocatedAt"`
		}
		out := make([]jsonEntry, len(rows))
		for i, r := range rows {
			out[i] = jsonEntry{
				Key:         r.key,
				Port:        r.entry.Port,
				DisplayName: r.entry.DisplayName,
				Framework:   r.entry.Framework,
				ProjectPath: r.entry.ProjectPath,
				KeySource:   string(r.entry.KeySource),
				AllocatedAt: r.entry.AllocatedAt.String(),
			}
		}
		data, _ := json.Marshal(out)
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	p := output.NewPlain(cmd.OutOrStdout())

	cols := []output.Column{
		{Header: "PORT", MinWidth: 5},
		{Header: "PROJECT", MinWidth: 20},
		{Header: "FRAMEWORK", MinWidth: 10},
	}
	if listFlagVerbose {
		cols = append(cols,
			output.Column{Header: "KEY_SOURCE", MinWidth: 12},
			output.Column{Header: "ALLOCATED_AT", MinWidth: 20},
		)
	}

	tbl := output.NewTable(p, cols)
	for _, r := range rows {
		vals := []string{
			strconv.Itoa(r.entry.Port),
			r.entry.DisplayName,
			r.entry.Framework,
		}
		if listFlagVerbose {
			vals = append(vals,
				string(r.entry.KeySource),
				r.entry.AllocatedAt.Format("2006-01-02 15:04:05"),
			)
		}
		tbl.AddRow(vals...)
	}
	tbl.Render()
	return nil
}
