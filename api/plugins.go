package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	Name            string                         `json:"name"`
	Version         string                         `json:"version"`
	Description     string                         `json:"description,omitempty"`
	Author          string                         `json:"author,omitempty"`
	Icon            string                         `json:"icon,omitempty"`
	State           plugins.PluginState            `json:"state"`
	Capabilities    []plugins.Capability           `json:"capabilities"`
	Tabs            []plugins.PluginTabSpec        `json:"tabs,omitempty"`
	Widgets         []plugins.PluginWidget         `json:"widgets,omitempty"`
	Settings        []plugins.PluginSettingSection `json:"settings,omitempty"`
	Actions         []plugins.ActionSpec           `json:"actions,omitempty"`
	LastCollectedAt int64                          `json:"last_collected_at,omitempty"`
	LastError       string                         `json:"last_error,omitempty"`
	Manifest        plugins.PluginManifest         `json:"manifest"`
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
			Widgets:      m.Spec.Widgets,
			Settings:     m.Spec.Settings,
			Manifest:     m,
		}

		if inst, found := instanceMap[name]; found {
			dto.State = inst.State
			dto.Capabilities = inst.Capabilities
			dto.Tabs = inst.Tabs
			dto.Actions = inst.Actions
			dto.LastError = inst.LastError
			if len(inst.Widgets) > 0 {
				dto.Widgets = inst.Widgets
			}
			if len(inst.Settings) > 0 {
				dto.Settings = inst.Settings
			}
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

// CatalogModeDTO represents an installation variant or mode for a catalog plugin.
type CatalogModeDTO struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	SubLabel    string `json:"sub_label,omitempty"`
	Description string `json:"description"`
	PackageName string `json:"package_name"`
	DebFilename string `json:"deb_filename,omitempty"`
	AptCommand  string `json:"apt_command"`
	ProcessName string `json:"process_name,omitempty"`
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
	UpdateStatus     string               `json:"update_status,omitempty"`     // "up_to_date", "update_available", "dev_ahead", "not_installed"
	UpdateType       string               `json:"update_type,omitempty"`       // "patch", "minor", "major", "dev", "equal"
	CapabilitiesDiff []string             `json:"capabilities_diff,omitempty"` // newly added capabilities in catalog version
	HasDeb           bool                 `json:"has_deb"`
	DebFilename      string               `json:"deb_filename,omitempty"`
	DownloadURL      string               `json:"download_url,omitempty"`
	AptCommand       string               `json:"apt_command,omitempty"`
	Changelog        string               `json:"changelog,omitempty"`
	Modes            []CatalogModeDTO     `json:"modes,omitempty"`
}

// GetDefaultCatalog returns the list of default plugin catalog entries.
func GetDefaultCatalog() []CatalogPluginDTO {
	cp := make([]CatalogPluginDTO, len(defaultCatalogEntries))
	copy(cp, defaultCatalogEntries)
	return cp
}

