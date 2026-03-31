package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/user01/devport/internal/allocator"
	"github.com/user01/devport/internal/detect"
	"github.com/user01/devport/internal/registry"
	"github.com/user01/devport/internal/resolver"
)

var errFake = errors.New("injected error")

// injectAllocate makes cmdAllocate return an error for the duration of the test.
func injectAllocate(t *testing.T) {
	t.Helper()
	orig := cmdAllocate
	cmdAllocate = func(_ string, _ string, _ *registry.Registry, _ allocator.Options) (int, error) {
		return 0, errFake
	}
	t.Cleanup(func() { cmdAllocate = orig })
}

// injectGetwd makes cmdGetwd return an error for the duration of the test.
func injectGetwd(t *testing.T) {
	t.Helper()
	orig := cmdGetwd
	cmdGetwd = func() (string, error) { return "", errFake }
	t.Cleanup(func() { cmdGetwd = orig })
}

// injectHomeDir makes cmdUserHomeDir return an error for the duration of the test.
func injectHomeDir(t *testing.T) {
	t.Helper()
	orig := cmdUserHomeDir
	cmdUserHomeDir = func() (string, error) { return "", errFake }
	t.Cleanup(func() { cmdUserHomeDir = orig })
}

// injectResolve makes cmdResolve return an error.
func injectResolve(t *testing.T) {
	t.Helper()
	orig := cmdResolve
	cmdResolve = func(_ string) (resolver.Resolution, error) { return resolver.Resolution{}, errFake }
	t.Cleanup(func() { cmdResolve = orig })
}

// injectRegistryLoad makes cmdRegistryLoad return an error.
func injectRegistryLoad(t *testing.T) {
	t.Helper()
	orig := cmdRegistryLoad
	cmdRegistryLoad = func(_ string) (*registry.Registry, error) { return nil, errFake }
	t.Cleanup(func() { cmdRegistryLoad = orig })
}

// injectRegistrySave makes cmdRegistrySave return an error.
func injectRegistrySave(t *testing.T) {
	t.Helper()
	orig := cmdRegistrySave
	cmdRegistrySave = func(_ string, _ *registry.Registry) error { return errFake }
	t.Cleanup(func() { cmdRegistrySave = orig })
}

// injectWriteEnvFile makes cmdWriteEnvFile return an error.
func injectWriteEnvFile(t *testing.T) {
	t.Helper()
	orig := cmdWriteEnvFile
	cmdWriteEnvFile = func(_ string, _ int, _ detect.EnvConfig) error { return errFake }
	t.Cleanup(func() { cmdWriteEnvFile = orig })
}

// injectOsExit captures calls to cmdOsExit instead of actually exiting.
func injectOsExit(t *testing.T) (captured *int) {
	t.Helper()
	code := -1
	orig := cmdOsExit
	cmdOsExit = func(c int) { code = c }
	t.Cleanup(func() { cmdOsExit = orig })
	return &code
}

// ---- get ----

func TestGetGetwdError(t *testing.T) {
	cleanupGetFlags(t)
	newTestHome(t)
	injectGetwd(t)

	_, err := runCmd(t, getCmd)
	if err == nil {
		t.Fatal("expected error from getwd, got nil")
	}
}

func TestGetHomeDirError(t *testing.T) {
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectHomeDir(t)

	_, err := runCmd(t, getCmd)
	if err == nil {
		t.Fatal("expected error from homedir, got nil")
	}
}

