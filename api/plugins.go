package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/pkg/plugins"
)

// PluginHandlers provides REST endpoints for supervising external plugins.
type PluginHandlers struct {
	Manager *plugins.Manager
}

// NewPluginHandlers creates a handler set backed by the plugin Manager.
func NewPluginHandlers(mgr *plugins.Manager) *PluginHandlers {
	if mgr == nil {
		mgr = plugins.NewManager("v0.2.0")
	}
	return &PluginHandlers{Manager: mgr}
}

// PluginDTO represents the JSON response for a managed plugin.
type PluginDTO struct {
	Name            string                  `json:"name"`
	Version         string                  `json:"version"`
	Description     string                  `json:"description,omitempty"`
	Author          string                  `json:"author,omitempty"`
	Icon            string                  `json:"icon,omitempty"`
	State           plugins.PluginState     `json:"state"`
	Capabilities    []plugins.Capability    `json:"capabilities"`
	Tabs            []plugins.PluginTabSpec `json:"tabs,omitempty"`
	Actions         []plugins.ActionSpec    `json:"actions,omitempty"`
	LastCollectedAt int64                   `json:"last_collected_at,omitempty"`
	LastError       string                  `json:"last_error,omitempty"`
	Manifest        plugins.PluginManifest  `json:"manifest"`
}

// ListPlugins returns all discovered and managed plugins.
// GET /api/v2/plugins
func (h *PluginHandlers) ListPlugins(w http.ResponseWriter, r *http.Request) {
	// Trigger discovery to ensure any newly added plugins in search paths are picked up
	discovered, _ := h.Manager.Discover()

	allInstances := h.Manager.GetAllPlugins()
	instanceMap := make(map[string]*plugins.PluginInstance)
	for _, inst := range allInstances {
		instanceMap[inst.Manifest.Metadata.Name] = inst
	}

	var results []PluginDTO
	for _, m := range discovered {
		name := m.Metadata.Name
		dto := PluginDTO{
			Name:         name,
			Version:      m.Metadata.Version,
			Description:  m.Metadata.Description,
			Author:       m.Metadata.Author,
			Icon:         m.Metadata.Icon,
			State:        plugins.StateStopped,
			Capabilities: m.Spec.Capabilities,
			Manifest:     m,
		}

		if inst, found := instanceMap[name]; found {
			dto.State = inst.State
			dto.Capabilities = inst.Capabilities
			dto.Tabs = inst.Tabs
			dto.Actions = inst.Actions
			dto.LastError = inst.LastError
			if !inst.LastCollectedAt.IsZero() {
				dto.LastCollectedAt = inst.LastCollectedAt.Unix()
			}
		}

		results = append(results, dto)
	}

	if results == nil {
		results = []PluginDTO{}
	}

	sendSuccess(w, http.StatusOK, results, nil)
}

// StartPlugin starts a plugin subprocess.
// POST /api/v2/plugins/{name}/start
func (h *PluginHandlers) StartPlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "INVALID_NAME", "plugin name required", nil)
		return
	}

	// Discover in case it's not yet loaded into manager
	_, _ = h.Manager.Discover()

	err := h.Manager.StartPlugin(r.Context(), name)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PLUGIN_START_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"name":    name,
		"started": true,
	}, nil)
}

// StopPlugin stops a plugin subprocess.
// POST /api/v2/plugins/{name}/stop
func (h *PluginHandlers) StopPlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "INVALID_NAME", "plugin name required", nil)
		return
	}

	err := h.Manager.StopPlugin(name)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PLUGIN_STOP_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"name":    name,
		"stopped": true,
	}, nil)
}

