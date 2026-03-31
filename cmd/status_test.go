package cmd

import (
	"encoding/json"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/user01/devport/internal/registry"
)

func cleanupStatusFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { statusFlagJSON = false })
}

func TestStatusNotRegistered(t *testing.T) {
	cleanupStatusFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "status-not-registered-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, statusCmd)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out, "NOT REGISTERED") {
		t.Errorf("expected NOT REGISTERED, got: %q", out)
	}
}

func TestStatusNotRegisteredJSON(t *testing.T) {
	cleanupStatusFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "status-not-reg-json-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	out, err := runCmd(t, statusCmd, "--json")
	if err != nil {
		t.Fatalf("status --json: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("invalid JSON: %v — output: %q", err, out)
	}
	if m["status"] != "NOT REGISTERED" {
		t.Errorf("expected status NOT REGISTERED, got: %v", m["status"])
	}
}

func TestStatusAllocated(t *testing.T) {
	cleanupStatusFlags(t)
	homeDir := newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "status-allocated-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Seed registry with port that is NOT bound (so ProbePort returns true = free = ALLOCATED).
	reg := newEmptyRegistry()
	reg.Entries["status-allocated-app"] = &registry.Entry{
		Port:           7100,
		KeySource:      registry.KeySourcePackageJSON,
		DisplayName:    "status-allocated-app",
		ProjectPath:    dir,
		Framework:      "next",
		AllocatedAt:    time.Now().UTC(),
		LastAccessedAt: time.Now().UTC(),
	}
	registry.Save(homeDir, reg)

	out, err := runCmd(t, statusCmd)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out, "ALLOCATED") {
		t.Errorf("expected ALLOCATED, got: %q", out)
	}
}

func TestStatusListening(t *testing.T) {
	cleanupStatusFlags(t)
	homeDir := newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "status-listening-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	// Bind a real port so ProbePort returns false = port is in use = LISTENING.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	reg := newEmptyRegistry()
	reg.Entries["status-listening-app"] = &registry.Entry{
		Port:           port,
		KeySource:      registry.KeySourcePackageJSON,
		DisplayName:    "status-listening-app",
		ProjectPath:    dir,
		Framework:      "express",
		AllocatedAt:    time.Now().UTC(),
		LastAccessedAt: time.Now().UTC(),
	}
	registry.Save(homeDir, reg)

	out, err := runCmd(t, statusCmd)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out, "LISTENING") {
		t.Errorf("expected LISTENING, got: %q", out)
	}
}

func TestStatusJSON(t *testing.T) {
	cleanupStatusFlags(t)
	homeDir := newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "status-json-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	reg := newEmptyRegistry()
	reg.Entries["status-json-app"] = &registry.Entry{
		Port:           7200,
		KeySource:      registry.KeySourcePackageJSON,
		DisplayName:    "status-json-app",
		ProjectPath:    dir,
		Framework:      "next",
		AllocatedAt:    time.Now().UTC(),
		LastAccessedAt: time.Now().UTC(),
	}
	registry.Save(homeDir, reg)

	out, err := runCmd(t, statusCmd, "--json")
	if err != nil {
		t.Fatalf("status --json: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["port"] == nil {
		t.Error("expected port in JSON output")
	}
}

func TestStatusGetwdError(t *testing.T) {
	cleanupStatusFlags(t)
	newTestHome(t)
	injectGetwd(t)

	_, err := runCmd(t, statusCmd)
	if err == nil {
		t.Fatal("expected getwd error")
	}
}

func TestStatusHomeDirError(t *testing.T) {
	cleanupStatusFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectHomeDir(t)

	_, err := runCmd(t, statusCmd)
	if err == nil {
		t.Fatal("expected homedir error")
	}
}

func TestStatusResolveError(t *testing.T) {
	cleanupStatusFlags(t)
	newTestHome(t)
	dir := newTestProject(t)
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectResolve(t)

	_, err := runCmd(t, statusCmd)
	if err == nil {
		t.Fatal("expected resolve error")
	}
}

func TestStatusRegistryLoadError(t *testing.T) {
	cleanupStatusFlags(t)
	newTestHome(t)
	dir := newTestProjectWithPackageJSON(t, "status-load-err-app")
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)
	injectRegistryLoad(t)

	_, err := runCmd(t, statusCmd)
	if err == nil {
		t.Fatal("expected load error")
	}
}
