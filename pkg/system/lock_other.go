//go:build !linux && !darwin

package system

// AcquireProcessLock fallback for non-Unix platforms.
func AcquireProcessLock(role string) (func(), bool) {
	return func() {}, true
}
