package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/events"
	"github.com/jakej985-rgb/m3tal-core/core/orchestrator"
	"github.com/jakej985-rgb/m3tal-core/core/plugins"
	"github.com/jakej985-rgb/m3tal-core/core/state"
	"github.com/jakej985-rgb/m3tal-core/core/traefik"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"github.com/jakej985-rgb/m3tal-core/pkg/system"
)

// StackHandlers provides operations for stack management.
type StackHandlers struct {
	Store *state.Store
}

// ListStacks scans the stack directory and returns all discovered stacks
// merged with stored state from the database.
// GET /api/v2/stacks
func (h *StackHandlers) ListStacks(w http.ResponseWriter, r *http.Request) {
	stackDir := system.GetStackDir()
	matches, err := system.FindComposeFiles(stackDir)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DIRECTORY_SCAN_FAILED", "cannot scan stack directory", nil)
		return
	}

	dirs := system.GetPluginDirs()
	reg, _ := plugin.LoadAll(dirs...)

	var stacks []models.Stack
	for _, match := range matches {
		base := filepath.Base(match)
		name := strings.TrimSuffix(base, "-compose.yml")

		if reg != nil {
			if sp := reg.GetStack(name); sp != nil && !sp.Enabled {
				continue
			}
		}

		info := models.Stack{
			Name:        name,
			ComposePath: match,
			Status:      "discovered",
		}

		// Try to parse for service names
		if cf, err := traefik.ParseCompose(match); err == nil {
			info.Services = cf.ServiceNames()
		}

		// Check DB for stored status
		if h.Store != nil {
			_ = h.Store.UpsertStack(name, match)
			dbStacks, _ := h.Store.ListStacks()
			for _, ds := range dbStacks {
				if ds.Name == name && ds.Status != "" {
					info.Status = ds.Status
				}
			}
		}

		stacks = append(stacks, info)
	}

	if stacks == nil {
		stacks = []models.Stack{}
	} else {
		sort.Slice(stacks, func(i, j int) bool {
			pI := 100
			if reg != nil {
				if sp := reg.GetStack(stacks[i].Name); sp != nil {
					pI = sp.Priority
				}
			}

			pJ := 100
			if reg != nil {
				if sp := reg.GetStack(stacks[j].Name); sp != nil {
					pJ = sp.Priority
				}
			}

			if pI != pJ {
				return pI < pJ
			}
			return stacks[i].Name < stacks[j].Name
		})
	}

	sendSuccess(w, http.StatusOK, stacks, nil)
}

// LoadStack parses a compose file and returns its services, ports, and labels.
// POST /api/v2/stacks/load
func (h *StackHandlers) LoadStack(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	if req.Path == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "path is required", nil)
		return
	}

	if !isPathAllowed(req.Path) {
		sendError(w, http.StatusForbidden, "FORBIDDEN_PATH", "path must be within the stack directory", nil)
		return
	}

	cf, err := traefik.ParseCompose(req.Path)
	if err != nil {
		sendError(w, http.StatusBadRequest, "COMPOSE_PARSE_FAILED", err.Error(), nil)
		return
	}

	// Load plugin registry for service matching suggestions
	reg, _ := plugin.LoadFromPaths(system.SystemPluginsDir, system.UserPluginsDir)

	// Build service details
	type serviceDetail struct {
		Name            string              `json:"name"`
		Image           string              `json:"image"`
		Ports           []string            `json:"ports,omitempty"`
		ContainerPort   int                 `json:"container_port,omitempty"`
		Labels          map[string]string   `json:"labels,omitempty"`
		Networks        []string            `json:"networks,omitempty"`
		SuggestedPlugin *plugin.RoutePlugin `json:"suggested_plugin,omitempty"`
	}

	var services []serviceDetail
	for name, svc := range cf.Services {
		sd := serviceDetail{
			Name:          name,
			Image:         svc.Image,
			Ports:         svc.Ports,
			ContainerPort: cf.GetContainerPort(name),
			Labels:        cf.GetTraefikLabels(name),
			Networks:      svc.Networks.Values,
		}
		if reg != nil {
			sd.SuggestedPlugin = reg.MatchService(name, svc.Image, sd.Labels)
		}
		services = append(services, sd)
	}

	// Validate
	validationErrors := docker.ValidateCompose(cf)

	sendSuccess(w, http.StatusOK, map[string]any{
		"path":              req.Path,
		"services":          services,
		"service_count":     len(services),
		"validation_errors": validationErrors,
	}, nil)
}

