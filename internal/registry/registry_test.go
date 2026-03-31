package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// registryPath returns the devports.json path inside tmpDir.
func registryPath(tmpDir string) string {
	return filepath.Join(tmpDir, ".devports.json")
}

// withTempHome sets HOME (and USERPROFILE on Windows) to tmpDir for the
// duration of the test and returns the registry path.
func withTempHome(t *testing.T) (string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)
	return tmpDir, registryPath(tmpDir)
}

// --- createEmptyRegistry ---

func TestCreateEmptyRegistry(t *testing.T) {
	r := createEmptyRegistry()

	if r.Version != currentVersion {
		t.Errorf("version = %d, want %d", r.Version, currentVersion)
	}
	if r.Entries == nil {
		t.Error("entries is nil")
	}
	if len(r.Entries) != 0 {
		t.Errorf("entries len = %d, want 0", len(r.Entries))
	}
	if r.Reserved == nil {
		t.Error("reserved is nil")
	}
	if len(r.Reserved) != 0 {
		t.Errorf("reserved len = %d, want 0", len(r.Reserved))
	}
	if r.RangePolicy == nil {
		t.Error("rangePolicy is nil")
	}
	if _, ok := r.RangePolicy["default"]; !ok {
		t.Error("rangePolicy missing 'default' key")
	}
}

// --- loadRegistry ---

func TestLoadRegistryMissingFile(t *testing.T) {
	tmpDir, _ := withTempHome(t)
	// Do NOT create the file — loadRegistry should return an empty registry.
	r, err := loadRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(r.Entries))
	}
}

func TestLoadRegistryValidFile(t *testing.T) {
	tmpDir, path := withTempHome(t)
	reg := createEmptyRegistry()
	reg.Entries["my-app"] = &Entry{
		Port:           3001,
		KeySource:      KeySourcePackageJSON,
		DisplayName:    "my-app",
		ProjectPath:    "/work/my-app",
		Framework:      "next",
		AllocatedAt:    time.Now().UTC().Truncate(time.Second),
		LastAccessedAt: time.Now().UTC().Truncate(time.Second),
		RangeMin:       3000,
		RangeMax:       3999,
	}
	data, _ := json.MarshalIndent(reg, "", "  ")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := loadRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entry, ok := loaded.Entries["my-app"]
	if !ok {
		t.Fatal("entry 'my-app' not found")
	}
	if entry.Port != 3001 {
		t.Errorf("port = %d, want 3001", entry.Port)
	}
}

func TestLoadRegistryCorruptedFile(t *testing.T) {
	tmpDir, path := withTempHome(t)
	// Write invalid JSON to simulate corruption.
	if err := os.WriteFile(path, []byte(`{"version":1,"entries":`), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := loadRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error on corrupted file: %v", err)
	}
	// Should return empty registry.
	if len(r.Entries) != 0 {
		t.Errorf("expected empty entries after corruption, got %d", len(r.Entries))
	}
	// Backup file should exist.
	entries, _ := os.ReadDir(tmpDir)
	backupFound := false
	for _, e := range entries {
		if len(e.Name()) > len(".devports.json.bak.") &&
			e.Name()[:len(".devports.json.bak.")] == ".devports.json.bak." {
			backupFound = true
		}
	}
	if !backupFound {
		t.Error("expected backup file to be created, but none found")
	}
}

func TestLoadRegistryEmptyFile(t *testing.T) {
	tmpDir, path := withTempHome(t)
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := loadRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error on empty file: %v", err)
	}
	if len(r.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(r.Entries))
	}
}

// --- writeRegistry ---

func TestLoadRegistryReadError(t *testing.T) {
	tmpDir, path := withTempHome(t)
	// Create a directory with the same name as the registry file.
	// os.ReadFile on a directory returns a non-NotExist error.
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	_, err := loadRegistry(tmpDir)
	if err == nil {
		t.Fatal("expected error reading directory as file, got nil")
	}
}

