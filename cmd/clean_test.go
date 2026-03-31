package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/user01/devport/internal/registry"
)

func cleanupCleanFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		cleanFlagStale = false
		cleanFlagOlderThan = 0
		cleanFlagAll = false
		cleanFlagForce = false
	})
}

func TestCleanStale(t *testing.T) {
	cleanupCleanFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	reg.Entries["stale-app"] = &registry.Entry{
		Port:           6100,
		ProjectPath:    "/nonexistent/stale/path",
		AllocatedAt:    time.Now().UTC(),
		LastAccessedAt: time.Now().UTC(),
	}
	dir := t.TempDir()
	reg.Entries["good-app"] = &registry.Entry{
		Port:           6101,
		ProjectPath:    dir,
		AllocatedAt:    time.Now().UTC(),
		LastAccessedAt: time.Now().UTC(),
	}
	registry.Save(homeDir, reg)
	autoConfirm(t, true)

	out, err := runCmd(t, cleanCmd, "--stale")
	if err != nil {
		t.Fatalf("clean --stale: %v", err)
	}
	if !strings.Contains(out, "1 registration") {
		t.Errorf("expected 1 registration removed, got: %q", out)
	}

	loaded, _ := cmdRegistryLoad(homeDir)
	if loaded.Entries["stale-app"] != nil {
		t.Error("stale-app should be removed")
	}
	if loaded.Entries["good-app"] == nil {
		t.Error("good-app should remain")
	}
}

func TestCleanOlderThan(t *testing.T) {
	cleanupCleanFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	// Old entry: last accessed 10 days ago.
	reg.Entries["old-app"] = &registry.Entry{
		Port:           6200,
		LastAccessedAt: time.Now().UTC().AddDate(0, 0, -10),
		AllocatedAt:    time.Now().UTC(),
	}
	// Recent entry: last accessed 1 day ago.
	reg.Entries["recent-app"] = &registry.Entry{
		Port:           6201,
		LastAccessedAt: time.Now().UTC().AddDate(0, 0, -1),
		AllocatedAt:    time.Now().UTC(),
	}
	registry.Save(homeDir, reg)
	autoConfirm(t, true)

	out, err := runCmd(t, cleanCmd, "--older-than", "5")
	if err != nil {
		t.Fatalf("clean --older-than: %v", err)
	}
	if !strings.Contains(out, "1 registration") {
		t.Errorf("expected 1 removed, got: %q", out)
	}

	loaded, _ := cmdRegistryLoad(homeDir)
	if loaded.Entries["old-app"] != nil {
		t.Error("old-app should be removed")
	}
	if loaded.Entries["recent-app"] == nil {
		t.Error("recent-app should remain")
	}
}

func TestCleanAll(t *testing.T) {
	cleanupCleanFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	reg.Entries["app-a"] = &registry.Entry{Port: 6300, AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()}
	reg.Entries["app-b"] = &registry.Entry{Port: 6301, AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()}
	registry.Save(homeDir, reg)
	autoConfirm(t, true)

	out, err := runCmd(t, cleanCmd, "--all")
	if err != nil {
		t.Fatalf("clean --all: %v", err)
	}
	if !strings.Contains(out, "2 registration") {
		t.Errorf("expected 2 removed, got: %q", out)
	}

	loaded, _ := cmdRegistryLoad(homeDir)
	if len(loaded.Entries) != 0 {
		t.Errorf("expected empty registry, got %d entries", len(loaded.Entries))
	}
}

func TestCleanForce(t *testing.T) {
	cleanupCleanFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	reg.Entries["force-app"] = &registry.Entry{Port: 6400, AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()}
	registry.Save(homeDir, reg)

	// --force skips confirmation; no autoConfirm needed.
	_, err := runCmd(t, cleanCmd, "--all", "--force")
	if err != nil {
		t.Fatalf("clean --all --force: %v", err)
	}

	loaded, _ := cmdRegistryLoad(homeDir)
	if len(loaded.Entries) != 0 {
		t.Error("expected registry empty after --all --force")
	}
}

func TestCleanAborted(t *testing.T) {
	cleanupCleanFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	reg.Entries["abort-app"] = &registry.Entry{Port: 6500, AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()}
	registry.Save(homeDir, reg)
	autoConfirm(t, false) // user says no

	out, err := runCmd(t, cleanCmd, "--all")
	if err != nil {
		t.Fatalf("clean aborted: %v", err)
	}
	if !strings.Contains(out, "Aborted") {
		t.Errorf("expected Aborted, got: %q", out)
	}
}

func TestCleanNothing(t *testing.T) {
	cleanupCleanFlags(t)
	homeDir := newTestHome(t)
	dir := t.TempDir()
	reg := newEmptyRegistry()
	reg.Entries["ok-app"] = &registry.Entry{
		Port:           6600,
		ProjectPath:    dir,
		LastAccessedAt: time.Now().UTC(),
		AllocatedAt:    time.Now().UTC(),
	}
	registry.Save(homeDir, reg)

	// --stale with no stale entries → "Nothing to clean"
	out, err := runCmd(t, cleanCmd, "--stale")
	if err != nil {
		t.Fatalf("clean: %v", err)
	}
	if !strings.Contains(out, "Nothing") {
		t.Errorf("expected Nothing to clean, got: %q", out)
	}
}

func TestCleanHomeDirError(t *testing.T) {
	cleanupCleanFlags(t)
	newTestHome(t)
	injectHomeDir(t)

	_, err := runCmd(t, cleanCmd, "--all")
	if err == nil {
		t.Fatal("expected homedir error")
	}
}

func TestCleanRegistryLoadError(t *testing.T) {
	cleanupCleanFlags(t)
	newTestHome(t)
	injectRegistryLoad(t)

	_, err := runCmd(t, cleanCmd, "--all")
	if err == nil {
		t.Fatal("expected load error")
	}
}

func TestCleanTransactionError(t *testing.T) {
	cleanupCleanFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	reg.Entries["tx-err-app"] = &registry.Entry{Port: 6700, AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()}
	registry.Save(homeDir, reg)
	autoConfirm(t, true)
	injectTransaction(t)

	_, err := runCmd(t, cleanCmd, "--all")
	if err == nil {
		t.Fatal("expected transaction error")
	}
}

func TestCleanOlderThanNoMatch(t *testing.T) {
	cleanupCleanFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	reg.Entries["fresh-app"] = &registry.Entry{
		Port:           6800,
		LastAccessedAt: time.Now().UTC(),
		AllocatedAt:    time.Now().UTC(),
	}
	registry.Save(homeDir, reg)

	// --older-than 30 won't match a just-created entry.
	out, err := runCmd(t, cleanCmd, "--older-than", "30")
	if err != nil {
		t.Fatalf("clean --older-than: %v", err)
	}

	// cmdRegistryLoad injection is not used; load is real.
	_ = os.Getenv("HOME") // ensure HOME is set

	if !strings.Contains(out, "Nothing") {
		t.Errorf("expected Nothing to clean, got: %q", out)
	}
}
