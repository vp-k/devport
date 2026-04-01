package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Detect returns the framework name for the project in dir, or "" if unknown.
// Detection priority (highest to lowest):
//  1. Config files (next/vite/angular/cloudflare/nuxt/svelte/remix)
//  2. Bun/Deno runtime files
//  3. Go module (gin/echo/fiber/chi/go)
//  4. package.json dependencies
func Detect(dir string) string {
	// --- Config file detection (highest priority) ---
	configChecks := []struct {
		files     []string
		framework string
	}{
		{[]string{"next.config.js", "next.config.ts", "next.config.mjs"}, "next"},
		{[]string{"vite.config.js", "vite.config.ts", "vite.config.mjs"}, "vite"},
		{[]string{"angular.json"}, "angular"},
		{[]string{"wrangler.toml"}, "cloudflare"},
		{[]string{"nuxt.config.js", "nuxt.config.ts", "nuxt.config.mjs"}, "nuxt"},
		{[]string{"svelte.config.js", "svelte.config.ts"}, "svelte"},
		{[]string{"remix.config.js"}, "remix"},
	}

	for _, check := range configChecks {
		for _, f := range check.files {
			if fileExists(filepath.Join(dir, f)) {
				return check.framework
			}
		}
	}

	// --- Bun / Deno runtime detection ---
	if fileExists(filepath.Join(dir, "bun.lockb")) || fileExists(filepath.Join(dir, "bunfig.toml")) {
		return "bun"
	}
	if fileExists(filepath.Join(dir, "deno.json")) || fileExists(filepath.Join(dir, "deno.jsonc")) {
		return "deno"
	}

	// --- Go module detection ---
	if goFW := detectGoFramework(dir); goFW != "" {
		return goFW
	}

	// --- package.json dependency detection ---
	deps := readDependencies(dir)
	if deps == nil {
		return ""
	}

	// Order matters: more specific first.
	depChecks := []struct {
		pkg       string
		framework string
	}{
		{"@nestjs/core", "nest"},
		{"react-scripts", "cra"},
		{"express", "express"},
	}
	for _, check := range depChecks {
		if _, ok := deps[check.pkg]; ok {
			return check.framework
		}
	}

	// Hono with Node.js server adapter (requires both packages).
	if _, hasHono := deps["hono"]; hasHono {
		if _, hasNode := deps["@hono/node-server"]; hasNode {
			return "hono"
		}
	}

	lateChecks := []struct {
		pkg       string
		framework string
	}{
		{"@remix-run/dev", "remix"},
		{"fastify", "fastify"},
		{"next", "next"},
		{"vite", "vite"},
	}
	for _, check := range lateChecks {
		if _, ok := deps[check.pkg]; ok {
			return check.framework
		}
	}

	return ""
}

// detectGoFramework returns a Go framework name if go.mod is present in dir.
// Priority: gin → echo → fiber → chi → "go" (generic).
// Returns "" if go.mod is absent.
func detectGoFramework(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ""
	}
	content := string(data)

	goFrameworks := []struct {
		module    string
		framework string
	}{
		{"gin-gonic/gin", "gin"},
		{"labstack/echo", "echo"},
		{"gofiber/fiber", "fiber"},
		{"go-chi/chi", "chi"},
	}

	for _, fw := range goFrameworks {
		if strings.Contains(content, fw.module) {
			return fw.framework
		}
	}

	return "go"
}

// readDependencies returns the merged dependencies+devDependencies map from
// package.json, or nil if the file cannot be read or parsed.
func readDependencies(dir string) map[string]string {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	merged := make(map[string]string)
	for k, v := range pkg.Dependencies {
		merged[k] = v
	}
	for k, v := range pkg.DevDependencies {
		merged[k] = v
	}
	return merged
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
