package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vp-k/devport/internal/allocator"
	"github.com/vp-k/devport/internal/registry"
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
	injectStartProcessCapture(t, code, err, nil)
}

type startedProcess struct {
	name string
	args []string
	env  []string
}

func injectStartProcessCapture(t *testing.T, code int, err error, capture *startedProcess) {
	t.Helper()
	orig := cmdStartProcess
	cmdStartProcess = func(name string, args []string, env []string, _ <-chan os.Signal) (int, error) {
		if capture != nil {
			capture.name = name
			capture.args = append([]string(nil), args...)
			capture.env = append([]string(nil), env...)
		}
		return code, err
	}
	t.Cleanup(func() { cmdStartProcess = orig })
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, value := range env {
		if strings.HasPrefix(value, prefix) {
			return strings.TrimPrefix(value, prefix)
		}
	}
	return ""
}

func seedRegistryEntry(t *testing.T, homeDir, key, displayName, projectPath, framework string, port int) {
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
				DisplayName:    displayName,
				ProjectPath:    projectPath,
				Framework:      framework,
				AllocatedAt:    time.Now().UTC(),
				LastAccessedAt: time.Now().UTC(),
			},
		},
		Reserved:    []int{},
		RangePolicy: registry.DefaultRangePolicy(),
	}

	if err := registry.Save(homeDir, reg); err != nil {
		t.Fatalf("seed registry: %v", err)
	}
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

func TestExecInjectsFrameworkEnvVarAndPortFlagForVite(t *testing.T) {
	cleanupExecFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "exec-vite-app")
	if err := os.WriteFile(filepath.Join(dir, "vite.config.ts"), []byte("export default {}\n"), 0644); err != nil {
		t.Fatalf("write vite config: %v", err)
	}

	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	var started startedProcess
	injectStartProcessCapture(t, 0, nil, &started)

	_, err := runCmd(t, execCmd, "--", "npm", "run", "dev")
	if err != nil {
		t.Fatalf("exec vite: %v", err)
	}

	port := envValue(started.env, "PORT")
	if port == "" {
		t.Fatal("expected PORT to be injected")
	}
	if got := envValue(started.env, "VITE_PORT"); got != port {
		t.Fatalf("expected VITE_PORT=%s, got %q", port, got)
	}
	if started.name != "npm" {
		t.Fatalf("expected process name npm, got %q", started.name)
	}
	wantArgs := []string{"run", "dev", "--", "--port", port}
	if !reflect.DeepEqual(started.args, wantArgs) {
		t.Fatalf("expected args %v, got %v", wantArgs, started.args)
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("parse port %q: %v", port, err)
	}
	if portNum < 5000 || portNum > 5999 {
		t.Fatalf("expected vite port in [5000,5999], got %d", portNum)
	}
}

func TestExecUsesStoredFrameworkForPortInjection(t *testing.T) {
	cleanupExecFlags(t)
	homeDir := newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "exec-stored-vite-app")

	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	seedRegistryEntry(t, homeDir, "exec-stored-vite-app", "exec-stored-vite-app", dir, "vite", 5500)

	var started startedProcess
	injectStartProcessCapture(t, 0, nil, &started)

	_, err := runCmd(t, execCmd, "--", "npm", "run", "dev")
	if err != nil {
		t.Fatalf("exec with stored framework: %v", err)
	}

	if got := envValue(started.env, "PORT"); got != "5500" {
		t.Fatalf("expected PORT=5500, got %q", got)
	}
	if got := envValue(started.env, "VITE_PORT"); got != "5500" {
		t.Fatalf("expected VITE_PORT=5500, got %q", got)
	}
	wantArgs := []string{"run", "dev", "--", "--port", "5500"}
	if !reflect.DeepEqual(started.args, wantArgs) {
		t.Fatalf("expected args %v, got %v", wantArgs, started.args)
	}
}

func TestInjectPortFlagDirect(t *testing.T) {
	got := injectPortFlag([]string{"vite"}, "vite", 5000)
	want := []string{"vite", "--port", "5000"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestInjectPortFlagViaNpmRun(t *testing.T) {
	got := injectPortFlag([]string{"npm", "run", "dev"}, "vite", 5000)
	want := []string{"npm", "run", "dev", "--", "--port", "5000"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestInjectPortFlagAppendsAfterExistingDoubleDash(t *testing.T) {
	got := injectPortFlag([]string{"npm", "run", "dev", "--", "--host"}, "vite", 5000)
	want := []string{"npm", "run", "dev", "--", "--host", "--port", "5000"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestInjectPortFlagSkipsUnknownFramework(t *testing.T) {
	input := []string{"npm", "run", "dev"}
	got := injectPortFlag(input, "", 5000)
	if !reflect.DeepEqual(got, input) {
		t.Fatalf("expected args to remain unchanged, got %v", got)
	}
}

func TestInjectPortFlagSkipsExistingPortFlag(t *testing.T) {
	input := []string{"npm", "run", "dev", "--", "--port", "5100"}
	got := injectPortFlag(input, "vite", 5000)
	if !reflect.DeepEqual(got, input) {
		t.Fatalf("expected args to remain unchanged, got %v", got)
	}
}

func TestBuildExecEnvAddsFrameworkVariable(t *testing.T) {
	env := buildExecEnv("vite", 5000)
	if got := envValue(env, "PORT"); got != "5000" {
		t.Fatalf("expected PORT=5000, got %q", got)
	}
	if got := envValue(env, "VITE_PORT"); got != "5000" {
		t.Fatalf("expected VITE_PORT=5000, got %q", got)
	}
}

func TestExecPreservesStoredPortForExistingEntry(t *testing.T) {
	cleanupExecFlags(t)
	homeDir := newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "exec-existing-vite-port")
	if err := os.WriteFile(filepath.Join(dir, "vite.config.ts"), []byte("export default {}\n"), 0644); err != nil {
		t.Fatalf("write vite config: %v", err)
	}

	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	seedRegistryEntry(t, homeDir, "exec-existing-vite-port", "exec-existing-vite-port", dir, "vite", 5601)

	var started startedProcess
	injectStartProcessCapture(t, 0, nil, &started)

	_, err := runCmd(t, execCmd, "--", "vite")
	if err != nil {
		t.Fatalf("exec existing entry: %v", err)
	}

	if got := envValue(started.env, "VITE_PORT"); got != "5601" {
		t.Fatalf("expected VITE_PORT=5601, got %q", got)
	}
	wantArgs := []string{"--port", "5601"}
	if !reflect.DeepEqual(started.args, wantArgs) {
		t.Fatalf("expected args %v, got %v", wantArgs, started.args)
	}

	reg, err := cmdRegistryLoad(homeDir)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	entry := reg.Entries["exec-existing-vite-port"]
	if entry == nil {
		t.Fatal("expected registry entry to exist")
	}
	port, err := allocator.Allocate("exec-existing-vite-port", "vite", reg, allocator.Options{})
	if err != nil {
		t.Fatalf("allocate existing entry: %v", err)
	}
	if port != entry.Port {
		t.Fatalf("expected allocator to preserve port %d, got %d", entry.Port, port)
	}
}