func TestLoadRegistryNilEntries(t *testing.T) {
	tmpDir, path := withTempHome(t)
	// Write registry JSON with null entries to trigger the nil check.
	raw := `{"version":1,"meta":{"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"},"entries":null,"reserved":[]}`
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := loadRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(r.Entries))
	}
}

func TestLoadRegistryNilReserved(t *testing.T) {
	tmpDir, path := withTempHome(t)
	raw := `{"version":1,"meta":{"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"},"entries":{},"reserved":null,"rangePolicy":{"default":{"min":3000,"max":9999}}}`
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := loadRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Reserved == nil {
		t.Error("reserved should not be nil after load")
	}
}

func TestLoadRegistryNilRangePolicy(t *testing.T) {
	tmpDir, path := withTempHome(t)
	raw := `{"version":1,"meta":{"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"},"entries":{},"reserved":[],"rangePolicy":null}`
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := loadRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.RangePolicy == nil {
		t.Error("rangePolicy should not be nil after load")
	}
	if _, ok := r.RangePolicy["default"]; !ok {
		t.Error("expected default range policy to be restored")
	}
}

func TestWriteRegistryLockError(t *testing.T) {
	tmpDir, _ := withTempHome(t)
	orig := newFlock
	t.Cleanup(func() { newFlock = orig })
	newFlock = func(_ string) flockLocker { return &failLocker{} }

	reg := createEmptyRegistry()
	if err := writeRegistry(tmpDir, reg); err == nil {
		t.Fatal("expected error from lock failure, got nil")
	}
}

func TestWriteRegistryMarshalError(t *testing.T) {
	tmpDir, _ := withTempHome(t)
	orig := jsonMarshalIndent
	t.Cleanup(func() { jsonMarshalIndent = orig })
	jsonMarshalIndent = func(_ any, _, _ string) ([]byte, error) {
		return nil, errors.New("marshal error")
	}

	reg := createEmptyRegistry()
	if err := writeRegistry(tmpDir, reg); err == nil {
		t.Fatal("expected error from MarshalIndent, got nil")
	}
}

func TestWriteRegistryWriteFileError(t *testing.T) {
	tmpDir, _ := withTempHome(t)
	orig := osWriteFile
	t.Cleanup(func() { osWriteFile = orig })
	osWriteFile = func(_ string, _ []byte, _ os.FileMode) error {
		return errors.New("disk full")
	}

	reg := createEmptyRegistry()
	if err := writeRegistry(tmpDir, reg); err == nil {
		t.Fatal("expected error from WriteFile, got nil")
	}
}

func TestWriteRegistryRenameError(t *testing.T) {
	tmpDir, _ := withTempHome(t)
	orig := osRename
	t.Cleanup(func() { osRename = orig })
	osRename = func(_, _ string) error {
		return errors.New("rename failed")
	}

	reg := createEmptyRegistry()
	if err := writeRegistry(tmpDir, reg); err == nil {
		t.Fatal("expected error from Rename, got nil")
	}
}

func TestWriteRegistryRoundTrip(t *testing.T) {
	tmpDir, path := withTempHome(t)
	reg := createEmptyRegistry()
	reg.Entries["test-app"] = &Entry{
		Port:        4000,
		DisplayName: "test-app",
		ProjectPath: "/work/test-app",
	}

	if err := writeRegistry(tmpDir, reg); err != nil {
		t.Fatalf("writeRegistry: %v", err)
	}

	// File must exist.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("registry file not created")
	}

	// Round-trip: load and verify.
	loaded, err := loadRegistry(tmpDir)
	if err != nil {
		t.Fatalf("loadRegistry after write: %v", err)
	}
	entry, ok := loaded.Entries["test-app"]
	if !ok {
		t.Fatal("entry 'test-app' not found after round-trip")
	}
	if entry.Port != 4000 {
		t.Errorf("port = %d, want 4000", entry.Port)
	}
}

