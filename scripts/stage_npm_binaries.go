package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

type target struct {
	packageName string // @vp-k/devport-win32-x64  (package.json name, used in logs)
	dirName     string // devport-win32-x64         (actual directory name, used for paths)
	goos        string
	goarch      string
	binaryName  string
}

var targets = map[string]target{
	"devport-darwin-arm64": {packageName: "@vp-k/devport-darwin-arm64", dirName: "devport-darwin-arm64", goos: "darwin", goarch: "arm64", binaryName: "devport"},
	"devport-darwin-x64":   {packageName: "@vp-k/devport-darwin-x64", dirName: "devport-darwin-x64", goos: "darwin", goarch: "amd64", binaryName: "devport"},
	"devport-linux-arm64":  {packageName: "@vp-k/devport-linux-arm64", dirName: "devport-linux-arm64", goos: "linux", goarch: "arm64", binaryName: "devport"},
	"devport-linux-x64":    {packageName: "@vp-k/devport-linux-x64", dirName: "devport-linux-x64", goos: "linux", goarch: "amd64", binaryName: "devport"},
	"devport-win32-x64":    {packageName: "@vp-k/devport-win32-x64", dirName: "devport-win32-x64", goos: "windows", goarch: "amd64", binaryName: "devport.exe"},
}

func main() {
	only := flag.String("only", "", "Stage a single platform package by name")
	flag.Parse()

	root, err := findRepoRoot()
	if err != nil {
		fail(err)
	}

	version, err := readNpmVersion(root)
	if err != nil {
		fail(fmt.Errorf("read npm version: %w", err))
	}

	selected, err := selectedTargets(*only)
	if err != nil {
		fail(err)
	}

	for _, target := range selected {
		if err := stageBinary(root, version, target); err != nil {
			fail(err)
		}
		fmt.Printf("staged %s@%s\n", target.packageName, version)
	}
}

func selectedTargets(only string) ([]target, error) {
	if only != "" {
		selectedTarget, ok := targets[only]
		if !ok {
			return nil, fmt.Errorf("unknown platform package %q", only)
		}
		return []target{selectedTarget}, nil
	}

	names := make([]string, 0, len(targets))
	for name := range targets {
		names = append(names, name)
	}
	sort.Strings(names)

	selected := make([]target, 0, len(names))
	for _, name := range names {
		selected = append(selected, targets[name])
	}
	return selected, nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goMod := filepath.Join(dir, "go.mod")
		npmPkg := filepath.Join(dir, "npm", "package.json")
		if fileExists(goMod) && fileExists(npmPkg) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repo root from %s", dir)
		}
		dir = parent
	}
}

func stageBinary(root, version string, target target) error {
	packageDir := filepath.Join(root, "npm", "platforms", target.dirName)
	binDir := filepath.Join(packageDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("create bin dir for %s: %w", target.packageName, err)
	}

	outputPath := filepath.Join(binDir, target.binaryName)
	ldflags := fmt.Sprintf("-s -w -X github.com/vp-k/devport/cmd.version=%s", version)
	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", outputPath, ".")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOOS="+target.goos,
		"GOARCH="+target.goarch,
		"CGO_ENABLED=0",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build %s: %w\n%s", target.packageName, err, string(output))
	}

	return nil
}

func readNpmVersion(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "npm", "package.json"))
	if err != nil {
		return "", err
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", err
	}
	if pkg.Version == "" {
		return "", fmt.Errorf("version field is empty in npm/package.json")
	}
	return pkg.Version, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