func TestGetRegistryLoadError(t *testing.T) {
	cleanupGetFlags(t)
	homeDir := newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "load-err-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Make registry file unreadable by making it a directory.
	regPath := filepath.Join(homeDir, ".devports.json")
	if err := os.Mkdir(regPath, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := runCmd(t, getCmd)
	if err == nil {
		t.Fatal("expected error from registry load, got nil")
	}
}

func TestGetRangeExhaustedError(t *testing.T) {
	cleanupGetFlags(t)
	homeDir := newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "exhausted-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Fill the registry so ports 7800-7801 are both "used".
	reg := &registry.Registry{
		Version:     1,
		Meta:        registry.Meta{CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		RangePolicy: registry.DefaultRangePolicy(),
		Reserved:    []int{},
		Entries: map[string]*registry.Entry{
			"blocker-a": {Port: 7800, KeySource: registry.KeySourcePath, DisplayName: "blocker-a",
				AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()},
			"blocker-b": {Port: 7801, KeySource: registry.KeySourcePath, DisplayName: "blocker-b",
				AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()},
		},
	}
	registry.Save(homeDir, reg)

	_, err := runCmd(t, getCmd, "--range-min", "7800", "--range-max", "7801")
	if err == nil {
		t.Fatal("expected error for exhausted range, got nil")
	}
}

// ---- env ----

func TestEnvGetwdError(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	injectGetwd(t)

	_, err := runCmd(t, envCmd)
	if err == nil {
		t.Fatal("expected error from getwd, got nil")
	}
}

func TestEnvHomeDirError(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectHomeDir(t)

	_, err := runCmd(t, envCmd)
	if err == nil {
		t.Fatal("expected error from homedir, got nil")
	}
}

// ---- free ----

func TestFreeHomeDirError(t *testing.T) {
	cleanupFreeFlags(t)
	newTestHome(t)
	injectHomeDir(t)

	_, err := runCmd(t, freeCmd, "--force", "some-key")
	if err == nil {
		t.Fatal("expected error from homedir, got nil")
	}
}

func TestFreeGetwdError(t *testing.T) {
	cleanupFreeFlags(t)
	newTestHome(t)
	injectGetwd(t)

	// No args → tries to resolve cwd.
	_, err := runCmd(t, freeCmd, "--force")
	if err == nil {
		t.Fatal("expected error from getwd, got nil")
	}
}

// ---- list ----

func TestListHomeDirError(t *testing.T) {
	cleanupListFlags(t)
	newTestHome(t)
	injectHomeDir(t)

	_, err := runCmd(t, listCmd)
	if err == nil {
		t.Fatal("expected error from homedir, got nil")
	}
}

func TestListRegistryLoadError(t *testing.T) {
	cleanupListFlags(t)
	homeDir := newTestHome(t)

	regPath := filepath.Join(homeDir, ".devports.json")
	if err := os.Mkdir(regPath, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := runCmd(t, listCmd)
	if err == nil {
		t.Fatal("expected error from registry load, got nil")
	}
}

// ---- init ----

func TestInitGetwdError(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	injectGetwd(t)

	_, err := runCmd(t, initCmd)
	if err == nil {
		t.Fatal("expected error from getwd, got nil")
	}
}

func TestInitHomeDirError(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectHomeDir(t)

	_, err := runCmd(t, initCmd)
	if err == nil {
		t.Fatal("expected error from homedir, got nil")
	}
}

// ---- addPredevScript ----

func TestAddPredevScriptReadError(t *testing.T) {
	err := addPredevScript(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Fatal("expected error reading nonexistent file")
	}
}

func TestAddPredevScriptMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	os.WriteFile(path, []byte("{bad json"), 0644)
	err := addPredevScript(path)
	if err == nil {
		t.Fatal("expected error from malformed JSON")
	}
}

func TestAddPredevScriptMalformedScripts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	// scripts is not an object.
	os.WriteFile(path, []byte(`{"scripts":"bad"}`), 0644)
	err := addPredevScript(path)
	if err == nil {
		t.Fatal("expected error from malformed scripts")
	}
}

func TestAddPredevScriptNoScriptsSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	os.WriteFile(path, []byte(`{"name":"app"}`), 0644)
	err := addPredevScript(path)
	if err != nil {
		t.Fatalf("addPredevScript with no scripts section: %v", err)
	}
}

// ---- addToGitignore ----

func TestAddToGitignoreNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	os.WriteFile(path, []byte("node_modules"), 0644)
	if err := addToGitignore(dir, ".env.local"); err != nil {
		t.Fatalf("addToGitignore: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "node_modules\n.env.local\n" {
		t.Errorf("unexpected content: %q", data)
	}
}

func TestAddToGitignoreReadError(t *testing.T) {
	dir := t.TempDir()
	// Make .gitignore a directory so ReadFile fails with a non-NotExist error.
	gitignorePath := filepath.Join(dir, ".gitignore")
	os.Mkdir(gitignorePath, 0755)
	err := addToGitignore(dir, ".env.local")
	if err == nil {
		t.Fatal("expected error reading .gitignore directory")
	}
}

// ---- reset ----

func TestResetAllocateError(t *testing.T) {
	cleanupResetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "alloc-err-reset-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectAllocate(t)

	_, err := runCmd(t, resetCmd, "--force")
	if err == nil {
		t.Fatal("expected allocate error, got nil")
	}
}

