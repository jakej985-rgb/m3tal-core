package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/pkg/plugins"
)

// CoreSettings represents general M3TAL host settings.
type CoreSettings struct {
	NodeName     string `json:"node_name"`
	PollInterval int    `json:"poll_interval"` // In seconds
	Theme        string `json:"theme"`         // "dark", "matrix", etc.
}

func getSettingsDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "/etc/m3tal"
	}
	return filepath.Join(home, ".config", "m3tal")
}

func getCoreSettingsFile() string {
	return filepath.Join(getSettingsDir(), "core_settings.json")
}

func loadCoreSettings() CoreSettings {
	defaults := CoreSettings{
		NodeName:     "m3tal-host",
		PollInterval: 5,
		Theme:        "dark",
	}

	hostname, _ := os.Hostname()
	if hostname != "" {
		defaults.NodeName = hostname
	}

	file := getCoreSettingsFile()
	data, err := os.ReadFile(file)
	if err != nil {
		return defaults
	}

	var s CoreSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return defaults
	}
	if s.NodeName == "" {
		s.NodeName = defaults.NodeName
	}
	if s.PollInterval <= 0 {
		s.PollInterval = 5
	}
	if s.Theme == "" {
		s.Theme = "dark"
	}
	return s
}

func saveCoreSettings(s CoreSettings) error {
	dir := getSettingsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getCoreSettingsFile(), data, 0644)
}

func getPluginConfigFile(pluginName string) string {
	return filepath.Join(getSettingsDir(), "plugins", pluginName, "config.json")
}

func loadPluginConfig(pluginName string) map[string]any {
	file := getPluginConfigFile(pluginName)
	data, err := os.ReadFile(file)
	if err != nil {
		return make(map[string]any)
	}
	var res map[string]any
	if err := json.Unmarshal(data, &res); err != nil {
		return make(map[string]any)
	}
	return res
}

func savePluginConfig(pluginName string, config map[string]any) error {
	dir := filepath.Join(getSettingsDir(), "plugins", pluginName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getPluginConfigFile(pluginName), data, 0644)
}

// Preset settings categories and the plugins that belong to them. Plugins may
// override this by declaring `category:` on their settings sections.
var presetSettingsCategories = []struct {
	ID      string
	Title   string
	Icon    string
	Plugins []string
}{
	{ID: "containers", Title: "Containers", Icon: "IconPlugins", Plugins: []string{
		"docker-monitor", "m3tal-docker", "docker", "compose", "m3tal-compose",
		"stack-orchestrator", "stacks", "m3tal-stacks",
	}},
	{ID: "network", Title: "Network & Proxy", Icon: "IconServer", Plugins: []string{
		"mesh", "m3tal-mesh", "mesh-node", "m3tal-mesh-node",
		"gluetun-vpn", "vpn", "m3tal-vpn",
		"traefik-gateway", "traefik", "m3tal-traefik",
	}},
	{ID: "file-sharing", Title: "File Sharing", Icon: "IconDocument", Plugins: []string{
		"samba-nfs", "m3tal-samba-nfs",
	}},
	{ID: "ai", Title: "AI & LLM", Icon: "IconPlugins", Plugins: []string{
		"ai-engine", "ai", "m3tal-ai",
	}},
	{ID: "hardware", Title: "Hardware & Diagnostics", Icon: "IconSettings", Plugins: []string{
		"smart-disk", "smart", "m3tal-smart",
		"system-bench", "bench", "m3tal-bench",
	}},
}

// presetPluginCategory returns the preset category id for a plugin name, or ""
// when the plugin is not part of a preset group.
func presetPluginCategory(pluginName string) string {
	name := strings.TrimPrefix(strings.TrimSpace(pluginName), "m3tal-")
	name = strings.TrimPrefix(name, "plugin-")
	for _, cat := range presetSettingsCategories {
		for _, p := range cat.Plugins {
			pName := strings.TrimPrefix(p, "m3tal-")
			if name == pName || pluginName == p {
				return cat.ID
			}
		}
	}
	return ""
}

