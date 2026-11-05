package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/core/state"
)

// CreateAPIKeyRequest defines payload for generating an API key.
// Local keys cannot specify a custom token (Jellyfin-style server generation).
// Remote keys record a credential issued by another machine/app, so Token is
// required when Direction is "remote".
type CreateAPIKeyRequest struct {
	Name        string   `json:"name"`
	Token       string   `json:"token,omitempty"`
	Key         string   `json:"key,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Direction   string   `json:"direction,omitempty"`
	Target      string   `json:"target,omitempty"`
}

// ListAPIKeys returns all stored API keys.
// GET /api/v2/auth/keys
func (s *Server) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	if s.Store == nil {
		sendError(w, http.StatusServiceUnavailable, "STORE_UNAVAILABLE", "State store is not active", nil)
		return
	}
	keys, err := s.Store.ListAPIKeys()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error(), nil)
		return
	}
	if keys == nil {
		keys = []state.APIKeyRecord{}
	}
	sendSuccess(w, http.StatusOK, keys, nil)
}

// CreateAPIKey generates a cryptographically secure random API key like Jellyfin.
// POST /api/v2/auth/keys
func (s *Server) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if s.Store == nil {
		sendError(w, http.StatusServiceUnavailable, "STORE_UNAVAILABLE", "State store is not active", nil)
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_BODY", "Failed to parse JSON body", nil)
		return
	}

	direction := state.NormalizeDirection(req.Direction)
	suppliedToken := strings.TrimSpace(req.Token)
	if suppliedToken == "" {
		suppliedToken = strings.TrimSpace(req.Key)
	}

	// Strictly prohibit custom token submissions for local keys (Jellyfin
	// generation behavior). Remote keys are foreign credentials and must
	// carry the token issued by the remote machine.
	if direction == state.DirectionLocal && suppliedToken != "" {
		sendError(w, http.StatusBadRequest, "CUSTOM_ENTRIES_PROHIBITED", "Custom API key entries are prohibited for this PC. API keys are generated automatically by Core. Use the remote tab to register a credential issued by another machine.", nil)
		return
	}
	if direction == state.DirectionRemote && suppliedToken == "" {
		sendError(w, http.StatusBadRequest, "TOKEN_REQUIRED", "Remote API keys require the token issued by the remote machine", nil)
		return
	}

	trimmedName := strings.TrimSpace(req.Name)
	if trimmedName == "" {
		sendError(w, http.StatusBadRequest, "NAME_REQUIRED", "App name is required", nil)
		return
	}
	if len(trimmedName) > 255 {
		sendError(w, http.StatusBadRequest, "NAME_TOO_LONG", "API key name cannot exceed 255 characters", nil)
		return
	}

	rec, err := s.Store.CreateAPIKey(trimmedName, req.Permissions, direction, req.Target, suppliedToken)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "CREATE_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusCreated, rec, nil)
}

// DeleteAPIKey removes an API key by ID.
// DELETE /api/v2/auth/keys/{id}
func (s *Server) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if s.Store == nil {
		sendError(w, http.StatusServiceUnavailable, "STORE_UNAVAILABLE", "State store is not active", nil)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_ID", "Invalid API key ID", nil)
		return
	}

	if err := s.Store.DeleteAPIKey(id); err != nil {
		if errors.Is(err, state.ErrAPIKeyNotFound) || strings.Contains(err.Error(), "not found") {
			sendError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		} else {
			sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error(), nil)
		}
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{"deleted": true, "id": id}, nil)
}
