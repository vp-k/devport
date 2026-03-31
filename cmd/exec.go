package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/user01/devport/internal/allocator"
	"github.com/user01/devport/internal/detect"
	"github.com/user01/devport/internal/registry"
)

var execCmd = &cobra.Command{
	Use:   "exec -- <command> [args...]",
	Short: "Run a command with PORT injected as an environment variable",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runExec,
}

var execFlagAutoFree bool

func init() {
	rootCmd.AddCommand(execCmd)
	execCmd.Flags().BoolVar(&execFlagAutoFree, "auto-free", false, "Release port registration on exit")
}

// cmdStartProcess starts a child process with the given env and returns its exit code.
// Injectable for testing.
var cmdStartProcess = startProcess

func runExec(cmd *cobra.Command, args []string) error {
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

	framework := detect.Detect(dir)

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

	// Build environment: inherit parent env + inject PORT.
	env := append(os.Environ(), "PORT="+strconv.Itoa(port))

	// Set up signal handling so we can forward SIGINT/SIGTERM to the child.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	exitCode, err := cmdStartProcess(args[0], args[1:], env, sigCh)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if execFlagAutoFree {
		if freeErr := cmdTransaction(home, func(reg *registry.Registry) error {
			delete(reg.Entries, res.Key)
			return nil
		}); freeErr != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Warning: could not release port: %v\n", freeErr)
		}
	}

	if exitCode != 0 {
		cmdOsExit(exitCode)
	}
	return nil
}