// CollectPlugin requests live telemetry from a running plugin over JSON-RPC 2.0.
// GET /api/v2/plugins/{name}/collect
func (h *PluginHandlers) CollectPlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	p := h.Manager.GetPlugin(name)
	if p == nil {
		sendError(w, http.StatusNotFound, "PLUGIN_NOT_FOUND", "plugin not found: "+name, nil)
		return
	}

	res, err := p.Collect(r.Context())
	if err != nil {
		sendError(w, http.StatusInternalServerError, "COLLECT_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, res, nil)
}

// ExecuteAction executes a declared action on a running plugin.
// POST /api/v2/plugins/{name}/action
func (h *PluginHandlers) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	p := h.Manager.GetPlugin(name)
	if p == nil {
		sendError(w, http.StatusNotFound, "PLUGIN_NOT_FOUND", "plugin not found: "+name, nil)
		return
	}

	var req struct {
		Action string         `json:"action"`
		Target string         `json:"target,omitempty"`
		Args   map[string]any `json:"args,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_BODY", "failed to decode action payload", nil)
		return
	}

	res, err := p.Action(r.Context(), req.Action, req.Target, req.Args)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "ACTION_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, res, nil)
}

// ListTabs returns all dynamic UI tabs declared by running plugins.
// GET /api/v2/plugins/tabs
func (h *PluginHandlers) ListTabs(w http.ResponseWriter, r *http.Request) {
	tabs := h.Manager.GetAllTabs()
	if tabs == nil {
		tabs = []plugins.PluginTabSpec{}
	}
	sendSuccess(w, http.StatusOK, tabs, nil)
}

// CatalogPluginDTO represents a catalog entry with version, changelog, download, and install info.
type CatalogPluginDTO struct {
	Name             string               `json:"name"`
	Title            string               `json:"title"`
	ProcessName      string               `json:"process_name"`
	PackageName      string               `json:"package_name"`
	Category         string               `json:"category"`
	Description      string               `json:"description"`
	Capabilities     []plugins.Capability `json:"capabilities"`
	Status           string               `json:"status"` // "available", "installed", "planned"
	Version          string               `json:"version"`
	InstalledVersion string               `json:"installed_version,omitempty"`
	HasDeb           bool                 `json:"has_deb"`
	DebFilename      string               `json:"deb_filename,omitempty"`
	DownloadURL      string               `json:"download_url,omitempty"`
	AptCommand       string               `json:"apt_command,omitempty"`
	Changelog        string               `json:"changelog,omitempty"`
}

var defaultCatalogEntries = []CatalogPluginDTO{
	{
		Name:         "docker-monitor",
		Title:        "Containers",
		ProcessName:  "m3tal-docker",
		PackageName:  "m3tal-plugin-docker-monitor",
		Category:     "Container Management",
		Description:  "Docker container fleet monitor, log inspector, container lifecycle controls, and image pruning.",
		Capabilities: []plugins.Capability{"collector", "actions", "ui_view", "websocket_stream"},
		Version:      "0.1.0",
		Status:       "available",
		AptCommand:   "sudo apt install -y m3tal-plugin-docker-monitor",
	},
	{
		Name:         "stack-orchestrator",
		Title:        "Compose Stacks",
		ProcessName:  "m3tal-stacks",
		PackageName:  "m3tal-plugin-stacks",
		Category:     "Compose Stacks",
		Description:  "Multi-service compose stack orchestration, live health checks, and lifecycle deployment.",
		Capabilities: []plugins.Capability{"orchestrator", "actions", "ui_view"},
		Version:      "0.1.0",
		Status:       "planned",
		AptCommand:   "sudo apt install -y m3tal-plugin-stacks",
	},
	{
		Name:         "ai-engine",
		Title:        "Local AI & LLM",
		ProcessName:  "m3tal-ai",
		PackageName:  "m3tal-plugin-ai",
		Category:     "Local AI & LLM",
		Description:  "Local LLM inference manager (Ollama, vLLM), job queue, and model catalog.",
		Capabilities: []plugins.Capability{"inference_runner", "job_queue", "ui_view"},
		Version:      "0.1.0",
		Status:       "planned",
		AptCommand:   "sudo apt install -y m3tal-plugin-ai",
	},
	{
		Name:         "smart-disk",
		Title:        "Smart Disk Telemetry",
		ProcessName:  "m3tal-smart",
		PackageName:  "m3tal-plugin-smart",
		Category:     "Hardware Diagnostics",
		Description:  "Drive health telemetry, NVMe wear life, S.M.A.R.T. self-tests, and temperature history.",
		Capabilities: []plugins.Capability{"collector", "ui_widget", "alerts"},
		Version:      "0.1.0",
		Status:       "planned",
		AptCommand:   "sudo apt install -y m3tal-plugin-smart",
	},
	{
		Name:         "traefik-gateway",
		Title:        "Traefik Gateway",
		ProcessName:  "m3tal-traefik",
		PackageName:  "m3tal-plugin-traefik",
		Category:     "Networking & Proxy",
		Description:  "Traefik v3 reverse proxy, automated Let's Encrypt SSL/TLS certificates, and dynamic routing.",
		Capabilities: []plugins.Capability{"proxy_generator", "route_provider"},
		Version:      "0.1.0",
		Status:       "planned",
		AptCommand:   "sudo apt install -y m3tal-plugin-traefik",
	},
	{
		Name:         "gluetun-vpn",
		Title:        "VPN Tunnel",
		ProcessName:  "m3tal-vpn",
		PackageName:  "m3tal-plugin-vpn",
		Category:     "Network Privacy",
		Description:  "WireGuard/OpenVPN tunnel connectivity with container egress routing and leak protection.",
		Capabilities: []plugins.Capability{"network_tunnel", "status_checker"},
		Version:      "0.1.0",
		Status:       "planned",
		AptCommand:   "sudo apt install -y m3tal-plugin-vpn",
	},
	{
		Name:         "samba-manager",
		Title:        "Samba Manager",
		ProcessName:  "m3tal-samba",
		PackageName:  "m3tal-plugin-samba",
		Category:     "File Sharing",
		Description:  "Graphical Samba/NFS share manager, guest permissions, user credentials, and active mounts.",
		Capabilities: []plugins.Capability{"actions", "ui_view", "system_admin"},
		Version:      "0.1.0",
		Status:       "planned",
		AptCommand:   "sudo apt install -y m3tal-plugin-samba",
	},
	{
		Name:         "system-bench",
		Title:        "System Benchmarks",
		ProcessName:  "m3tal-bench",
		PackageName:  "m3tal-plugin-bench",
		Category:     "Diagnostics",
		Description:  "One-click benchmarking for CPU multi-threading, GPU compute, and disk I/O throughput.",
		Capabilities: []plugins.Capability{"benchmark_runner", "ui_view"},
		Version:      "0.1.0",
		Status:       "planned",
		AptCommand:   "sudo apt install -y m3tal-plugin-bench",
	},
}

// findPluginDeb searches candidate locations for a built .deb package.
func findPluginDeb(name string) (string, string) {
	home, _ := os.UserHomeDir()
	candidateDirs := []string{
		"/home/m3tal/apps/m3tal-plugins",
		filepath.Join(home, "apps", "m3tal-plugins"),
		"/home/m3tal/apps/m3tal-plugins/dist/packages",
		"/var/lib/m3tal/packages",
		"./plugins",
	}

	for _, dir := range candidateDirs {
		matches, err := filepath.Glob(filepath.Join(dir, fmt.Sprintf("m3tal-plugin-%s_*.deb", name)))
		if err == nil && len(matches) > 0 {
			return matches[0], filepath.Base(matches[0])
		}
		exact := filepath.Join(dir, fmt.Sprintf("m3tal-plugin-%s.deb", name))
		if _, err := os.Stat(exact); err == nil {
			return exact, filepath.Base(exact)
		}
	}
	return "", ""
}

// findPluginChangelog finds changelog markdown for a plugin.
func findPluginChangelog(name string) string {
	home, _ := os.UserHomeDir()
	candidateFiles := []string{
		fmt.Sprintf("/home/m3tal/apps/m3tal-plugins/%s/CHANGELOG.md", name),
		filepath.Join(home, "apps", "m3tal-plugins", name, "CHANGELOG.md"),
		fmt.Sprintf("/usr/lib/m3tal/plugins/%s/CHANGELOG.md", name),
		fmt.Sprintf("/usr/share/doc/m3tal-plugin-%s/CHANGELOG.md", name),
		filepath.Join(home, ".config", "m3tal", "plugins", name, "CHANGELOG.md"),
	}
	for _, f := range candidateFiles {
		if data, err := os.ReadFile(f); err == nil && len(data) > 0 {
			return string(data)
		}
	}
	return ""
}

// GetPluginCatalog returns all plugins in the catalog with versions, deb, and changelog.
// GET /api/v2/plugins/catalog
func (h *PluginHandlers) GetPluginCatalog(w http.ResponseWriter, r *http.Request) {
	discovered, _ := h.Manager.Discover()
	installedMap := make(map[string]plugins.PluginManifest)
	for _, d := range discovered {
		installedMap[d.Metadata.Name] = d
	}

	results := make([]CatalogPluginDTO, len(defaultCatalogEntries))
	for i, item := range defaultCatalogEntries {
		entry := item
		debPath, debName := findPluginDeb(entry.Name)
		if debPath != "" {
			entry.HasDeb = true
			entry.DebFilename = debName
			entry.DownloadURL = fmt.Sprintf("/api/v2/plugins/%s/download", entry.Name)
			if entry.Status != "installed" {
				entry.Status = "available"
			}
		}

		if inst, found := installedMap[entry.Name]; found {
			entry.Status = "installed"
			entry.InstalledVersion = inst.Metadata.Version
		}

		changelog := findPluginChangelog(entry.Name)
		if changelog != "" {
			entry.Changelog = changelog
		}

		results[i] = entry
	}

	sendSuccess(w, http.StatusOK, results, nil)
}

// GetPluginChangelog returns the changelog for a specific plugin.
// GET /api/v2/plugins/{name}/changelog
func (h *PluginHandlers) GetPluginChangelog(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "INVALID_NAME", "plugin name required", nil)
		return
	}

	cl := findPluginChangelog(name)
	if cl == "" {
		sendError(w, http.StatusNotFound, "NOT_FOUND", "changelog not available for "+name, nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]string{
		"name":      name,
		"changelog": cl,
	}, nil)
}

// DownloadPluginDeb streams the compiled .deb file.
// GET /api/v2/plugins/{name}/download
func (h *PluginHandlers) DownloadPluginDeb(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "INVALID_NAME", "plugin name required", nil)
		return
	}

	debPath, debName := findPluginDeb(name)
	if debPath == "" {
		sendError(w, http.StatusNotFound, "DEB_NOT_FOUND", "Debian package not found for "+name, nil)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", debName))
	w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
	http.ServeFile(w, r, debPath)
}

// InstallPlugin installs a plugin package via sudo confirmation or user profile extraction.
// POST /api/v2/plugins/{name}/install
func (h *PluginHandlers) InstallPlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "INVALID_NAME", "plugin name required", nil)
		return
	}

	var req struct {
		Method   string `json:"method"`   // "sudo" (default) or "user"
		Password string `json:"password"` // sudo password
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.Method == "" {
		req.Method = "sudo"
	}

	debPath, debName := findPluginDeb(name)
	if debPath == "" {
		sendError(w, http.StatusNotFound, "DEB_NOT_FOUND", "Debian package not found for "+name, nil)
		return
	}

	if req.Method == "sudo" {
		// Run sudo -S dpkg -i <debPath>
		cmd := exec.Command("sudo", "-S", "-p", "", "dpkg", "-i", debPath)
		var stdinBuf bytes.Buffer
		if req.Password != "" {
			stdinBuf.WriteString(req.Password + "\n")
		}
		cmd.Stdin = &stdinBuf

		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		err := cmd.Run()
		if err != nil {
			errStr := stderrBuf.String()
			if strings.Contains(strings.ToLower(errStr), "incorrect password") ||
				strings.Contains(strings.ToLower(errStr), "a password is required") ||
				strings.Contains(strings.ToLower(errStr), "sorry, try again") {
				sendError(w, http.StatusUnauthorized, "AUTH_FAILED", "Sudo authentication failed. Incorrect password.", nil)
				return
			}
			sendError(w, http.StatusInternalServerError, "INSTALL_FAILED", fmt.Sprintf("sudo dpkg failed: %v: %s", err, errStr), nil)
			return
		}
	} else {
		// User space installation to ~/.config/m3tal/plugins/<name>
		home, err := os.UserHomeDir()
		if err != nil {
			sendError(w, http.StatusInternalServerError, "HOME_NOT_FOUND", err.Error(), nil)
			return
		}

		targetPluginDir := filepath.Join(home, ".config", "m3tal", "plugins", name)
		_ = os.RemoveAll(targetPluginDir)
		_ = os.MkdirAll(filepath.Dir(targetPluginDir), 0755)

		// Extract using dpkg-deb -x
		tempExtract, err := os.MkdirTemp("", "m3tal-extract-*")
		if err != nil {
			sendError(w, http.StatusInternalServerError, "TEMP_FAILED", err.Error(), nil)
			return
		}
		defer os.RemoveAll(tempExtract)

		extractCmd := exec.Command("dpkg-deb", "-x", debPath, tempExtract)
		if out, err := extractCmd.CombinedOutput(); err != nil {
			sendError(w, http.StatusInternalServerError, "EXTRACT_FAILED", string(out), nil)
			return
		}

		sourceDir := filepath.Join(tempExtract, "usr", "lib", "m3tal", "plugins", name)
		if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
			sendError(w, http.StatusInternalServerError, "PAYLOAD_NOT_FOUND", "Extracted package does not contain plugin directory", nil)
			return
		}

		// Move to target
		cpCmd := exec.Command("cp", "-r", sourceDir, targetPluginDir)
		if out, err := cpCmd.CombinedOutput(); err != nil {
			sendError(w, http.StatusInternalServerError, "COPY_FAILED", string(out), nil)
			return
		}

		// Ensure binaries are executable
		binDir := filepath.Join(targetPluginDir, "bin")
		if entries, err := os.ReadDir(binDir); err == nil {
			for _, e := range entries {
				_ = os.Chmod(filepath.Join(binDir, e.Name()), 0755)
			}
		}
	}

	// Trigger discovery
	_, _ = h.Manager.Discover()

	sendSuccess(w, http.StatusOK, map[string]any{
		"name":      name,
		"installed": true,
		"package":   debName,
		"method":    req.Method,
	}, nil)
}

// UninstallPlugin uninstalls a plugin.
// POST /api/v2/plugins/{name}/uninstall
func (h *PluginHandlers) UninstallPlugin(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "INVALID_NAME", "plugin name required", nil)
		return
	}

	var req struct {
		Method   string `json:"method"`
		Password string `json:"password"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	// Stop plugin if running
	_ = h.Manager.StopPlugin(name)

	home, _ := os.UserHomeDir()
	userPluginDir := filepath.Join(home, ".config", "m3tal", "plugins", name)
	_ = os.RemoveAll(userPluginDir)

	if req.Method == "sudo" || req.Password != "" {
		// Run sudo dpkg -r
		cmd := exec.Command("sudo", "-S", "-p", "", "dpkg", "-r", fmt.Sprintf("m3tal-plugin-%s", name))
		var stdinBuf bytes.Buffer
		stdinBuf.WriteString(req.Password + "\n")
		cmd.Stdin = &stdinBuf
		_ = cmd.Run()
	}

	// Re-discover
	_, _ = h.Manager.Discover()

	sendSuccess(w, http.StatusOK, map[string]any{
		"name":        name,
		"uninstalled": true,
	}, nil)
}

