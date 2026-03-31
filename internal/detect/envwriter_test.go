package detect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- EnvConfig resolution ----

func TestEnvConfigForNext(t *testing.T) {
	cfg := EnvConfigFor("next", EnvOptions{})
	if cfg.File != ".env.local" {
		t.Errorf("file = %q, want .env.local", cfg.File)
	}
	if cfg.VarName != "PORT" {
		t.Errorf("varName = %q, want PORT", cfg.VarName)
	}
}

func TestEnvConfigForVite(t *testing.T) {
	cfg := EnvConfigFor("vite", EnvOptions{})
	if cfg.File != ".env.local" {
		t.Errorf("file = %q, want .env.local", cfg.File)
	}
	if cfg.VarName != "VITE_PORT" {
		t.Errorf("varName = %q, want VITE_PORT", cfg.VarName)
	}
}

func TestEnvConfigForExpress(t *testing.T) {
	cfg := EnvConfigFor("express", EnvOptions{})
	if cfg.File != ".env" {
		t.Errorf("file = %q, want .env", cfg.File)
	}
	if cfg.VarName != "PORT" {
		t.Errorf("varName = %q, want PORT", cfg.VarName)
	}
}

func TestEnvConfigForAngular(t *testing.T) {
	cfg := EnvConfigFor("angular", EnvOptions{})
	if cfg.File != ".env.local" {
		t.Errorf("file = %q, want .env.local", cfg.File)
	}
}

func TestEnvConfigForNest(t *testing.T) {
	cfg := EnvConfigFor("nest", EnvOptions{})
	if cfg.File != ".env" {
		t.Errorf("file = %q, want .env", cfg.File)
	}
}

func TestEnvConfigForCRA(t *testing.T) {
	cfg := EnvConfigFor("cra", EnvOptions{})
	if cfg.File != ".env.local" {
		t.Errorf("file = %q, want .env.local", cfg.File)
	}
}

func TestEnvConfigForUnknown(t *testing.T) {
	cfg := EnvConfigFor("", EnvOptions{})
	if cfg.File != ".env.local" {
		t.Errorf("file = %q, want .env.local (fallback)", cfg.File)
	}
	if cfg.VarName != "PORT" {
		t.Errorf("varName = %q, want PORT (fallback)", cfg.VarName)
	}
}

func TestEnvConfigCustomVarName(t *testing.T) {
	cfg := EnvConfigFor("next", EnvOptions{VarName: "MY_PORT"})
	if cfg.VarName != "MY_PORT" {
		t.Errorf("varName = %q, want MY_PORT", cfg.VarName)
	}
}

func TestEnvConfigCustomOutput(t *testing.T) {
	cfg := EnvConfigFor("next", EnvOptions{Output: ".env.custom"})
	if cfg.File != ".env.custom" {
		t.Errorf("file = %q, want .env.custom", cfg.File)
	}
}

// ---- WriteEnvFile ----

