package models

import "time"

// JobStatus represents the state of a background queue job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// JobRecord represents the status and details of a job in the queue.
type JobRecord struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Priority    int            `json:"priority"`
	Status      JobStatus      `json:"status"`
	Payload     map[string]any `json:"payload,omitempty"`
	SubmittedAt time.Time      `json:"submitted_at"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	FinishedAt  *time.Time     `json:"finished_at,omitempty"`
	Result      any            `json:"result,omitempty"`
	Error       string         `json:"error,omitempty"`
}
