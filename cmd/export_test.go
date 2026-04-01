package cmd

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/user01/devport/internal/registry"
)

func cleanupExportFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		exportFlagOutput = ""
		exportFlagFormat = "json"
	})
}

func seedExportRegistry(t *testing.T, homeDir string) {
	t.Helper()
	reg := newEmptyRegistry()
	now := time.Now().UTC()
	reg.Entries["app-a"] = &registry.Entry{
		Port:           3001,
		KeySource:      registry.KeySourcePackageJSON,
		DisplayName:    "app-a",
		ProjectPath:    "/work/app-a",
		Framework:      "next",
		AllocatedAt:    now,
		LastAccessedAt: now,
	}
	reg.Entries["app-b"] = &registry.Entry{
		Port:           4001,
		KeySource:      registry.KeySourcePackageJSON,
		DisplayName:    "app-b",
		ProjectPath:    "/work/app-b",
		Framework:      "express",
		AllocatedAt:    now,
		LastAccessedAt: now,
	}
	if err := registry.Save(homeDir, reg); err != nil {
		t.Fatal(err)
	}
}

func TestExportJSONStdout(t *testing.T) {
	cleanupExportFlags(t)
	homeDir := newTestHome(t)
	seedExportRegistry(t, homeDir)

	out, err := runCmd(t, exportCmd)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	var entries []exportEntry
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &entries); jsonErr != nil {
		t.Fatalf("invalid JSON: %v — output: %q", jsonErr, out)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
	// Entries should be sorted by key.
	if entries[0].Key != "app-a" || entries[1].Key != "app-b" {
		t.Errorf("unexpected order: %v %v", entries[0].Key, entries[1].Key)
	}
}

func TestExportJSONEmpty(t *testing.T) {
	cleanupExportFlags(t)
	newTestHome(t) // empty registry

	out, err := runCmd(t, exportCmd)
	if err != nil {
		t.Fatalf("export empty: %v", err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("expected [], got %q", out)
	}
}

func TestExportCSVStdout(t *testing.T) {
	cleanupExportFlags(t)
	homeDir := newTestHome(t)
	seedExportRegistry(t, homeDir)

	out, err := runCmd(t, exportCmd, "--format", "csv")
	if err != nil {
		t.Fatalf("export csv: %v", err)
	}

	r := csv.NewReader(strings.NewReader(out))
	records, csvErr := r.ReadAll()
	if csvErr != nil {
		t.Fatalf("parse CSV: %v", csvErr)
	}
	// Header + 2 data rows.
	if len(records) != 3 {
		t.Errorf("expected 3 CSV rows, got %d", len(records))
	}
	if records[0][0] != "key" {
		t.Errorf("expected CSV header, got: %v", records[0])
	}
}

func TestExportCSVEmpty(t *testing.T) {
	cleanupExportFlags(t)
	newTestHome(t)

	out, err := runCmd(t, exportCmd, "--format", "csv")
	if err != nil {
		t.Fatalf("export csv empty: %v", err)
	}
	// Only header row.
	r := csv.NewReader(strings.NewReader(out))
	records, _ := r.ReadAll()
	if len(records) != 1 {
		t.Errorf("expected 1 CSV row (header only), got %d", len(records))
	}
}

func TestExportToFile(t *testing.T) {
	cleanupExportFlags(t)
	homeDir := newTestHome(t)
	seedExportRegistry(t, homeDir)

	outFile := filepath.Join(t.TempDir(), "backup.json")

	_, err := runCmd(t, exportCmd, "--output", outFile)
	if err != nil {
		t.Fatalf("export --output: %v", err)
	}

	data, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatalf("read output file: %v", readErr)
	}
	var entries []exportEntry
	if jsonErr := json.Unmarshal(data, &entries); jsonErr != nil {
		t.Fatalf("invalid JSON in file: %v", jsonErr)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries in file, got %d", len(entries))
	}
}

func TestExportToFileCreateError(t *testing.T) {
	cleanupExportFlags(t)
	homeDir := newTestHome(t)
	seedExportRegistry(t, homeDir)

	// Use a path inside a nonexistent directory.
	badPath := filepath.Join(t.TempDir(), "nonexistent-dir", "out.json")

	_, err := runCmd(t, exportCmd, "--output", badPath)
	if err == nil {
		t.Fatal("expected error creating output file, got nil")
	}
}

func TestExportHomeDirError(t *testing.T) {
	cleanupExportFlags(t)
	newTestHome(t)
	injectHomeDir(t)

	_, err := runCmd(t, exportCmd)
	if err == nil {
		t.Fatal("expected homedir error")
	}
}

func TestExportRegistryLoadError(t *testing.T) {
	cleanupExportFlags(t)
	newTestHome(t)
	injectRegistryLoad(t)

	_, err := runCmd(t, exportCmd)
	if err == nil {
		t.Fatal("expected registry load error")
	}
}
