package queue

import (
	"context"
	"errors"
	"sync"
	"time"
)

type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

// Job represents a unit of work that can be processed by the queue.
type Job interface {
	ID() string
	Priority() int // Higher value = higher priority
	Execute(ctx context.Context) (any, error)
	Payload() map[string]any
	Type() string
}

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

var globalManager *Manager

// Manager coordinates job queues, workers, and status tracking.
type Manager struct {
	mu           sync.RWMutex
	pending      []Job
	records      map[string]*JobRecord
	activeJobs   map[string]context.CancelFunc
	maxWorkers   int
	maxQueueSize int
	jobSem       chan struct{} // Limits concurrent active workers
	jobNotifyCh  chan struct{} // Signal to workers that a job is ready
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewManager creates and starts a new queue Manager.
func NewManager(maxWorkers, maxQueueSize int) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		records:      make(map[string]*JobRecord),
		activeJobs:   make(map[string]context.CancelFunc),
		maxWorkers:   maxWorkers,
		maxQueueSize: maxQueueSize,
		jobSem:       make(chan struct{}, maxWorkers),
		jobNotifyCh:  make(chan struct{}, 1),
		ctx:          ctx,
		cancel:       cancel,
	}

	globalManager = m

	go m.workerLoop(ctx)
	return m
}

// Close stops the worker loop and cancels any running jobs.
func (m *Manager) Close() {
	m.cancel()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, cancel := range m.activeJobs {
		cancel()
	}
}

// Submit inserts a job into the queue, sorting it by priority.
func (m *Manager) Submit(job Job) error {
	m.mu.Lock()
	if len(m.pending) >= m.maxQueueSize {
		m.mu.Unlock()
		return errors.New("queue capacity exceeded")
	}

	if _, exists := m.records[job.ID()]; exists {
		m.mu.Unlock()
		return errors.New("job with this ID already exists")
	}

	record := &JobRecord{
		ID:          job.ID(),
		Type:        job.Type(),
		Priority:    job.Priority(),
		Status:      StatusPending,
		Payload:     job.Payload(),
		SubmittedAt: time.Now(),
	}
	m.records[job.ID()] = record
	m.pending = append(m.pending, job)
	m.mu.Unlock()

	// Notify worker loop
	select {
	case m.jobNotifyCh <- struct{}{}:
	default:
	}

	return nil
}

// Cancel terminates a pending or running job by ID.
func (m *Manager) Cancel(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, found := m.records[id]
	if !found {
		return false
	}

	if record.Status == StatusPending {
		// Remove from pending list
		for i, j := range m.pending {
			if j.ID() == id {
				m.pending = append(m.pending[:i], m.pending[i+1:]...)
				break
			}
		}
		record.Status = StatusFailed
		record.Error = "job cancelled by user"
		now := time.Now()
		record.FinishedAt = &now
		return true
	} else if record.Status == StatusRunning {
		if cancel, active := m.activeJobs[id]; active {
			cancel()
			delete(m.activeJobs, id)
			record.Status = StatusFailed
			record.Error = "job cancelled by user"
			now := time.Now()
			record.FinishedAt = &now
			return true
		}
	}

	return false
}

// List returns a copy of all job records (history and current).
func (m *Manager) List() []*JobRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*JobRecord, 0, len(m.records))
	for _, rec := range m.records {
		// Copy record
		recCopy := *rec
		list = append(list, &recCopy)
	}
	return list
}

// Get retrieves a specific job status record by ID.
func (m *Manager) Get(id string) (*JobRecord, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rec, found := m.records[id]
	if !found {
		return nil, false
	}
	recCopy := *rec
	return &recCopy, true
}

func (m *Manager) nextJob() Job {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.pending) == 0 {
		return nil
	}

	// Find the job with the highest priority (stable FIFO for ties)
	bestIdx := 0
	for i := 1; i < len(m.pending); i++ {
		if m.pending[i].Priority() > m.pending[bestIdx].Priority() {
			bestIdx = i
		}
	}

	job := m.pending[bestIdx]
	m.pending = append(m.pending[:bestIdx], m.pending[bestIdx+1:]...)
	return job
}

func (m *Manager) workerLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.jobNotifyCh:
		}

		for {
			acquired := false
			select {
			case <-ctx.Done():
				return
			case m.jobSem <- struct{}{}:
				acquired = true
			default:
			}
			if !acquired {
				break
			}

			job := m.nextJob()
			if job == nil {
				<-m.jobSem // Release slot
				break
			}

			go m.runJob(ctx, job)
		}
	}
}

func (m *Manager) runJob(ctx context.Context, job Job) {
	defer func() {
		<-m.jobSem
		// Try triggering workers again in case other jobs were queued
		select {
		case m.jobNotifyCh <- struct{}{}:
		default:
		}
	}()

	jobCtx, cancel := context.WithCancel(ctx)

	m.mu.Lock()
	record, exists := m.records[job.ID()]
	if !exists || record.Status != StatusPending {
		m.mu.Unlock()
		cancel()
		return
	}

	record.Status = StatusRunning
	now := time.Now()
	record.StartedAt = &now
	m.activeJobs[job.ID()] = cancel
	m.mu.Unlock()

	// Execute task
	result, err := job.Execute(jobCtx)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Clean up cancel mapping
	delete(m.activeJobs, job.ID())
	cancel()

	// If job was cancelled mid-run, its status might have been set to failed by Cancel()
	if record.Status == StatusFailed && record.Error == "job cancelled by user" {
		return
	}

	finishedNow := time.Now()
	record.FinishedAt = &finishedNow
	if err != nil {
		record.Status = StatusFailed
		record.Error = err.Error()
	} else {
		record.Status = StatusCompleted
		record.Result = result
	}
}
