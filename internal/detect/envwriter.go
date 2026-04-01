package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvOptions allows callers to override defaults returned by EnvConfigFor.
type EnvOptions struct {
	VarName string // override the env variable name (e.g. "MY_PORT")
	Output  string // override the output file path (e.g. ".env.custom")
}

// EnvConfig holds the resolved configuration for writing a port env var.
type EnvConfig struct {
	File    string // relative path of the env file (e.g. ".env.local")
	VarName string // env variable name (e.g. "PORT" or "VITE_PORT")
}

// frameworkEnvDefaults maps framework name → (file, varName).
var frameworkEnvDefaults = map[string][2]string{
	"next":       {".env.local", "PORT"},
	"vite":       {".env.local", "VITE_PORT"},
	"express":    {".env", "PORT"},
	"angular":    {".env.local", "PORT"},
	"nest":       {".env", "PORT"},
	"cra":        {".env.local", "PORT"},
	"bun":        {".env", "PORT"},
	"deno":       {".env", "PORT"},
	"go":         {".env", "PORT"},
	"gin":        {".env", "PORT"},
	"echo":       {".env", "PORT"},
	"fiber":      {".env", "PORT"},
	"chi":        {".env", "PORT"},
	"cloudflare": {".dev.vars", "PORT"},
	"nuxt":       {".env", "PORT"},
	"svelte":     {".env", "PORT"},
	"remix":      {".env", "PORT"},
	"fastify":    {".env", "PORT"},
	"hono":       {".env", "PORT"},
}

// EnvConfigFor returns the EnvConfig for the given framework, applying any
// overrides from opts. Unknown or empty frameworks fall back to .env.local/PORT.
func EnvConfigFor(framework string, opts EnvOptions) EnvConfig {
	defaults, ok := frameworkEnvDefaults[framework]
	if !ok {
		defaults = [2]string{".env.local", "PORT"}
	}

	cfg := EnvConfig{
		File:    defaults[0],
		VarName: defaults[1],
	}

	if opts.Output != "" {
		cfg.File = opts.Output
	}
	if opts.VarName != "" {
		cfg.VarName = opts.VarName
	}

	return cfg
}

// Injectable for testing.
var (
	osMkdirAll  = os.MkdirAll
	osReadFile  = os.ReadFile
	osWriteFile = os.WriteFile
)

// WriteEnvFile writes PORT=<port> into the env file described by cfg inside
// dir. If the file already contains a line for cfg.VarName it is replaced in
// place; otherwise the assignment is appended. Other lines (including blank
// lines and comments) are preserved. Parent directories are created as needed.
func WriteEnvFile(dir string, port int, cfg EnvConfig) error {
	fullPath := filepath.Join(dir, filepath.FromSlash(cfg.File))

	if err := osMkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("create parent dirs: %w", err)
	}

	assignment := fmt.Sprintf("%s=%d", cfg.VarName, port)
	prefix := cfg.VarName + "="

	data, err := osReadFile(fullPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read env file: %w", err)
		}
		// File does not exist — create it with just the assignment.
		return osWriteFile(fullPath, []byte(assignment+"\n"), 0644)
	}

	lines := strings.Split(string(data), "\n")
	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = assignment
			replaced = true
			break
		}
	}

	var out string
	if replaced {
		out = strings.Join(lines, "\n")
	} else {
		// Append: ensure there is a trailing newline before appending.
		content := string(data)
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		out = content + assignment + "\n"
	}

	return osWriteFile(fullPath, []byte(out), 0644)
}
