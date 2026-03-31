//go:build !windows

package cmd

import (
	"os"
	"os/exec"
	"syscall"
)

// startProcess launches cmd with args and env, forwards signals from sigCh,
// and returns the child's exit code.
func startProcess(name string, args []string, env []string, sigCh <-chan os.Signal) (int, error) {
	c := exec.Command(name, args...)
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

	if err := c.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), nil
			}
			return 1, nil
		}
		return 0, err
	}
	return 0, nil
}
