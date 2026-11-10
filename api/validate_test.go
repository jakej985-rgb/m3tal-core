package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/pkg/plugins"
)

func TestSanitizePluginName(t *testing.T) {
	valid := []string{
		"docker-monitor", "m3tal-plugin-ai", "mesh-cluster",
		"plugin.v2", "a", "A1_-x",
	}
	for _, name := range valid {
		got, err := sanitizePluginName(name)
		if err != nil {
			t.Errorf("sanitizePluginName(%q) unexpected error: %v", name, err)
		}
		if got != name {
			t.Errorf("sanitizePluginName(%q) = %q", name, got)
		}
	}

	long := ""
	for i := 0; i < 200; i++ {
		long += "a"
	}
	invalid := []string{
		"", "..", ".", "../uninstall", "foo/bar", `foo\bar`,
		"a/b", "name/../x", "-leading-dash", "with space",
		"semi;colon", "$(cmd)", "`tick`", long,
	}
	for _, name := range invalid {
		if _, err := sanitizePluginName(name); err == nil {
			t.Errorf("sanitizePluginName(%q) expected error, got nil", name)
		}
	}
}

// TestPluginHandlersRejectTraversalName ensures chi URL parameters containing
// path traversal sequences are rejected before reaching filesystem sinks.
func TestPluginHandlersRejectTraversalName(t *testing.T) {
	tempDir := t.TempDir()
	h := NewPluginHandlers(plugins.NewManager("v0.2.0", tempDir))

	cases := []struct {
		name string
		call func(w http.ResponseWriter, r *http.Request)
	}{
		{"uninstall", h.UninstallPlugin},
		{"install", h.InstallPlugin},
		{"download", h.DownloadPluginDeb},
		{"changelog", h.GetPluginChangelog},
		{"start", h.StartPlugin},
		{"stop", h.StopPlugin},
		{"settings-get", h.GetPluginSettings},
		{"settings-save", h.SavePluginSettings},
	}
	for _, tc := range cases {
		for _, bad := range []string{"..", "../x", "a/b"} {
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("name", bad)
			req := httptest.NewRequest("POST", "/api/v2/plugins/x", nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()
			tc.call(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("%s with name %q: got status %d, want 400", tc.name, bad, w.Code)
			}
		}
	}
}

func TestFindPluginDebRejectsTraversal(t *testing.T) {
	for _, bad := range []string{"..", "../evil", "a/b", "a\\b", ""} {
		if path, _ := findPluginDeb(bad); path != "" {
			t.Errorf("findPluginDeb(%q) returned %q, want empty", bad, path)
		}
	}
}

func TestProtectedKillReason(t *testing.T) {
	if reason := protectedKillReason(1); reason == "" {
		t.Error("expected PID 1 to be protected")
	}
	if reason := protectedKillReason(0); reason == "" {
		t.Error("expected PID 0 to be protected")
	}
	if reason := protectedKillReason(int32(os.Getpid())); reason == "" {
		t.Error("expected own PID to be protected")
	}
}
