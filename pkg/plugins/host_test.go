package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPluginDiscovery(t *testing.T) {
	tempDir := t.TempDir()

	mockPluginDir := filepath.Join(tempDir, "mock-plugin")
	if err := os.MkdirAll(mockPluginDir, 0755); err != nil {
		t.Fatalf("Failed to create mock plugin dir: %v", err)
	}

	manifestContent := `
apiVersion: m3tal.io/v1alpha1
kind: Plugin
metadata:
  name: test-sensor
  version: 0.1.0
  description: Temperature sensors
spec:
  command: ./sensor.sh
  capabilities:
    - collector
`
	if err := os.WriteFile(filepath.Join(mockPluginDir, "plugin.yaml"), []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write plugin.yaml: %v", err)
	}

	mgr := NewManager("1.0.0", tempDir)
	discovered, err := mgr.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(discovered) != 1 {
		t.Fatalf("Expected 1 discovered plugin, got %d", len(discovered))
	}

	if discovered[0].Metadata.Name != "test-sensor" {
		t.Errorf("Expected name 'test-sensor', got %q", discovered[0].Metadata.Name)
	}
	if !discovered[0].HasCapability(CapabilityCollector) {
		t.Errorf("Expected capability 'collector'")
	}
}

func TestPluginSubprocessLifecycle(t *testing.T) {
	tempDir := t.TempDir()

	mockPluginDir := filepath.Join(tempDir, "echo-plugin")
	if err := os.MkdirAll(mockPluginDir, 0755); err != nil {
		t.Fatalf("Failed to create mock plugin dir: %v", err)
	}

	// Create a mock shell script that speaks JSON-RPC 2.0
	scriptContent := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"m3tal.init"'*)
      echo '{"jsonrpc":"2.0","id":1,"result":{"name":"echo-plugin","version":"1.0.0","capabilities":["collector"]}}'
      ;;
    *'"method":"m3tal.collect"'*)
      echo '{"jsonrpc":"2.0","id":2,"result":{"status":"ok","metrics":[{"name":"temp_celsius","value":42.5,"unit":"C"}]}}'
      ;;
    *)
      echo '{"jsonrpc":"2.0","id":0,"error":{"code":-32601,"message":"Method not found"}}'
      ;;
  esac
done
`
	scriptPath := filepath.Join(mockPluginDir, "run.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write mock script: %v", err)
	}

	manifest := PluginManifest{
		APIVersion: APIVersionV1Alpha1,
		Kind:       KindPlugin,
		Metadata: PluginMeta{
			Name: "echo-plugin",
		},
		Spec: PluginSpec{
			Command:      "./run.sh",
			Capabilities: []Capability{CapabilityCollector},
		},
		Dir: mockPluginDir,
	}

	inst := &PluginInstance{
		Manifest: manifest,
		State:    StateStopped,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := inst.Start(ctx, "1.0.0"); err != nil {
		t.Fatalf("Plugin start failed: %v", err)
	}

	if inst.State != StateRunning {
		t.Fatalf("Expected state StateRunning, got %s", inst.State)
	}

	// Test collect
	res, err := inst.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if res.Status != "ok" {
		t.Errorf("Expected status ok, got %s", res.Status)
	}
	if len(res.Metrics) != 1 || res.Metrics[0].Value != 42.5 {
		t.Errorf("Expected metric temp_celsius=42.5, got %+v", res.Metrics)
	}

	// Clean up
	if err := inst.Stop(); err != nil {
		t.Fatalf("Plugin stop failed: %v", err)
	}

	if inst.State != StateStopped {
		t.Errorf("Expected state StateStopped after Stop(), got %s", inst.State)
	}
}

func TestManagerFullLifecycleIntegration(t *testing.T) {
	tempDir := t.TempDir()
	mockPluginDir := filepath.Join(tempDir, "mock-manager-plugin")
	if err := os.MkdirAll(mockPluginDir, 0755); err != nil {
		t.Fatalf("Failed to create mock plugin dir: %v", err)
	}

	manifestContent := `
apiVersion: m3tal.io/v1alpha1
kind: Plugin
metadata:
  name: mock-manager
  version: 1.0.0
  title: "Mock Manager"
spec:
  command: ./run.sh
  capabilities:
    - collector
    - ui_view
  ui:
    tabs:
      - id: mock-tab
        title: "Mock Tab"
`
	if err := os.WriteFile(filepath.Join(mockPluginDir, "plugin.yaml"), []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write plugin.yaml: %v", err)
	}

	scriptContent := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"m3tal.init"'*)
      echo '{"jsonrpc":"2.0","id":1,"result":{"name":"mock-manager","version":"1.0.0","capabilities":["collector","ui_view"],"tabs":[{"id":"mock-tab","title":"Mock Tab"}]}}'
      ;;
    *'"method":"m3tal.collect"'*)
      echo '{"jsonrpc":"2.0","id":2,"result":{"status":"ok","tables":[{"name":"mock_table","columns":["a","b"],"rows":[["1","2"]]}]}}'
      ;;
    *'"method":"m3tal.render"'*)
      echo '{"jsonrpc":"2.0","id":3,"result":{"content":"<h1>Mock Render</h1>"}}'
      ;;
    *)
      echo '{"jsonrpc":"2.0","id":0,"error":{"code":-32601,"message":"Method not found"}}'
      ;;
  esac
done
`
	scriptPath := filepath.Join(mockPluginDir, "run.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write mock script: %v", err)
	}

	mgr := NewManager("2.0.0", tempDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := mgr.LoadAndStart(ctx); err != nil {
		t.Fatalf("LoadAndStart failed: %v", err)
	}
	defer mgr.StopAll()

	active := mgr.GetAllPlugins()
	if len(active) == 0 {
		t.Fatalf("Expected at least 1 active plugin, found none")
	}

	var mockInst *PluginInstance
	for _, p := range active {
		if p.Manifest.Metadata.Name == "mock-manager" {
			mockInst = p
			break
		}
	}

	if mockInst == nil {
		t.Fatalf("mock-manager plugin not found in active plugins")
	}

	if mockInst.State != StateRunning {
		t.Fatalf("mock-manager not running, state=%s (error: %s)", mockInst.State, mockInst.LastError)
	}

	tabs := mgr.GetAllTabs()
	if len(tabs) == 0 {
		t.Errorf("Expected plugin tabs registered, got 0")
	}

	// Test Collect
	res, err := mockInst.Collect(ctx)
	if err != nil {
		t.Fatalf("mockInst.Collect failed: %v", err)
	}
	if len(res.Tables) == 0 {
		t.Errorf("Expected tables in collect result")
	}

	// Test Render
	renderRes, err := mockInst.Render(ctx, "mock-tab", 80, 24, "dark")
	if err != nil {
		t.Fatalf("mockInst.Render failed: %v", err)
	}
	if renderRes.Content == "" {
		t.Errorf("Expected non-empty render content")
	}
}
