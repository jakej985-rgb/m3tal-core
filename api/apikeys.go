package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/core/state"
)

// CreateAPIKeyRequest defines payload for generating an API key.
// Note: Custom key tokens cannot be specified; tokens are strictly server-generated (Jellyfin-style).
type CreateAPIKeyRequest struct {
	Name  string `json:"name"`
	Token string `json:"token,omitempty"`
	Key   string `json:"key,omitempty"`
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

	// Strictly prohibit custom token submissions (Jellyfin generation behavior)
	if strings.TrimSpace(req.Token) != "" || strings.TrimSpace(req.Key) != "" {
		sendError(w, http.StatusBadRequest, "CUSTOM_ENTRIES_PROHIBITED", "Custom API key entries are prohibited. API keys are generated automatically by Core.", nil)
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

	rec, err := s.Store.CreateAPIKey(trimmedName)
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
		sendError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{"deleted": true, "id": id}, nil)
}
