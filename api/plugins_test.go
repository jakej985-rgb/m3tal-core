package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/core/plugins"
)

func TestListPluginsByKindOrName(t *testing.T) {
	// Mock registry
	mockReg := &plugin.Registry{
		Routes: []plugin.RoutePlugin{
			{
				Metadata: plugin.PluginMetadata{Name: "my-route-plugin"},
				Service:  "my-service",
				Enabled:  true,
			},
		},
		Stacks: []plugin.StackPlugin{
			{
				Metadata: plugin.PluginMetadata{Name: "my-stack-plugin"},
				Enabled:  false,
			},
		},
	}

	h := &PluginHandlers{registry: mockReg}

	// 1. Test routes kind
	req := httptest.NewRequest("GET", "/api/v2/plugins/routes", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("kind", "routes")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.ListPluginsByKind(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "my-route-plugin") {
		t.Errorf("expected my-route-plugin in routes list, got %s", w.Body.String())
	}

	// 2. Test fallback by name
	req2 := httptest.NewRequest("GET", "/api/v2/plugins/my-stack-plugin", nil)
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("kind", "my-stack-plugin") // matching wildcard
	req2 = req2.WithContext(context.WithValue(req2.Context(), chi.RouteCtxKey, rctx2))
	w2 := httptest.NewRecorder()

	h.ListPluginsByKind(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "my-stack-plugin") {
		t.Errorf("expected stack details in name search, got %s", w2.Body.String())
	}

	// 3. Test not found fallback
	req3 := httptest.NewRequest("GET", "/api/v2/plugins/nonexistent", nil)
	rctx3 := chi.NewRouteContext()
	rctx3.URLParams.Add("kind", "nonexistent")
	req3 = req3.WithContext(context.WithValue(req3.Context(), chi.RouteCtxKey, rctx3))
	w3 := httptest.NewRecorder()

	h.ListPluginsByKind(w3, req3)

	if w3.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w3.Code)
	}
}

func TestEnablePlugin_UnsatisfiedDependencies(t *testing.T) {
	// Create mock registry where A requires B, and B is disabled
	mockReg := &plugin.Registry{
		Routes: []plugin.RoutePlugin{
			{
				Metadata:   plugin.PluginMetadata{Name: "A"},
				DependsOn:  []string{"B"},
				SourcePath: "/tmp/a.yml",
				Enabled:    false,
			},
		},
		Stacks: []plugin.StackPlugin{
			{
				Metadata:   plugin.PluginMetadata{Name: "B"},
				SourcePath: "/tmp/b.yml.disabled",
				Enabled:    false,
			},
		},
	}
	mockReg.PopulateWarnings()

	// Write mock yml files so LoadPlugin doesn't fail on read
	err := os.WriteFile("/tmp/a.yml", []byte("apiVersion: m3tal/v1\nkind: Route\nmetadata:\n  name: A\ndepends_on: [\"B\"]\nspec:\n  service: s\n  domain: d\n  port: 80"), 0644)
	if err != nil {
		t.Fatalf("failed to write mock manifest A: %v", err)
	}
	defer os.Remove("/tmp/a.yml")

	h := &PluginHandlers{registry: mockReg}

	reqBody := `{"name": "A", "kind": "Route"}`
	req := httptest.NewRequest("POST", "/api/v2/plugins/enable", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.Enable(w, req)

	// Since B is disabled, enabling A should be rejected with 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var response struct {
		Status string `json:"status"`
		Error  struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Details struct {
				Warnings []string `json:"warnings"`
			} `json:"details"`
		} `json:"error"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "error" || !strings.Contains(response.Error.Message, "unsatisfied dependencies") {
		t.Errorf("expected error status for unsatisfied dependencies, got: %+v", response)
	}
	if len(response.Error.Details.Warnings) == 0 || !strings.Contains(response.Error.Details.Warnings[0], "Dependency plugin B is disabled") {
		t.Errorf("expected warning about B being disabled, got warnings: %v", response.Error.Details.Warnings)
	}
}