// DeployStack validates and deploys a stack.
// POST /api/v2/stacks/deploy
func (h *StackHandlers) DeployStack(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	if req.Path == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "path is required", nil)
		return
	}

	if !isPathAllowed(req.Path) {
		sendError(w, http.StatusForbidden, "FORBIDDEN_PATH", "path must be within the stack directory", nil)
		return
	}

	// Verify file exists
	if _, err := os.Stat(req.Path); err != nil {
		sendError(w, http.StatusNotFound, "STACK_NOT_FOUND", "compose file not found: "+req.Path, nil)
		return
	}

	// Parse to get stack name
	base := filepath.Base(req.Path)
	stackName := strings.TrimSuffix(base, "-compose.yml")

	// Deploy
	result, err := docker.DeployStack(req.Path, 0)

	// Update DB status regardless of outcome
	if h.Store != nil {
		_ = h.Store.UpsertStack(stackName, req.Path)
		if err != nil {
			_ = h.Store.UpdateStackStatus(stackName, "failed")
		} else {
			_ = h.Store.UpdateStackStatus(stackName, "running")
		}
	}

	if err != nil {
		sendError(w, http.StatusInternalServerError, "DEPLOY_FAILED", err.Error(), result)
		return
	}

	sendSuccess(w, http.StatusOK, result, nil)
}

// MiddlewareHandlers provides CRUD for Traefik middleware templates.
type MiddlewareHandlers struct {
	Store *state.Store
}

// ListMiddleware returns all stored middleware definitions.
// GET /api/v2/middleware
func (h *MiddlewareHandlers) ListMiddleware(w http.ResponseWriter, r *http.Request) {
	mws, err := h.Store.ListMiddleware()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error(), nil)
		return
	}
	if mws == nil {
		mws = []state.MiddlewareRecord{}
	}
	sendSuccess(w, http.StatusOK, mws, nil)
}

// CreateMiddleware creates a new middleware definition and generates its labels.
// POST /api/v2/middleware
func (h *MiddlewareHandlers) CreateMiddleware(w http.ResponseWriter, r *http.Request) {
	var input traefik.MiddlewareInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	if input.Name == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "name is required", nil)
		return
	}
	if input.Type == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "type is required", nil)
		return
	}

	// Generate labels
	labels := traefik.GenerateMiddlewareLabels(input)

	// Store
	id, err := h.Store.CreateMiddleware(input.Name, input.Type, input.Config)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusCreated, map[string]any{
		"id":     id,
		"labels": labels,
	}, nil)
}

// isPathAllowed validates that the target path resides securely within the configured stack directories.
func isPathAllowed(path string) bool {
	if path == "" {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	stackDir := system.GetStackDir()
	absStackDir, err := filepath.Abs(stackDir)
	if err != nil {
		return false
	}

	allowedDirs := []string{absStackDir, "/docker", system.StackPath}
	for _, allowed := range allowedDirs {
		allowedAbs, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		if absPath == allowedAbs {
			return true
		}
		parentWithSep := allowedAbs + string(filepath.Separator)
		if strings.HasPrefix(absPath, parentWithSep) {
			return true
		}
	}
	return false
}

// ScanStacks manually triggers discovery of compose files and returns them.
// POST /api/v2/stacks/scan
func (h *StackHandlers) ScanStacks(w http.ResponseWriter, r *http.Request) {
	stackDir := system.GetStackDir()
	matches, err := system.FindComposeFiles(stackDir)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DIRECTORY_SCAN_FAILED", "failed to scan stacks: "+err.Error(), nil)
		return
	}

	dirs := system.GetPluginDirs()
	reg, _ := plugin.LoadAll(dirs...)

	var stacks []models.Stack
	for _, match := range matches {
		base := filepath.Base(match)
		name := strings.TrimSuffix(base, "-compose.yml")

		if reg != nil {
			if sp := reg.GetStack(name); sp != nil && !sp.Enabled {
				continue
			}
		}

		info := models.Stack{
			Name:        name,
			ComposePath: match,
			Status:      "discovered",
		}

		// Try to parse for service names
		if cf, err := traefik.ParseCompose(match); err == nil {
			info.Services = cf.ServiceNames()
		}

		// Update DB / Check status
		if h.Store != nil {
			_ = h.Store.UpsertStack(name, match)
			dbStacks, _ := h.Store.ListStacks()
			for _, ds := range dbStacks {
				if ds.Name == name && ds.Status != "" {
					info.Status = ds.Status
				}
			}
		}

		stacks = append(stacks, info)
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"ok":     true,
		"stacks": stacks,
	}, nil)
}

func (h *StackHandlers) findStackPathByName(name string) (string, error) {
	// 1. Try to fetch from DB first if store is active
	if h.Store != nil {
		stacks, err := h.Store.ListStacks()
		if err == nil {
			for _, s := range stacks {
				if s.Name == name && s.ComposePath != "" {
					// Verify file exists
					if _, err := os.Stat(s.ComposePath); err == nil {
						return s.ComposePath, nil
					}
				}
			}
		}
	}

	// 2. Scan stack directory
	stackDir := system.GetStackDir()
	matches, err := system.FindComposeFiles(stackDir)
	if err != nil {
		return "", err
	}
	for _, match := range matches {
		base := filepath.Base(match)
		stackName := strings.TrimSuffix(base, "-compose.yml")
		if stackName == name {
			return match, nil
		}
	}

	return "", fmt.Errorf("stack not found: %s", name)
}

