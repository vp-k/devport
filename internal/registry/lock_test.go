package registry

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAcquireReleaseLock(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "test.json")
	// Create the target file so flock can lock it
	if err := os.WriteFile(target, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	release, err := acquireLock(target)
	if err != nil {
		t.Fatalf("acquireLock: %v", err)
	}
	if err := release(); err != nil {
		t.Fatalf("release: %v", err)
	}
}

func TestLockIsExclusive(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "test.json")
	if err := os.WriteFile(target, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	release1, err := acquireLock(target)
	if err != nil {
		t.Fatalf("first acquireLock: %v", err)
	}

	// Second acquire in a goroutine should block until release1 is called.
	acquired := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		release2, err := acquireLock(target)
		if err != nil {
			t.Errorf("second acquireLock: %v", err)
			return
		}
		close(acquired)
		_ = release2()
	}()

	// Release first lock — second goroutine should now proceed.
	if err := release1(); err != nil {
		t.Fatalf("release1: %v", err)
	}

	wg.Wait()

	select {
	case <-acquired:
		// success: second lock was acquired after first was released
	default:
		t.Fatal("second lock was never acquired")
	}
}

func TestAcquireLockErrorOnInvalidPath(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	// Use a path inside a non-existent directory to trigger OpenFile error.
	invalid := filepath.Join(t.TempDir(), "nosuchdir", "test.json")
	_, err := acquireLock(invalid)
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestAcquireLockErrorOnFlockFail(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "test.json")
	if err := os.WriteFile(target, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Inject a flock that always fails Lock().
	orig := newFlock
	t.Cleanup(func() { newFlock = orig })
	newFlock = func(_ string) flockLocker { return &failLocker{} }

	_, err := acquireLock(target)
	if err == nil {
		t.Fatal("expected error from flock.Lock(), got nil")
	}
}

// failLocker is a flockLocker whose Lock always returns an error.
type failLocker struct{}

func (f *failLocker) Lock() error   { return os.ErrPermission }
func (f *failLocker) Unlock() error { return nil }

func TestAcquireLockCreatesFileIfMissing(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nonexistent.json")
	// File does NOT exist — lock should create it automatically.
	release, err := acquireLock(target)
	if err != nil {
		t.Fatalf("acquireLock on missing file: %v", err)
	}
	defer release() //nolint:errcheck

	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Fatal("expected lock target file to be created, but it does not exist")
	}
}
