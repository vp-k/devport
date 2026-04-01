package cmd

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/user01/devport/internal/registry"
)

func cleanupDoctorFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { doctorFlagFix = false })
}

func newEmptyRegistry() *registry.Registry {
	now := time.Now().UTC()
	return &registry.Registry{
		Version:     1,
		Meta:        registry.Meta{CreatedAt: now, UpdatedAt: now},
		Entries:     make(map[string]*registry.Entry),
		Reserved:    []int{},
		RangePolicy: registry.DefaultRangePolicy(),
	}
}

// seedRegistryFull creates an entry with a real (existing) project path.
func seedRegistryFull(t *testing.T, homeDir, key string, port int, projectPath string) {
	t.Helper()
	reg := newEmptyRegistry()
	reg.Entries[key] = &registry.Entry{
		Port:           port,
		KeySource:      registry.KeySourcePackageJSON,
		DisplayName:    key,
		ProjectPath:    projectPath,
		Framework:      "next",
		AllocatedAt:    time.Now().UTC(),
		LastAccessedAt: time.Now().UTC(),
	}
	if err := registry.Save(homeDir, reg); err != nil {
		t.Fatal(err)
	}
}

func TestDoctorAllOK(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	dir := t.TempDir()
	seedRegistryFull(t, homeDir, "doctor-ok-app", 4500, dir)

	out, err := runCmd(t, doctorCmd)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if !strings.Contains(out, "[OK]") {
		t.Errorf("expected [OK] in output, got: %q", out)
	}
}

func TestDoctorRegistryFileMissing(t *testing.T) {
	cleanupDoctorFlags(t)
	newTestHome(t)

	out, err := runCmd(t, doctorCmd)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if !strings.Contains(out, "WARN") {
		t.Errorf("expected WARN for missing file, got: %q", out)
	}
}

func TestDoctorRegistryFileMissingFix(t *testing.T) {
	cleanupDoctorFlags(t)
	newTestHome(t)

	out, err := runCmd(t, doctorCmd, "--fix")
	if err != nil {
		t.Fatalf("doctor --fix: %v", err)
	}
	if !strings.Contains(out, "FIXED") {
		t.Errorf("expected FIXED, got: %q", out)
	}
}

func TestDoctorLockFilePresent(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	lockPath := filepath.Join(homeDir, ".devports.json.lock")
	os.WriteFile(lockPath, []byte(""), 0644)

	out, err := runCmd(t, doctorCmd)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if !strings.Contains(out, "[OK] lock file") {
		t.Errorf("expected healthy lock file status, got: %q", out)
	}
}

func TestDoctorLockFileFix(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	lockPath := filepath.Join(homeDir, ".devports.json.lock")
	os.WriteFile(lockPath, []byte(""), 0644)

	out, err := runCmd(t, doctorCmd, "--fix")
	if err != nil {
		t.Fatalf("doctor --fix: %v", err)
	}
	if !strings.Contains(out, "[OK] lock file") {
		t.Errorf("expected OK for lock file, got: %q", out)
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("lock file should remain usable after --fix: %v", err)
	}
}

func TestDoctorLockFileDirFixNonEmpty(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	// A non-empty directory at the lock path should also be removed by --fix
	// (we use os.RemoveAll, so even non-empty directories are cleaned up).
	lockPath := filepath.Join(homeDir, ".devports.json.lock")
	os.RemoveAll(lockPath) // ensure no existing file/dir
	os.Mkdir(lockPath, 0755)
	os.WriteFile(filepath.Join(lockPath, "notempty"), []byte(""), 0644)

	out, err := runCmd(t, doctorCmd, "--fix")
	if err != nil {
		t.Fatalf("doctor --fix: %v", err)
	}
	if !strings.Contains(out, "[FIXED] lock file") {
		t.Errorf("expected [FIXED] lock file for non-empty dir, got: %q", out)
	}
	if _, statErr := os.Stat(lockPath); !os.IsNotExist(statErr) {
		t.Error("expected lock path directory to be removed by --fix")
	}
}

func TestDoctorSchemaVersionWrong(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	reg.Version = 99
	registry.Save(homeDir, reg)

	out, err := runCmd(t, doctorCmd)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if !strings.Contains(out, "WARN") {
		t.Errorf("expected WARN for wrong schema version, got: %q", out)
	}
}

