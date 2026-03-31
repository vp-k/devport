package cmd

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

func cleanupGetFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		getFlagJSON = false
		getFlagRangeMin = 0
		getFlagRangeMax = 0
		getFlagFramework = ""
	})
}

func TestGetAllocatesPort(t *testing.T) {
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "test-get-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, getCmd)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	port, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		t.Fatalf("output %q is not an integer: %v", out, err)
	}
	if port < 3000 || port > 9999 {
		t.Errorf("port %d out of expected range", port)
	}
}

func TestGetReturnsSamePortOnSecondCall(t *testing.T) {
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "test-fixed-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out1, err := runCmd(t, getCmd)
	if err != nil {
		t.Fatalf("first get: %v", err)
	}

	out2, err := runCmd(t, getCmd)
	if err != nil {
		t.Fatalf("second get: %v", err)
	}

	if strings.TrimSpace(out1) != strings.TrimSpace(out2) {
		t.Errorf("port changed: first=%s second=%s", out1, out2)
	}
}

func TestGetJSONOutput(t *testing.T) {
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "json-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	getFlagJSON = true

	out, err := runCmd(t, getCmd, "--json")
	if err != nil {
		t.Fatalf("get --json: %v", err)
	}

	var result getJSONOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("parse JSON output %q: %v", out, err)
	}
	if result.Port == 0 {
		t.Error("expected non-zero port in JSON output")
	}
	if result.Key == "" {
		t.Error("expected non-empty key in JSON output")
	}
	if !result.New {
		t.Error("expected new=true on first allocation")
	}
}

func TestGetJSONSecondCallNewFalse(t *testing.T) {
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "json-app2")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// First call.
	runCmd(t, getCmd, "--json")

	// Second call.
	out, err := runCmd(t, getCmd, "--json")
	if err != nil {
		t.Fatalf("second get --json: %v", err)
	}

	var result getJSONOutput
	json.Unmarshal([]byte(strings.TrimSpace(out)), &result)
	if result.New {
		t.Error("expected new=false on second call")
	}
}

func TestGetCustomRange(t *testing.T) {
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "range-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, getCmd, "--range-min", "7100", "--range-max", "7200")
	if err != nil {
		t.Fatalf("get --range-min 7100 --range-max 7200: %v", err)
	}

	port, _ := strconv.Atoi(strings.TrimSpace(out))
	if port < 7100 || port > 7200 {
		t.Errorf("port %d outside custom range [7100,7200]", port)
	}
}

func TestGetFrameworkOverride(t *testing.T) {
	cleanupGetFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "fw-override-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, getCmd, "--framework", "vite")
	if err != nil {
		t.Fatalf("get --framework vite: %v", err)
	}

	port, _ := strconv.Atoi(strings.TrimSpace(out))
	if port < 5000 || port > 5999 {
		t.Errorf("port %d outside vite range [5000,5999]", port)
	}
}
