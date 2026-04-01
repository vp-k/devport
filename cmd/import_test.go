package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vp-k/devport/internal/registry"
)

func cleanupImportFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		importFlagMerge = false
		importFlagOverwrite = false
		importFlagDryRun = false
	})
}

// writeImportFile writes a JSON export file to a temp dir and returns the path.
func writeImportFile(t *testing.T, entries []exportEntry) string {
	t.Helper()
	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "import.json")
	if err := os.WriteFile(f, data, 0644); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestImportBasic(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	entries := []exportEntry{
		{Key: "imp-app", Port: 5500, DisplayName: "imp-app", Framework: "next",
			ProjectPath: "/work/imp-app", AllocatedAt: "2026-01-01T00:00:00Z"},
	}
	f := writeImportFile(t, entries)

	out, err := runCmd(t, importCmd, f)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(out, "1 add") {
		t.Errorf("expected '1 add', got: %q", out)
	}

	reg, _ := cmdRegistryLoad(homeDir)
	if reg.Entries["imp-app"] == nil {
		t.Error("expected imp-app to be imported")
	}
	if reg.Entries["imp-app"].Port != 5500 {
		t.Errorf("expected port 5500, got %d", reg.Entries["imp-app"].Port)
	}
}

func TestImportMergeSkipsExisting(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	// Seed existing entry.
	reg := newEmptyRegistry()
	now := time.Now().UTC()
	reg.Entries["existing-app"] = &registry.Entry{
		Port: 5600, KeySource: registry.KeySourcePackageJSON,
		DisplayName: "existing-app", AllocatedAt: now, LastAccessedAt: now,
	}
	registry.Save(homeDir, reg)

	entries := []exportEntry{
		{Key: "existing-app", Port: 9999, DisplayName: "existing-app",
			AllocatedAt: "2026-01-01T00:00:00Z"},
	}
	f := writeImportFile(t, entries)

	out, err := runCmd(t, importCmd, f)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(out, "Nothing to import") {
		t.Errorf("expected 'Nothing to import', got: %q", out)
	}

	// Port should still be 5600 (not overwritten).
	reg2, _ := cmdRegistryLoad(homeDir)
	if reg2.Entries["existing-app"].Port != 5600 {
		t.Errorf("expected port 5600 unchanged, got %d", reg2.Entries["existing-app"].Port)
	}
}

func TestImportOverwrite(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	now := time.Now().UTC()
	reg := newEmptyRegistry()
	reg.Entries["ow-app"] = &registry.Entry{
		Port: 5700, KeySource: registry.KeySourcePackageJSON,
		DisplayName: "ow-app", AllocatedAt: now, LastAccessedAt: now,
	}
	registry.Save(homeDir, reg)

	entries := []exportEntry{
		{Key: "ow-app", Port: 5701, DisplayName: "ow-app",
			AllocatedAt: "2026-01-01T00:00:00Z"},
	}
	f := writeImportFile(t, entries)

	out, err := runCmd(t, importCmd, "--overwrite", f)
	if err != nil {
		t.Fatalf("import --overwrite: %v", err)
	}
	if !strings.Contains(out, "overwrite") {
		t.Errorf("expected 'overwrite' in output, got: %q", out)
	}

	reg2, _ := cmdRegistryLoad(homeDir)
	if reg2.Entries["ow-app"].Port != 5701 {
		t.Errorf("expected port 5701, got %d", reg2.Entries["ow-app"].Port)
	}
}

func TestImportMergeConflictsWithOverwrite(t *testing.T) {
	cleanupImportFlags(t)
	newTestHome(t)

	entries := []exportEntry{
		{Key: "merge-overwrite-app", Port: 5701, DisplayName: "merge-overwrite-app",
			AllocatedAt: "2026-01-01T00:00:00Z"},
	}
	f := writeImportFile(t, entries)

	_, err := runCmd(t, importCmd, "--merge", "--overwrite", f)
	if err == nil {
		t.Fatal("expected conflict error for --merge with --overwrite")
	}
}

func TestImportDryRun(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	entries := []exportEntry{
		{Key: "dry-app", Port: 5800, DisplayName: "dry-app",
			AllocatedAt: "2026-01-01T00:00:00Z"},
	}
	f := writeImportFile(t, entries)

	out, err := runCmd(t, importCmd, "--dry-run", f)
	if err != nil {
		t.Fatalf("import --dry-run: %v", err)
	}
	if !strings.Contains(out, "Dry-run") {
		t.Errorf("expected 'Dry-run', got: %q", out)
	}

	// Nothing should be saved.
	reg, _ := cmdRegistryLoad(homeDir)
	if reg.Entries["dry-app"] != nil {
		t.Error("dry-run should not save to registry")
	}
}