func TestDoctorSchemaVersionFix(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	reg.Version = 99
	registry.Save(homeDir, reg)

	out, err := runCmd(t, doctorCmd, "--fix")
	if err != nil {
		t.Fatalf("doctor --fix: %v", err)
	}
	if !strings.Contains(out, "FIXED") {
		t.Errorf("expected FIXED, got: %q", out)
	}
}

func TestDoctorDuplicatePorts(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	reg.Entries["key-a"] = &registry.Entry{Port: 5500, AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()}
	reg.Entries["key-b"] = &registry.Entry{Port: 5500, AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()}
	registry.Save(homeDir, reg)

	out, err := runCmd(t, doctorCmd)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if !strings.Contains(out, "duplicate") {
		t.Errorf("expected 'duplicate' in output, got: %q", out)
	}
}

func TestDoctorDuplicatePortsFix(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	reg.Entries["key-a"] = &registry.Entry{Port: 5501, AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()}
	reg.Entries["key-b"] = &registry.Entry{Port: 5501, AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()}
	registry.Save(homeDir, reg)

	out, err := runCmd(t, doctorCmd, "--fix")
	if err != nil {
		t.Fatalf("doctor --fix: %v", err)
	}
	if !strings.Contains(out, "FIXED") {
		t.Errorf("expected FIXED, got: %q", out)
	}
}

func TestDoctorDuplicatePortsFixKeepsNewestEntry(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	oldTime := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	newTime := oldTime.Add(2 * time.Hour)
	reg.Entries["older-entry"] = &registry.Entry{
		Port:           5510,
		AllocatedAt:    oldTime,
		LastAccessedAt: oldTime,
	}
	reg.Entries["newer-entry"] = &registry.Entry{
		Port:           5510,
		AllocatedAt:    newTime,
		LastAccessedAt: newTime,
	}
	registry.Save(homeDir, reg)

	out, err := runCmd(t, doctorCmd, "--fix")
	if err != nil {
		t.Fatalf("doctor --fix: %v", err)
	}
	if !strings.Contains(out, "removed 1 older duplicate entries") {
		t.Fatalf("expected duplicate removal detail, got: %q", out)
	}

	regAfter, err := registry.Load(homeDir)
	if err != nil {
		t.Fatalf("load registry after fix: %v", err)
	}
	if _, ok := regAfter.Entries["newer-entry"]; !ok {
		t.Fatal("expected newest entry to remain after doctor --fix")
	}
	if _, ok := regAfter.Entries["older-entry"]; ok {
		t.Fatal("expected older duplicate entry to be removed")
	}
}

func TestDoctorDuplicatePortsFixTieBreaksByKey(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	allocatedAt := time.Date(2026, time.February, 1, 9, 0, 0, 0, time.UTC)
	reg.Entries["key-b"] = &registry.Entry{
		Port:           5511,
		AllocatedAt:    allocatedAt,
		LastAccessedAt: allocatedAt,
	}
	reg.Entries["key-a"] = &registry.Entry{
		Port:           5511,
		AllocatedAt:    allocatedAt,
		LastAccessedAt: allocatedAt,
	}
	registry.Save(homeDir, reg)

	_, err := runCmd(t, doctorCmd, "--fix")
	if err != nil {
		t.Fatalf("doctor --fix: %v", err)
	}

	regAfter, err := registry.Load(homeDir)
	if err != nil {
		t.Fatalf("load registry after fix: %v", err)
	}
	if _, ok := regAfter.Entries["key-a"]; !ok {
		t.Fatal("expected lexicographically first key to remain on identical timestamps")
	}
	if _, ok := regAfter.Entries["key-b"]; ok {
		t.Fatal("expected duplicate key-b to be removed")
	}
}

func TestDoctorLockFileIsDir(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	seedRegistryFull(t, homeDir, "lock-dir-app", 4600, homeDir)

	// Create a directory at the lock file path to simulate the broken state.
	// Remove any existing lock file first (Windows may leave one after Save).
	lockPath := filepath.Join(homeDir, ".devports.json.lock")
	os.RemoveAll(lockPath)
	if err := os.Mkdir(lockPath, 0755); err != nil {
		t.Fatalf("create dir at lock path: %v", err)
	}

	out, err := runCmd(t, doctorCmd)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if !strings.Contains(out, "[ERROR] lock file") {
		t.Errorf("expected [ERROR] for lock-is-dir, got: %q", out)
	}
}

