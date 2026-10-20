package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/core/docker"
)

func TestGetContainerLogs_Success(t *testing.T) {
	// Set mock provider
	mock := &docker.MockProvider{
		Containers: []docker.ContainerInfo{
			{ID: "12345", Names: []string{"/mock-container"}},
		},
	}
	docker.SetProvider(mock)

	srv := NewServer("test-token")
	req := httptest.NewRequest("GET", "/api/containers/mock-container/logs", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "mock-container")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.GetContainerLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var apiResponse struct {
		Status string            `json:"status"`
		Data   map[string]string `json:"data"`
		Error  any               `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &apiResponse); err != nil {
		t.Fatalf("failed to unmarshal response: %v, body: %s", err, w.Body.String())
	}
	if apiResponse.Status != "success" {
		t.Fatalf("expected success status, got %q (error: %v)", apiResponse.Status, apiResponse.Error)
	}

	logs := apiResponse.Data["logs"]
	if logs != "mock logs for mock-container" {
		t.Errorf("expected logs 'mock logs for mock-container', got %q", logs)
	}
}
