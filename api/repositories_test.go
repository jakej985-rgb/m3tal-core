package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/pkg/plugins"
)

func TestRepositoriesEndpoints(t *testing.T) {
	mgr := plugins.NewManager("v0.2.0")
	h := NewPluginHandlers(mgr)

	// 1. GET /api/v2/repositories
	req := httptest.NewRequest("GET", "/api/v2/repositories", nil)
	w := httptest.NewRecorder()
	h.GetRepositories(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var getResp struct {
		Status string `json:"status"`
		Data   struct {
			Official RepoConfig            `json:"official"`
			Custom   []RepoConfig          `json:"custom"`
			Packages []AptPackageCandidate `json:"packages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if getResp.Data.Official.ID != "official-m3tal" {
		t.Errorf("expected official-m3tal repo ID, got %s", getResp.Data.Official.ID)
	}
	if getResp.Data.Official.URL != "https://jakej985-rgb.github.io/m3tal-apt-key" {
		t.Errorf("expected official repo URL, got %s", getResp.Data.Official.URL)
	}

	// 2. POST /api/v2/repositories (Save custom repo)
	newRepo := RepoConfig{
		ID:           "test-custom-repo",
		Name:         "Community Plugins Lab",
		URL:          "https://example.org/apt-repo",
		Distribution: "stable",
		Component:    "main",
		Enabled:      true,
	}
	body, _ := json.Marshal(newRepo)
	postReq := httptest.NewRequest("POST", "/api/v2/repositories", bytes.NewReader(body))
	wPost := httptest.NewRecorder()
	h.SaveRepository(wPost, postReq)

	if wPost.Code != http.StatusOK {
		t.Fatalf("expected status 200 on save, got %d: %s", wPost.Code, wPost.Body.String())
	}

	// 3. POST /api/v2/repositories/refresh
	refReq := httptest.NewRequest("POST", "/api/v2/repositories/refresh", nil)
	wRef := httptest.NewRecorder()
	h.RefreshRepositories(wRef, refReq)
	if wRef.Code != http.StatusOK {
		t.Fatalf("expected status 200 on refresh, got %d: %s", wRef.Code, wRef.Body.String())
	}

	// 4. DELETE /api/v2/repositories/{id}
	r := chi.NewRouter()
	r.Delete("/api/v2/repositories/{id}", h.DeleteRepository)

	delReq := httptest.NewRequest("DELETE", "/api/v2/repositories/test-custom-repo", nil)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, delReq)

	if wDel.Code != http.StatusOK {
		t.Fatalf("expected status 200 on delete, got %d: %s", wDel.Code, wDel.Body.String())
	}
}
