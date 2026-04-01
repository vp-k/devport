package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/user01/devport/internal/registry"
)

type smokeScenario struct {
	name              string
	files             map[string]string
	expectedFramework string
	expectedEnvFile   string
	expectedVarName   string
	portMin           int
	portMax           int
	checkList         bool
	checkExec         bool
}

func TestReadmeQuickStartNextFlow(t *testing.T) {
	runSmokeScenario(t, smokeScenario{
		name: "next",
		files: map[string]string{
			"package.json":   `{"name":"readme-next-app","dependencies":{"next":"14.2.0"}}`,
			"next.config.js": "module.exports = {};\n",
		},
		expectedFramework: "next",
		expectedEnvFile:   ".env.local",
		expectedVarName:   "PORT",
		portMin:           3000,
		portMax:           3999,
		checkList:         true,
		checkExec:         true,
	})
}

func TestReadmeViteFlow(t *testing.T) {
	runSmokeScenario(t, smokeScenario{
		name: "vite",
		files: map[string]string{
			"package.json":   `{"name":"readme-vite-app","devDependencies":{"vite":"5.0.0"}}`,
			"vite.config.ts": "export default {}\n",
		},
		expectedFramework: "vite",
		expectedEnvFile:   ".env.local",
		expectedVarName:   "VITE_PORT",
		portMin:           5000,
		portMax:           5999,
	})
}

func TestReadmeUnknownFrameworkFallbackFlow(t *testing.T) {
	runSmokeScenario(t, smokeScenario{
		name: "unknown",
		files: map[string]string{
			"package.json": `{"name":"readme-plain-app","dependencies":{"lodash":"4.17.21"}}`,
		},
		expectedFramework: "",
		expectedEnvFile:   ".env.local",
		expectedVarName:   "PORT",
		portMin:           3000,
		portMax:           9999,
	})
}

func TestReadmeGoFlow(t *testing.T) {
	runSmokeScenario(t, smokeScenario{
		name: "go",
		files: map[string]string{
			"go.mod": "module example.com/readme-go\n\ngo 1.25.5\n",
		},
		expectedFramework: "go",
		expectedEnvFile:   ".env",
		expectedVarName:   "PORT",
		portMin:           8000,
		portMax:           8999,
	})
}

func TestReadmeAdminFlow(t *testing.T) {
	cleanupGetFlags(t)
	cleanupDoctorFlags(t)
	cleanupExportFlags(t)
	cleanupImportFlags(t)
	cleanupListFlags(t)

	homeDir := newTestHome(t)
	dir := t.TempDir()
	writeSmokeFiles(t, dir, map[string]string{
		"package.json":   `{"name":"readme-admin-app","dependencies":{"next":"14.2.0"}}`,
		"next.config.js": "module.exports = {};\n",
	})

	origWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWd)

	if _, err := runCmd(t, getCmd); err != nil {
		t.Fatalf("get: %v", err)
	}

	reg, err := cmdRegistryLoad(homeDir)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	current := reg.Entries["readme-admin-app"]
	if current == nil {
		t.Fatal("expected readme-admin-app entry to exist")
	}

	reg.Entries["duplicate-admin-app"] = &registry.Entry{
		Port:           current.Port,
		KeySource:      registry.KeySourcePackageJSON,
		DisplayName:    "duplicate-admin-app",
		ProjectPath:    filepath.Join(dir, "missing-project"),
		Framework:      "next",
		AllocatedAt:    time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		LastAccessedAt: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := registry.Save(homeDir, reg); err != nil {
		t.Fatalf("save registry with duplicate: %v", err)
	}

	doctorOut, err := runCmd(t, doctorCmd, "--fix")
	if err != nil {
		t.Fatalf("doctor --fix: %v", err)
	}
	if !strings.Contains(doctorOut, "removed 1 older duplicate entries") {
		t.Fatalf("expected duplicate cleanup, got: %q", doctorOut)
	}
	if strings.Contains(doctorOut, "[FIXED] lock file") || strings.Contains(doctorOut, "[WARN] lock file") {
		t.Fatalf("expected healthy lock file handling, got: %q", doctorOut)
	}

	regAfter, err := cmdRegistryLoad(homeDir)
	if err != nil {
		t.Fatalf("load registry after doctor: %v", err)
	}
	if _, ok := regAfter.Entries["duplicate-admin-app"]; ok {
		t.Fatal("expected duplicate-admin-app to be removed")
	}

	backup := filepath.Join(t.TempDir(), "backup.json")
	if _, err := runCmd(t, exportCmd, "--output", backup); err != nil {
		t.Fatalf("export: %v", err)
	}
	backupData, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	var exported []exportEntry
	if err := json.Unmarshal(backupData, &exported); err != nil {
		t.Fatalf("parse exported JSON: %v", err)
	}
	if len(exported) != 1 {
		t.Fatalf("expected one exported entry, got %d", len(exported))
	}

	// Use a port well outside normal framework ranges (3000–9999) to avoid
	// colliding with whatever port getCmd allocated for the admin app above.
	importFile := filepath.Join(t.TempDir(), "import.json")
	writeSmokeFiles(t, filepath.Dir(importFile), map[string]string{
		filepath.Base(importFile): `[{"key":"imported-admin-app","port":49001,"displayName":"imported-admin-app","framework":"next","projectPath":"C:\\work\\imported-admin-app","allocatedAt":"2026-01-01T00:00:00Z"}]`,
	})

	dryRunOut, err := runCmd(t, importCmd, "--dry-run", importFile)
	if err != nil {
		t.Fatalf("import --dry-run: %v", err)
	}
	if !strings.Contains(dryRunOut, "Dry-run: 1 add") {
		t.Fatalf("expected dry-run add preview, got: %q", dryRunOut)
	}

	importFlagDryRun = false

	importOut, err := runCmd(t, importCmd, importFile)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(importOut, "Imported: 1 add") {
		t.Fatalf("expected import summary, got: %q", importOut)
	}

	listOut, err := runCmd(t, listCmd, "--json")
	if err != nil {
		t.Fatalf("list --json: %v", err)
	}
	var entries []struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(listOut)), &entries); err != nil {
		t.Fatalf("parse list JSON: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected two entries after import, got %d", len(entries))
	}
}

