//go:build linux

package system

import "os"

// SetProcessName sets the current process comm name in /proc/self/comm
// (visible in top, ps, and task managers). TASK_COMM_LEN is 16 bytes
// including the NUL terminator, so at most 15 characters are retained.
func SetProcessName(name string) {
	if len(name) == 0 {
		return
	}
	if len(name) > 15 {
		name = name[:15]
	}
	_ = os.WriteFile("/proc/self/comm", []byte(name), 0)
}
