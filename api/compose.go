package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/state"
	"github.com/jakej985-rgb/m3tal-core/pkg/compose"
	"github.com/jakej985-rgb/m3tal-core/pkg/system"
)

// ComposeHandlers provides HTTP handlers for Smart Compose operations.
type ComposeHandlers struct {
	Store *state.Store
}

// ValidateCompose handles validation and linting of a compose YAML string.
// POST /api/v2/compose/validate
func (h *ComposeHandlers) ValidateCompose(w http.ResponseWriter, r *http.Request) {
	var req struct {
		YAML string `json:"yaml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	cfg, err := compose.Parse([]byte(req.YAML))
	if err != nil {
		sendSuccess(w, http.StatusOK, map[string]any{
			"valid":  false,
			"errors": []string{err.Error()},
			"issues": []compose.LintIssue{},
		}, nil)
		return
	}

	issues := compose.Lint(cfg)
	hasErrors := false
	var errs []string
	for _, issue := range issues {
		if issue.Severity == compose.SeverityError {
			hasErrors = true
			errs = append(errs, issue.Message)
		}
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"valid":  !hasErrors,
		"errors": errs,
		"issues": issues,
	}, nil)
}

// FixCompose handles auto-fixing a compose YAML string.
// POST /api/v2/compose/fix
func (h *ComposeHandlers) FixCompose(w http.ResponseWriter, r *http.Request) {
	var req struct {
		YAML string `json:"yaml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	fixed, fixes, err := compose.AutoFix([]byte(req.YAML))
	if err != nil {
		sendError(w, http.StatusBadRequest, "COMPOSE_FIX_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"yaml":  string(fixed),
		"fixes": fixes,
	}, nil)
}

// ListTemplates handles listing the pre-defined compose templates.
// GET /api/v2/compose/templates
func (h *ComposeHandlers) ListTemplates(w http.ResponseWriter, r *http.Request) {
	sendSuccess(w, http.StatusOK, compose.Templates, nil)
}

// GenerateTemplate handles generating a compose YAML from a template.
// POST /api/v2/compose/generate
func (h *ComposeHandlers) GenerateTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Template   string            `json:"template"`
		Parameters map[string]string `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	yamlData, err := compose.Generate(req.Template, req.Parameters)
	if err != nil {
		sendError(w, http.StatusBadRequest, "TEMPLATE_GEN_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"yaml": string(yamlData),
	}, nil)
}

// SaveCompose handles saving a compose file and deploying it.
// POST /api/v2/compose/save
func (h *ComposeHandlers) SaveCompose(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Stack string `json:"stack"`
		YAML  string `json:"yaml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	if req.Stack == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "stack name is required", nil)
		return
	}

	// Clean/sanitize stack name to prevent path traversal
	stackName := strings.TrimSpace(req.Stack)
	stackName = filepath.Base(stackName)
	stackName = strings.TrimSuffix(stackName, "-compose.yml")

	// Target path
	stackDir := system.GetStackDir()
	filePath := filepath.Join(stackDir, fmt.Sprintf("%s-compose.yml", stackName))

	// Write file
	if err := os.WriteFile(filePath, []byte(req.YAML), 0644); err != nil {
		sendError(w, http.StatusInternalServerError, "FILE_WRITE_FAILED", "failed to save file: "+err.Error(), nil)
		return
	}

	// Deploy stack
	result, err := docker.DeployStack(filePath, 0)

	// Update DB status
	if h.Store != nil {
		_ = h.Store.UpsertStack(stackName, filePath)
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

	sendSuccess(w, http.StatusOK, map[string]any{
		"status": "success",
		"result": result,
	}, nil)
}
