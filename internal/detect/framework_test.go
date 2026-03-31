package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func writePackageJSON(t *testing.T, dir string, deps map[string]string) {
	t.Helper()
	pkg := map[string]any{
		"name":         "test-app",
		"dependencies": deps,
	}
	data, _ := json.Marshal(pkg)
	writeFile(t, dir, "package.json", string(data))
}

func writePackageJSONDevDeps(t *testing.T, dir string, devDeps map[string]string) {
	t.Helper()
	pkg := map[string]any{
		"name":            "test-app",
		"devDependencies": devDeps,
	}
	data, _ := json.Marshal(pkg)
	writeFile(t, dir, "package.json", string(data))
}

// ---- Config file detection (highest priority) ----

func TestDetectNextJSConfigFile(t *testing.T) {
	for _, cfg := range []string{"next.config.js", "next.config.ts", "next.config.mjs"} {
		t.Run(cfg, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, cfg, "module.exports = {}")
			got := Detect(dir)
			if got != "next" {
				t.Errorf("Detect with %s = %q, want %q", cfg, got, "next")
			}
		})
	}
}

func TestDetectViteConfigFile(t *testing.T) {
	for _, cfg := range []string{"vite.config.js", "vite.config.ts", "vite.config.mjs"} {
		t.Run(cfg, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, cfg, "export default {}")
			got := Detect(dir)
			if got != "vite" {
				t.Errorf("Detect with %s = %q, want %q", cfg, got, "vite")
			}
		})
	}
}

func TestDetectAngularJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "angular.json", "{}")
	got := Detect(dir)
	if got != "angular" {
		t.Errorf("Detect = %q, want %q", got, "angular")
	}
}

// ---- package.json dependency detection ----

func TestDetectNestJS(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{"@nestjs/core": "^10.0.0"})
	got := Detect(dir)
	if got != "nest" {
		t.Errorf("Detect = %q, want %q", got, "nest")
	}
}

func TestDetectNestJSTakesPriorityOverExpress(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{
		"@nestjs/core": "^10.0.0",
		"express":      "^4.0.0",
	})
	got := Detect(dir)
	if got != "nest" {
		t.Errorf("Detect = %q, want %q (NestJS > Express)", got, "nest")
	}
}

func TestDetectCRA(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{"react-scripts": "^5.0.0"})
	got := Detect(dir)
	if got != "cra" {
		t.Errorf("Detect = %q, want %q", got, "cra")
	}
}

func TestDetectExpress(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{"express": "^4.0.0"})
	got := Detect(dir)
	if got != "express" {
		t.Errorf("Detect = %q, want %q", got, "express")
	}
}

func TestDetectNextFromDependency(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{"next": "^14.0.0"})
	got := Detect(dir)
	if got != "next" {
		t.Errorf("Detect = %q, want %q", got, "next")
	}
}

func TestDetectViteFromDependency(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{"vite": "^5.0.0"})
	got := Detect(dir)
	if got != "vite" {
		t.Errorf("Detect = %q, want %q", got, "vite")
	}
}

func TestDetectViteFromDevDependency(t *testing.T) {
	dir := t.TempDir()
	writePackageJSONDevDeps(t, dir, map[string]string{"vite": "^5.0.0"})
	got := Detect(dir)
	if got != "vite" {
		t.Errorf("Detect = %q, want %q (vite in devDependencies)", got, "vite")
	}
}

// ---- No match ----

func TestDetectNone(t *testing.T) {
	dir := t.TempDir()
	got := Detect(dir)
	if got != "" {
		t.Errorf("Detect empty dir = %q, want %q", got, "")
	}
}

func TestDetectUnknownDependency(t *testing.T) {
	dir := t.TempDir()
	// Valid package.json but no known framework dependency.
	writePackageJSON(t, dir, map[string]string{"lodash": "^4.0.0"})
	got := Detect(dir)
	if got != "" {
		t.Errorf("Detect = %q, want empty string for unknown dep", got)
	}
}

func TestDetectMalformedPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", "{bad json")
	got := Detect(dir)
	if got != "" {
		t.Errorf("Detect malformed package.json = %q, want %q", got, "")
	}
}

// ---- Priority: config file > package.json ----

func TestDetectConfigFileTakesPriorityOverDependency(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "vite.config.ts", "export default {}")
	writePackageJSON(t, dir, map[string]string{"next": "^14.0.0"})
	got := Detect(dir)
	if got != "vite" {
		t.Errorf("Detect = %q, want vite (config file > dependency)", got)
	}
}