func TestWriteRegistryNoTmpFileLeft(t *testing.T) {
	tmpDir, _ := withTempHome(t)
	reg := createEmptyRegistry()

	if err := writeRegistry(tmpDir, reg); err != nil {
		t.Fatalf("writeRegistry: %v", err)
	}

	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		if len(e.Name()) > 4 && e.Name()[len(e.Name())-4:] == ".tmp" {
			t.Errorf("tmp file left behind: %s", e.Name())
		}
	}
}

func TestWriteRegistryConcurrent(t *testing.T) {
	tmpDir, _ := withTempHome(t)

	const workers = 10
	var wg sync.WaitGroup
	errs := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			reg := createEmptyRegistry()
			if err := writeRegistry(tmpDir, reg); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent write error: %v", err)
	}

	// Registry must be loadable after concurrent writes.
	if _, err := loadRegistry(tmpDir); err != nil {
		t.Fatalf("registry unreadable after concurrent writes: %v", err)
	}
}

// TestLoadSaveExported exercises the exported Load and Save wrappers.
func TestLoadSaveExported(t *testing.T) {
	tmpDir, _ := withTempHome(t)

	// Load from empty dir returns a fresh registry.
	reg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(reg.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(reg.Entries))
	}

	// Save and reload round-trips successfully.
	reg.Entries["test-key"] = &Entry{
		Port:           3333,
		KeySource:      KeySourcePackageJSON,
		DisplayName:    "test",
		AllocatedAt:    reg.Meta.CreatedAt,
		LastAccessedAt: reg.Meta.CreatedAt,
	}
	if err := Save(tmpDir, reg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if loaded.Entries["test-key"] == nil || loaded.Entries["test-key"].Port != 3333 {
		t.Error("expected test-key entry with port 3333 after round-trip")
	}
}

// --- Transaction ---

func TestTransaction(t *testing.T) {
	tmpDir, _ := withTempHome(t)

	// Successful transaction modifies the registry.
	err := Transaction(tmpDir, func(reg *Registry) error {
		reg.Entries["tx-key"] = &Entry{Port: 4444}
		return nil
	})
	if err != nil {
		t.Fatalf("Transaction: %v", err)
	}
	loaded, _ := Load(tmpDir)
	if loaded.Entries["tx-key"] == nil || loaded.Entries["tx-key"].Port != 4444 {
		t.Error("expected tx-key with port 4444 after transaction")
	}
}

func TestTransactionFnError(t *testing.T) {
	tmpDir, _ := withTempHome(t)

	sentinel := errors.New("fn error")
	err := Transaction(tmpDir, func(reg *Registry) error {
		reg.Entries["should-not-persist"] = &Entry{Port: 5555}
		return sentinel
	})
	if err != sentinel {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	// The registry must NOT be saved when fn returns an error.
	loaded, _ := Load(tmpDir)
	if loaded.Entries["should-not-persist"] != nil {
		t.Error("entry must not be persisted when fn returns error")
	}
}

func TestTransactionLockError(t *testing.T) {
	tmpDir := t.TempDir()
	// Make the lock file a directory so os.OpenFile fails.
	lockPath := filepath.Join(tmpDir, ".devports.json.lock")
	if err := os.Mkdir(lockPath, 0755); err != nil {
		t.Fatal(err)
	}

	err := Transaction(tmpDir, func(_ *Registry) error { return nil })
	if err == nil {
		t.Fatal("expected lock error, got nil")
	}
}

func TestTransactionLoadError(t *testing.T) {
	tmpDir := t.TempDir()
	// Make the registry file a directory so os.ReadFile returns a non-IsNotExist error.
	regPath := filepath.Join(tmpDir, ".devports.json")
	if err := os.Mkdir(regPath, 0755); err != nil {
		t.Fatal(err)
	}

	err := Transaction(tmpDir, func(_ *Registry) error { return nil })
	if err == nil {
		t.Fatal("expected load error, got nil")
	}
}
