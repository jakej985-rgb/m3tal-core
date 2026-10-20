package stack

import (
	"fmt"
	"sync"
	"time"

	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/state"
	"github.com/jakej985-rgb/m3tal-core/core/traefik"
)

// DockerClient defines the interface contract for container engine deployments.
type DockerClient interface {
	ComposeUp(stack Stack) error
	ComposeDown(stack Stack) error
}

// TraefikClient defines the interface contract for proxy route orchestration.
type TraefikClient interface {
	Apply(stack Stack) error
}

// StateStore defines the interface contract for persistent state operations.
type StateStore interface {
	SaveDesired(stack Stack) error
	SetStatus(stackID string, status string) error
	GetStatus(stackID string) (string, error)
}

// StackManager orchestrates multi-container stacks and routes.
type StackManager struct {
	docker  DockerClient
	traefik TraefikClient
	state   StateStore
	locks   sync.Map
}

// NewStackManager creates a new stack manager.
func NewStackManager(docker DockerClient, traefik TraefikClient, state StateStore) *StackManager {
	return &StackManager{
		docker:  docker,
		traefik: traefik,
		state:   state,
	}
}

// DockerAdapter wraps the core/docker package to implement DockerClient.
type DockerAdapter struct{}

func (d *DockerAdapter) ComposeUp(stack Stack) error {
	for _, svc := range stack.Services {
		tSvc := traefik.ServiceConfig{
			Name: svc.Name,
			Port: svc.Port,
		}
		labels := traefik.GenerateLabels(tSvc)
		if err := traefik.InjectLabels(stack.ComposePath, svc.Name, labels); err != nil {
			return fmt.Errorf("failed to inject labels for service %s: %w", svc.Name, err)
		}
	}
	// Call existing core/docker.DeployStack with default timeout
	_, err := docker.DeployStack(stack.ComposePath, 120*time.Second)
	return err
}

func (d *DockerAdapter) ComposeDown(stack Stack) error {
	_, err := docker.StopStack(stack.ComposePath, 120*time.Second)
	return err
}

// TraefikAdapter wraps the core/traefik package to implement TraefikClient.
type TraefikAdapter struct{}

func (t *TraefikAdapter) Apply(stack Stack) error {
	// Placeholder for Phase 2 - will be fully implemented in Phase 3.
	return nil
}

// StateStoreAdapter wraps the state Store database to implement StateStore.
type StateStoreAdapter struct {
	Store *state.Store
}

func (s *StateStoreAdapter) SaveDesired(stack Stack) error {
	if s.Store == nil {
		return fmt.Errorf("state store not initialized")
	}
	return s.Store.UpsertStack(stack.Name, stack.ComposePath)
}

func (s *StateStoreAdapter) SetStatus(stackID string, status string) error {
	if s.Store == nil {
		return fmt.Errorf("state store not initialized")
	}
	return s.Store.UpdateStackStatus(stackID, status)
}

func (s *StateStoreAdapter) GetStatus(stackID string) (string, error) {
	if s.Store == nil {
		return "", fmt.Errorf("state store not initialized")
	}
	stacks, err := s.Store.ListStacks()
	if err != nil {
		return "", err
	}
	for _, st := range stacks {
		if st.Name == stackID {
			return st.Status, nil
		}
	}
	return "unknown", nil
}
