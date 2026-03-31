package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/user01/devport/internal/registry"
)

func cleanupResetFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		resetFlagForce = false
	})
}

func TestResetReassignsPort(t *testing.T) {
	cleanupResetFlags(t)
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "reset-test-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// First, allocate a port via get.
	runCmd(t, getCmd)

	out, err := runCmd(t, resetCmd, "--force")
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	if !strings.Contains(out, "Reset") {
		t.Errorf("expected 'Reset' in output, got: %q", out)
	}
}

func TestResetByKey(t *testing.T) {
	cleanupResetFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "reset-by-key", 3600)

	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, resetCmd, "--force", "reset-by-key")
	if err != nil {
		t.Fatalf("reset reset-by-key: %v", err)
	}
	if !strings.Contains(out, "Reset") {
		t.Errorf("expected 'Reset' in output, got: %q", out)
	}

	reg, _ := registry.Load(homeDir)
	entry, ok := reg.Entries["reset-by-key"]
	if !ok {
		t.Fatal("entry not found after reset")
	}
	if entry.Port == 3600 {
		t.Error("port should have changed after reset")
	}
}

func TestResetAborted(t *testing.T) {
	cleanupResetFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "no-reset-app", 3700)
	autoConfirm(t, false)

	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, resetCmd, "no-reset-app")
	if err != nil {
		t.Fatalf("reset no-reset-app: %v", err)
	}
	if !strings.Contains(out, "Aborted") {
		t.Errorf("expected 'Aborted', got: %q", out)
	}

	reg, _ := registry.Load(homeDir)
	if reg.Entries["no-reset-app"].Port != 3700 {
		t.Error("port should not change after abort")
	}
}

func TestResetNewEntryNoConfirm(t *testing.T) {
	cleanupResetFlags(t)
	autoConfirm(t, true)
	newTestHome(t)

	dir := newTestProjectWithPackageJSON(t, "brand-new-reset-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, resetCmd, "--force")
	if err != nil {
		t.Fatalf("reset (new app): %v", err)
	}
	if !strings.Contains(out, "Reset") {
		t.Errorf("expected 'Reset' in output, got: %q", out)
	}
}
