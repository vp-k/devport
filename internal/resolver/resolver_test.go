package resolver

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// makeDir creates a temp directory and returns its path.
func makeDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// writePackageJSON writes a package.json with the given name into dir.
func writePackageJSON(t *testing.T, dir, name string) {
	t.Helper()
	pkg := map[string]string{"name": name}
	data, _ := json.Marshal(pkg)
	if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

// initGitRepo initialises a git repo in dir with a remote origin URL.
func initGitRepo(t *testing.T, dir, remoteURL string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@t.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@t.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("remote", "add", "origin", remoteURL)
}

// ---- Stage 1: package.json name ----

func TestResolveFromPackageJSONName(t *testing.T) {
	dir := makeDir(t)
	writePackageJSON(t, dir, "my-cool-app")

	result, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result.Key != "my-cool-app" {
		t.Errorf("key = %q, want %q", result.Key, "my-cool-app")
	}
	if result.Source != SourcePackageJSON {
		t.Errorf("source = %q, want %q", result.Source, SourcePackageJSON)
	}
	if result.Name != "my-cool-app" {
		t.Errorf("name = %q, want %q", result.Name, "my-cool-app")
	}
}

func TestResolveFromScopedPackageJSONName(t *testing.T) {
	dir := makeDir(t)
	writePackageJSON(t, dir, "@myorg/my-app")

	result, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result.Key != "@myorg/my-app" {
		t.Errorf("key = %q, want @myorg/my-app", result.Key)
	}
	if result.Source != SourcePackageJSON {
		t.Errorf("source = %q, want %q", result.Source, SourcePackageJSON)
	}
}

func TestResolveSkipsEmptyPackageJSONName(t *testing.T) {
	dir := makeDir(t)
	writePackageJSON(t, dir, "")
	// No git remote — should fall back to path.
	result, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result.Source == SourcePackageJSON {
		t.Errorf("should not use package.json when name is empty")
	}
}

func TestResolveSkipsInvalidPackageJSONName(t *testing.T) {
	cases := []string{
		"Has Spaces",
		"UpperCase",
		"../traversal",
		strings.Repeat("a", 215), // too long
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			dir := makeDir(t)
			writePackageJSON(t, dir, name)
			result, err := Resolve(dir)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if result.Source == SourcePackageJSON {
				t.Errorf("name %q should be rejected, but source is package.json", name)
			}
		})
	}
}

func TestResolveSkipsMalformedPackageJSON(t *testing.T) {
	dir := makeDir(t)
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result.Source == SourcePackageJSON {
		t.Error("malformed package.json should not resolve to SourcePackageJSON")
	}
}

// ---- Stage 2: git remote ----

func TestResolveFromGitRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}
	dir := makeDir(t)
	initGitRepo(t, dir, "https://github.com/user/my-repo.git")

	result, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result.Source != SourceGitRemote {
		t.Errorf("source = %q, want %q", result.Source, SourceGitRemote)
	}
	if len(result.Key) != 8 {
		t.Errorf("key length = %d, want 8", len(result.Key))
	}
}

func TestResolveGitRemoteNormalization(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}
	// Same repo URL with and without .git suffix must produce identical keys.
	dir1, dir2 := makeDir(t), makeDir(t)
	initGitRepo(t, dir1, "https://github.com/user/my-repo.git")
	initGitRepo(t, dir2, "https://github.com/user/my-repo")

	r1, _ := Resolve(dir1)
	r2, _ := Resolve(dir2)
	if r1.Key != r2.Key {
		t.Errorf("keys differ: %q vs %q", r1.Key, r2.Key)
	}
}

func TestResolveGitRemoteNameExtraction(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}
	dir := makeDir(t)
	initGitRepo(t, dir, "https://github.com/user/my-repo.git")

	result, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result.Name != "my-repo" {
		t.Errorf("name = %q, want %q", result.Name, "my-repo")
	}
}

func TestResolveGitRemoteTimeout(t *testing.T) {
	dir := makeDir(t)
	orig := gitTimeout
	origFn := gitRemoteURL
	t.Cleanup(func() {
		gitTimeout = orig
		gitRemoteURL = origFn
	})

	// Make timeout fire before the (slow) git command returns.
	gitTimeout = 1 * time.Millisecond
	gitRemoteURL = func(_ string) (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "https://github.com/user/repo", nil
	}

	result, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// Should fall back to path source.
	if result.Source != SourcePath {
		t.Errorf("source = %q, want %q after timeout", result.Source, SourcePath)
	}
}

func TestResolveGitRemoteEmptyURL(t *testing.T) {
	dir := makeDir(t)
	origFn := gitRemoteURL
	t.Cleanup(func() { gitRemoteURL = origFn })

	gitRemoteURL = func(_ string) (string, error) {
		return "", nil // empty URL, no error
	}

	result, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result.Source != SourcePath {
		t.Errorf("source = %q, want SourcePath for empty git URL", result.Source)
	}
}

// ---- Stage 3: path fallback ----

func TestResolveFromPath(t *testing.T) {
	dir := makeDir(t)
	// No package.json, no git — must use path hash.
	result, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result.Source != SourcePath {
		t.Errorf("source = %q, want %q", result.Source, SourcePath)
	}
	if len(result.Key) != 8 {
		t.Errorf("key length = %d, want 8", len(result.Key))
	}
}

func TestResolvePathIsDeterministic(t *testing.T) {
	dir := makeDir(t)
	r1, _ := Resolve(dir)
	r2, _ := Resolve(dir)
	if r1.Key != r2.Key {
		t.Errorf("path key not deterministic: %q vs %q", r1.Key, r2.Key)
	}
}

func TestResolvePathNameIsBasename(t *testing.T) {
	dir := makeDir(t)
	result, _ := Resolve(dir)
	if result.Name != filepath.Base(dir) {
		t.Errorf("name = %q, want %q", result.Name, filepath.Base(dir))
	}
}

func TestExtractRepoNameFallback(t *testing.T) {
	// A URL consisting only of slashes — all parts are empty, fallback returns url itself.
	got := extractRepoName("///")
	if got != "///" {
		t.Errorf("extractRepoName(%q) = %q, want %q", "///", got, "///")
	}
}

func TestResolveWindowsPathNormalization(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}
	// On Windows the same path with different casing should yield the same key.
	dir := makeDir(t)
	upper := strings.ToUpper(dir)
	lower := strings.ToLower(dir)

	ru, _ := Resolve(upper)
	rl, _ := Resolve(lower)
	if ru.Key != rl.Key {
		t.Errorf("Windows path keys differ for same path: upper=%q lower=%q", ru.Key, rl.Key)
	}
}
