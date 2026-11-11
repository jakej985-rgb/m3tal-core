package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/core/state"
)

func openTestStore(t *testing.T) *state.Store {
	t.Helper()
	db, err := state.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("state.Open failed: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestPluginRuntimeAPIKeyProvisionsScopedKey(t *testing.T) {
	db := openTestStore(t)

	token := pluginRuntimeAPIKey(db)
	if token == "" {
		t.Fatal("pluginRuntimeAPIKey returned empty token")
	}

	// The provisioned key must validate and carry only the plugin scopes.
	rec, ok := db.LookupAPIKey(token)
	if !ok {
		t.Fatal("provisioned token does not validate")
	}
	if rec.Name != pluginRuntimeKeyName {
		t.Fatalf("key name = %q; want %q", rec.Name, pluginRuntimeKeyName)
	}
	for _, scope := range []string{state.ScopeKeysManage, state.ScopeSystemWrite} {
		for _, p := range rec.Permissions {
			if p == scope || p == state.ScopeFull {
				t.Fatalf("plugin runtime key must not grant %q, got perms %v", scope, rec.Permissions)
			}
		}
	}

	// Idempotent: second call returns the same token, no duplicate key.
	if again := pluginRuntimeAPIKey(db); again != token {
		t.Fatal("pluginRuntimeAPIKey not idempotent")
	}
	keys, _ := db.ListAPIKeys()
	count := 0
	for _, k := range keys {
		if k.Name == pluginRuntimeKeyName {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 plugin-runtime key, got %d", count)
	}
}

func TestPluginRuntimeAPIKeyNilStore(t *testing.T) {
	if got := pluginRuntimeAPIKey(nil); got != "" {
		t.Fatalf("nil store should yield empty token, got %q", got)
	}
}

func TestPluginUIAuthRequiresToken(t *testing.T) {
	db := openTestStore(t)
	srv := NewServerWithStore("primary-token", db)

	r := chi.NewRouter()
	r.With(srv.pluginUIAuth).Get("/plugins/{name}/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// No credentials -> 401.
	req := httptest.NewRequest("GET", "/plugins/demo/index.html", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated request status = %d; want 401", w.Code)
	}

	// Static token via header (browser clients authenticate API calls this
	// way; the session cookie covers requests that cannot send headers).
	req = httptest.NewRequest("GET", "/plugins/demo/index.html", nil)
	req.Header.Set("X-API-Token", "primary-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("token request status = %d; want 200", w.Code)
	}
	cookies := w.Result().Cookies()
	var uiCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == pluginUICookieName {
			uiCookie = c
		}
	}
	if uiCookie == nil {
		t.Fatal("expected m3tal_ui_token cookie to be set")
	}
	if uiCookie.Path != "/plugins/" || !uiCookie.HttpOnly || uiCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie attributes wrong: %+v", uiCookie)
	}

	// Cookie alone authenticates subresource requests.
	req = httptest.NewRequest("GET", "/plugins/demo/assets/app.js", nil)
	req.AddCookie(uiCookie)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("cookie request status = %d; want 200", w.Code)
	}

	// Bogus token -> 401.
	req = httptest.NewRequest("GET", "/plugins/demo/index.html", nil)
	req.Header.Set("X-API-Token", "wrong")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad token status = %d; want 401", w.Code)
	}

	// The Path=/ session cookie (set by Authenticate on header auth)
	// authenticates plugin UI requests without any header.
	req = httptest.NewRequest("GET", "/plugins/demo/index.html", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "primary-token"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("session-cookie request status = %d; want 200", w.Code)
	}
}
