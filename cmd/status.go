package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user01/devport/internal/allocator"
	"github.com/user01/devport/internal/detect"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show port status for the current project",
	RunE:  runStatus,
}

var statusFlagJSON bool

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVar(&statusFlagJSON, "json", false, "Output as JSON")
}

const (
	statusListening    = "LISTENING"
	statusAllocated    = "ALLOCATED"
	statusNotRegistered = "NOT REGISTERED"
)

func runStatus(cmd *cobra.Command, _ []string) error {
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

	entry := reg.Entries[res.Key]

	if entry == nil {
		if statusFlagJSON {
			data, _ := json.Marshal(map[string]interface{}{
				"key":    res.Key,
				"status": statusNotRegistered,
			})
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Project  : %s\nKey      : %s\nStatus   : %s\n",
			res.Name, res.Key, statusNotRegistered)
		return nil
	}

	portStatus := statusAllocated
	if !allocator.ProbePort(entry.Port) {
		portStatus = statusListening
	}

	cfg := detect.EnvConfigFor(entry.Framework, detect.EnvOptions{})

	if statusFlagJSON {
		data, _ := json.Marshal(map[string]interface{}{
			"key":         res.Key,
			"project":     entry.DisplayName,
			"port":        entry.Port,
			"status":      portStatus,
			"framework":   entry.Framework,
			"envFile":     cfg.File,
			"projectPath": entry.ProjectPath,
		})
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Project  : %s\nKey      : %s\nPort     : %d\nStatus   : %s\nFramework: %s\nEnv file : %s\n",
		entry.DisplayName, res.Key, entry.Port, portStatus, entry.Framework, cfg.File)
	return nil
}
