package resolver

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Source indicates how the project key was resolved.
type Source string

const (
	SourcePackageJSON Source = "package.json"
	SourceGitRemote   Source = "git-remote"
	SourcePath        Source = "path"
)

// Resolution holds the result of resolving a project key.
type Resolution struct {
	Key    string
	Source Source
	Name   string
}

// validNpmName matches valid npm package names (including scoped).
var validNpmName = regexp.MustCompile(
	`^(@[a-z0-9\-~][a-z0-9\-._~]*/)?[a-z0-9\-~][a-z0-9\-._~]*$`,
)

const maxNpmNameLen = 214

// gitTimeout is the maximum time allowed for the git command (overridable in tests).
var gitTimeout = 2 * time.Second

// gitRemoteURL executes "git remote get-url origin" in dir and returns the URL.
// Overridable in tests to avoid spawning real git processes.
var gitRemoteURL = func(dir string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// Resolve determines the project key for the given directory using a three-
// stage fallback strategy:
//  1. package.json "name" field (if valid npm name)
//  2. git remote origin URL → SHA-256[:8] hex
//  3. absolute path → SHA-256[:8] hex
func Resolve(dir string) (Resolution, error) {
	// Stage 1: package.json name
	if r, ok := resolveFromPackageJSON(dir); ok {
		return r, nil
	}

	// Stage 2: git remote origin
	if r, ok := resolveFromGitRemote(dir); ok {
		return r, nil
	}

	// Stage 3: path fallback
	return resolveFromPath(dir), nil
}

// --- Stage 1 ---

func resolveFromPackageJSON(dir string) (Resolution, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return Resolution{}, false
	}

	var pkg struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return Resolution{}, false
	}

	if !isValidNpmName(pkg.Name) {
		return Resolution{}, false
	}

	return Resolution{
		Key:    pkg.Name,
		Source: SourcePackageJSON,
		Name:   pkg.Name,
	}, true
}

func isValidNpmName(name string) bool {
	if name == "" || len(name) > maxNpmNameLen {
		return false
	}
	return validNpmName.MatchString(name)
}

// --- Stage 2 ---

func resolveFromGitRemote(dir string) (Resolution, bool) {
	type result struct {
		url string
		err error
	}
	ch := make(chan result, 1)

	go func() {
		url, err := gitRemoteURL(dir)
		ch <- result{url, err}
	}()

	select {
	case res := <-ch:
		if res.err != nil || res.url == "" {
			return Resolution{}, false
		}
		normalized := normalizeGitURL(res.url)
		hash := sha8(normalized)
		name := extractRepoName(normalized)
		return Resolution{
			Key:    hash,
			Source: SourceGitRemote,
			Name:   name,
		}, true
	case <-time.After(gitTimeout):
		return Resolution{}, false
	}
}

func normalizeGitURL(url string) string {
	url = strings.ToLower(url)
	url = strings.TrimSuffix(url, ".git")
	return url
}

func extractRepoName(url string) string {
	// Handle both https://host/user/repo and git@host:user/repo patterns.
	url = strings.ReplaceAll(url, ":", "/")
	parts := strings.Split(url, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return url
}

// --- Stage 3 ---

func resolveFromPath(dir string) Resolution {
	normalized := normalizePath(dir)
	return Resolution{
		Key:    sha8(normalized),
		Source: SourcePath,
		Name:   filepath.Base(dir),
	}
}

func normalizePath(p string) string {
	// On Windows normalise drive letter and separators for consistent hashing.
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return filepath.ToSlash(p)
}

// --- Helpers ---

func sha8(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum[:4]) // 4 bytes → 8 hex chars
}