func runSmokeScenario(t *testing.T, scenario smokeScenario) {
	t.Helper()

	cleanupGetFlags(t)
	cleanupEnvFlags(t)
	cleanupStatusFlags(t)
	cleanupListFlags(t)
	cleanupExecFlags(t)

	homeDir := newTestHome(t)
	dir := t.TempDir()
	writeSmokeFiles(t, dir, scenario.files)

	origWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWd)

	getOut, err := runCmd(t, getCmd)
	if err != nil {
		t.Fatalf("%s get: %v", scenario.name, err)
	}

	port, err := strconv.Atoi(strings.TrimSpace(getOut))
	if err != nil {
		t.Fatalf("%s parse port %q: %v", scenario.name, getOut, err)
	}
	if port < scenario.portMin || port > scenario.portMax {
		t.Fatalf("%s expected port in [%d,%d], got %d", scenario.name, scenario.portMin, scenario.portMax, port)
	}

	envOut, err := runCmd(t, envCmd)
	if err != nil {
		t.Fatalf("%s env: %v", scenario.name, err)
	}
	if !strings.Contains(envOut, scenario.expectedEnvFile) || !strings.Contains(envOut, scenario.expectedVarName+"=") {
		t.Fatalf("%s unexpected env output: %q", scenario.name, envOut)
	}

	envData, err := os.ReadFile(filepath.Join(dir, scenario.expectedEnvFile))
	if err != nil {
		t.Fatalf("%s read env file: %v", scenario.name, err)
	}
	expectedAssignment := scenario.expectedVarName + "=" + strconv.Itoa(port)
	if !strings.Contains(string(envData), expectedAssignment) {
		t.Fatalf("%s expected %s in env file, got %q", scenario.name, expectedAssignment, string(envData))
	}

	statusOut, err := runCmd(t, statusCmd, "--json")
	if err != nil {
		t.Fatalf("%s status --json: %v", scenario.name, err)
	}
	var status map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(statusOut)), &status); err != nil {
		t.Fatalf("%s parse status JSON: %v", scenario.name, err)
	}
	if status["status"] != "ALLOCATED" {
		t.Fatalf("%s expected ALLOCATED status, got %v", scenario.name, status["status"])
	}
	if status["framework"] != scenario.expectedFramework {
		t.Fatalf("%s expected framework %q, got %v", scenario.name, scenario.expectedFramework, status["framework"])
	}
	if status["envFile"] != scenario.expectedEnvFile {
		t.Fatalf("%s expected envFile %q, got %v", scenario.name, scenario.expectedEnvFile, status["envFile"])
	}

	if scenario.checkList {
		listOut, err := runCmd(t, listCmd, "--json")
		if err != nil {
			t.Fatalf("%s list --json: %v", scenario.name, err)
		}
		var entries []map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(listOut)), &entries); err != nil {
			t.Fatalf("%s parse list JSON: %v", scenario.name, err)
		}
		if len(entries) != 1 {
			t.Fatalf("%s expected one list entry, got %d", scenario.name, len(entries))
		}
		if entries[0]["framework"] != scenario.expectedFramework {
			t.Fatalf("%s expected list framework %q, got %v", scenario.name, scenario.expectedFramework, entries[0]["framework"])
		}
		if allocatedAt, ok := entries[0]["allocatedAt"].(string); !ok {
			t.Fatalf("%s expected allocatedAt string in list JSON", scenario.name)
		} else if _, err := parseRegistryTime(allocatedAt); err != nil {
			t.Fatalf("%s expected RFC3339 allocatedAt, got %q: %v", scenario.name, allocatedAt, err)
		}
	}

	if scenario.checkExec {
		var capturedEnv []string
		origStartProcess := cmdStartProcess
		cmdStartProcess = func(_ string, _ []string, env []string, _ <-chan os.Signal) (int, error) {
			capturedEnv = append([]string(nil), env...)
			return 0, nil
		}
		t.Cleanup(func() { cmdStartProcess = origStartProcess })

		if _, err := runCmd(t, execCmd, "--", "echo", "hello"); err != nil {
			t.Fatalf("%s exec: %v", scenario.name, err)
		}
		if !containsString(capturedEnv, "PORT="+strconv.Itoa(port)) {
			t.Fatalf("%s expected exec environment to include PORT=%d, got %v", scenario.name, port, capturedEnv)
		}
	}

	reg, err := cmdRegistryLoad(homeDir)
	if err != nil {
		t.Fatalf("%s load registry: %v", scenario.name, err)
	}
	if len(reg.Entries) != 1 {
		t.Fatalf("%s expected one registry entry, got %d", scenario.name, len(reg.Entries))
	}
}

func writeSmokeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
