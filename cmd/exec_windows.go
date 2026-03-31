//go:build windows

package cmd

import (
	"os"
	"os/exec"
)

// execCommandFn is the factory used to create an exec.Cmd. Injectable for tests.
var execCommandFn = func(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// execWaitFn calls c.Wait(). Injectable for tests.
var execWaitFn = func(c *exec.Cmd) error { return c.Wait() }

// startProcess launches cmd /C <name> [args...] with env, forwards signals,
// and returns the child's exit code.
func startProcess(name string, args []string, env []string, sigCh <-chan os.Signal) (int, error) {
	cmdArgs := append([]string{"/C", name}, args...)
	c := execCommandFn("cmd", cmdArgs...)
	c.Env = env
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Start(); err != nil {
		return 0, err
	}

	// Forward signals to the child process.
	go func() {
		for sig := range sigCh {
			if c.Process != nil {
				c.Process.Signal(sig) //nolint:errcheck
			}
		}
	}()

	if err := execWaitFn(c); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}
