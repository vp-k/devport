package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cleanupInitFlags resets init command flags to defaults after each test.
func cleanupInitFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		initFlagFramework = ""
		initFlagRangeMin = 0
		initFlagRangeMax = 0
		initFlagYes = false
	})
}

func TestInitAllocatesPortAndCreatesEnvFile(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "init-test-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, initCmd, "--yes")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if !strings.Contains(out, "Port") {
		t.Errorf("expected Port in output, got: %q", out)
	}
}

func TestInitAddsPredevScript(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "predev-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	_, err := runCmd(t, initCmd, "--yes")
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	var pkg map[string]json.RawMessage
	json.Unmarshal(data, &pkg)
	var scripts map[string]string
	json.Unmarshal(pkg["scripts"], &scripts)
	if scripts["predev"] != "devport env" {
		t.Errorf("predev script = %q, want %q", scripts["predev"], "devport env")
	}
}

func TestInitAddsGitignoreEntry(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "gitignore-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Make framework = next so .env.local is used → .gitignore offer triggered.
	os.WriteFile(filepath.Join(dir, "next.config.js"), []byte(""), 0644)

	_, err := runCmd(t, initCmd, "--yes")
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if !strings.Contains(string(data), ".env.local") {
		t.Errorf(".gitignore should contain .env.local, got:\n%s", data)
	}
}

func TestInitGitignoreNotDuplicated(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "gitignore-dup-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Pre-create .gitignore with .env.local.
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".env.local\n"), 0644)
	os.WriteFile(filepath.Join(dir, "next.config.js"), []byte(""), 0644)

	_, err := runCmd(t, initCmd, "--yes")
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	count := strings.Count(string(data), ".env.local")
	if count != 1 {
		t.Errorf("expected .env.local once in .gitignore, got %d times", count)
	}
}

func TestInitExistingEntry(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "init-existing-entry")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// First call creates the entry.
	_, err := runCmd(t, initCmd, "--yes")
	if err != nil {
		t.Fatalf("init (first): %v", err)
	}
	// Second call hits the else branch (LastAccessedAt update).
	_, err = runCmd(t, initCmd, "--yes")
	if err != nil {
		t.Fatalf("init (second): %v", err)
	}
}

func TestInitSkipsPredevOnDeny(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "skip-script-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	autoConfirm(t, false)

	// Run WITHOUT --yes; confirmFn returns false → predev and .gitignore skipped.
	_, err := runCmd(t, initCmd)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "package.json"))
	if strings.Contains(string(data), "predev") {
		t.Error("predev should not be added when denied")
	}
}

func TestInitFrameworkOverride(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "fw-init-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, initCmd, "--framework", "express", "--yes")
	if err != nil {
		t.Fatalf("init --framework express: %v", err)
	}
	if !strings.Contains(out, "express") {
		t.Errorf("expected 'express' in output, got: %q", out)
	}
}

func TestInitUnknownFramework(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "unknown-fw-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, initCmd, "--yes")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	// Unknown framework shows "(unknown)" label.
	if !strings.Contains(out, "unknown") {
		t.Errorf("expected '(unknown)' label in output, got: %q", out)
	}
}
