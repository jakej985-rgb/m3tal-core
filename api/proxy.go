package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/networking/proxy"
	"github.com/jakej985-rgb/m3tal-core/core/state"
	"github.com/jakej985-rgb/m3tal-core/core/traefik"
)

// ProxyHandlers wraps the SQLite store to manage reverse proxy endpoints.
type ProxyHandlers struct {
	Store *state.Store
}

// NewProxyHandlers creates a new set of proxy handlers.
func NewProxyHandlers(db *state.Store) *ProxyHandlers {
	return &ProxyHandlers{Store: db}
}

// DiscoverServices auto-discovers containers.
// GET /api/v2/proxy/discover
func (h *ProxyHandlers) DiscoverServices(w http.ResponseWriter, r *http.Request) {
	mgr := proxy.NewManager(h.Store)
	services, err := mgr.DiscoverServices()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DISCOVERY_FAILED", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, services, nil)
}

// ExposeService registers/exposes a service on a domain.
// POST /api/v2/proxy/expose
func (h *ProxyHandlers) ExposeService(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Service     string   `json:"service"`
		Domain      string   `json:"domain"`
		Port        int      `json:"port"`
		SSL         bool     `json:"ssl"`
		Middlewares []string `json:"middlewares"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	// Route validation
	routeInput := traefik.RouteInput{
		Service: input.Service,
		Domain:  input.Domain,
		Port:    input.Port,
	}
	errs := docker.ValidateRoute(routeInput)
	if len(errs) > 0 {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "validation failed", map[string]any{"violations": errs})
		return
	}

	mgr := proxy.NewManager(h.Store)
	err := mgr.ExposeService(input.Service, input.Domain, input.Port, input.SSL, input.Middlewares)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PROXY_EXPOSE_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"ok":      true,
		"exposed": true,
		"route":   input,
	}, nil)
}

// UnexposeService removes route mapping.
// POST /api/v2/proxy/unexpose
func (h *ProxyHandlers) UnexposeService(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Domain string `json:"domain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	if input.Domain == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "domain is required", nil)
		return
	}

	mgr := proxy.NewManager(h.Store)
	err := mgr.UnexposeService(input.Domain)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PROXY_UNEXPOSE_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"ok":        true,
		"unexposed": true,
	}, nil)
}

// ConfigureSSL triggers SSL automated Let's Encrypt certificates setup.
// POST /api/v2/proxy/secure
func (h *ProxyHandlers) ConfigureSSL(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "valid email is required for SSL configuration", nil)
		return
	}

	mgr := proxy.NewManager(h.Store)
	err := mgr.ConfigureSSL(email)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PROXY_SSL_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"ok":      true,
		"secured": true,
		"message": "Let's Encrypt SSL configured and routing stack redeployed.",
	}, nil)
}
