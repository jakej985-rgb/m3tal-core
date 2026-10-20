package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/core/plugins"
	"github.com/jakej985-rgb/m3tal-core/core/state"
	"github.com/jakej985-rgb/m3tal-core/pkg/system"
)

// PluginHandlers provides endpoints for viewing loaded plugins.
type PluginHandlers struct {
	registry *plugin.Registry
	db       *state.Store
}

// NewPluginHandlers creates a handler set with a lazily-loaded plugin registry and SQLite state.
func NewPluginHandlers(db *state.Store) *PluginHandlers {
	return &PluginHandlers{db: db}
}

// ensureLoaded lazily loads the plugin registry on first access.
func (h *PluginHandlers) ensureLoaded() *plugin.Registry {
	if h.registry != nil {
		return h.registry
	}

	dirs := system.GetPluginDirs()
	reg, err := plugin.LoadAll(dirs...)
	if err != nil {
		h.registry = &plugin.Registry{}
		return h.registry
	}
	h.registry = reg
	return h.registry
}

// ListPlugins returns all loaded plugins across all kinds.
// GET /api/v2/plugins
func (h *PluginHandlers) ListPlugins(w http.ResponseWriter, r *http.Request) {
	reg := h.ensureLoaded()

	sendSuccess(w, http.StatusOK, map[string]any{
		"summary":    reg.Summary(),
		"routes":     reg.ListRoutes(),
		"stacks":     reg.ListStacks(),
		"middleware": reg.ListMiddlewares(),
	}, nil)
}

// ListPluginsByKind returns plugins filtered by kind.
// GET /api/v2/plugins/{kind}
func (h *PluginHandlers) ListPluginsByKind(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")
	reg := h.ensureLoaded()

	switch strings.ToLower(kind) {
	case "routes", "route":
		sendSuccess(w, http.StatusOK, reg.ListRoutes(), nil)
	case "stacks", "stack":
		sendSuccess(w, http.StatusOK, reg.ListStacks(), nil)
	case "middleware", "middlewares":
		sendSuccess(w, http.StatusOK, reg.ListMiddlewares(), nil)
	case "services", "service":
		sendSuccess(w, http.StatusOK, reg.ListServices(), nil)
	default:
		if p := reg.GetRoute(kind); p != nil {
			sendSuccess(w, http.StatusOK, p, nil)
			return
		}
		if p := reg.GetStack(kind); p != nil {
			sendSuccess(w, http.StatusOK, p, nil)
			return
		}
		if p := reg.GetMiddleware(kind); p != nil {
			sendSuccess(w, http.StatusOK, p, nil)
			return
		}
		if p := reg.GetService(kind); p != nil {
			sendSuccess(w, http.StatusOK, p, nil)
			return
		}
		sendError(w, http.StatusNotFound, "PLUGIN_NOT_FOUND", "plugin not found: "+kind, nil)
	}
}

// Reload forces a reload of the plugin registry from disk.
// POST /api/v2/plugins/reload
func (h *PluginHandlers) Reload(w http.ResponseWriter, r *http.Request) {
	h.registry = nil
	reg := h.ensureLoaded()
	sendSuccess(w, http.StatusOK, map[string]any{
		"reloaded": true,
		"summary":  reg.Summary(),
	}, nil)
}

// Enable renames a plugin file ending in `.disabled` by removing the suffix.
// POST /api/v2/plugins/enable
func (h *PluginHandlers) Enable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	reg := h.ensureLoaded()
	var path string
	switch req.Kind {
	case "Route":
		p := reg.GetRoute(req.Name)
		if p != nil {
			path = p.SourcePath
		}
	case "Stack":
		p := reg.GetStack(req.Name)
		if p != nil {
			path = p.SourcePath
		}
	case "Middleware":
		p := reg.GetMiddleware(req.Name)
		if p != nil {
			path = p.SourcePath
		}
	default:
		if p := reg.GetRoute(req.Name); p != nil {
			path = p.SourcePath
		} else if p := reg.GetStack(req.Name); p != nil {
			path = p.SourcePath
		} else if p := reg.GetMiddleware(req.Name); p != nil {
			path = p.SourcePath
		}
	}

	if path == "" {
		sendError(w, http.StatusNotFound, "PLUGIN_NOT_FOUND", "plugin not found: "+req.Name, nil)
		return
	}

	p, err := plugin.LoadPlugin(path)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PLUGIN_LOAD_FAILED", "failed to load plugin: "+err.Error(), nil)
		return
	}

	header := plugin.PluginHeader{
		Name:         p.GetName(),
		Kind:         p.Kind,
		Enabled:      true,
		Provides:     p.Provides,
		Requires:     p.Requires,
		DependsOn:    p.DependsOn,
		Dependencies: p.Dependencies,
	}

	warnings := reg.GetWarningsForHeader(header)
	if len(warnings) > 0 {
		sendError(w, http.StatusBadRequest, "UNSATISFIED_DEPENDENCIES", "unsatisfied dependencies", map[string]any{"warnings": warnings})
		return
	}

	mgr := plugin.NewStateManager(h.db)
	err = mgr.SetPluginEnabled(p, true)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PLUGIN_ENABLE_FAILED", "failed to enable: "+err.Error(), nil)
		return
	}

	h.registry = nil
	sendSuccess(w, http.StatusOK, map[string]any{"enabled": true, "path": p.SourcePath}, nil)
}