// GetAllSettings returns Core settings + all registered plugin settings sections and current values.
// GET /api/v2/settings
func (h *PluginHandlers) GetAllSettings(w http.ResponseWriter, r *http.Request) {
	core := loadCoreSettings()
	sections := h.Manager.GetAllSettings()
	if sections == nil {
		sections = []plugins.PluginSettingSection{}
	}

	// Apply preset category fallback for plugins that don't self-declare one.
	for i := range sections {
		if sections[i].Category == "" {
			sections[i].Category = presetPluginCategory(sections[i].Plugin)
		}
	}

	categories := make([]map[string]string, 0, len(presetSettingsCategories)+1)
	for _, cat := range presetSettingsCategories {
		categories = append(categories, map[string]string{
			"id": cat.ID, "title": cat.Title, "icon": cat.Icon,
		})
	}
	categories = append(categories, map[string]string{
		"id": "other", "title": "Other Plugins", "icon": "IconPlugins",
	})

	pluginConfigs := make(map[string]map[string]any)
	for _, sec := range sections {
		pName := sec.Plugin
		if pName != "" && pluginConfigs[pName] == nil {
			pluginConfigs[pName] = loadPluginConfig(pName)
		}
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"core":           core,
		"sections":       sections,
		"categories":     categories,
		"plugin_configs": pluginConfigs,
	}, nil)
}

// SaveCoreSettings updates host-level settings.
// POST /api/v2/settings/core
func (h *PluginHandlers) SaveCoreSettings(w http.ResponseWriter, r *http.Request) {
	var s CoreSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_BODY", "failed to parse core settings payload", nil)
		return
	}

	if err := saveCoreSettings(s); err != nil {
		sendError(w, http.StatusInternalServerError, "SAVE_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"message": "Core settings saved successfully",
		"core":    s,
	}, nil)
}

// GetPluginSettings returns settings and stored values for a single plugin.
// GET /api/v2/plugins/{name}/settings
func (h *PluginHandlers) GetPluginSettings(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	p := h.Manager.GetPlugin(name)
	var sections []plugins.PluginSettingSection
	if p != nil {
		if len(p.Settings) > 0 {
			sections = p.Settings
		} else {
			sections = p.Manifest.Spec.Settings
		}
	} else {
		// Try discovering
		disc, _ := h.Manager.Discover()
		for _, m := range disc {
			if m.Metadata.Name == name || m.Metadata.Name == strings.TrimPrefix(name, "m3tal-") {
				sections = m.Spec.Settings
				break
			}
		}
	}

	for i := range sections {
		if sections[i].Category == "" {
			sections[i].Category = presetPluginCategory(sections[i].Plugin)
		}
		if sections[i].Plugin == "" {
			sections[i].Plugin = name
		}
	}

	cfg := loadPluginConfig(name)

	sendSuccess(w, http.StatusOK, map[string]any{
		"plugin":   name,
		"sections": sections,
		"config":   cfg,
	}, nil)
}

// SavePluginSettings updates stored values for a plugin and notifies the daemon.
// POST /api/v2/plugins/{name}/settings
func (h *PluginHandlers) SavePluginSettings(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var req struct {
		Config map[string]any `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_BODY", "failed to parse plugin settings payload", nil)
		return
	}

	if err := savePluginConfig(name, req.Config); err != nil {
		sendError(w, http.StatusInternalServerError, "SAVE_FAILED", err.Error(), nil)
		return
	}

	// If plugin is running, notify via action set_settings
	p := h.Manager.GetPlugin(name)
	if p != nil && p.State == plugins.StateRunning {
		_, _ = p.Action(r.Context(), "set_settings", "", req.Config)
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"message": fmt.Sprintf("Settings for %s saved successfully", name),
		"plugin":  name,
		"config":  req.Config,
	}, nil)
}