// GetStackByName returns the details and status of a specific stack.
// GET /api/v2/stacks/{name}
func (h *StackHandlers) GetStackByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "stack name is required", nil)
		return
	}

	composePath, err := h.findStackPathByName(name)
	if err != nil {
		sendError(w, http.StatusNotFound, "STACK_NOT_FOUND", err.Error(), nil)
		return
	}

	info := models.Stack{
		Name:        name,
		ComposePath: composePath,
		Status:      "discovered",
	}

	if cf, err := traefik.ParseCompose(composePath); err == nil {
		info.Services = cf.ServiceNames()
	}

	if h.Store != nil {
		dbStacks, _ := h.Store.ListStacks()
		for _, ds := range dbStacks {
			if ds.Name == name && ds.Status != "" {
				info.Status = ds.Status
			}
		}
	}

	sendSuccess(w, http.StatusOK, info, nil)
}

// DeployStackByName handles validating and deploying a stack by its name.
// POST /api/v2/stacks/{name}/up
func (h *StackHandlers) DeployStackByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "stack name is required", nil)
		return
	}

	composePath, err := h.findStackPathByName(name)
	if err != nil {
		sendError(w, http.StatusNotFound, "STACK_NOT_FOUND", err.Error(), nil)
		return
	}

	// Deploy stack
	result, err := docker.DeployStack(composePath, 0)

	// Update DB status
	status := "running"
	if err != nil {
		status = "failed"
	}
	if h.Store != nil {
		_ = h.Store.UpsertStack(name, composePath)
		_ = h.Store.UpdateStackStatus(name, status)
	}
	events.GlobalEventBus.Publish("stack.updated", map[string]string{
		"name":   name,
		"action": "up",
		"status": status,
	})

	if err != nil {
		sendError(w, http.StatusInternalServerError, "DEPLOY_FAILED", err.Error(), result)
		return
	}

	sendSuccess(w, http.StatusOK, result, nil)
}

// StopStackByName handles stopping a stack by its name.
// POST /api/v2/stacks/{name}/down
func (h *StackHandlers) StopStackByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "stack name is required", nil)
		return
	}

	composePath, err := h.findStackPathByName(name)
	if err != nil {
		sendError(w, http.StatusNotFound, "STACK_NOT_FOUND", err.Error(), nil)
		return
	}

	// Stop stack
	result, err := docker.StopStack(composePath, 0)

	// Update DB status
	status := "stopped"
	if err != nil {
		status = "failed"
	}
	if h.Store != nil {
		_ = h.Store.UpsertStack(name, composePath)
		_ = h.Store.UpdateStackStatus(name, status)
	}
	events.GlobalEventBus.Publish("stack.updated", map[string]string{
		"name":   name,
		"action": "down",
		"status": status,
	})

	if err != nil {
		sendError(w, http.StatusInternalServerError, "STOP_FAILED", err.Error(), result)
		return
	}

	sendSuccess(w, http.StatusOK, result, nil)
}

// PullStacks pulls latest images for all stacks or a specific stack.
// POST /api/v2/stacks/pull
func (h *StackHandlers) PullStacks(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	stackMgr := orchestrator.NewStackManager()
	if req.Name != "" {
		found := false
		for _, f := range stackMgr.Files {
			base := filepath.Base(f)
			if strings.TrimSuffix(base, "-compose.yml") == req.Name {
				stackMgr.Files = []string{f}
				found = true
				break
			}
		}
		if !found {
			sendError(w, http.StatusNotFound, "STACK_NOT_FOUND", "stack not found: "+req.Name, nil)
			return
		}
	}

	err := stackMgr.Run("pull")
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PULL_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{"pulled": true}, nil)
}

// GetStackLogs returns compose logs of a stack.
// GET /api/v2/stacks/{name}/logs
func (h *StackHandlers) GetStackLogs(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "100"
	}

	stackMgr := orchestrator.NewStackManager()
	if name != "all" && name != "" {
		found := false
		for _, f := range stackMgr.Files {
			base := filepath.Base(f)
			if strings.TrimSuffix(base, "-compose.yml") == name {
				stackMgr.Files = []string{f}
				found = true
				break
			}
		}
		if !found {
			sendError(w, http.StatusNotFound, "STACK_NOT_FOUND", "stack not found: "+name, nil)
			return
		}
	}

	var logsBuilder strings.Builder
	for _, file := range stackMgr.Files {
		stackBase := filepath.Base(file)
		stackName := stackBase[:len(stackBase)-len("-compose.yml")]
		cmdArgs := []string{"compose", "-p", stackName, "-f", file, "logs", "--tail", tail}
		cmd := exec.Command("docker", cmdArgs...)
		if out, err := cmd.CombinedOutput(); err == nil {
			logsBuilder.WriteString(string(out))
		}
	}

	sendSuccess(w, http.StatusOK, map[string]string{"logs": logsBuilder.String()}, nil)
}