// Disable renames a plugin file to append `.disabled`.
// POST /api/v2/plugins/disable
func (h *PluginHandlers) Disable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	reg := h.ensureLoaded()
	var path string
	switch req.Kind {
	case "Route":
		p := reg.GetRoute(req.Name)
		if p != nil {
			path = p.SourcePath
		}
	case "Stack":
		p := reg.GetStack(req.Name)
		if p != nil {
			path = p.SourcePath
		}
	case "Middleware":
		p := reg.GetMiddleware(req.Name)
		if p != nil {
			path = p.SourcePath
		}
	default:
		if p := reg.GetRoute(req.Name); p != nil {
			path = p.SourcePath
		} else if p := reg.GetStack(req.Name); p != nil {
			path = p.SourcePath
		} else if p := reg.GetMiddleware(req.Name); p != nil {
			path = p.SourcePath
		}
	}

	if path == "" {
		sendError(w, http.StatusNotFound, "PLUGIN_NOT_FOUND", "plugin not found: "+req.Name, nil)
		return
	}

	p, err := plugin.LoadPlugin(path)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PLUGIN_LOAD_FAILED", "failed to load plugin: "+err.Error(), nil)
		return
	}

	mgr := plugin.NewStateManager(h.db)
	err = mgr.SetPluginEnabled(p, false)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PLUGIN_DISABLE_FAILED", "failed to disable: "+err.Error(), nil)
		return
	}

	h.registry = nil
	sendSuccess(w, http.StatusOK, map[string]any{"disabled": true, "path": p.SourcePath}, nil)
}

// Sync writes the dynamic Traefik configuration file
// POST /api/v2/plugins/sync
func (h *PluginHandlers) Sync(w http.ResponseWriter, r *http.Request) {
	reg := h.ensureLoaded()
	configData, err := reg.GenerateTraefikConfig()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "CONFIG_GENERATION_FAILED", "failed to generate Traefik config: "+err.Error(), nil)
		return
	}

	stackDir := system.GetStackDir()
	dynamicDir := filepath.Join(stackDir, "dynamic")
	if err := os.MkdirAll(dynamicDir, 0755); err != nil {
		sendError(w, http.StatusInternalServerError, "DIR_CREATE_FAILED", "failed to create dynamic directory: "+err.Error(), nil)
		return
	}

	outputPath := filepath.Join(dynamicDir, "m3tal-plugins.yml")
	if err := os.WriteFile(outputPath, configData, 0644); err != nil {
		sendError(w, http.StatusInternalServerError, "CONFIG_WRITE_FAILED", "failed to write config file: "+err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"synced": true,
		"path":   outputPath,
	}, nil)
}

// ListCatalog returns all official plugins and their local installation state.
// GET /api/v2/plugins/catalog
func (h *PluginHandlers) ListCatalog(w http.ResponseWriter, r *http.Request) {
	reg := h.ensureLoaded()
	catalogStatus := plugin.ListCatalog(reg)
	sendSuccess(w, http.StatusOK, catalogStatus, nil)
}

// Install downloads and installs a plugin from the remote catalog.
// POST /api/v2/plugins/install
func (h *PluginHandlers) Install(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	if req.Name == "" || req.Kind == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "missing name or kind", nil)
		return
	}

	userDir := system.GetUserPluginsDir()

	err := plugin.InstallPlugin(req.Name, req.Kind, userDir)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "INSTALL_FAILED", "failed to install plugin: "+err.Error(), nil)
		return
	}

	h.registry = nil
	reg := h.ensureLoaded()

	sendSuccess(w, http.StatusOK, map[string]any{
		"installed": true,
		"summary":   reg.Summary(),
	}, nil)
}

