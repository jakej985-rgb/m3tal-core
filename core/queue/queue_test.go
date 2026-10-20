package queue

import (
	"context"
	"sync"
	"testing"
	"time"
)

type MockJob struct {
	id       string
	priority int
	payload  map[string]any
	jobType  string
	execute  func(ctx context.Context) (any, error)
}

func (m *MockJob) ID() string              { return m.id }
func (m *MockJob) Priority() int           { return m.priority }
func (m *MockJob) Payload() map[string]any { return m.payload }
func (m *MockJob) Type() string            { return m.jobType }
func (m *MockJob) Execute(ctx context.Context) (any, error) {
	if m.execute != nil {
		return m.execute(ctx)
	}
	return nil, nil
}

func TestQueue_Basic(t *testing.T) {
	mgr := NewManager(2, 5)
	defer mgr.Close()

	runChan := make(chan struct{})
	job := &MockJob{
		id:       "job-1",
		priority: 1,
		payload:  map[string]any{"data": "test"},
		jobType:  "test-job",
		execute: func(ctx context.Context) (any, error) {
			close(runChan)
			return "done", nil
		},
	}

	err := mgr.Submit(job)
	if err != nil {
		t.Fatalf("failed to submit job: %v", err)
	}

	select {
	case <-runChan:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for job to execute")
	}

	// Wait for status update
	time.Sleep(50 * time.Millisecond)

	rec, found := mgr.Get("job-1")
	if !found {
		t.Fatal("job record not found")
	}

	if rec.Status != StatusCompleted {
		t.Errorf("expected status completed, got %s", rec.Status)
	}
	if rec.Result != "done" {
		t.Errorf("expected result 'done', got %v", rec.Result)
	}
}

func TestQueue_Priority(t *testing.T) {
	// 1 worker, so we can queue jobs and verify execution order
	mgr := NewManager(1, 10)
	defer mgr.Close()

	// Block the worker loop first with a running job
	blockChan := make(chan struct{})
	unblockChan := make(chan struct{})
	blockJob := &MockJob{
		id:       "blocker",
		priority: 1,
		execute: func(ctx context.Context) (any, error) {
			close(blockChan)
			<-unblockChan
			return nil, nil
		},
	}
	_ = mgr.Submit(blockJob)

	// Wait for blocker to start
	<-blockChan

	var mu sync.Mutex
	executionOrder := make([]string, 0)

	submitJob := func(id string, priority int) {
		job := &MockJob{
			id:       id,
			priority: priority,
			execute: func(ctx context.Context) (any, error) {
				mu.Lock()
				executionOrder = append(executionOrder, id)
				mu.Unlock()
				return nil, nil
			},
		}
		_ = mgr.Submit(job)
	}

	// Submit jobs in arbitrary order
	submitJob("job-low", 1)
	submitJob("job-high", 10)
	submitJob("job-medium", 5)

	// Unblock worker loop
	close(unblockChan)

	// Wait for jobs to finish
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(executionOrder) != 3 {
		t.Fatalf("expected 3 executed jobs, got %d", len(executionOrder))
	}

	if executionOrder[0] != "job-high" || executionOrder[1] != "job-medium" || executionOrder[2] != "job-low" {
		t.Errorf("expected order [job-high, job-medium, job-low], got %v", executionOrder)
	}
}

func TestQueue_Cancel(t *testing.T) {
	mgr := NewManager(1, 5)
	defer mgr.Close()

	// 1. Cancel pending job
	blockChan := make(chan struct{})
	unblockChan := make(chan struct{})
	blockJob := &MockJob{
		id:       "blocker",
		priority: 1,
		execute: func(ctx context.Context) (any, error) {
			close(blockChan)
			<-unblockChan
			return nil, nil
		},
	}
	_ = mgr.Submit(blockJob)
	<-blockChan

	pendingJob := &MockJob{
		id:       "pending-1",
		priority: 1,
	}
	_ = mgr.Submit(pendingJob)

	cancelledPending := mgr.Cancel("pending-1")
	if !cancelledPending {
		t.Error("expected pending job to be cancelled")
	}

	rec, found := mgr.Get("pending-1")
	if !found || rec.Status != StatusFailed || rec.Error != "job cancelled by user" {
		t.Errorf("expected status failed, got %+v", rec)
	}

	// 2. Cancel running job
	runningJobChan := make(chan struct{})
	runningJobInterrupted := make(chan struct{})
	runningJob := &MockJob{
		id:       "running-1",
		priority: 1,
		execute: func(ctx context.Context) (any, error) {
			close(runningJobChan)
			<-ctx.Done()
			close(runningJobInterrupted)
			return nil, ctx.Err()
		},
	}

	// Unblock blocker to start running-1
	close(unblockChan)
	_ = mgr.Submit(runningJob)
	<-runningJobChan

	cancelledRunning := mgr.Cancel("running-1")
	if !cancelledRunning {
		t.Error("expected running job to be cancelled")
	}

	select {
	case <-runningJobInterrupted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("running job did not receive cancellation signal")
	}

	rec2, found2 := mgr.Get("running-1")
	if !found2 || rec2.Status != StatusFailed || rec2.Error != "job cancelled by user" {
		t.Errorf("expected status failed for running job, got %+v", rec2)
	}
}

func TestQueue_Capacity(t *testing.T) {
	mgr := NewManager(1, 2)
	defer mgr.Close()

	// Fill queue: 1 running, 1 pending (capacity is 2)
	blockChan := make(chan struct{})
	blockJob := &MockJob{
		id:       "job-1",
		priority: 1,
		execute: func(ctx context.Context) (any, error) {
			<-blockChan
			return nil, nil
		},
	}
	_ = mgr.Submit(blockJob)

	job2 := &MockJob{id: "job-2", priority: 1}
	if err := mgr.Submit(job2); err != nil {
		t.Fatalf("failed to submit second job: %v", err)
	}

	job3 := &MockJob{id: "job-3", priority: 1}
	if err := mgr.Submit(job3); err == nil {
		t.Error("expected queue capacity exceeded error")
	}

	close(blockChan)
}

func TestQueue_Submit(t *testing.T) {
	mgr := NewManager(1, 5)
	defer mgr.Close()

	runChan := make(chan struct{})
	job := &MockJob{
		id:       "job-submit",
		priority: 1,
		execute: func(ctx context.Context) (any, error) {
			close(runChan)
			return nil, nil
		},
	}

	err := Submit(job)
	if err != nil {
		t.Fatalf("failed to submit job via pkg level Submit: %v", err)
	}

	select {
	case <-runChan:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for job to execute")
	}
}
