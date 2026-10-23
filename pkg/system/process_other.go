//go:build !linux

package system

// SetProcessName is a no-op on non-Linux platforms.
func SetProcessName(name string) {}