// Uninstall deletes a user-installed plugin.
// POST /api/v2/plugins/uninstall
func (h *PluginHandlers) Uninstall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	if req.Name == "" || req.Kind == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "missing name or kind", nil)
		return
	}

	reg := h.ensureLoaded()
	userDir := system.GetUserPluginsDir()

	var path string
	switch req.Kind {
	case "Route":
		p := reg.GetRoute(req.Name)
		if p != nil {
			path = p.SourcePath
		}
	case "Stack":
		p := reg.GetStack(req.Name)
		if p != nil {
			path = p.SourcePath
		}
	case "Middleware":
		p := reg.GetMiddleware(req.Name)
		if p != nil {
			path = p.SourcePath
		}
	default:
		if p := reg.GetRoute(req.Name); p != nil {
			path = p.SourcePath
		} else if p := reg.GetStack(req.Name); p != nil {
			path = p.SourcePath
		} else if p := reg.GetMiddleware(req.Name); p != nil {
			path = p.SourcePath
		}
	}

	if path == "" {
		sendError(w, http.StatusNotFound, "PLUGIN_NOT_FOUND", "plugin not found: "+req.Name, nil)
		return
	}

	p, err := plugin.LoadPlugin(path)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PLUGIN_LOAD_FAILED", "failed to load plugin manifest: "+err.Error(), nil)
		return
	}

	mgr := plugin.NewStateManager(h.db)
	err = mgr.UninstallPlugin(p, userDir, reg)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "UNINSTALL_FAILED", "failed to uninstall plugin: "+err.Error(), nil)
		return
	}

	h.registry = nil
	newReg := h.ensureLoaded()

	sendSuccess(w, http.StatusOK, map[string]any{
		"uninstalled": true,
		"summary":     newReg.Summary(),
	}, nil)
}

// Validate validates a plugin manifest YAML.
// POST /api/v2/plugins/validate
func (h *PluginHandlers) Validate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		YAML string `json:"yaml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	p, err := plugin.ParsePlugin([]byte(req.YAML))
	if err != nil {
		sendError(w, http.StatusBadRequest, "PARSE_FAILED", err.Error(), nil)
		return
	}

	if err := p.Validate(); err != nil {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"valid":       true,
		"kind":        p.Kind,
		"name":        p.Metadata.Name,
		"description": p.Metadata.Description,
		"version":     p.Metadata.Version,
	}, nil)
}

// Match matches a service name/image/labels against Route plugins.
// POST /api/v2/plugins/match
func (h *PluginHandlers) Match(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Service string            `json:"service"`
		Image   string            `json:"image"`
		Labels  map[string]string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	reg := h.ensureLoaded()
	match := reg.MatchService(req.Service, req.Image, req.Labels)
	if match == nil {
		sendSuccess(w, http.StatusOK, map[string]any{"matched": false}, nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"matched": true,
		"plugin":  match,
	}, nil)
}

// InstallStackPlugin parameterizes and installs a stack plugin template.
// POST /api/v2/plugins/install-stack
func (h *PluginHandlers) InstallStackPlugin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string            `json:"name"`
		Env  map[string]string `json:"env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	reg := h.ensureLoaded()
	stack := reg.GetStack(req.Name)
	if stack == nil {
		sendError(w, http.StatusNotFound, "STACK_NOT_FOUND", "stack plugin not found: "+req.Name, nil)
		return
	}

	composeFile := stack.ComposePath
	if !filepath.IsAbs(composeFile) {
		composeFile = filepath.Join(filepath.Dir(stack.SourcePath), composeFile)
	}

	composeData, err := os.ReadFile(composeFile)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "READ_FAILED", "failed to read compose template: "+err.Error(), nil)
		return
	}

	vars := make(map[string]string)
	for k, v := range req.Env {
		vars[k] = v
	}
	for _, envLine := range os.Environ() {
		parts := strings.SplitN(envLine, "=", 2)
		if len(parts) == 2 {
			if _, ok := vars[parts[0]]; !ok {
				vars[parts[0]] = parts[1]
			}
		}
	}

	finalCompose := plugin.Parameterize(string(composeData), vars)

	stackDir := system.GetStackDir()
	outputPath := filepath.Join(stackDir, fmt.Sprintf("%s-compose.yml", req.Name))
	if err := os.WriteFile(outputPath, []byte(finalCompose), 0644); err != nil {
		sendError(w, http.StatusInternalServerError, "WRITE_FAILED", "failed to write compose file: "+err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"installed": true,
		"path":      outputPath,
	}, nil)
}
