package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/pkg/plugins"
)

func TestGetAllSettings(t *testing.T) {
	mgr := plugins.NewManager("v0.18.0")
	h := NewPluginHandlers(mgr)

	req := httptest.NewRequest("GET", "/api/v2/settings", nil)
	w := httptest.NewRecorder()

	h.GetAllSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Core          CoreSettings                   `json:"core"`
			Sections      []plugins.PluginSettingSection `json:"sections"`
			PluginConfigs map[string]map[string]any      `json:"plugin_configs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "success" {
		t.Errorf("expected status 'success', got %s", resp.Status)
	}
	if resp.Data.Core.PollInterval <= 0 {
		t.Errorf("expected positive poll interval, got %d", resp.Data.Core.PollInterval)
	}
}

func TestSaveCoreSettings(t *testing.T) {
	mgr := plugins.NewManager("v0.18.0")
	h := NewPluginHandlers(mgr)

	payload := CoreSettings{
		NodeName:     "test-node",
		PollInterval: 10,
		Theme:        "matrix",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v2/settings/core", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.SaveCoreSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Verify loaded settings match
	loaded := loadCoreSettings()
	if loaded.NodeName != "test-node" {
		t.Errorf("expected node_name 'test-node', got '%s'", loaded.NodeName)
	}
	if loaded.PollInterval != 10 {
		t.Errorf("expected poll_interval 10, got %d", loaded.PollInterval)
	}
}

func TestPluginSettings_NotFound(t *testing.T) {
	mgr := plugins.NewManager("v0.18.0")
	h := NewPluginHandlers(mgr)

	req := httptest.NewRequest("GET", "/api/v2/plugins/unknown-plugin/settings", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "unknown-plugin")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.GetPluginSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
