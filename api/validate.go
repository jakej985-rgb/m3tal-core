package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

// pluginDebCandidateDirs returns the trusted directories searched for plugin
// packages. Shared by findPluginDeb and validatePluginDebPath so the install
// path can prove containment.
func pluginDebCandidateDirs() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"/home/m3tal/apps/m3tal-plugins",
		filepath.Join(home, "apps", "m3tal-plugins"),
		"/home/m3tal/apps/m3tal-plugins/dist/packages",
		"/var/lib/m3tal/packages",
	}
}

// validatePluginDebPath verifies a resolved .deb path is a regular file
// inside one of the trusted package directories before it reaches dpkg.
func validatePluginDebPath(path string) (string, error) {
	if path == "" || !strings.HasSuffix(path, ".deb") || strings.ContainsRune(path, 0) {
		return "", fmt.Errorf("invalid package path %q", path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	for _, dir := range pluginDebCandidateDirs() {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		if rel, err := filepath.Rel(absDir, abs); err == nil &&
			rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			if info, err := os.Stat(abs); err == nil && !info.IsDir() {
				return abs, nil
			}
		}
	}
	return "", fmt.Errorf("package path %q is outside trusted package directories", path)
}

// pluginUISubPathRe constrains wildcard subpaths served from plugin UI
// bundles: forward-slash separated segments, no traversal, bounded length.
var pluginUISubPathRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._\-/]{0,255}$`)

// validatePluginUISubPath validates the {name}/* wildcard portion of plugin
// UI requests before it is joined onto the UI directory.
func validatePluginUISubPath(p string) (string, error) {
	p = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(p)), "/")
	if p == "" || strings.Contains(p, "..") || strings.HasPrefix(p, "/") ||
		strings.ContainsRune(p, 0) || !pluginUISubPathRe.MatchString(p) {
		return "", fmt.Errorf("invalid plugin UI path %q", p)
	}
	return p, nil
}
