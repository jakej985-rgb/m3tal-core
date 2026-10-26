package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jakej985-rgb/m3tal-core/pkg/version"
)

func TestGetVersion(t *testing.T) {
	srv := NewServer("test-token")
	req := httptest.NewRequest("GET", "/api/v2/version", nil)
	w := httptest.NewRecorder()

	srv.GetVersion(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Status string              `json:"status"`
		Data   version.VersionInfo `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode version response: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected success status, got %s", resp.Status)
	}

	if resp.Data.Version == "" {
		t.Errorf("expected non-empty version")
	}

	if resp.Data.Display == "" {
		t.Errorf("expected non-empty display version")
	}
}
