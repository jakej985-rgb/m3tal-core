package version

import (
	"os"
	"path/filepath"
	"strings"
)

// Current is the default embedded fallback version for M3TAL Core.
var Current = "0.20.4"

// VersionInfo represents canonical version metadata.
type VersionInfo struct {
	Version    string `json:"version"`
	Display    string `json:"display"`
	APIVersion string `json:"api_version"`
	Codename   string `json:"codename,omitempty"`
}

// GetVersion returns the canonical version string (e.g., "0.18.0").
func GetVersion() string {
	// 1. Try reading /etc/m3tal/VERSION
	if data, err := os.ReadFile("/etc/m3tal/VERSION"); err == nil {
		v := strings.TrimSpace(string(data))
		if v != "" {
			return strings.TrimPrefix(v, "v")
		}
	}

	// 2. Try reading local VERSION file relative to working directory
	searchPaths := []string{
		"VERSION",
		"../VERSION",
		"../../VERSION",
		"/usr/share/m3tal/VERSION",
	}

	for _, p := range searchPaths {
		if data, err := os.ReadFile(p); err == nil {
			v := strings.TrimSpace(string(data))
			if v != "" {
				return strings.TrimPrefix(v, "v")
			}
		}
	}

	// 3. Try finding executable directory
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if data, err := os.ReadFile(filepath.Join(dir, "VERSION")); err == nil {
			v := strings.TrimSpace(string(data))
			if v != "" {
				return strings.TrimPrefix(v, "v")
			}
		}
	}

	return Current
}

// GetInfo returns complete version metadata.
func GetInfo() VersionInfo {
	v := GetVersion()
	return VersionInfo{
		Version:    v,
		Display:    "v" + v,
		APIVersion: "v2.0",
		Codename:   "Titanium",
	}
}
