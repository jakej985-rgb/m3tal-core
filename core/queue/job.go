package queue

import "errors"

// ErrQueueNotInitialized is returned when package-level Submit is called before a manager is started.
var ErrQueueNotInitialized = errors.New("queue manager not initialized")

// Submit enqueues a job into the global queue manager.
func Submit(job Job) error {
	if globalManager == nil {
		return ErrQueueNotInitialized
	}
	return globalManager.Submit(job)
}