var defaultCatalogEntries = []CatalogPluginDTO{
	{
		Name:         "docker-monitor",
		Title:        "Containers",
		ProcessName:  "m3tal-docker",
		PackageName:  "m3tal-plugin-docker-monitor",
		Category:     "Container Management",
		Description:  "Docker container fleet monitor, log inspector, container lifecycle controls, and image pruning.",
		Capabilities: []plugins.Capability{"collector", "actions", "ui_view", "ui_widget", "settings"},
		Version:      "",
		Status:       "planned",
		HasDeb:       false,
		AptCommand:   "sudo apt install -y m3tal-plugin-docker-monitor",
	},
	{
		Name:         "compose",
		Title:        "Compose Editor",
		ProcessName:  "m3tal-compose",
		PackageName:  "m3tal-plugin-compose",
		Category:     "Container Management",
		Description:  "Docker Compose visual editor, syntax validator, and multi-container stack lifecycle manager.",
		Capabilities: []plugins.Capability{"collector", "actions", "ui_view"},
		Version:      "",
		Status:       "planned",
		HasDeb:       false,
		AptCommand:   "sudo apt install -y m3tal-plugin-compose",
	},
	{
		Name:         "mesh",
		Title:        "Cluster & Mesh Federation",
		ProcessName:  "m3tal-mesh",
		PackageName:  "m3tal-plugin-mesh-cluster",
		Category:     "Networking & Fleet",
		Description:  "Local network discovery, multi-node clustering, and telemetry federation across M3TAL workstations, servers, and NAS nodes.",
		Capabilities: []plugins.Capability{"collector", "actions", "ui_view"},
		Version:      "",
		Status:       "planned",
		HasDeb:       false,
		AptCommand:   "sudo apt install -y m3tal-plugin-mesh-cluster",
		Modes: []CatalogModeDTO{
			{
				ID:          "cluster",
				Label:       "Cluster",
				SubLabel:    "Full Peer",
				Description: "Full peer (Send + Receive). Broadcasts local machine metrics AND renders the multi-node fleet dashboard. Recommended for primary workstations.",
				PackageName: "m3tal-plugin-mesh-cluster",
				AptCommand:  "sudo apt install -y m3tal-plugin-mesh-cluster",
				ProcessName: "m3tal-mesh",
			},
			{
				ID:          "mesh",
				Label:       "Mesh",
				SubLabel:    "Viewer",
				Description: "Reader & dashboard UI. Scans LAN to view and monitor other connected M3TAL nodes from your desktop.",
				PackageName: "m3tal-plugin-mesh",
				AptCommand:  "sudo apt install -y m3tal-plugin-mesh",
				ProcessName: "m3tal-mesh",
			},
			{
				ID:          "node",
				Label:       "Node",
				SubLabel:    "Sender",
				Description: "Headless sender daemon. Broadcasts CPU, GPU, and disk metrics without running a GUI. Ideal for servers & NAS.",
				PackageName: "m3tal-plugin-mesh-node",
				AptCommand:  "sudo apt install -y m3tal-plugin-mesh-node",
				ProcessName: "m3tal-mesh-node",
			},
		},
	},
	{
		Name:         "stack-orchestrator",
		Title:        "Compose Stacks (Template DB)",
		ProcessName:  "m3tal-stacks",
		PackageName:  "m3tal-plugin-stacks",
		Category:     "Compose Stacks",
		Description:  "Curated database of verified Docker Compose templates, argument presets, and 1-click community stacks.",
		Capabilities: []plugins.Capability{"orchestrator", "actions", "ui_view"},
		Version:      "",
		Status:       "planned",
		AptCommand:   "sudo apt install -y m3tal-plugin-stacks",
	},
	{
		Name:         "ai-engine",
		Title:        "Local AI & LLM",
		ProcessName:  "m3tal-ai",
		PackageName:  "m3tal-plugin-ai",
		Category:     "Local AI & LLM",
		Description:  "Local LLM inference manager (Ollama, vLLM), GPU VRAM stats, asynchronous job queue, and model catalog.",
		Capabilities: []plugins.Capability{"inference_runner", "job_queue", "ui_view"},
		Version:      "",
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
		Version:      "",
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
		Version:      "",
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
		Version:      "",
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
		Version:      "",
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
		Version:      "",
		Status:       "planned",
		AptCommand:   "sudo apt install -y m3tal-plugin-bench",
	},
}

type pluginMeta struct {
	ProcessName string
	PackageName string
	FolderName  string
}

