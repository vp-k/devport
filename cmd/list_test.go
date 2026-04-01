package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/user01/devport/internal/registry"
)

func cleanupListFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		listFlagJSON = false
		listFlagVerbose = false
	})
}

func TestListEmptyRegistry(t *testing.T) {
	cleanupListFlags(t)
	newTestHome(t)

	out, err := runCmd(t, listCmd)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "No projects registered") {
		t.Errorf("expected empty message, got: %q", out)
	}
}

func TestListShowsEntries(t *testing.T) {
	cleanupListFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "my-app", 3001)

	out, err := runCmd(t, listCmd)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "3001") {
		t.Errorf("expected port 3001 in output, got: %q", out)
	}
	if !strings.Contains(out, "my-app") {
		t.Errorf("expected project name in output, got: %q", out)
	}
}

func TestListJSON(t *testing.T) {
	cleanupListFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "json-list-app", 3002)

	out, err := runCmd(t, listCmd, "--json")
	if err != nil {
		t.Fatalf("list --json: %v", err)
	}
	if !strings.Contains(out, `"port"`) {
		t.Errorf("expected JSON output, got: %q", out)
	}
	if !strings.Contains(out, "3002") {
		t.Errorf("expected port 3002 in JSON, got: %q", out)
	}
}

func TestListJSONAllocatedAtUsesRegistryTimeFormat(t *testing.T) {
	cleanupListFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "json-time-app", 3005)

	out, err := runCmd(t, listCmd, "--json")
	if err != nil {
		t.Fatalf("list --json: %v", err)
	}

	var entries []struct {
		AllocatedAt string `json:"allocatedAt"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &entries); err != nil {
		t.Fatalf("parse list JSON: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	if _, err := parseRegistryTime(entries[0].AllocatedAt); err != nil {
		t.Fatalf("expected RFC3339 allocatedAt, got %q: %v", entries[0].AllocatedAt, err)
	}
}

func TestListVerbose(t *testing.T) {
	cleanupListFlags(t)
	homeDir := newTestHome(t)
	seedRegistry(t, homeDir, "verbose-app", 3003)

	out, err := runCmd(t, listCmd, "--verbose")
	if err != nil {
		t.Fatalf("list --verbose: %v", err)
	}
	if !strings.Contains(out, "KEY_SOURCE") {
		t.Errorf("expected KEY_SOURCE column in verbose output, got: %q", out)
	}
}

func TestListSortsByPort(t *testing.T) {
	cleanupListFlags(t)
	homeDir := newTestHome(t)

	reg := &registry.Registry{
		Version:     1,
		Meta:        registry.Meta{CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		RangePolicy: registry.DefaultRangePolicy(),
		Reserved:    []int{},
		Entries: map[string]*registry.Entry{
			"app-b": {Port: 3020, DisplayName: "app-b", Framework: "next",
				KeySource: registry.KeySourcePackageJSON, ProjectPath: "/tmp/app-b",
				AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()},
			"app-a": {Port: 3010, DisplayName: "app-a", Framework: "next",
				KeySource: registry.KeySourcePackageJSON, ProjectPath: "/tmp/app-a",
				AllocatedAt: time.Now().UTC(), LastAccessedAt: time.Now().UTC()},
		},
	}
	if err := registry.Save(homeDir, reg); err != nil {
		t.Fatal(err)
	}

	out, err := runCmd(t, listCmd)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	idx3010 := strings.Index(out, "3010")
	idx3020 := strings.Index(out, "3020")
	if idx3010 < 0 || idx3020 < 0 {
		t.Fatalf("missing ports in output: %q", out)
	}
	if idx3010 > idx3020 {
		t.Error("expected 3010 before 3020 (sorted by port)")
	}
}
