package api

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestGetPluginCatalog(t *testing.T) {
	mgr := plugins.NewManager("v0.2.0")
	h := NewPluginHandlers(mgr)

	req := httptest.NewRequest("GET", "/api/v2/plugins/catalog", nil)
	w := httptest.NewRecorder()

	h.GetPluginCatalog(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Status string             `json:"status"`
		Data   []CatalogPluginDTO `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) == 0 {
		t.Fatalf("expected catalog items, got 0")
	}

	// Verify docker-monitor is present
	foundDocker := false
	for _, item := range resp.Data {
		if item.Name == "docker-monitor" {
			foundDocker = true
			if item.ProcessName != "m3tal-docker" {
				t.Errorf("expected processName m3tal-docker, got %s", item.ProcessName)
			}
			if item.Version != "0.5.1" && item.Version != "0.2.2" && item.Version != "0.1.0" {
				t.Errorf("expected version 0.5.1, 0.2.2, or 0.1.0, got %s", item.Version)
			}
		}
	}
	if !foundDocker {
		t.Errorf("expected docker-monitor in catalog")
	}
}

func TestCompareSemVer(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		wantCmp  int
		wantType string
	}{
		{"0.5.1", "0.5.1", 0, "equal"},
		{"v0.5.1", "0.5.1", 0, "equal"},
		{"0.4.2", "0.5.1", -1, "minor"},
		{"0.5.0", "0.5.1", -1, "patch"},
		{"1.0.0", "0.5.1", 1, "major"},
		{"0.5.2", "0.5.1", 1, "patch"},
	}

	for _, tc := range tests {
		cmp, diffType := CompareSemVer(tc.v1, tc.v2)
		if cmp != tc.wantCmp || diffType != tc.wantType {
			t.Errorf("CompareSemVer(%q, %q) = (%d, %q), want (%d, %q)",
				tc.v1, tc.v2, cmp, diffType, tc.wantCmp, tc.wantType)
		}
	}
}

func TestStartPlugin_Discovered(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "m3tal-plugins-start-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pDir := filepath.Join(tempDir, "mock-run")
	_ = os.MkdirAll(pDir, 0755)

	script := filepath.Join(pDir, "run.sh")
	scriptContent := `#!/bin/sh
while read -r line; do
  case "$line" in
    *'m3tal.init'*)
      echo '{"jsonrpc":"2.0","id":1,"result":{"name":"mock-run","version":"1.0.0","capabilities":["collector"]}}'
      ;;
    *)
      echo '{"jsonrpc":"2.0","id":99,"result":{}}'
      ;;
  esac
done
`
	_ = os.WriteFile(script, []byte(scriptContent), 0755)

	manifestYAML := fmt.Sprintf(`
apiVersion: m3tal.io/v1alpha1
kind: Plugin
metadata:
  name: mock-run
  version: 1.0.0
spec:
  command: %s
  capabilities:
    - collector
`, script)
	_ = os.WriteFile(filepath.Join(pDir, "plugin.yaml"), []byte(manifestYAML), 0644)

	mgr := plugins.NewManager("v0.2.0", tempDir)
	h := NewPluginHandlers(mgr)

	req := httptest.NewRequest("POST", "/api/v2/plugins/mock-run/start", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "mock-run")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.StartPlugin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for discovered plugin start, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPluginChangelog(t *testing.T) {
	mgr := plugins.NewManager("v0.2.0")
	h := NewPluginHandlers(mgr)

	pluginsToTest := []string{"docker-monitor", "compose", "stack-orchestrator", "ai-engine", "smart-disk"}
	for _, p := range pluginsToTest {
		req := httptest.NewRequest("GET", "/api/v2/plugins/"+p+"/changelog", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("name", p)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		h.GetPluginChangelog(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 for plugin %s changelog, got %d: %s", p, w.Code, w.Body.String())
		}

		var resp struct {
			Status string `json:"status"`
			Data   struct {
				Name      string `json:"name"`
				Changelog string `json:"changelog"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("failed to unmarshal changelog response for %s: %v", p, err)
		}
		if len(resp.Data.Changelog) == 0 {
			t.Errorf("expected non-empty changelog for %s", p)
		}
	}
}
