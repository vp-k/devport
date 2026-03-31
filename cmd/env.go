package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/user01/devport/internal/allocator"
	"github.com/user01/devport/internal/detect"
	"github.com/user01/devport/internal/registry"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Write PORT to the project's env file (.env.local or .env)",
	RunE:  runEnv,
}

var (
	envFlagOutput    string
	envFlagVarName   string
	envFlagFramework string
)

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.Flags().StringVar(&envFlagOutput, "output", "", "Override output file path")
	envCmd.Flags().StringVar(&envFlagVarName, "var-name", "", "Override env variable name")
	envCmd.Flags().StringVar(&envFlagFramework, "framework", "", "Override framework detection")
}

func runEnv(cmd *cobra.Command, _ []string) error {
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

	framework := envFlagFramework
	if framework == "" {
		framework = detect.Detect(dir)
	}

	var port int

	if err := cmdTransaction(home, func(reg *registry.Registry) error {
		isNew := reg.Entries[res.Key] == nil
		var allocErr error
		port, allocErr = cmdAllocate(res.Key, framework, reg, allocator.Options{})
		if allocErr != nil {
			return allocErr
		}
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
		return nil
	}); err != nil {
		return err
	}

	cfg := detect.EnvConfigFor(framework, detect.EnvOptions{
		Output:  envFlagOutput,
		VarName: envFlagVarName,
	})

	if err := cmdWriteEnvFile(dir, port, cfg); err != nil {
		return fmt.Errorf("write env file: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s=%d to %s\n", cfg.VarName, port, cfg.File)
	return nil
}
