package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Detect returns the framework name for the project in dir, or "" if unknown.
// Detection priority: config files first, then package.json dependencies.
func Detect(dir string) string {
	// --- Config file detection (highest priority) ---
	configChecks := []struct {
		files     []string
		framework string
	}{
		{[]string{"next.config.js", "next.config.ts", "next.config.mjs"}, "next"},
		{[]string{"vite.config.js", "vite.config.ts", "vite.config.mjs"}, "vite"},
		{[]string{"angular.json"}, "angular"},
	}

	for _, check := range configChecks {
		for _, f := range check.files {
			if fileExists(filepath.Join(dir, f)) {
				return check.framework
			}
		}
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
		{"next", "next"},
		{"vite", "vite"},
	}

	for _, check := range depChecks {
		if _, ok := deps[check.pkg]; ok {
			return check.framework
		}
	}

	return ""
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