func TestImportPortConflict(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	now := time.Now().UTC()
	reg := newEmptyRegistry()
	reg.Entries["owner-app"] = &registry.Entry{
		Port: 5900, KeySource: registry.KeySourcePackageJSON,
		DisplayName: "owner-app", AllocatedAt: now, LastAccessedAt: now,
	}
	registry.Save(homeDir, reg)

	// Incoming entry wants port 5900, but it belongs to "owner-app".
	entries := []exportEntry{
		{Key: "conflict-app", Port: 5900, DisplayName: "conflict-app",
			AllocatedAt: "2026-01-01T00:00:00Z"},
	}
	f := writeImportFile(t, entries)

	out, err := runCmd(t, importCmd, f)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(out, "port 5900 taken by") {
		t.Errorf("expected port-conflict skip, got: %q", out)
	}
	if !strings.Contains(out, "Nothing to import") {
		t.Errorf("expected 'Nothing to import', got: %q", out)
	}
}

func TestImportInvalidAllocatedAt(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	// AllocatedAt with bad format — should fall back to now.
	entries := []exportEntry{
		{Key: "bad-time-app", Port: 6000, DisplayName: "bad-time-app",
			AllocatedAt: "not-a-date"},
	}
	f := writeImportFile(t, entries)

	_, err := runCmd(t, importCmd, f)
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	reg, _ := cmdRegistryLoad(homeDir)
	if reg.Entries["bad-time-app"] == nil {
		t.Error("expected bad-time-app to be imported")
	}
}

