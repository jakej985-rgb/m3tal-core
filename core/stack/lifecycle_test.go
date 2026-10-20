package stack

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jakej985-rgb/m3tal-core/core/traefik"
)

type MockDocker struct {
	mu            sync.Mutex
	ComposeUpFn   func(stack Stack) error
	ComposeDownFn func(stack Stack) error
	CalledCount   int
	DownCalled    int
}

func (m *MockDocker) ComposeUp(stack Stack) error {
	m.mu.Lock()
	m.CalledCount++
	m.mu.Unlock()
	if m.ComposeUpFn != nil {
		return m.ComposeUpFn(stack)
	}
	return nil
}

func (m *MockDocker) ComposeDown(stack Stack) error {
	m.mu.Lock()
	m.DownCalled++
	m.mu.Unlock()
	if m.ComposeDownFn != nil {
		return m.ComposeDownFn(stack)
	}
	return nil
}

type MockTraefik struct {
	mu          sync.Mutex
	ApplyFn     func(stack Stack) error
	CalledCount int
}

func (m *MockTraefik) Apply(stack Stack) error {
	m.mu.Lock()
	m.CalledCount++
	m.mu.Unlock()
	if m.ApplyFn != nil {
		return m.ApplyFn(stack)
	}
	return nil
}

type MockStateStore struct {
	mu            sync.Mutex
	SaveDesiredFn func(stack Stack) error
	SetStatusFn   func(stackID string, status string) error
	GetStatusFn   func(stackID string) (string, error)
	SavedStacks   []Stack
	Statuses      map[string]string
}

func (m *MockStateStore) SaveDesired(stack Stack) error {
	m.mu.Lock()
	m.SavedStacks = append(m.SavedStacks, stack)
	m.mu.Unlock()
	if m.SaveDesiredFn != nil {
		return m.SaveDesiredFn(stack)
	}
	return nil
}

func TestDockerAdapter_ComposeUp(t *testing.T) {
	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "docker-compose.yml")
	composeContent := `services:
  web:
    image: nginx:alpine
`
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write temp compose: %v", err)
	}

	adapter := &DockerAdapter{}
	s := Stack{
		ID:          "stack-1",
		Name:        "test-stack",
		ComposePath: composePath,
		Services: []Service{
			{Name: "web", Port: 80},
		},
	}

	// We expect this to return an error because we are in a test environment without Docker,
	// but the labels should be injected on-disk before the command executes.
	_ = adapter.ComposeUp(s)

	// Read and parse the file to verify labels were injected
	cf, err := traefik.ParseCompose(composePath)
	if err != nil {
		t.Fatalf("failed to parse modified compose: %v", err)
	}

	webSvc, ok := cf.Services["web"]
	if !ok {
		t.Fatal("missing web service in compose file")
	}

	if webSvc.Labels.Values["traefik.enable"] != "true" {
		t.Errorf("expected traefik.enable to be true, got %q", webSvc.Labels.Values["traefik.enable"])
	}

	expectedRule := "Host(`web.local`)"
	if webSvc.Labels.Values["traefik.http.routers.web.rule"] != expectedRule {
		t.Errorf("expected router rule %q, got %q", expectedRule, webSvc.Labels.Values["traefik.http.routers.web.rule"])
	}
}

func (m *MockStateStore) SetStatus(stackID string, status string) error {
	m.mu.Lock()
	if m.Statuses == nil {
		m.Statuses = make(map[string]string)
	}
	m.Statuses[stackID] = status
	m.mu.Unlock()
	if m.SetStatusFn != nil {
		return m.SetStatusFn(stackID, status)
	}
	return nil
}

