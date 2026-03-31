package registry

import (
	"os"

	"github.com/gofrs/flock"
)

// releaseFn releases an acquired file lock.
type releaseFn func() error

// flockLocker is the interface satisfied by *flock.Flock for testing.
type flockLocker interface {
	Lock() error
	Unlock() error
}

// newFlock is the factory used to create a flock instance.
// Overridable in tests.
var newFlock = func(path string) flockLocker {
	return flock.New(path)
}

// acquireLock obtains an exclusive file lock on the given path.
// If the file does not exist it is created automatically.
// The returned function must be called to release the lock.
func acquireLock(path string) (releaseFn, error) {
	// Ensure the file exists so flock can operate on it.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	f.Close()

	fl := newFlock(path)
	if err := fl.Lock(); err != nil {
		return nil, err
	}
	return func() error {
		return fl.Unlock()
	}, nil
}
