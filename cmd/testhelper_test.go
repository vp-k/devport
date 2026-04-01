package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/vp-k/devport/internal/registry"
)

// newTestHome creates a temp directory and sets HOME / USERPROFILE to it.
func newTestHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

// newTestProject creates a temp directory simulating a project.
func newTestProject(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// newTestProjectWithPackageJSON creates a project dir with a package.json.
func newTestProjectWithPackageJSON(t *testing.T, name string) string {
	t.Helper()
	dir := newTestProject(t)
	pkg := map[string]any{"name": name}
	data, _ := json.Marshal(pkg)
	if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// runCmd executes a cobra subcommand via the root, returning captured stdout
// and the execution error. Extra args are appended after the subcommand name.
func runCmd(t *testing.T, sub *cobra.Command, args ...string) (string, error) {
	t.Helper()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)

	cmdArgs := append([]string{sub.Name()}, args...)
	rootCmd.SetArgs(cmdArgs)

	err := rootCmd.Execute()
	return buf.String(), err
}

// seedRegistry writes an entry directly to the registry in homeDir.
func seedRegistry(t *testing.T, homeDir, key string, port int) {
	t.Helper()
	reg := &registry.Registry{
		Version: 1,
		Meta: registry.Meta{
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		Entries: map[string]*registry.Entry{
			key: {
				Port:           port,
				KeySource:      registry.KeySourcePackageJSON,
				DisplayName:    key,
				ProjectPath:    "/tmp/" + key,
				Framework:      "next",
				AllocatedAt:    time.Now().UTC(),
				LastAccessedAt: time.Now().UTC(),
			},
		},
		Reserved:    []int{},
		RangePolicy: registry.DefaultRangePolicy(),
	}
	if err := registry.Save(homeDir, reg); err != nil {
		t.Fatal(err)
	}
}

// autoConfirm injects a confirmFn that always returns the given value.
func autoConfirm(t *testing.T, answer bool) {
	t.Helper()
	orig := confirmFn
	confirmFn = func(_ string) bool { return answer }
	t.Cleanup(func() { confirmFn = orig })
}
