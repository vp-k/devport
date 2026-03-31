package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Overridable OS functions for testing.
var (
	osWriteFile       = os.WriteFile
	osRename          = os.Rename
	jsonMarshalIndent = func(v any, prefix, indent string) ([]byte, error) {
		return json.MarshalIndent(v, prefix, indent)
	}
)

const currentVersion = 1

// registryFileName is the name of the JSON registry file in the home directory.
const registryFileName = ".devports.json"

// registryFilePath returns the full path to the registry file.
func registryFilePath(homeDir string) string {
	return filepath.Join(homeDir, registryFileName)
}

// createEmptyRegistry returns a new Registry with defaults populated.
func createEmptyRegistry() *Registry {
	now := time.Now().UTC()
	return &Registry{
		Version: currentVersion,
		Meta: Meta{
			CreatedAt: now,
			UpdatedAt: now,
		},
		Entries:     make(map[string]*Entry),
		Reserved:    []int{},
		RangePolicy: DefaultRangePolicy(),
	}
}

// loadRegistry reads the registry from homeDir. If the file does not exist an
// empty registry is returned. If the file is corrupted a backup is created and
// an empty registry is returned.
func loadRegistry(homeDir string) (*Registry, error) {
	path := registryFilePath(homeDir)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return createEmptyRegistry(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading registry: %w", err)
	}

	if len(data) == 0 {
		return createEmptyRegistry(), nil
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return recoverCorrupted(homeDir, path, data)
	}

	// Basic schema validation.
	if reg.Entries == nil {
		return recoverCorrupted(homeDir, path, data)
	}
	if reg.Reserved == nil {
		reg.Reserved = []int{}
	}
	if reg.RangePolicy == nil {
		reg.RangePolicy = DefaultRangePolicy()
	}

	return &reg, nil
}

// recoverCorrupted backs up the corrupted registry file and returns a fresh one.
func recoverCorrupted(homeDir, path string, data []byte) (*Registry, error) {
	backupPath := fmt.Sprintf("%s.bak.%d", path, time.Now().UnixMilli())
	_ = os.WriteFile(backupPath, data, 0644)
	fmt.Fprintf(os.Stderr, "devport: registry corrupted, backed up to %s\n", filepath.Base(backupPath))
	return createEmptyRegistry(), nil
}

// lockFilePath returns the path of the sentinel lock file (separate from the
// registry file so Windows allows renaming over the registry).
func lockFilePath(homeDir string) string {
	return filepath.Join(homeDir, registryFileName+".lock")
}

// writeRegistryBody performs the disk write. The caller must already hold the
// write lock; this function does not acquire it.
func writeRegistryBody(homeDir string, reg *Registry) error {
	path := registryFilePath(homeDir)

	reg.Meta.UpdatedAt = time.Now().UTC()

	data, err := jsonMarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling registry: %w", err)
	}

	tmpPath := fmt.Sprintf("%s.tmp.%d.%d", path, os.Getpid(), time.Now().UnixNano())
	if err := osWriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("writing tmp file: %w", err)
	}

	if err := osRename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming tmp file: %w", err)
	}

	return nil
}

// writeRegistry atomically writes the registry to homeDir using a tmp→rename
// pattern protected by a separate lock file.
func writeRegistry(homeDir string, reg *Registry) error {
	// Lock a separate sentinel file so the registry file itself stays unlocked
	// and can be replaced atomically via rename on all platforms including Windows.
	release, err := acquireLock(lockFilePath(homeDir))
	if err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer release() //nolint:errcheck

	return writeRegistryBody(homeDir, reg)
}

// Transaction holds the write lock for the entire load → fn → save sequence,
// preventing concurrent CLI invocations from observing stale registry state
// during a read-modify-write cycle (e.g. port allocation).
// fn receives the loaded registry and may modify it in-place; a non-nil error
// returned by fn aborts the transaction without saving.
func Transaction(homeDir string, fn func(*Registry) error) error {
	release, err := acquireLock(lockFilePath(homeDir))
	if err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer release() //nolint:errcheck

	reg, err := loadRegistry(homeDir)
	if err != nil {
		return err
	}

	if err := fn(reg); err != nil {
		return err
	}

	return writeRegistryBody(homeDir, reg)
}

// Load is the exported entry point for loading the registry.
func Load(homeDir string) (*Registry, error) {
	return loadRegistry(homeDir)
}

// Save is the exported entry point for persisting the registry.
func Save(homeDir string, reg *Registry) error {
	return writeRegistry(homeDir, reg)
}