var pluginMetadataMap = map[string]pluginMeta{
	"docker-monitor":     {ProcessName: "m3tal-docker", PackageName: "m3tal-plugin-docker-monitor", FolderName: "m3tal-docker"},
	"docker":             {ProcessName: "m3tal-docker", PackageName: "m3tal-plugin-docker-monitor", FolderName: "m3tal-docker"},
	"m3tal-docker":       {ProcessName: "m3tal-docker", PackageName: "m3tal-plugin-docker-monitor", FolderName: "m3tal-docker"},
	"compose":            {ProcessName: "m3tal-compose", PackageName: "m3tal-plugin-compose", FolderName: "m3tal-compose"},
	"m3tal-compose":      {ProcessName: "m3tal-compose", PackageName: "m3tal-plugin-compose", FolderName: "m3tal-compose"},
	"mesh":               {ProcessName: "m3tal-mesh", PackageName: "m3tal-plugin-mesh-cluster", FolderName: "m3tal-mesh"},
	"m3tal-mesh":         {ProcessName: "m3tal-mesh", PackageName: "m3tal-plugin-mesh", FolderName: "m3tal-mesh"},
	"m3tal-mesh-node":    {ProcessName: "m3tal-mesh-node", PackageName: "m3tal-plugin-mesh-node", FolderName: "m3tal-mesh-node"},
	"m3tal-mesh-cluster": {ProcessName: "m3tal-mesh", PackageName: "m3tal-plugin-mesh-cluster", FolderName: "m3tal-mesh"},
	"stack-orchestrator": {ProcessName: "m3tal-stacks", PackageName: "m3tal-plugin-stacks", FolderName: "m3tal-stacks"},
	"stacks":             {ProcessName: "m3tal-stacks", PackageName: "m3tal-plugin-stacks", FolderName: "m3tal-stacks"},
	"m3tal-stacks":       {ProcessName: "m3tal-stacks", PackageName: "m3tal-plugin-stacks", FolderName: "m3tal-stacks"},
	"ai-engine":          {ProcessName: "m3tal-ai", PackageName: "m3tal-plugin-ai", FolderName: "m3tal-ai"},
	"ai":                 {ProcessName: "m3tal-ai", PackageName: "m3tal-plugin-ai", FolderName: "m3tal-ai"},
	"m3tal-ai":           {ProcessName: "m3tal-ai", PackageName: "m3tal-plugin-ai", FolderName: "m3tal-ai"},
	"smart-disk":         {ProcessName: "m3tal-smart", PackageName: "m3tal-plugin-smart", FolderName: "m3tal-smart"},
	"smart":              {ProcessName: "m3tal-smart", PackageName: "m3tal-plugin-smart", FolderName: "m3tal-smart"},
	"m3tal-smart":        {ProcessName: "m3tal-smart", PackageName: "m3tal-plugin-smart", FolderName: "m3tal-smart"},
	"traefik-gateway":    {ProcessName: "m3tal-traefik", PackageName: "m3tal-plugin-traefik", FolderName: "m3tal-traefik"},
	"traefik":            {ProcessName: "m3tal-traefik", PackageName: "m3tal-plugin-traefik", FolderName: "m3tal-traefik"},
	"m3tal-traefik":      {ProcessName: "m3tal-traefik", PackageName: "m3tal-plugin-traefik", FolderName: "m3tal-traefik"},
	"gluetun-vpn":        {ProcessName: "m3tal-vpn", PackageName: "m3tal-plugin-vpn", FolderName: "m3tal-vpn"},
	"vpn":                {ProcessName: "m3tal-vpn", PackageName: "m3tal-plugin-vpn", FolderName: "m3tal-vpn"},
	"m3tal-vpn":          {ProcessName: "m3tal-vpn", PackageName: "m3tal-plugin-vpn", FolderName: "m3tal-vpn"},
	"samba-manager":      {ProcessName: "m3tal-samba", PackageName: "m3tal-plugin-samba", FolderName: "m3tal-samba"},
	"samba":              {ProcessName: "m3tal-samba", PackageName: "m3tal-plugin-samba", FolderName: "m3tal-samba"},
	"m3tal-samba":        {ProcessName: "m3tal-samba", PackageName: "m3tal-plugin-samba", FolderName: "m3tal-samba"},
	"system-bench":       {ProcessName: "m3tal-bench", PackageName: "m3tal-plugin-bench", FolderName: "m3tal-bench"},
	"bench":              {ProcessName: "m3tal-bench", PackageName: "m3tal-plugin-bench", FolderName: "m3tal-bench"},
	"m3tal-bench":        {ProcessName: "m3tal-bench", PackageName: "m3tal-plugin-bench", FolderName: "m3tal-bench"},
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

	clean := strings.TrimPrefix(name, "m3tal-")
	nameVariations := []string{
		name,
		clean,
		"m3tal-" + clean,
		"m3tal-plugin-" + name,
		"m3tal-plugin-" + clean,
	}

	if meta, ok := pluginMetadataMap[name]; ok {
		nameVariations = append(nameVariations, meta.FolderName, meta.ProcessName, meta.PackageName, strings.TrimPrefix(meta.PackageName, "m3tal-plugin-"), strings.TrimPrefix(meta.FolderName, "m3tal-"))
	}
	if meta, ok := pluginMetadataMap[clean]; ok {
		nameVariations = append(nameVariations, meta.FolderName, meta.ProcessName, meta.PackageName, strings.TrimPrefix(meta.PackageName, "m3tal-plugin-"), strings.TrimPrefix(meta.FolderName, "m3tal-"))
	}

	for _, dir := range candidateDirs {
		for _, v := range nameVariations {
			// Pattern: m3tal-plugin-<v>_*.deb
			matches, err := filepath.Glob(filepath.Join(dir, fmt.Sprintf("m3tal-plugin-%s_*.deb", v)))
			if err == nil && len(matches) > 0 {
				return matches[0], filepath.Base(matches[0])
			}
			matchesDirect, err := filepath.Glob(filepath.Join(dir, fmt.Sprintf("%s_*.deb", v)))
			if err == nil && len(matchesDirect) > 0 {
				return matchesDirect[0], filepath.Base(matchesDirect[0])
			}
			exact := filepath.Join(dir, fmt.Sprintf("m3tal-plugin-%s.deb", v))
			if _, err := os.Stat(exact); err == nil {
				return exact, filepath.Base(exact)
			}
		}
	}
	return "", ""
}

