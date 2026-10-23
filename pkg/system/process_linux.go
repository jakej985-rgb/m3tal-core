//go:build linux

package system

import (
	"syscall"
	"unsafe"
)

// SetProcessName sets the current process comm name in /proc/self/comm (visible in top, ps, and task managers).
func SetProcessName(name string) {
	if len(name) == 0 {
		return
	}
	bytes := append([]byte(name), 0)
	if len(bytes) > 16 {
		bytes = bytes[:16]
		bytes[15] = 0
	}
	// PR_SET_NAME = 15
	_, _, _ = syscall.RawSyscall(syscall.SYS_PRCTL, 15, uintptr(unsafe.Pointer(&bytes[0])), 0)
}
