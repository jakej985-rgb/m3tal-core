//go:build linux || darwin

package system

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// AcquireProcessLock attempts to acquire an exclusive non-blocking lock for a role (e.g. m3tal-core, m3tal-gui).
// Returns an unlock cleanup function and true if acquired; false if an instance is already running.
func AcquireProcessLock(role string) (func(), bool) {
	lockDir := os.TempDir()
	lockPath := filepath.Join(lockDir, fmt.Sprintf("%s.lock", role))

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, false
	}

	// Non-blocking exclusive lock
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = f.Close()
		return nil, false
	}

	unlock := func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
		_ = os.Remove(lockPath)
	}

	return unlock, true
}
