package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cleanupEnvFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		envFlagOutput = ""
		envFlagVarName = ""
		envFlagFramework = ""
	})
}

func TestEnvCreatesEnvFile(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "env-test-app")
	// Add next.config.js so framework detects as next.
	os.WriteFile(filepath.Join(dir, "next.config.js"), []byte("module.exports={}"), 0644)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, envCmd)
	if err != nil {
		t.Fatalf("env: %v", err)
	}
	if !strings.Contains(out, "PORT") {
		t.Errorf("expected PORT in output, got: %q", out)
	}

	if _, err := os.Stat(filepath.Join(dir, ".env.local")); os.IsNotExist(err) {
		t.Error("expected .env.local to be created")
	}
}

func TestEnvCustomOutput(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "env-custom-out")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	_, err := runCmd(t, envCmd, "--output", ".env.custom")
	if err != nil {
		t.Fatalf("env --output .env.custom: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".env.custom")); os.IsNotExist(err) {
		t.Error("expected .env.custom to be created")
	}
}

func TestEnvCustomVarName(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "env-custom-var")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, envCmd, "--var-name", "MY_PORT")
	if err != nil {
		t.Fatalf("env --var-name MY_PORT: %v", err)
	}
	if !strings.Contains(out, "MY_PORT") {
		t.Errorf("expected MY_PORT in output, got: %q", out)
	}
}

func TestEnvFrameworkOverride(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "env-fw-override")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, envCmd, "--framework", "vite")
	if err != nil {
		t.Fatalf("env --framework vite: %v", err)
	}
	if !strings.Contains(out, "VITE_PORT") {
		t.Errorf("expected VITE_PORT in output, got: %q", out)
	}
}

func TestEnvExistingEntry(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "env-existing-entry")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// First call creates the entry.
	_, err := runCmd(t, envCmd)
	if err != nil {
		t.Fatalf("env (first): %v", err)
	}
	// Second call hits the else branch (LastAccessedAt update).
	_, err = runCmd(t, envCmd)
	if err != nil {
		t.Fatalf("env (second): %v", err)
	}
}

func TestEnvUpdatesExistingFile(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "env-update-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Pre-create .env.local with a PORT.
	os.WriteFile(filepath.Join(dir, ".env.local"), []byte("PORT=1234\nFOO=bar\n"), 0644)

	_, err := runCmd(t, envCmd)
	if err != nil {
		t.Fatalf("env: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".env.local"))
	content := string(data)
	if strings.Contains(content, "PORT=1234") {
		t.Error("old PORT=1234 should have been replaced")
	}
	if !strings.Contains(content, "FOO=bar") {
		t.Error("FOO=bar should be preserved")
	}
}
