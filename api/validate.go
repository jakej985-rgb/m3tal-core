package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
)

// pluginNameRe constrains plugin identifiers to a single safe path segment.
// Names containing '/', '\', or '..' are rejected, which prevents chi URL
// parameters from escaping plugin directories via filepath.Join.
var pluginNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// sanitizePluginName validates a plugin identifier supplied by a client.
func sanitizePluginName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." ||
		strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") ||
		!pluginNameRe.MatchString(name) {
		return "", fmt.Errorf("invalid plugin name %q", name)
	}
	return name, nil
}

// pluginNameParam extracts and validates the {name} URL parameter, writing a
// 400 response and returning ok=false when invalid.
func pluginNameParam(w http.ResponseWriter, r *http.Request) (string, bool) {
	name, err := sanitizePluginName(chi.URLParam(r, "name"))
	if err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_NAME", "invalid plugin name", nil)
		return "", false
	}
	return name, true
}
