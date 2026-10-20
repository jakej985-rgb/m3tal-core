package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/state"
	"github.com/jakej985-rgb/m3tal-core/core/traefik"
)

// RouteHandlers provides CRUD operations for Traefik routes.
type RouteHandlers struct {
	Store *state.Store
}

// ListRoutes returns all stored routes.
// GET /api/v2/routes
func (h *RouteHandlers) ListRoutes(w http.ResponseWriter, r *http.Request) {
	routes, err := h.Store.ListRoutes()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error(), nil)
		return
	}
	if routes == nil {
		routes = []state.RouteRecord{}
	}
	sendSuccess(w, http.StatusOK, routes, nil)
}

// CreateRoute validates and creates a new route.
// POST /api/v2/routes
func (h *RouteHandlers) CreateRoute(w http.ResponseWriter, r *http.Request) {
	var input traefik.RouteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	// Validate route input
	errs := docker.ValidateRoute(input)
	if len(errs) > 0 {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "validation failed", map[string]any{"violations": errs})
		return
	}

	// Check domain uniqueness
	existing, _ := h.Store.ListRoutes()
	var existingInputs []traefik.RouteInput
	for _, r := range existing {
		existingInputs = append(existingInputs, traefik.RouteInput{
			Service: r.Service,
			Domain:  r.Domain,
			Port:    r.Port,
		})
	}
	uniqueErrs := docker.ValidateDomainUnique(input.Domain, existingInputs)
	if len(uniqueErrs) > 0 {
		sendError(w, http.StatusConflict, "DOMAIN_CONFLICT", "domain conflict", map[string]any{"violations": uniqueErrs})
		return
	}

	// Generate labels
	labels := traefik.GenerateRouteLabels(input)

	// Store in DB
	id, err := h.Store.CreateRoute(input.Service, input.Domain, input.Port, input.Entrypoints, "", false, "")
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusCreated, map[string]any{
		"id":     id,
		"labels": labels,
		"route":  input,
	}, nil)
}

// DeleteRoute removes a route by ID.
// DELETE /api/v2/routes/{id}
func (h *RouteHandlers) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_ID", "invalid route ID", nil)
		return
	}

	if err := h.Store.DeleteRoute(id); err != nil {
		sendError(w, http.StatusNotFound, "ROUTE_NOT_FOUND", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

// GetRouteLabels generates labels for a route without storing it.
// POST /api/v2/routes/preview
func (h *RouteHandlers) GetRouteLabels(w http.ResponseWriter, r *http.Request) {
	var input traefik.RouteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	errs := docker.ValidateRoute(input)
	if len(errs) > 0 {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "validation failed", map[string]any{"violations": errs})
		return
	}

	labels := traefik.GenerateRouteLabels(input)
	sendSuccess(w, http.StatusOK, labels, nil)
}
