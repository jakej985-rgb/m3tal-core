package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/core/state"
)

func setupTestServerWithStore(t *testing.T) (*Server, *chi.Mux, *state.Store) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("state.Open failed: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	srv := NewServerWithStore("primary-token", db)
	r := chi.NewRouter()

	r.Route("/api/v2", func(r chi.Router) {
		r.Use(srv.chiAuthMiddleware)
		r.Get("/auth/keys", srv.ListAPIKeys)
		r.Post("/auth/keys", srv.CreateAPIKey)
		r.Delete("/auth/keys/{id}", srv.DeleteAPIKey)
		r.Get("/version", srv.GetVersion)
	})

	return srv, r, db
}

func TestAPIKeysEndpoints(t *testing.T) {
	_, r, _ := setupTestServerWithStore(t)

	// 1. List initial keys
	req := httptest.NewRequest("GET", "/api/v2/auth/keys", nil)
	req.Header.Set("X-API-Token", "primary-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var listResp struct {
		Status string             `json:"status"`
		Data   []state.APIKeyRecord `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if len(listResp.Data) == 0 {
		t.Fatal("expected at least 1 default provisioned key")
	}

	// 2. Reject custom token entries (Jellyfin style strictly enforced)
	invalidBody, _ := json.Marshal(map[string]string{
		"name":  "Mesh Fleet",
		"token": "my-custom-token-attempt",
	})
	req = httptest.NewRequest("POST", "/api/v2/auth/keys", bytes.NewReader(invalidBody))
	req.Header.Set("X-API-Token", "primary-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for custom key entry attempt, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Create server-generated API key
	validBody, _ := json.Marshal(map[string]string{
		"name": "Mesh Fleet",
	})
	req = httptest.NewRequest("POST", "/api/v2/auth/keys", bytes.NewReader(validBody))
	req.Header.Set("X-API-Token", "primary-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Status string            `json:"status"`
		Data   state.APIKeyRecord `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	createdKey := createResp.Data
	if createdKey.Name != "Mesh Fleet" {
		t.Errorf("expected name 'Mesh Fleet', got %s", createdKey.Name)
	}
	if len(createdKey.Token) != 32 {
		t.Errorf("expected 32 hex chars generated token, got len=%d", len(createdKey.Token))
	}

	// 4. Authenticate using the newly generated API key
	req = httptest.NewRequest("GET", "/api/v2/version", nil)
	req.Header.Set("Authorization", "Bearer "+createdKey.Token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 using generated API key, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Delete the API key
	delURL := "/api/v2/auth/keys/" + strconvFormat(createdKey.ID)
	req = httptest.NewRequest("DELETE", delURL, nil)
	req.Header.Set("X-API-Token", "primary-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d: %s", w.Code, w.Body.String())
	}

	// 6. Deleted key no longer authenticates
	req = httptest.NewRequest("GET", "/api/v2/version", nil)
	req.Header.Set("Authorization", "Bearer "+createdKey.Token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after key deletion, got %d", w.Code)
	}
}

func strconvFormat(id int64) string {
	var buf [20]byte
	i := len(buf)
	neg := false
	if id < 0 {
		neg = true
		id = -id
	}
	for id >= 10 {
		i--
		buf[i] = byte('0' + id%10)
		id /= 10
	}
	i--
	buf[i] = byte('0' + id)
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
