package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/vp-k/devport/internal/allocator"
	"github.com/vp-k/devport/internal/detect"
	"github.com/vp-k/devport/internal/registry"
)

var execCmd = &cobra.Command{
	Use:   "exec -- <command> [args...]",
	Short: "Run a command with the allocated port injected",
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
		entry := reg.Entries[res.Key]
		isNew := entry == nil
		if entry != nil && entry.Framework != "" {
			framework = entry.Framework
		}
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
			if reg.Entries[res.Key].Framework == "" && framework != "" {
				reg.Entries[res.Key].Framework = framework
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Build environment: inherit parent env + inject PORT plus any framework-specific variable.
	env := buildExecEnv(framework, port)
	args = injectPortFlag(args, framework, port)

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

func buildExecEnv(framework string, port int) []string {
	portStr := strconv.Itoa(port)
	env := append(os.Environ(), "PORT="+portStr)

	cfg := detect.EnvConfigFor(framework, detect.EnvOptions{})
	if cfg.VarName != "" && cfg.VarName != "PORT" {
		env = append(env, cfg.VarName+"="+portStr)
	}

	return env
}

// injectPortFlag appends the framework's port flag to args when the framework
// has a standard CLI switch for overriding the dev server port.
func injectPortFlag(args []string, framework string, port int) []string {
	flag := detect.PortFlagFor(framework)
	if flag == "" || len(args) == 0 || hasPortFlagArg(args) {
		return args
	}

	portStr := strconv.Itoa(port)
	if isPackageManagerRun(args) {
		if hasDoubleDash(args) {
			return append(args, flag, portStr)
		}
		return append(args, "--", flag, portStr)
	}

	return append(args, flag, portStr)
}

func hasPortFlagArg(args []string) bool {
	for _, arg := range args {
		if arg == "--port" || arg == "-p" {
			return true
		}
		if strings.HasPrefix(arg, "--port=") || strings.HasPrefix(arg, "-p=") {
			return true
		}
	}
	return false
}

func hasDoubleDash(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return true
		}
	}
	return false
}

func isPackageManagerRun(args []string) bool {
	if len(args) < 3 || args[1] != "run" {
		return false
	}

	switch args[0] {
	case "npm", "pnpm", "yarn", "bun":
		return true
	default:
		return false
	}
}