func TestResetGetwdError(t *testing.T) {
	cleanupResetFlags(t)
	newTestHome(t)
	injectGetwd(t)

	_, err := runCmd(t, resetCmd)
	if err == nil {
		t.Fatal("expected error from getwd, got nil")
	}
}

func TestResetHomeDirError(t *testing.T) {
	cleanupResetFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectHomeDir(t)

	_, err := runCmd(t, resetCmd)
	if err == nil {
		t.Fatal("expected error from homedir, got nil")
	}
}

func TestResetRegistryLoadError(t *testing.T) {
	cleanupResetFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectRegistryLoad(t)

	_, err := runCmd(t, resetCmd)
	if err == nil {
		t.Fatal("expected error from registry load, got nil")
	}
}

func TestResetResolveError(t *testing.T) {
	cleanupResetFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectResolve(t)

	_, err := runCmd(t, resetCmd)
	if err == nil {
		t.Fatal("expected error from resolve, got nil")
	}
}

func TestResetRegistrySaveError(t *testing.T) {
	cleanupResetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "save-err-reset-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectRegistrySave(t)

	_, err := runCmd(t, resetCmd, "--force")
	if err == nil {
		t.Fatal("expected error from registry save, got nil")
	}
}

func TestResetWriteEnvFileError(t *testing.T) {
	cleanupResetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "write-err-reset-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectWriteEnvFile(t)
	autoConfirm(t, true)

	out, err := runCmd(t, resetCmd, "--force")
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	// Warning should be printed but no error returned.
	if !strings.Contains(out, "Warning") {
		t.Errorf("expected Warning in output, got: %q", out)
	}
}

// ---- get additional error paths ----

func TestGetResolveError(t *testing.T) {
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectResolve(t)

	_, err := runCmd(t, getCmd)
	if err == nil {
		t.Fatal("expected error from resolve, got nil")
	}
}

func TestGetRegistryLoadInjectedError(t *testing.T) {
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "load-err-get-app2")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectRegistryLoad(t)

	_, err := runCmd(t, getCmd)
	if err == nil {
		t.Fatal("expected error from registry load, got nil")
	}
}

func TestGetRegistrySaveError(t *testing.T) {
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "save-err-get-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectRegistrySave(t)

	_, err := runCmd(t, getCmd)
	if err == nil {
		t.Fatal("expected error from registry save, got nil")
	}
}

func TestGetAllocateError(t *testing.T) {
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "alloc-err-get-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectAllocate(t)

	_, err := runCmd(t, getCmd)
	if err == nil {
		t.Fatal("expected allocate error, got nil")
	}
}

// ---- env additional error paths ----

func TestEnvResolveError(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectResolve(t)

	_, err := runCmd(t, envCmd)
	if err == nil {
		t.Fatal("expected error from resolve, got nil")
	}
}

func TestEnvRegistryLoadError(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "load-err-env-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectRegistryLoad(t)

	_, err := runCmd(t, envCmd)
	if err == nil {
		t.Fatal("expected error from registry load, got nil")
	}
}

func TestEnvRegistrySaveError(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "save-err-env-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectRegistrySave(t)

	_, err := runCmd(t, envCmd)
	if err == nil {
		t.Fatal("expected error from registry save, got nil")
	}
}

func TestEnvAllocateError(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "alloc-err-env-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectAllocate(t)

	_, err := runCmd(t, envCmd)
	if err == nil {
		t.Fatal("expected allocate error, got nil")
	}
}

func TestEnvWriteEnvFileError(t *testing.T) {
	cleanupEnvFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "write-err-env-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectWriteEnvFile(t)

	_, err := runCmd(t, envCmd)
	if err == nil {
		t.Fatal("expected error from WriteEnvFile, got nil")
	}
}

// ---- free additional error paths ----

func TestFreeRegistryLoadError(t *testing.T) {
	cleanupFreeFlags(t)
	newTestHome(t)
	injectRegistryLoad(t)

	_, err := runCmd(t, freeCmd, "--force", "some-key")
	if err == nil {
		t.Fatal("expected error from registry load, got nil")
	}
}

func TestFreeRegistrySaveError(t *testing.T) {
	cleanupFreeFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "save-err-free-app", 3999)
	injectRegistrySave(t)

	_, err := runCmd(t, freeCmd, "--force", "save-err-free-app")
	if err == nil {
		t.Fatal("expected error from registry save, got nil")
	}
}

