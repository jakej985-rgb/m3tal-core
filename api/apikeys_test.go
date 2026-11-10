package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	db.ConfigDir = dir
	t.Cleanup(func() { db.Close() })

	srv := NewServerWithStore("primary-token", db)
	r := chi.NewRouter()

	r.Route("/api/v2", func(r chi.Router) {
		r.Use(srv.Authenticate)
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
		Status string               `json:"status"`
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

	// 3. Reject empty name and oversized name
	emptyBody, _ := json.Marshal(map[string]string{"name": "   "})
	req = httptest.NewRequest("POST", "/api/v2/auth/keys", bytes.NewReader(emptyBody))
	req.Header.Set("X-API-Token", "primary-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty name, got %d", w.Code)
	}

	longBody, _ := json.Marshal(map[string]string{"name": strings.Repeat("a", 300)})
	req = httptest.NewRequest("POST", "/api/v2/auth/keys", bytes.NewReader(longBody))
	req.Header.Set("X-API-Token", "primary-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized name, got %d", w.Code)
	}

	// 4. Create server-generated API key
	validBody, _ := json.Marshal(map[string]any{
		"name":        "Mesh Fleet",
		"permissions": []string{"*"},
	})
	req = httptest.NewRequest("POST", "/api/v2/auth/keys", bytes.NewReader(validBody))
	req.Header.Set("X-API-Token", "primary-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Status string             `json:"status"`
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

	// 5. Authenticate using the newly generated API key (with padding / lowercase bearer)
	req = httptest.NewRequest("GET", "/api/v2/version", nil)
	req.Header.Set("Authorization", "bearer   "+createdKey.Token+"  ")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 using generated API key, got %d: %s", w.Code, w.Body.String())
	}

	// 6. Delete invalid ID string -> 400
	req = httptest.NewRequest("DELETE", "/api/v2/auth/keys/abc", nil)
	req.Header.Set("X-API-Token", "primary-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid ID string, got %d", w.Code)
	}

	// 7. Delete non-existent ID -> 404
	req = httptest.NewRequest("DELETE", "/api/v2/auth/keys/99999", nil)
	req.Header.Set("X-API-Token", "primary-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on non-existent ID, got %d", w.Code)
	}

	// 8. Delete the valid API key
	delURL := "/api/v2/auth/keys/" + strconv.FormatInt(createdKey.ID, 10)
	req = httptest.NewRequest("DELETE", delURL, nil)
	req.Header.Set("X-API-Token", "primary-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d: %s", w.Code, w.Body.String())
	}

	// 9. Deleted key no longer authenticates
	req = httptest.NewRequest("GET", "/api/v2/version", nil)
	req.Header.Set("Authorization", "Bearer "+createdKey.Token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after key deletion, got %d", w.Code)
	}
}

func TestScopedAPIKeyPermissions(t *testing.T) {
	dir := t.TempDir()
	db, err := state.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("state.Open failed: %v", err)
	}
	db.ConfigDir = dir
	t.Cleanup(func() { db.Close() })

	srv := NewServerWithStore("primary-token", db)
	r := chi.NewRouter()
	r.Route("/api/v2", func(r chi.Router) {
		r.Use(srv.Authenticate)
		r.Get("/system/stats", srv.requireScope(state.ScopeSystemRead, srv.GetStats))
		r.Get("/auth/keys", srv.requireScope(state.ScopeKeysManage, srv.ListAPIKeys))
		r.Get("/plugins", srv.requireScope(state.ScopePluginsRead, srv.GetVersion))
	})

	// Scoped key: system read only
	scoped, err := db.CreateAPIKey("Monitor Bot", []string{state.ScopeSystemRead}, state.DirectionLocal, "", "")
	if err != nil {
		t.Fatalf("CreateAPIKey scoped: %v", err)
	}

	// Allowed: system:read endpoint
	req := httptest.NewRequest("GET", "/api/v2/system/stats", nil)
	req.Header.Set("X-API-Token", scoped.Token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusForbidden || w.Code == http.StatusUnauthorized {
		t.Fatalf("scoped key should access system:read, got %d", w.Code)
	}

	// Denied: keys:manage endpoint -> 403
	req = httptest.NewRequest("GET", "/api/v2/auth/keys", nil)
	req.Header.Set("X-API-Token", scoped.Token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 INSUFFICIENT_SCOPE, got %d", w.Code)
	}

	// plugins:write implies plugins:read
	writer, err := db.CreateAPIKey("Plugin Writer", []string{state.ScopePluginsWrite}, state.DirectionLocal, "", "")
	if err != nil {
		t.Fatalf("CreateAPIKey writer: %v", err)
	}
	req = httptest.NewRequest("GET", "/api/v2/plugins", nil)
	req.Header.Set("X-API-Token", writer.Token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusForbidden {
		t.Fatal("plugins:write should imply plugins:read")
	}

	// Static token bypasses scope checks
	req = httptest.NewRequest("GET", "/api/v2/auth/keys", nil)
	req.Header.Set("X-API-Token", "primary-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusForbidden {
		t.Fatal("static token should bypass scope checks")
	}
}

func TestRemoteAPIKeyCannotAuthenticateInbound(t *testing.T) {
	dir := t.TempDir()
	db, err := state.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("state.Open failed: %v", err)
	}
	db.ConfigDir = dir
	t.Cleanup(func() { db.Close() })

	// Remote key stores a foreign credential
	remote, err := db.CreateAPIKey("NAS Node", []string{state.ScopeFull}, state.DirectionRemote, "192.168.1.20", "foreign-token-abc123")
	if err != nil {
		t.Fatalf("CreateAPIKey remote: %v", err)
	}
	if remote.Direction != state.DirectionRemote {
		t.Fatalf("expected direction remote, got %q", remote.Direction)
	}
	if remote.Token != "foreign-token-abc123" {
		t.Fatalf("expected supplied remote token, got %q", remote.Token)
	}

	// Remote token must not validate inbound
	if db.ValidateAPIKey(remote.Token) {
		t.Fatal("remote-direction key must not authenticate inbound")
	}
	if _, ok := db.LookupAPIKey(remote.Token); ok {
		t.Fatal("LookupAPIKey must not return remote-direction keys")
	}

	// Remote keys are excluded from the plugin-facing export
	if err := db.SyncAPIKeysToFile(); err != nil {
		t.Fatalf("SyncAPIKeysToFile: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "api_keys.json"))
	if err != nil {
		t.Fatalf("read api_keys.json: %v", err)
	}
	if strings.Contains(string(data), "foreign-token-abc123") {
		t.Fatal("remote keys must not be exported to api_keys.json")
	}
}

func TestAuthMiddlewareEmptyTokenProtection(t *testing.T) {
	dir := t.TempDir()
	db, err := state.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("state.Open failed: %v", err)
	}
	db.ConfigDir = dir
	t.Cleanup(func() { db.Close() })

	// Server initialized with empty static token
	srv := NewServerWithStore("", db)
	r := chi.NewRouter()
	r.Route("/api/v2", func(r chi.Router) {
		r.Use(srv.Authenticate)
		r.Get("/version", srv.GetVersion)
	})

	// Request with empty token MUST be rejected with 401
	req := httptest.NewRequest("GET", "/api/v2/version", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request when APIToken is empty, got %d", w.Code)
	}

	// Request with whitespace token MUST be rejected with 401
	req = httptest.NewRequest("GET", "/api/v2/version", nil)
	req.Header.Set("X-API-Token", "   ")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for whitespace token, got %d", w.Code)
	}
}
