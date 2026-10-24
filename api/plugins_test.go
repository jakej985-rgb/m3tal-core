package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/pkg/plugins"
)

func TestListPlugins_Empty(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "m3tal-plugins-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := plugins.NewManager("v0.2.0", tempDir)
	h := NewPluginHandlers(mgr)

	req := httptest.NewRequest("GET", "/api/v2/plugins", nil)
	w := httptest.NewRecorder()

	h.ListPlugins(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Status string      `json:"status"`
		Data   []PluginDTO `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected status 'success', got %q", resp.Status)
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(resp.Data))
	}
}

func TestListTabs_Empty(t *testing.T) {
	mgr := plugins.NewManager("v0.2.0")
	h := NewPluginHandlers(mgr)

	req := httptest.NewRequest("GET", "/api/v2/plugins/tabs", nil)
	w := httptest.NewRecorder()

	h.ListTabs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Status string                  `json:"status"`
		Data   []plugins.PluginTabSpec `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Status != "success" {
		t.Errorf("expected success, got %q", resp.Status)
	}
}

func TestCollectPlugin_NotFound(t *testing.T) {
	mgr := plugins.NewManager("v0.2.0")
	h := NewPluginHandlers(mgr)

	req := httptest.NewRequest("GET", "/api/v2/plugins/nonexistent/collect", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.CollectPlugin(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestStartPlugin_NotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "m3tal-test-plugin-start")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := plugins.NewManager("v0.2.0", tempDir)
	h := NewPluginHandlers(mgr)

	req := httptest.NewRequest("POST", "/api/v2/plugins/missing/start", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "missing")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.StartPlugin(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 for missing plugin start, got %d", w.Code)
	}
}

func TestListPlugins_WithDiscovered(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "m3tal-plugins-disc")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a dummy plugin directory with plugin.yaml
	pDir := filepath.Join(tempDir, "dummy-plugin")
	_ = os.MkdirAll(pDir, 0755)

	manifestYAML := `
apiVersion: m3tal.io/v1alpha1
kind: Plugin
metadata:
  name: dummy-plugin
  version: "1.2.0"
  description: "A test dummy plugin"
spec:
  command: "./dummy"
  capabilities:
    - collector
`
	_ = os.WriteFile(filepath.Join(pDir, "plugin.yaml"), []byte(manifestYAML), 0644)

	mgr := plugins.NewManager("v0.2.0", tempDir)
	h := NewPluginHandlers(mgr)

	req := httptest.NewRequest("GET", "/api/v2/plugins", nil)
	w := httptest.NewRecorder()

	h.ListPlugins(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Status string      `json:"status"`
		Data   []PluginDTO `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 discovered plugin, got %d", len(resp.Data))
	}
	if resp.Data[0].Name != "dummy-plugin" {
		t.Errorf("expected plugin name dummy-plugin, got %s", resp.Data[0].Name)
	}
	if resp.Data[0].State != plugins.StateStopped {
		t.Errorf("expected state stopped, got %s", resp.Data[0].State)
	}
}
