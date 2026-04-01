package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/vp-k/devport/internal/registry"
)

func cleanupFreeFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		freeFlagAll = false
		freeFlagForce = false
	})
}

func TestFreeByKey(t *testing.T) {
	cleanupFreeFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "free-app", 3100)

	out, err := runCmd(t, freeCmd, "--force", "free-app")
	if err != nil {
		t.Fatalf("free free-app: %v", err)
	}
	if !strings.Contains(out, "Released") {
		t.Errorf("expected 'Released' in output, got: %q", out)
	}

	reg, _ := registry.Load(homeDir)
	if _, ok := reg.Entries["free-app"]; ok {
		t.Error("entry should have been removed")
	}
}

func TestFreeByPort(t *testing.T) {
	cleanupFreeFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "port-app", 3200)

	_, err := runCmd(t, freeCmd, "--force", "3200")
	if err != nil {
		t.Fatalf("free 3200: %v", err)
	}

	reg, _ := registry.Load(homeDir)
	if _, ok := reg.Entries["port-app"]; ok {
		t.Error("entry should have been removed via port lookup")
	}
}

func TestFreePortNotFound(t *testing.T) {
	cleanupFreeFlags(t)
	newTestHome(t)

	_, err := runCmd(t, freeCmd, "--force", "9999")
	if err == nil {
		t.Fatal("expected error for unknown port, got nil")
	}
}

func TestFreeKeyNotFound(t *testing.T) {
	cleanupFreeFlags(t)
	newTestHome(t)

	_, err := runCmd(t, freeCmd, "--force", "nonexistent-key")
	if err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
}

func TestFreeCurrentDir(t *testing.T) {
	cleanupFreeFlags(t)
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "cwd-free-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Allocate a port first so the key exists.
	runCmd(t, getCmd)

	out, err := runCmd(t, freeCmd, "--force")
	if err != nil {
		t.Fatalf("free (cwd): %v", err)
	}
	if !strings.Contains(out, "Released") {
		t.Errorf("expected 'Released' in output, got: %q", out)
	}
}

func TestFreeAbortedOnDeny(t *testing.T) {
	cleanupFreeFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "abort-app", 3300)
	autoConfirm(t, false)

	out, err := runCmd(t, freeCmd, "abort-app")
	if err != nil {
		t.Fatalf("free abort-app: %v", err)
	}
	if !strings.Contains(out, "Aborted") {
		t.Errorf("expected 'Aborted' in output, got: %q", out)
	}

	reg, _ := registry.Load(homeDir)
	if _, ok := reg.Entries["abort-app"]; !ok {
		t.Error("entry should still exist after abort")
	}
}

func TestFreeAll(t *testing.T) {
	cleanupFreeFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "all-app-1", 3400)

	out, err := runCmd(t, freeCmd, "--all", "--force")
	if err != nil {
		t.Fatalf("free --all: %v", err)
	}
	if !strings.Contains(out, "All registrations released") {
		t.Errorf("expected 'All registrations released', got: %q", out)
	}

	reg, _ := registry.Load(homeDir)
	if len(reg.Entries) != 0 {
		t.Errorf("expected empty registry, got %d entries", len(reg.Entries))
	}
}

func TestFreeAllAborted(t *testing.T) {
	cleanupFreeFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "no-del-app", 3500)
	autoConfirm(t, false)

	out, err := runCmd(t, freeCmd, "--all")
	if err != nil {
		t.Fatalf("free --all aborted: %v", err)
	}
	if !strings.Contains(out, "Aborted") {
		t.Errorf("expected 'Aborted', got: %q", out)
	}
}
