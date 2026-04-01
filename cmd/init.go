package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vp-k/devport/internal/allocator"
	"github.com/vp-k/devport/internal/detect"
	"github.com/vp-k/devport/internal/registry"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up devport for the current project",
	RunE:  runInit,
}

var (
	initFlagFramework string
	initFlagRangeMin  int
	initFlagRangeMax  int
	initFlagYes       bool
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&initFlagFramework, "framework", "", "Override framework detection")
	initCmd.Flags().IntVar(&initFlagRangeMin, "range-min", 0, "Custom port range minimum")
	initCmd.Flags().IntVar(&initFlagRangeMax, "range-max", 0, "Custom port range maximum")
	initCmd.Flags().BoolVarP(&initFlagYes, "yes", "y", false, "Accept all prompts automatically")
}

func runInit(cmd *cobra.Command, _ []string) error {
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

	framework := initFlagFramework
	if framework == "" {
		framework = detect.Detect(dir)
	}

	var port int

	if err := cmdTransaction(home, func(reg *registry.Registry) error {
		isNew := reg.Entries[res.Key] == nil
		var allocErr error
		port, allocErr = cmdAllocate(res.Key, framework, reg, allocator.Options{
			RangeMin: initFlagRangeMin,
			RangeMax: initFlagRangeMax,
		})
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

	// Write env file.
	cfg := detect.EnvConfigFor(framework, detect.EnvOptions{})
	if err := cmdWriteEnvFile(dir, port, cfg); err != nil {
		return fmt.Errorf("write env file: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Framework : %s\n", frameworkLabel(framework))
	fmt.Fprintf(cmd.OutOrStdout(), "Port      : %d\n", port)
	fmt.Fprintf(cmd.OutOrStdout(), "Env file  : %s (%s=%d)\n", cfg.File, cfg.VarName, port)

	// Offer to add predev script to package.json.
	pkgPath := filepath.Join(dir, "package.json")
	if _, err := os.Stat(pkgPath); err == nil {
		do := initFlagYes || confirmFn(`Add "predev": "devport env" to package.json? [y/N] `)
		if do {
			if err := addPredevScript(pkgPath); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Warning: could not update package.json: %v\n", err)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), `Added "predev": "devport env" to package.json.`)
			}
		}
	}

	// Offer to add .env.local to .gitignore.
	if cfg.File == ".env.local" {
		do := initFlagYes || confirmFn("Add .env.local to .gitignore? [y/N] ")
		if do {
			if err := addToGitignore(dir, ".env.local"); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Warning: could not update .gitignore: %v\n", err)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Added .env.local to .gitignore.")
			}
		}
	}

	return nil
}

func frameworkLabel(f string) string {
	if f == "" {
		return "(unknown)"
	}
	return f
}

// addPredevScript inserts or updates the "predev" script in package.json.
func addPredevScript(pkgPath string) error {
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return err
	}

	var pkg map[string]json.RawMessage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return err
	}

	// Ensure scripts section exists.
	var scripts map[string]json.RawMessage
	if raw, ok := pkg["scripts"]; ok {
		if err := json.Unmarshal(raw, &scripts); err != nil {
			return err
		}
	} else {
		scripts = make(map[string]json.RawMessage)
	}

	scripts["predev"] = json.RawMessage(`"devport env"`)

	encoded, _ := json.Marshal(scripts)
	pkg["scripts"] = json.RawMessage(encoded)

	out, _ := json.MarshalIndent(pkg, "", "  ")

	return os.WriteFile(pkgPath, append(out, '\n'), 0644)
}

// addToGitignore appends line to .gitignore if not already present.
func addToGitignore(dir, line string) error {
	path := filepath.Join(dir, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	content := string(data)
	for _, l := range strings.Split(content, "\n") {
		if strings.TrimSpace(l) == line {
			return nil // already present
		}
	}

	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += line + "\n"

	return os.WriteFile(path, []byte(content), 0644)
}
