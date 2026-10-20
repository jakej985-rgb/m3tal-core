package agents

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/state"
)

func TestRunReconcileDrift(t *testing.T) {
	// Setup temporary database path
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_reconcile.db")
	os.Setenv("M3TAL_STATE_DB", dbPath)
	defer os.Unsetenv("M3TAL_STATE_DB")

	// Open store and insert a running stack that is missing its containers
	store, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test store: %v", err)
	}

	// Create control plane logs directory
	os.Setenv("M3TAL_DATA_PATH", tempDir)
	defer os.Unsetenv("M3TAL_DATA_PATH")
	_ = os.MkdirAll(filepath.Join(tempDir, "logs"), 0755)

	err = store.UpsertStack("testapp", "/tmp/testapp-compose.yml")
	if err != nil {
		t.Fatalf("failed to upsert test stack: %v", err)
	}
	err = store.UpdateStackStatus("testapp", "running")
	if err != nil {
		t.Fatalf("failed to update stack status: %v", err)
	}
	store.Close()

	// Set mock provider with no containers (i.e. drift)
	mock := &docker.MockProvider{
		Containers: []docker.ContainerInfo{},
	}
	docker.SetProvider(mock)

	// Run reconciliation
	actions, err := RunReconcile()
	if err != nil {
		t.Fatalf("RunReconcile failed: %v", err)
	}

	// Reconcile should detect drift and log action
	if len(actions) == 0 {
		t.Fatal("expected reconciliation actions to be taken, got none")
	}

	foundDriftMsg := false
	for _, act := range actions {
		if contains(act, "Drift detected on stack testapp") {
			foundDriftMsg = true
			break
		}
	}

	if !foundDriftMsg {
		t.Errorf("expected drift action message for testapp, got: %v", actions)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || stringsContains(s, substr)
}

func stringsContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