func (m *MockStateStore) GetStatus(stackID string) (string, error) {
	if m.GetStatusFn != nil {
		return m.GetStatusFn(stackID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Statuses == nil {
		return "unknown", nil
	}
	status, exists := m.Statuses[stackID]
	if !exists {
		return "unknown", nil
	}
	return status, nil
}

func TestDeploy_Success(t *testing.T) {
	docker := &MockDocker{}
	traefik := &MockTraefik{}
	state := &MockStateStore{}
	mgr := NewStackManager(docker, traefik, state)

	s := Stack{
		ID:          "stack-1",
		Name:        "test-stack",
		ComposePath: "/path/to/compose.yml",
	}

	err := mgr.Deploy(s)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(state.SavedStacks) != 1 || state.SavedStacks[0].ID != "stack-1" {
		t.Errorf("expected stack to be saved in desired state")
	}

	if docker.CalledCount != 1 {
		t.Errorf("expected ComposeUp to be called once, got %d", docker.CalledCount)
	}

	if traefik.CalledCount != 1 {
		t.Errorf("expected Traefik Apply to be called once, got %d", traefik.CalledCount)
	}

	if state.Statuses["stack-1"] != "running" {
		t.Errorf("expected status to be set to running, got %q", state.Statuses["stack-1"])
	}
}

func TestDeploy_DockerFailure(t *testing.T) {
	docker := &MockDocker{
		ComposeUpFn: func(stack Stack) error {
			return errors.New("docker compose failed")
		},
	}
	traefik := &MockTraefik{}
	state := &MockStateStore{}
	mgr := NewStackManager(docker, traefik, state)

	s := Stack{
		ID:          "stack-1",
		Name:        "test-stack",
		ComposePath: "/path/to/compose.yml",
	}

	err := mgr.Deploy(s)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(state.SavedStacks) != 1 {
		t.Errorf("expected stack to still be saved in desired state")
	}

	if traefik.CalledCount != 0 {
		t.Errorf("expected Traefik Apply to not be called on docker failure")
	}

	if state.Statuses["stack-1"] == "running" {
		t.Errorf("expected status to not be set to running")
	}
}

func TestDeploy_Idempotency(t *testing.T) {
	docker := &MockDocker{}
	traefik := &MockTraefik{}
	state := &MockStateStore{
		Statuses: map[string]string{
			"stack-1": "running",
		},
	}
	mgr := NewStackManager(docker, traefik, state)

	s := Stack{
		ID:          "stack-1",
		Name:        "test-stack",
		ComposePath: "/path/to/compose.yml",
	}

	err := mgr.Deploy(s)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(state.SavedStacks) != 0 {
		t.Errorf("expected no stack to be saved (early return), got %d", len(state.SavedStacks))
	}

	if docker.CalledCount != 0 {
		t.Errorf("expected ComposeUp to not be called, got %d", docker.CalledCount)
	}

	if traefik.CalledCount != 0 {
		t.Errorf("expected Traefik Apply to not be called, got %d", traefik.CalledCount)
	}
}

func TestDeploy_Rollback(t *testing.T) {
	docker := &MockDocker{}
	traefik := &MockTraefik{
		ApplyFn: func(stack Stack) error {
			return errors.New("traefik routing failed")
		},
	}
	state := &MockStateStore{}
	mgr := NewStackManager(docker, traefik, state)

	s := Stack{
		ID:          "stack-1",
		Name:        "test-stack",
		ComposePath: "/path/to/compose.yml",
	}

	err := mgr.Deploy(s)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if docker.CalledCount != 1 {
		t.Errorf("expected ComposeUp to be called once, got %d", docker.CalledCount)
	}

	if docker.DownCalled != 1 {
		t.Errorf("expected ComposeDown to be called once for rollback, got %d", docker.DownCalled)
	}
}

func TestDeploy_ConcurrencyLocking(t *testing.T) {
	dockerBlock := make(chan struct{})
	dockerDone := make(chan struct{})

	var once sync.Once
	docker := &MockDocker{
		ComposeUpFn: func(stack Stack) error {
			once.Do(func() {
				close(dockerDone)
			})
			<-dockerBlock
			return nil
		},
	}
	traefik := &MockTraefik{}
	state := &MockStateStore{
		GetStatusFn: func(stackID string) (string, error) {
			return "pending", nil
		},
	}
	mgr := NewStackManager(docker, traefik, state)

	s := Stack{
		ID:          "stack-1",
		Name:        "test-stack",
		ComposePath: "/path/to/compose.yml",
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// First deploy
	go func() {
		defer wg.Done()
		_ = mgr.Deploy(s)
	}()

	// Wait until the first deploy reaches ComposeUp
	<-dockerDone

	// Start second deploy in background
	secondDeployStarted := make(chan struct{})
	secondDeployDone := make(chan struct{})
	go func() {
		close(secondDeployStarted)
		defer wg.Done()
		_ = mgr.Deploy(s)
		close(secondDeployDone)
	}()

	<-secondDeployStarted
	// Give it a tiny bit of time to make sure it gets blocked by the lock
	time.Sleep(50 * time.Millisecond)

	select {
	case <-secondDeployDone:
		t.Fatal("second deploy completed before first deploy released the lock!")
	default:
		// Passed, second deploy is blocked
	}

	// Release first deploy
	close(dockerBlock)

	// Wait for both to finish
	wg.Wait()

	if docker.CalledCount != 2 {
		t.Errorf("expected Docker ComposeUp to be called twice, got %d", docker.CalledCount)
	}
}