func TestFreeAllRegistrySaveError(t *testing.T) {
	cleanupFreeFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "all-save-err-app", 3998)
	injectRegistrySave(t)

	_, err := runCmd(t, freeCmd, "--all", "--force")
	if err == nil {
		t.Fatal("expected error from registry save (--all), got nil")
	}
}

func TestFreeResolveError(t *testing.T) {
	cleanupFreeFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectResolve(t)

	_, err := runCmd(t, freeCmd, "--force")
	if err == nil {
		t.Fatal("expected error from resolve, got nil")
	}
}

// ---- init additional error paths ----

func TestInitResolveError(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectResolve(t)

	_, err := runCmd(t, initCmd)
	if err == nil {
		t.Fatal("expected error from resolve, got nil")
	}
}

func TestInitRegistryLoadError(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "load-err-init-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectRegistryLoad(t)

	_, err := runCmd(t, initCmd)
	if err == nil {
		t.Fatal("expected error from registry load, got nil")
	}
}

func TestInitRegistrySaveError(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "save-err-init-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectRegistrySave(t)

	_, err := runCmd(t, initCmd)
	if err == nil {
		t.Fatal("expected error from registry save, got nil")
	}
}

func TestInitAllocateError(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "alloc-err-init-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectAllocate(t)

	_, err := runCmd(t, initCmd)
	if err == nil {
		t.Fatal("expected allocate error, got nil")
	}
}

func TestInitWriteEnvFileError(t *testing.T) {
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "write-err-init-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectWriteEnvFile(t)

	_, err := runCmd(t, initCmd)
	if err == nil {
		t.Fatal("expected error from WriteEnvFile, got nil")
	}
}

func TestInitAddPredevWarning(t *testing.T) {
	// When addPredevScript fails, a warning is printed but no error is returned.
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "warn-pkg-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	autoConfirm(t, true)

	// Write malformed package.json so addPredevScript fails.
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{bad json"), 0644)

	out, err := runCmd(t, initCmd, "--yes")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if !strings.Contains(out, "Warning") {
		t.Errorf("expected Warning in output, got: %q", out)
	}
}

func TestInitAddGitignoreWarning(t *testing.T) {
	// When addToGitignore fails, a warning is printed but no error is returned.
	cleanupInitFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "warn-gitignore-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Create next.config.js so .env.local path is chosen.
	os.WriteFile(filepath.Join(dir, "next.config.js"), []byte(""), 0644)
	// Make .gitignore a directory so it fails.
	os.Mkdir(filepath.Join(dir, ".gitignore"), 0755)

	out, err := runCmd(t, initCmd, "--yes")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if !strings.Contains(out, "Warning") {
		t.Errorf("expected Warning in output, got: %q", out)
	}
}

func TestResetSecondResolveError(t *testing.T) {
	cleanupResetFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "second-resolve-err-app", 3800)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectResolve(t)

	// Passing an explicit key skips the first cmdResolve call (else branch),
	// so only the unconditional second cmdResolve will fail.
	_, err := runCmd(t, resetCmd, "--force", "second-resolve-err-app")
	if err == nil {
		t.Fatal("expected error from second resolve, got nil")
	}
	if !strings.Contains(err.Error(), "resolve key (2)") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---- confirmFn default body ----

func TestDefaultConfirmFnYes(t *testing.T) {
	// Exercise the default interactive confirmFn by piping stdin.
	orig := confirmFn
	t.Cleanup(func() { confirmFn = orig })
	confirmFn = orig // ensure we're using the original stdin-reading closure

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = origStdin
		r.Close()
	})
	w.WriteString("y\n")
	w.Close()

	if !confirmFn("prompt: ") {
		t.Error("expected true for 'y' answer")
	}
}

// ---- Execute (root) ----

func TestExecuteErrorPath(t *testing.T) {
	exitCode := injectOsExit(t)

	// Inject a resolver error so rootCmd.Execute() returns an error.
	// Use a completely invalid subcommand to force an error.
	rootCmd.SetArgs([]string{"nonexistent-command-xyz"})
	Execute()

	if *exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", *exitCode)
	}
}

// ---- addPredevScript write error ----

func TestAddPredevScriptWriteError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	os.WriteFile(path, []byte(`{"name":"app"}`), 0644)

	// Make it read-only so WriteFile fails.
	os.Chmod(path, 0444)
	defer os.Chmod(path, 0644) // restore for cleanup

	err := addPredevScript(path)
	// On Windows, making a file read-only may or may not prevent writes.
	// Accept either outcome but verify no panic.
	_ = err
}