func TestDoctorLockFileDirFix(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	seedRegistryFull(t, homeDir, "lock-dir-fix-app", 4601, homeDir)

	lockPath := filepath.Join(homeDir, ".devports.json.lock")
	os.RemoveAll(lockPath)
	if err := os.Mkdir(lockPath, 0755); err != nil {
		t.Fatalf("create dir at lock path: %v", err)
	}

	out, err := runCmd(t, doctorCmd, "--fix")
	if err != nil {
		t.Fatalf("doctor --fix: %v", err)
	}
	if !strings.Contains(out, "[FIXED] lock file") {
		t.Errorf("expected [FIXED] for lock-is-dir, got: %q", out)
	}
	if _, statErr := os.Stat(lockPath); !os.IsNotExist(statErr) {
		t.Error("expected lock path directory to be removed by --fix")
	}
}

func TestDoctorStalePaths(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	reg.Entries["stale-app"] = &registry.Entry{
		Port:           5600,
		ProjectPath:    "/nonexistent/path/that/does/not/exist",
		AllocatedAt:    time.Now().UTC(),
		LastAccessedAt: time.Now().UTC(),
	}
	registry.Save(homeDir, reg)

	out, err := runCmd(t, doctorCmd)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if !strings.Contains(out, "stale") {
		t.Errorf("expected 'stale' in output, got: %q", out)
	}
}

func TestDoctorStalePathsFix(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	reg := newEmptyRegistry()
	reg.Entries["stale-app-fix"] = &registry.Entry{
		Port:           5601,
		ProjectPath:    "/nonexistent/path/that/does/not/exist/fix",
		AllocatedAt:    time.Now().UTC(),
		LastAccessedAt: time.Now().UTC(),
	}
	registry.Save(homeDir, reg)

	out, err := runCmd(t, doctorCmd, "--fix")
	if err != nil {
		t.Fatalf("doctor --fix: %v", err)
	}
	if !strings.Contains(out, "FIXED") {
		t.Errorf("expected FIXED, got: %q", out)
	}
}

func TestDoctorHomeDirError(t *testing.T) {
	cleanupDoctorFlags(t)
	newTestHome(t)
	injectHomeDir(t)

	_, err := runCmd(t, doctorCmd)
	if err == nil {
		t.Fatal("expected homedir error")
	}
}

func TestDoctorRegistryLoadError(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	// Create file first (passes the exists check), then make it unreadable.
	regPath := filepath.Join(homeDir, ".devports.json")
	os.WriteFile(regPath, []byte(""), 0644)
	os.Remove(regPath)
	os.Mkdir(regPath, 0755)

	out, err := runCmd(t, doctorCmd)
	if err != nil {
		t.Fatalf("doctor with bad registry: %v", err)
	}
	if !strings.Contains(out, "ERROR") {
		t.Errorf("expected ERROR for load failure, got: %q", out)
	}
}

func TestDoctorPortInUse(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	dir := t.TempDir()

	// Bind a real port so ProbePort returns false (port in use).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	reg := newEmptyRegistry()
	reg.Entries["in-use-app"] = &registry.Entry{
		Port:           port,
		ProjectPath:    dir,
		AllocatedAt:    time.Now().UTC(),
		LastAccessedAt: time.Now().UTC(),
	}
	registry.Save(homeDir, reg)

	out, err := runCmd(t, doctorCmd)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	// Should report the port as in use (WARN) and also show port availability OK line.
	if !strings.Contains(out, fmt.Sprintf("%d", port)) {
		t.Errorf("expected port %d in output, got: %q", port, out)
	}
}

func TestDoctorNoEntries(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	// Save empty registry (no entries).
	reg := newEmptyRegistry()
	registry.Save(homeDir, reg)

	out, err := runCmd(t, doctorCmd)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if !strings.Contains(out, "[OK]") {
		t.Errorf("expected [OK] output, got: %q", out)
	}
}

func TestDoctorHealthyLockFileAfterNormalWrite(t *testing.T) {
	cleanupDoctorFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "lock-healthy-app", 3004)

	out, err := runCmd(t, doctorCmd, "--fix")
	if err != nil {
		t.Fatalf("doctor --fix: %v", err)
	}
	if strings.Contains(out, "[FIXED] lock file") || strings.Contains(out, "[WARN] lock file") {
		t.Fatalf("expected healthy lock file check, got: %q", out)
	}
	if !strings.Contains(out, "[OK] lock file") {
		t.Fatalf("expected OK lock file result, got: %q", out)
	}
}
