package stack

import (
	"sync"
)

// Deploy executes the stack deployment lifecycle:
// 1. Locks the stack deployment to prevent concurrent race conditions.
// 2. Saves the desired state to the store.
// 3. Deploys the stack using the container client (Docker).
// 4. Applies Traefik routing rules.
// 5. Updates the stack deployment status to "running".
func (m *StackManager) Deploy(stack Stack) error {
	// Concurrency map locking per stack ID
	val, _ := m.locks.LoadOrStore(stack.ID, &sync.Mutex{})
	mu := val.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	// 1. Idempotency check
	currentStatus, err := m.state.GetStatus(stack.ID)
	if err == nil && currentStatus == "running" {
		return nil
	}

	if err := m.state.SaveDesired(stack); err != nil {
		return err
	}

	if err := m.docker.ComposeUp(stack); err != nil {
		return err
	}

	if err := m.traefik.Apply(stack); err != nil {
		// 2. Rollback on Traefik failure
		_ = m.docker.ComposeDown(stack)
		return err
	}

	return m.state.SetStatus(stack.ID, "running")
}
