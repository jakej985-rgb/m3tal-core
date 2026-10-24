package api

import (
	"encoding/json"
	"net/http"

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