func TestWriteEnvFileCreatesNew(t *testing.T) {
	dir := t.TempDir()
	cfg := EnvConfig{File: ".env.local", VarName: "PORT"}

	if err := WriteEnvFile(dir, 3001, cfg); err != nil {
		t.Fatalf("WriteEnvFile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".env.local"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "PORT=3001") {
		t.Errorf("expected PORT=3001 in file, got:\n%s", data)
	}
}

func TestWriteEnvFileUpdatesExisting(t *testing.T) {
	dir := t.TempDir()
	cfg := EnvConfig{File: ".env.local", VarName: "PORT"}

	// Write initial file with other content.
	initial := "# comment\nFOO=bar\nPORT=3000\nBAZ=qux\n"
	if err := os.WriteFile(filepath.Join(dir, ".env.local"), []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteEnvFile(dir, 3001, cfg); err != nil {
		t.Fatalf("WriteEnvFile: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".env.local"))
	content := string(data)

	if strings.Contains(content, "PORT=3000") {
		t.Error("old PORT=3000 should have been replaced")
	}
	if !strings.Contains(content, "PORT=3001") {
		t.Error("expected PORT=3001 in updated file")
	}
	if !strings.Contains(content, "FOO=bar") {
		t.Error("FOO=bar should be preserved")
	}
	if !strings.Contains(content, "BAZ=qux") {
		t.Error("BAZ=qux should be preserved")
	}
	if !strings.Contains(content, "# comment") {
		t.Error("comment should be preserved")
	}
}

func TestWriteEnvFileAppendsIfVarMissing(t *testing.T) {
	dir := t.TempDir()
	cfg := EnvConfig{File: ".env.local", VarName: "PORT"}

	existing := "FOO=bar\n"
	if err := os.WriteFile(filepath.Join(dir, ".env.local"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteEnvFile(dir, 3002, cfg); err != nil {
		t.Fatalf("WriteEnvFile: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".env.local"))
	content := string(data)
	if !strings.Contains(content, "PORT=3002") {
		t.Errorf("expected PORT=3002 appended, got:\n%s", content)
	}
	if !strings.Contains(content, "FOO=bar") {
		t.Error("FOO=bar should be preserved")
	}
}

func TestWriteEnvFileCreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	cfg := EnvConfig{File: "subdir/.env.local", VarName: "PORT"}

	if err := WriteEnvFile(dir, 3003, cfg); err != nil {
		t.Fatalf("WriteEnvFile with subdir: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "subdir", ".env.local")); os.IsNotExist(err) {
		t.Error("expected file in subdir to be created")
	}
}

func TestWriteEnvFilePreservesBlankLines(t *testing.T) {
	dir := t.TempDir()
	cfg := EnvConfig{File: ".env", VarName: "PORT"}

	initial := "A=1\n\nB=2\nPORT=3000\n\nC=3\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteEnvFile(dir, 4000, cfg); err != nil {
		t.Fatalf("WriteEnvFile: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".env"))
	content := string(data)
	if !strings.Contains(content, "PORT=4000") {
		t.Errorf("expected PORT=4000, got:\n%s", content)
	}
	// Blank lines should be preserved.
	if strings.Count(content, "\n\n") < 1 {
		t.Error("expected blank lines to be preserved")
	}
}

func TestWriteEnvFileAppendsNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	cfg := EnvConfig{File: ".env", VarName: "PORT"}

	// File without trailing newline.
	existing := "FOO=bar"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteEnvFile(dir, 9999, cfg); err != nil {
		t.Fatalf("WriteEnvFile: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".env"))
	content := string(data)
	if !strings.Contains(content, "PORT=9999") {
		t.Errorf("expected PORT=9999 appended, got:\n%s", content)
	}
	if !strings.Contains(content, "FOO=bar") {
		t.Error("FOO=bar should be preserved")
	}
}

func TestWriteEnvFileMkdirAllError(t *testing.T) {
	orig := osMkdirAll
	osMkdirAll = func(path string, perm os.FileMode) error {
		return os.ErrPermission
	}
	t.Cleanup(func() { osMkdirAll = orig })

	cfg := EnvConfig{File: ".env", VarName: "PORT"}
	err := WriteEnvFile(t.TempDir(), 3000, cfg)
	if err == nil {
		t.Fatal("expected error from MkdirAll, got nil")
	}
}

func TestWriteEnvFileReadError(t *testing.T) {
	dir := t.TempDir()
	// Create the file first so it exists (not a NotExist error).
	_ = os.WriteFile(filepath.Join(dir, ".env"), []byte("PORT=3000\n"), 0644)

	orig := osReadFile
	osReadFile = func(path string) ([]byte, error) {
		return nil, os.ErrPermission
	}
	t.Cleanup(func() { osReadFile = orig })

	cfg := EnvConfig{File: ".env", VarName: "PORT"}
	err := WriteEnvFile(dir, 4000, cfg)
	if err == nil {
		t.Fatal("expected error from ReadFile, got nil")
	}
}

func TestWriteEnvFileWriteNewError(t *testing.T) {
	dir := t.TempDir()

	orig := osWriteFile
	origRead := osReadFile
	// ReadFile returns NotExist so we take the "create new" path.
	osReadFile = func(path string) ([]byte, error) {
		return nil, os.ErrNotExist
	}
	osWriteFile = func(path string, data []byte, perm os.FileMode) error {
		return os.ErrPermission
	}
	t.Cleanup(func() {
		osWriteFile = orig
		osReadFile = origRead
	})

	cfg := EnvConfig{File: ".env", VarName: "PORT"}
	err := WriteEnvFile(dir, 3000, cfg)
	if err == nil {
		t.Fatal("expected error from WriteFile (new), got nil")
	}
}

func TestWriteEnvFileWriteUpdateError(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".env"), []byte("PORT=3000\n"), 0644)

	orig := osWriteFile
	osWriteFile = func(path string, data []byte, perm os.FileMode) error {
		return os.ErrPermission
	}
	t.Cleanup(func() { osWriteFile = orig })

	cfg := EnvConfig{File: ".env", VarName: "PORT"}
	err := WriteEnvFile(dir, 4000, cfg)
	if err == nil {
		t.Fatal("expected error from WriteFile (update), got nil")
	}
}