func TestImportFileNotFound(t *testing.T) {
	cleanupImportFlags(t)
	newTestHome(t)

	_, err := runCmd(t, importCmd, "/nonexistent/path/import.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestImportInvalidJSON(t *testing.T) {
	cleanupImportFlags(t)
	newTestHome(t)

	f := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(f, []byte("{not json}"), 0644)

	_, err := runCmd(t, importCmd, f)
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
}

func TestImportHomeDirError(t *testing.T) {
	cleanupImportFlags(t)
	newTestHome(t)

	entries := []exportEntry{{Key: "x", Port: 1234, AllocatedAt: "2026-01-01T00:00:00Z"}}
	f := writeImportFile(t, entries)
	injectHomeDir(t)

	_, err := runCmd(t, importCmd, f)
	if err == nil {
		t.Fatal("expected homedir error")
	}
}

func TestImportRegistryLoadError(t *testing.T) {
	cleanupImportFlags(t)
	newTestHome(t)

	entries := []exportEntry{{Key: "x", Port: 1234, AllocatedAt: "2026-01-01T00:00:00Z"}}
	f := writeImportFile(t, entries)
	injectRegistryLoad(t)

	_, err := runCmd(t, importCmd, f)
	if err == nil {
		t.Fatal("expected registry load error")
	}
}

func TestImportMixedSkipAndAdd(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	now := time.Now().UTC()
	reg := newEmptyRegistry()
	reg.Entries["existing-mix"] = &registry.Entry{
		Port: 6200, KeySource: registry.KeySourcePackageJSON,
		DisplayName: "existing-mix", AllocatedAt: now, LastAccessedAt: now,
	}
	registry.Save(homeDir, reg)

	// One skipped (existing) + one added (new).
	entries := []exportEntry{
		{Key: "existing-mix", Port: 6200, DisplayName: "existing-mix",
			AllocatedAt: "2026-01-01T00:00:00Z"},
		{Key: "new-mix", Port: 6201, DisplayName: "new-mix",
			AllocatedAt: "2026-01-01T00:00:00Z"},
	}
	f := writeImportFile(t, entries)

	out, err := runCmd(t, importCmd, f)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(out, "1 add") {
		t.Errorf("expected '1 add', got: %q", out)
	}
	if !strings.Contains(out, "1 skip") {
		t.Errorf("expected '1 skip', got: %q", out)
	}

	reg2, _ := cmdRegistryLoad(homeDir)
	if reg2.Entries["new-mix"] == nil {
		t.Error("expected new-mix to be imported")
	}
	if reg2.Entries["existing-mix"].Port != 6200 {
		t.Error("expected existing-mix to remain unchanged")
	}
}

func TestImportIncomingKeyDuplicate(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	// Two entries share the same key but different ports.
	// Only the first should be imported; the second is a duplicate key.
	entries := []exportEntry{
		{Key: "dup-key-app", Port: 7500, DisplayName: "dup-key-app", AllocatedAt: "2026-01-01T00:00:00Z"},
		{Key: "dup-key-app", Port: 7501, DisplayName: "dup-key-app", AllocatedAt: "2026-01-01T01:00:00Z"},
	}
	f := writeImportFile(t, entries)

	out, err := runCmd(t, importCmd, f)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(out, "duplicate key") {
		t.Errorf("expected 'duplicate key' warning, got: %q", out)
	}
	if !strings.Contains(out, "1 add") {
		t.Errorf("expected '1 add', got: %q", out)
	}
	if !strings.Contains(out, "1 skip") {
		t.Errorf("expected '1 skip', got: %q", out)
	}

	reg, _ := cmdRegistryLoad(homeDir)
	if reg.Entries["dup-key-app"] == nil {
		t.Fatal("expected dup-key-app to be imported")
	}
	if reg.Entries["dup-key-app"].Port != 7500 {
		t.Errorf("expected first occurrence port 7500, got %d", reg.Entries["dup-key-app"].Port)
	}
}

func TestImportKeyDuplicateAfterSkip(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	// Seed the key so the first occurrence is skip-exists (merge mode).
	now := time.Now().UTC()
	reg := newEmptyRegistry()
	reg.Entries["skip-dup-app"] = &registry.Entry{
		Port: 7600, KeySource: registry.KeySourcePackageJSON,
		DisplayName: "skip-dup-app", AllocatedAt: now, LastAccessedAt: now,
	}
	registry.Save(homeDir, reg)

	// Both entries have the same key. First → skip-exists, second → skip-key.
	entries := []exportEntry{
		{Key: "skip-dup-app", Port: 7600, DisplayName: "skip-dup-app", AllocatedAt: "2026-01-01T00:00:00Z"},
		{Key: "skip-dup-app", Port: 7601, DisplayName: "skip-dup-app", AllocatedAt: "2026-01-01T01:00:00Z"},
	}
	f := writeImportFile(t, entries)

	out, err := runCmd(t, importCmd, f)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(out, "Nothing to import") {
		t.Errorf("expected 'Nothing to import', got: %q", out)
	}
	// Both entries should have been skipped.
	if strings.Contains(out, "add") {
		t.Errorf("expected no adds, got: %q", out)
	}

	// Registry should be unchanged.
	reg2, _ := cmdRegistryLoad(homeDir)
	if reg2.Entries["skip-dup-app"].Port != 7600 {
		t.Errorf("expected port 7600 unchanged, got %d", reg2.Entries["skip-dup-app"].Port)
	}
}

func TestImportKeyDuplicateAfterPortConflict(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	now := time.Now().UTC()
	reg := newEmptyRegistry()
	reg.Entries["owner-app"] = &registry.Entry{
		Port: 7700, KeySource: registry.KeySourcePackageJSON,
		DisplayName: "owner-app", AllocatedAt: now, LastAccessedAt: now,
	}
	registry.Save(homeDir, reg)

	entries := []exportEntry{
		{Key: "retry-key-app", Port: 7700, DisplayName: "retry-key-app", AllocatedAt: "2026-01-01T00:00:00Z"},
		{Key: "retry-key-app", Port: 7701, DisplayName: "retry-key-app", AllocatedAt: "2026-01-01T01:00:00Z"},
	}
	f := writeImportFile(t, entries)

	out, err := runCmd(t, importCmd, f)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(out, "port 7700 taken by") {
		t.Errorf("expected port-conflict skip, got: %q", out)
	}
	if !strings.Contains(out, "duplicate key in payload") {
		t.Errorf("expected duplicate-key skip, got: %q", out)
	}
	if strings.Contains(out, "1 add") || strings.Contains(out, "Imported: 1 add") {
		t.Errorf("expected no adds after port-conflict duplicate key, got: %q", out)
	}
	if !strings.Contains(out, "Nothing to import") {
		t.Errorf("expected 'Nothing to import', got: %q", out)
	}

	reg2, _ := cmdRegistryLoad(homeDir)
	if reg2.Entries["retry-key-app"] != nil {
		t.Error("expected retry-key-app to be skipped entirely")
	}
	if reg2.Entries["owner-app"] == nil || reg2.Entries["owner-app"].Port != 7700 {
		t.Error("expected existing owner-app entry to remain unchanged")
	}
}

func TestImportIncomingPortConflict(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	// Two incoming entries claim the same port — only the first should be added.
	entries := []exportEntry{
		{Key: "first-app", Port: 7100, DisplayName: "first-app", AllocatedAt: "2026-01-01T00:00:00Z"},
		{Key: "second-app", Port: 7100, DisplayName: "second-app", AllocatedAt: "2026-01-01T00:00:00Z"},
	}
	f := writeImportFile(t, entries)

	out, err := runCmd(t, importCmd, f)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(out, "port 7100 taken by") {
		t.Errorf("expected port-conflict skip, got: %q", out)
	}
	if !strings.Contains(out, "1 add") {
		t.Errorf("expected '1 add', got: %q", out)
	}

	reg, _ := cmdRegistryLoad(homeDir)
	if reg.Entries["first-app"] == nil {
		t.Error("expected first-app to be imported")
	}
	if reg.Entries["second-app"] != nil {
		t.Error("expected second-app to be skipped due to port conflict")
	}
}

func TestImportKeySourcePreserved(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	entries := []exportEntry{
		{Key: "ks-app", Port: 7200, DisplayName: "ks-app",
			KeySource: "git-remote", AllocatedAt: "2026-01-01T00:00:00Z"},
	}
	f := writeImportFile(t, entries)

	if _, err := runCmd(t, importCmd, f); err != nil {
		t.Fatalf("import: %v", err)
	}

	reg, _ := cmdRegistryLoad(homeDir)
	if reg.Entries["ks-app"] == nil {
		t.Fatal("expected ks-app to be imported")
	}
	if reg.Entries["ks-app"].KeySource != registry.KeySourceGitRemote {
		t.Errorf("expected KeySource git-remote, got %q", reg.Entries["ks-app"].KeySource)
	}
}

func TestImportKeySourceFallbackToPath(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	// No keySource field in payload → should default to "path".
	entries := []exportEntry{
		{Key: "ks-fallback-app", Port: 7300, DisplayName: "ks-fallback-app",
			AllocatedAt: "2026-01-01T00:00:00Z"},
	}
	f := writeImportFile(t, entries)

	if _, err := runCmd(t, importCmd, f); err != nil {
		t.Fatalf("import: %v", err)
	}

	reg, _ := cmdRegistryLoad(homeDir)
	if reg.Entries["ks-fallback-app"] == nil {
		t.Fatal("expected ks-fallback-app to be imported")
	}
	if reg.Entries["ks-fallback-app"].KeySource != registry.KeySourcePath {
		t.Errorf("expected KeySource path, got %q", reg.Entries["ks-fallback-app"].KeySource)
	}
}

func TestImportKeySourceOverwritePreservesExisting(t *testing.T) {
	cleanupImportFlags(t)
	homeDir := newTestHome(t)

	now := time.Now().UTC()
	reg := newEmptyRegistry()
	reg.Entries["ks-ow-app"] = &registry.Entry{
		Port: 7400, KeySource: registry.KeySourceGitRemote,
		DisplayName: "ks-ow-app", AllocatedAt: now, LastAccessedAt: now,
	}
	registry.Save(homeDir, reg)

	// Overwrite with no keySource field → should keep existing git-remote.
	entries := []exportEntry{
		{Key: "ks-ow-app", Port: 7401, DisplayName: "ks-ow-app",
			AllocatedAt: "2026-01-01T00:00:00Z"},
	}
	f := writeImportFile(t, entries)

	if _, err := runCmd(t, importCmd, "--overwrite", f); err != nil {
		t.Fatalf("import --overwrite: %v", err)
	}

	reg2, _ := cmdRegistryLoad(homeDir)
	if reg2.Entries["ks-ow-app"] == nil {
		t.Fatal("expected ks-ow-app to exist")
	}
	if reg2.Entries["ks-ow-app"].KeySource != registry.KeySourceGitRemote {
		t.Errorf("expected KeySource git-remote preserved on overwrite, got %q", reg2.Entries["ks-ow-app"].KeySource)
	}
}

func TestImportTransactionError(t *testing.T) {
	cleanupImportFlags(t)
	newTestHome(t)

	entries := []exportEntry{
		{Key: "tx-err-imp-app", Port: 6100, DisplayName: "tx-err-imp-app",
			AllocatedAt: "2026-01-01T00:00:00Z"},
	}
	f := writeImportFile(t, entries)
	injectTransaction(t)

	_, err := runCmd(t, importCmd, f)
	if err == nil {
		t.Fatal("expected transaction error")
	}
}
