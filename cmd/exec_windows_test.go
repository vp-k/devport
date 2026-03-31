//go:build windows

package cmd

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestStartProcessSuccess(t *testing.T) {
	sigCh := make(chan os.Signal, 1)
	close(sigCh)
	code, err := startProcess("echo", []string{"hello"}, os.Environ(), sigCh)
	if err != nil {
		t.Fatalf("startProcess: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestStartProcessNonZeroExit(t *testing.T) {
	sigCh := make(chan os.Signal, 1)
	close(sigCh)
	code, err := startProcess("exit", []string{"1"}, os.Environ(), sigCh)
	if err != nil {
		t.Fatalf("startProcess: %v", err)
	}
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestStartProcessSignalForwarding(t *testing.T) {
	// Pre-buffer a signal so the goroutine reads and forwards it.
	sigCh := make(chan os.Signal, 1)
	sigCh <- syscall.SIGTERM
	_, err := startProcess("echo", []string{"signal-test"}, os.Environ(), sigCh)
	// Allow goroutine time to consume the buffered signal.
	time.Sleep(20 * time.Millisecond)
	close(sigCh)
	if err != nil {
		t.Fatalf("startProcess: %v", err)
	}
}

func TestStartProcessStartError(t *testing.T) {
	orig := execCommandFn
	execCommandFn = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("__nonexistent_binary_xyz_12345__")
	}
	t.Cleanup(func() { execCommandFn = orig })

	sigCh := make(chan os.Signal, 1)
	close(sigCh)
	_, err := startProcess("echo", nil, os.Environ(), sigCh)
	if err == nil {
		t.Fatal("expected start error, got nil")
	}
}

func TestStartProcessWaitNonExitError(t *testing.T) {
	origWait := execWaitFn
	execWaitFn = func(_ *exec.Cmd) error { return errors.New("fake wait error") }
	t.Cleanup(func() { execWaitFn = origWait })

	sigCh := make(chan os.Signal, 1)
	close(sigCh)
	_, err := startProcess("echo", nil, os.Environ(), sigCh)
	if err == nil {
		t.Fatal("expected wait error, got nil")
	}
}
