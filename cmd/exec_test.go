package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/user01/devport/internal/registry"
)

func cleanupExecFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		execFlagAutoFree = false
	})
}

// injectStartProcess replaces cmdStartProcess for the test duration.
func injectStartProcess(t *testing.T, code int, err error) {
	t.Helper()
	orig := cmdStartProcess
	cmdStartProcess = func(_ string, _ []string, _ []string, _ <-chan os.Signal) (int, error) {
		return code, err
	}
	t.Cleanup(func() { cmdStartProcess = orig })
}

func TestExecBasic(t *testing.T) {
	cleanupExecFlags(t)
	homeDir := newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "exec-basic-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectStartProcess(t, 0, nil)

	_, err := runCmd(t, execCmd, "--", "echo", "hello")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}

	// Verify the port was allocated and persisted.
	reg, _ := cmdRegistryLoad(homeDir)
	found := false
	for _, e := range reg.Entries {
		if e.Port > 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected port to be allocated after exec")
	}
}

func TestExecAutoFree(t *testing.T) {
	cleanupExecFlags(t)
	homeDir := newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "exec-autofree-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectStartProcess(t, 0, nil)

	_, err := runCmd(t, execCmd, "--auto-free", "--", "echo")
	if err != nil {
		t.Fatalf("exec --auto-free: %v", err)
	}

	reg, _ := cmdRegistryLoad(homeDir)
	if len(reg.Entries) != 0 {
		t.Errorf("expected registry empty after --auto-free, got %d entries", len(reg.Entries))
	}
}

func TestExecNonZeroExit(t *testing.T) {
	cleanupExecFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "exec-exit-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectStartProcess(t, 42, nil)
	exitCode := injectOsExit(t)

	_, err := runCmd(t, execCmd, "--", "exit", "42")
	if err != nil {
		t.Fatalf("exec non-zero: %v", err)
	}
	if *exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", *exitCode)
	}
}

func TestExecStartProcessError(t *testing.T) {
	cleanupExecFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "exec-proc-err-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectStartProcess(t, 0, errFake)

	_, err := runCmd(t, execCmd, "--", "bad-cmd")
	if err == nil {
		t.Fatal("expected error from startProcess, got nil")
	}
	if !strings.Contains(err.Error(), "exec") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecAutoFreeTransactionError(t *testing.T) {
	cleanupExecFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "exec-autofree-err-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectStartProcess(t, 0, nil)

	// First transaction (allocate) must succeed; second (auto-free) fails.
	callCount := 0
	orig := cmdTransaction
	cmdTransaction = func(home string, fn func(*registry.Registry) error) error {
		callCount++
		if callCount == 2 {
			return errFake
		}
		return orig(home, fn)
	}
	t.Cleanup(func() { cmdTransaction = orig })

	out, err := runCmd(t, execCmd, "--auto-free", "--", "echo")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if !strings.Contains(out, "Warning") {
		t.Errorf("expected Warning for auto-free failure, got: %q", out)
	}
}

func TestExecGetwdError(t *testing.T) {
	cleanupExecFlags(t)
	newTestHome(t)
	injectGetwd(t)

	_, err := runCmd(t, execCmd, "--", "echo")
	if err == nil {
		t.Fatal("expected getwd error")
	}
}

func TestExecHomeDirError(t *testing.T) {
	cleanupExecFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectHomeDir(t)

	_, err := runCmd(t, execCmd, "--", "echo")
	if err == nil {
		t.Fatal("expected homedir error")
	}
}

func TestExecResolveError(t *testing.T) {
	cleanupExecFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectResolve(t)

	_, err := runCmd(t, execCmd, "--", "echo")
	if err == nil {
		t.Fatal("expected resolve error")
	}
}

func TestExecTransactionError(t *testing.T) {
	cleanupExecFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "exec-tx-err-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectTransaction(t)

	_, err := runCmd(t, execCmd, "--", "echo")
	if err == nil {
		t.Fatal("expected transaction error")
	}
}

func TestExecAllocateError(t *testing.T) {
	cleanupExecFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "exec-alloc-err-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectAllocate(t)

	_, err := runCmd(t, execCmd, "--", "echo")
	if err == nil {
		t.Fatal("expected allocate error")
	}
}

func TestExecExistingEntry(t *testing.T) {
	cleanupExecFlags(t)
	homeDir := newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "exec-existing-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectStartProcess(t, 0, nil)

	// First exec: creates entry.
	_, err := runCmd(t, execCmd, "--", "echo")
	if err != nil {
		t.Fatalf("first exec: %v", err)
	}
	reg1, _ := cmdRegistryLoad(homeDir)
	var port1 int
	for _, e := range reg1.Entries {
		port1 = e.Port
	}

	// Second exec: updates LastAccessedAt on existing entry (else branch).
	_, err = runCmd(t, execCmd, "--", "echo")
	if err != nil {
		t.Fatalf("second exec: %v", err)
	}
	reg2, _ := cmdRegistryLoad(homeDir)
	var port2 int
	for _, e := range reg2.Entries {
		port2 = e.Port
	}

	if port1 != port2 {
		t.Errorf("expected same port on second exec, got %d then %d", port1, port2)
	}
}
