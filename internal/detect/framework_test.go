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

// ---- Bun detection ----

func TestDetectBunLockb(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lockb", "")
	if got := Detect(dir); got != "bun" {
		t.Errorf("Detect = %q, want bun", got)
	}
}

func TestDetectBunfigToml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bunfig.toml", "")
	if got := Detect(dir); got != "bun" {
		t.Errorf("Detect = %q, want bun", got)
	}
}

// ---- Deno detection ----

func TestDetectDenoJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "deno.json", "{}")
	if got := Detect(dir); got != "deno" {
		t.Errorf("Detect = %q, want deno", got)
	}
}

func TestDetectDenoJSONC(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "deno.jsonc", "{}")
	if got := Detect(dir); got != "deno" {
		t.Errorf("Detect = %q, want deno", got)
	}
}

// ---- Go framework detection ----

func writeGoMod(t *testing.T, dir, content string) {
	t.Helper()
	writeFile(t, dir, "go.mod", content)
}

func TestDetectGin(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module example\n\nrequire github.com/gin-gonic/gin v1.9.0\n")
	if got := Detect(dir); got != "gin" {
		t.Errorf("Detect = %q, want gin", got)
	}
}

func TestDetectEcho(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module example\n\nrequire github.com/labstack/echo/v4 v4.12.0\n")
	if got := Detect(dir); got != "echo" {
		t.Errorf("Detect = %q, want echo", got)
	}
}

func TestDetectFiber(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module example\n\nrequire github.com/gofiber/fiber/v2 v2.52.0\n")
	if got := Detect(dir); got != "fiber" {
		t.Errorf("Detect = %q, want fiber", got)
	}
}

func TestDetectChi(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module example\n\nrequire github.com/go-chi/chi/v5 v5.0.12\n")
	if got := Detect(dir); got != "chi" {
		t.Errorf("Detect = %q, want chi", got)
	}
}

func TestDetectGoGeneric(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module example\n\ngo 1.22\n")
	if got := Detect(dir); got != "go" {
		t.Errorf("Detect = %q, want go", got)
	}
}

// Gin takes priority over echo when both are present.
func TestDetectGoPriorityGinOverEcho(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module example\n\nrequire (\n\tgithub.com/gin-gonic/gin v1.9.0\n\tgithub.com/labstack/echo/v4 v4.12.0\n)\n")
	if got := Detect(dir); got != "gin" {
		t.Errorf("Detect = %q, want gin (gin > echo priority)", got)
	}
}

// Config file still takes priority over go.mod.
func TestDetectConfigFileTakesPriorityOverGoMod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "next.config.js", "module.exports = {}")
	writeGoMod(t, dir, "module example\n\nrequire github.com/gin-gonic/gin v1.9.0\n")
	if got := Detect(dir); got != "next" {
		t.Errorf("Detect = %q, want next (config file > go.mod)", got)
	}
}

// ---- Phase 3-B: New framework detection ----

func TestDetectCloudflare(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "wrangler.toml", "[build]")
	if got := Detect(dir); got != "cloudflare" {
		t.Errorf("Detect = %q, want cloudflare", got)
	}
}

func TestDetectNuxtJS(t *testing.T) {
	for _, cfg := range []string{"nuxt.config.js", "nuxt.config.ts", "nuxt.config.mjs"} {
		t.Run(cfg, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, cfg, "export default {}")
			if got := Detect(dir); got != "nuxt" {
				t.Errorf("Detect with %s = %q, want nuxt", cfg, got)
			}
		})
	}
}

func TestDetectSvelteKit(t *testing.T) {
	for _, cfg := range []string{"svelte.config.js", "svelte.config.ts"} {
		t.Run(cfg, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, cfg, "export default {}")
			if got := Detect(dir); got != "svelte" {
				t.Errorf("Detect with %s = %q, want svelte", cfg, got)
			}
		})
	}
}

func TestDetectRemixConfigFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "remix.config.js", "module.exports = {}")
	if got := Detect(dir); got != "remix" {
		t.Errorf("Detect = %q, want remix", got)
	}
}

func TestDetectRemixFromDependency(t *testing.T) {
	dir := t.TempDir()
	writePackageJSONDevDeps(t, dir, map[string]string{"@remix-run/dev": "^2.0.0"})
	if got := Detect(dir); got != "remix" {
		t.Errorf("Detect = %q, want remix", got)
	}
}

func TestDetectFastify(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{"fastify": "^4.0.0"})
	if got := Detect(dir); got != "fastify" {
		t.Errorf("Detect = %q, want fastify", got)
	}
}

func TestDetectHonoNode(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{
		"hono":              "^4.0.0",
		"@hono/node-server": "^1.0.0",
	})
	if got := Detect(dir); got != "hono" {
		t.Errorf("Detect = %q, want hono", got)
	}
}

func TestDetectHonoWithoutNodeServer(t *testing.T) {
	// hono alone (no @hono/node-server) → falls through to later checks or ""
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{"hono": "^4.0.0"})
	got := Detect(dir)
	if got == "hono" {
		t.Errorf("Detect = %q, hono without @hono/node-server should not match hono", got)
	}
}

// Priority: cloudflare config file > bun runtime
func TestDetectCloudflareTakesPriorityOverBun(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "wrangler.toml", "[build]")
	writeFile(t, dir, "bun.lockb", "")
	if got := Detect(dir); got != "cloudflare" {
		t.Errorf("Detect = %q, want cloudflare (config > runtime)", got)
	}
}

// Priority: express > hono (express checked before hono in depChecks)
func TestDetectExpressTakesPriorityOverHono(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{
		"express":           "^4.0.0",
		"hono":              "^4.0.0",
		"@hono/node-server": "^1.0.0",
	})
	if got := Detect(dir); got != "express" {
		t.Errorf("Detect = %q, want express (express > hono)", got)
	}
}