// findPluginChangelog finds changelog or roadmap documentation for a plugin.
func findPluginChangelog(name string) string {
	home, _ := os.UserHomeDir()
	clean := strings.TrimPrefix(name, "m3tal-")

	candidateDirs := []string{
		fmt.Sprintf("/home/m3tal/apps/m3tal-plugins/m3tal-%s", clean),
		fmt.Sprintf("/home/m3tal/apps/m3tal-plugins/%s", name),
		fmt.Sprintf("/home/m3tal/apps/m3tal-plugins/%s", clean),
		filepath.Join(home, "apps", "m3tal-plugins", "m3tal-"+clean),
		filepath.Join(home, "apps", "m3tal-plugins", name),
		fmt.Sprintf("/usr/lib/m3tal/plugins/%s", name),
		fmt.Sprintf("/usr/lib/m3tal/plugins/m3tal-%s", clean),
		fmt.Sprintf("/usr/share/doc/m3tal-plugin-%s", name),
		fmt.Sprintf("/usr/share/doc/m3tal-plugin-%s", clean),
		filepath.Join(home, ".config", "m3tal", "plugins", name),
		filepath.Join(home, ".config", "m3tal", "plugins", "m3tal-"+clean),
	}

	if meta, ok := pluginMetadataMap[name]; ok {
		candidateDirs = append(candidateDirs,
			filepath.Join("/home/m3tal/apps/m3tal-plugins", meta.FolderName),
			filepath.Join("/home/m3tal/apps/m3tal-plugins", meta.ProcessName),
			filepath.Join(home, "apps", "m3tal-plugins", meta.FolderName),
			filepath.Join("/usr/lib/m3tal/plugins", meta.FolderName),
			filepath.Join("/usr/lib/m3tal/plugins", meta.ProcessName),
			filepath.Join(home, ".config", "m3tal", "plugins", meta.FolderName),
		)
	}
	if meta, ok := pluginMetadataMap[clean]; ok {
		candidateDirs = append(candidateDirs,
			filepath.Join("/home/m3tal/apps/m3tal-plugins", meta.FolderName),
			filepath.Join("/home/m3tal/apps/m3tal-plugins", meta.ProcessName),
			filepath.Join(home, "apps", "m3tal-plugins", meta.FolderName),
			filepath.Join("/usr/lib/m3tal/plugins", meta.FolderName),
			filepath.Join("/usr/lib/m3tal/plugins", meta.ProcessName),
			filepath.Join(home, ".config", "m3tal", "plugins", meta.FolderName),
		)
	}

	// First look for CHANGELOG.md
	for _, dir := range candidateDirs {
		f := filepath.Join(dir, "CHANGELOG.md")
		if data, err := os.ReadFile(f); err == nil && len(data) > 0 {
			return string(data)
		}
	}

	// Fallback to README.md
	for _, dir := range candidateDirs {
		f := filepath.Join(dir, "README.md")
		if data, err := os.ReadFile(f); err == nil && len(data) > 0 {
			return string(data)
		}
	}

	// Fallback to synthesizing specification markdown from default catalog entries
	for _, entry := range defaultCatalogEntries {
		if entry.Name == name || entry.ProcessName == name || strings.TrimPrefix(entry.ProcessName, "m3tal-") == clean {
			var b strings.Builder
			b.WriteString(fmt.Sprintf("# %s (`%s`)\n\n", entry.Title, entry.ProcessName))
			b.WriteString(fmt.Sprintf("**Status**: %s (v%s)\n\n", entry.Status, entry.Version))
			b.WriteString(fmt.Sprintf("**Category**: %s\n\n", entry.Category))
			b.WriteString(fmt.Sprintf("## Description\n%s\n\n", entry.Description))
			if len(entry.Capabilities) > 0 {
				b.WriteString("## Target Capabilities\n")
				for _, c := range entry.Capabilities {
					b.WriteString(fmt.Sprintf("- `%s`\n", c))
				}
				b.WriteString("\n")
			}
			if entry.AptCommand != "" {
				b.WriteString(fmt.Sprintf("## APT Installation Command\n```bash\n%s\n```\n", entry.AptCommand))
			}
			return b.String()
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
		installedMap[strings.TrimPrefix(d.Metadata.Name, "m3tal-")] = d
		if d.Spec.ProcessName != "" {
			installedMap[d.Spec.ProcessName] = d
			installedMap[strings.TrimPrefix(d.Spec.ProcessName, "m3tal-")] = d
		}
	}

	// Try reading dynamic catalog.json from m3tal-plugins
	catalogEntries := defaultCatalogEntries
	home, _ := os.UserHomeDir()
	catalogPaths := []string{
		"/home/m3tal/apps/m3tal-plugins/catalog.json",
		filepath.Join(home, "apps", "m3tal-plugins", "catalog.json"),
		"/etc/m3tal/catalog.json",
		"/usr/share/m3tal/catalog.json",
	}

	for _, cp := range catalogPaths {
		if data, err := os.ReadFile(cp); err == nil && len(data) > 0 {
			var dynamicEntries []CatalogPluginDTO
			if err := json.Unmarshal(data, &dynamicEntries); err == nil && len(dynamicEntries) > 0 {
				catalogEntries = dynamicEntries
				break
			}
		}
	}

	// Collect all package names to query dynamically from APT keyring
	var pkgsToQuery []string
	for _, entry := range catalogEntries {
		if entry.PackageName != "" {
			pkgsToQuery = append(pkgsToQuery, entry.PackageName)
		} else if entry.Name != "" {
			pkgsToQuery = append(pkgsToQuery, "m3tal-plugin-"+entry.Name)
		}
		for _, m := range entry.Modes {
			if m.PackageName != "" {
				pkgsToQuery = append(pkgsToQuery, m.PackageName)
			}
		}
	}

	aptCandidates, _ := QueryAptCandidates(r.Context(), pkgsToQuery, false)

	results := make([]CatalogPluginDTO, len(catalogEntries))
	for i, item := range catalogEntries {
		entry := item

		// Check APT candidate version
		var aptCand *AptPackageCandidate
		if cand, found := aptCandidates[entry.PackageName]; found && cand.CandidateVersion != "" {
			aptCand = &cand
		} else if cand, found := aptCandidates["m3tal-plugin-"+entry.Name]; found && cand.CandidateVersion != "" {
			aptCand = &cand
		}

		if aptCand != nil {
			entry.Version = aptCand.CandidateVersion
			entry.Status = "available"
			entry.HasDeb = true
			entry.DebFilename = fmt.Sprintf("%s_%s_amd64.deb", entry.PackageName, entry.Version)
			entry.DownloadURL = fmt.Sprintf("/api/v2/plugins/%s/download", entry.Name)
		} else {
			debPath, debName := findPluginDeb(entry.Name)
			if debPath != "" {
				entry.HasDeb = true
				entry.DebFilename = debName
				entry.DownloadURL = fmt.Sprintf("/api/v2/plugins/%s/download", entry.Name)
				if entry.Status != "installed" {
					entry.Status = "available"
				}
			}
		}

		// Fall back to default modes if dynamic catalog entry omitted them
		if len(entry.Modes) == 0 {
			for _, def := range defaultCatalogEntries {
				if (def.Name == entry.Name || def.ProcessName == entry.ProcessName) && len(def.Modes) > 0 {
					entry.Modes = def.Modes
					break
				}
			}
		}

		// Update modes with live APT candidate versions
		for mi, m := range entry.Modes {
			if mCand, found := aptCandidates[m.PackageName]; found && mCand.CandidateVersion != "" {
				entry.Modes[mi].DebFilename = fmt.Sprintf("%s_%s_amd64.deb", m.PackageName, mCand.CandidateVersion)
			}
		}

		var activeManifest *plugins.PluginManifest
		clean := strings.TrimPrefix(entry.Name, "m3tal-")
		if inst, found := installedMap[entry.Name]; found {
			activeManifest = &inst
		} else if inst, found := installedMap[clean]; found {
			activeManifest = &inst
		} else if inst, found := installedMap[entry.ProcessName]; found {
			activeManifest = &inst
		}

		if activeManifest != nil {
			entry.Status = "installed"
			entry.InstalledVersion = activeManifest.Metadata.Version

			if entry.Version == "" {
				// Installed locally without an active APT release
				entry.UpdateStatus = "dev_ahead"
				entry.UpdateType = "dev"
			} else {
				cmp, diffType := CompareSemVer(entry.InstalledVersion, entry.Version)
				if cmp == 0 {
					entry.UpdateStatus = "up_to_date"
					entry.UpdateType = "equal"
				} else if cmp < 0 {
					entry.UpdateStatus = "update_available"
					entry.UpdateType = diffType
				} else {
					entry.UpdateStatus = "dev_ahead"
					entry.UpdateType = "dev"
				}
			}

			// Calculate capabilities diff (new capabilities in catalog version)
			instCapsMap := make(map[string]bool)
			for _, c := range activeManifest.Spec.Capabilities {
				instCapsMap[string(c)] = true
			}
			for _, c := range entry.Capabilities {
				if !instCapsMap[string(c)] {
					entry.CapabilitiesDiff = append(entry.CapabilitiesDiff, string(c))
				}
			}
		} else {
			if entry.Version == "" {
				entry.Status = "planned"
			}
			entry.UpdateStatus = "not_installed"
			entry.UpdateType = ""
		}

		changelog := findPluginChangelog(entry.Name)
		if changelog != "" {
			entry.Changelog = changelog
		}

		results[i] = entry
	}

	sendSuccess(w, http.StatusOK, results, nil)
}

// CompareSemVer compares two semantic version strings (v1 vs v2).
// Returns:
//
//	 1 if v1 > v2
//	-1 if v1 < v2
//	 0 if v1 == v2
//
// And diff level: "major", "minor", "patch", or "equal".
func CompareSemVer(v1, v2 string) (int, string) {
	parse := func(s string) (int, int, int) {
		s = strings.TrimPrefix(s, "v")
		parts := strings.Split(s, ".")
		var maj, min, pat int
		if len(parts) > 0 {
			maj, _ = strconv.Atoi(parts[0])
		}
		if len(parts) > 1 {
			min, _ = strconv.Atoi(parts[1])
		}
		if len(parts) > 2 {
			sub := strings.Split(parts[2], "-")[0]
			pat, _ = strconv.Atoi(sub)
		}
		return maj, min, pat
	}

	maj1, min1, pat1 := parse(v1)
	maj2, min2, pat2 := parse(v2)

	if maj1 > maj2 {
		return 1, "major"
	} else if maj1 < maj2 {
		return -1, "major"
	}

	if min1 > min2 {
		return 1, "minor"
	} else if min1 < min2 {
		return -1, "minor"
	}

	if pat1 > pat2 {
		return 1, "patch"
	} else if pat1 < pat2 {
		return -1, "patch"
	}

	return 0, "equal"
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
		Mode     string `json:"mode"`     // optional mode e.g. "cluster", "node", "mesh"
		Package  string `json:"package"`  // optional explicit package name
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.Method == "" {
		req.Method = "sudo"
	}

	searchTargets := []string{name}
	if req.Package != "" {
		searchTargets = append([]string{req.Package}, searchTargets...)
	}
	if req.Mode != "" {
		searchTargets = append([]string{
			req.Mode,
			name + "-" + req.Mode,
			"m3tal-" + name + "-" + req.Mode,
			"m3tal-plugin-" + name + "-" + req.Mode,
		}, searchTargets...)
	}

	var debPath, debName string
	for _, target := range searchTargets {
		dp, dn := findPluginDeb(target)
		if dp != "" {
			debPath, debName = dp, dn
			break
		}
	}

	if debPath == "" {
		sendError(w, http.StatusNotFound, "DEB_NOT_FOUND", "Debian package not found for "+name, nil)
		return
	}

	// Gracefully stop plugin subprocess if currently running before updating binary
	var wasRunning bool
	if existing := h.Manager.GetPlugin(name); existing != nil {
		wasRunning = (existing.State == plugins.StateRunning || existing.State == plugins.StateStarting)
		if wasRunning {
			_ = h.Manager.StopPlugin(name)
			time.Sleep(100 * time.Millisecond)
		}
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

			// If dpkg encountered dependency problems, automatically resolve and configure via apt-get install -f -y
			if strings.Contains(errStr, "dependency problems") || strings.Contains(errStr, "depends on") || strings.Contains(errStr, "unmet dependencies") {
				fixCmd := exec.Command("sudo", "-S", "-p", "", "apt-get", "install", "-f", "-y")
				var fixStdin bytes.Buffer
				if req.Password != "" {
					fixStdin.WriteString(req.Password + "\n")
				}
				fixCmd.Stdin = &fixStdin
				var fixOut, fixErr bytes.Buffer
				fixCmd.Stdout = &fixOut
				fixCmd.Stderr = &fixErr

				if fixCode := fixCmd.Run(); fixCode == nil {
					// Dependencies cleanly auto-resolved from APT keyring!
					goto SudoInstalled
				}
			}

			sendError(w, http.StatusInternalServerError, "INSTALL_FAILED", fmt.Sprintf("sudo dpkg failed: %v: %s", err, errStr), nil)
			return
		}

	SudoInstalled:
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

	// Trigger re-discovery to reload new manifest and binary paths
	_, _ = h.Manager.Discover()

	// If plugin was running before update, restart it with the new version
	if wasRunning {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = h.Manager.StartPlugin(ctx, name)
	}

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
