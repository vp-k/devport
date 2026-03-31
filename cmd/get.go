package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/user01/devport/internal/allocator"
	"github.com/user01/devport/internal/detect"
	"github.com/user01/devport/internal/registry"
)

type getJSONOutput struct {
	Port      int    `json:"port"`
	Key       string `json:"key"`
	Framework string `json:"framework"`
	New       bool   `json:"new"`
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get (or allocate) the port for the current project",
	RunE:  runGet,
}

var (
	getFlagJSON      bool
	getFlagRangeMin  int
	getFlagRangeMax  int
	getFlagFramework string
)

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().BoolVar(&getFlagJSON, "json", false, "Output as JSON")
	getCmd.Flags().IntVar(&getFlagRangeMin, "range-min", 0, "Custom port range minimum")
	getCmd.Flags().IntVar(&getFlagRangeMax, "range-max", 0, "Custom port range maximum")
	getCmd.Flags().StringVar(&getFlagFramework, "framework", "", "Override framework detection")
}

func runGet(cmd *cobra.Command, _ []string) error {
	dir, err := cmdGetwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	home, err := cmdUserHomeDir()
	if err != nil {
		return fmt.Errorf("homedir: %w", err)
	}

	res, err := cmdResolve(dir)
	if err != nil {
		return fmt.Errorf("resolve key: %w", err)
	}

	reg, err := cmdRegistryLoad(home)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	framework := getFlagFramework
	if framework == "" {
		framework = detect.Detect(dir)
	}

	isNew := reg.Entries[res.Key] == nil

	port, err := cmdAllocate(res.Key, framework, reg, allocator.Options{
		RangeMin: getFlagRangeMin,
		RangeMax: getFlagRangeMax,
	})
	if err != nil {
		return err
	}

	// Persist if new or update lastAccessedAt.
	now := time.Now().UTC()
	if isNew {
		reg.Entries[res.Key] = &registry.Entry{
			Port:           port,
			KeySource:      registry.KeySource(res.Source),
			DisplayName:    res.Name,
			ProjectPath:    dir,
			Framework:      framework,
			AllocatedAt:    now,
			LastAccessedAt: now,
		}
	} else {
		reg.Entries[res.Key].LastAccessedAt = now
	}

	if err := cmdRegistrySave(home, reg); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	if getFlagJSON {
		out := getJSONOutput{Port: port, Key: res.Key, Framework: framework, New: isNew}
		data, _ := json.Marshal(out)
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), port)
	return nil
}
