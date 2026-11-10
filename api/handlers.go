package api

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/core/state"
	"github.com/jakej985-rgb/m3tal-core/core/system"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"github.com/jakej985-rgb/m3tal-core/pkg/version"
)

// Server handles API requests
type Server struct {
	APIToken string
	Store    *state.Store
}

// NewServer creates a new API server
func NewServer(token string) *Server {
	return &Server{APIToken: token}
}

// NewServerWithStore creates a new API server with a persistent state store
func NewServerWithStore(token string, store *state.Store) *Server {
	return &Server{APIToken: token, Store: store}
}

// ExtractToken extracts authentication token from header, query, or Bearer auth.
func (s *Server) ExtractToken(r *http.Request) string {
	token := strings.TrimSpace(r.Header.Get("X-API-Token"))
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			token = strings.TrimSpace(authHeader[7:])
		}
	}
	return token
}

// ValidateToken validates a token against static secrets and dynamic API keys.
func (s *Server) ValidateToken(token string) bool {
	if token == "" {
		return false
	}
	if s.APIToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.APIToken)) == 1 {
		return true
	}
	if s.Store != nil && s.Store.ValidateAPIKey(token) {
		return true
	}
	return false
}

type apiKeyContextKey struct{}

// Authenticate validates the request token and stores the matched API key
// record in the request context. A nil record means the static APIToken was
// used, which has full access to every scope.
func (s *Server) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := s.ExtractToken(r)
		if token == "" {
			sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
			return
		}
		if s.APIToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.APIToken)) == 1 {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), apiKeyContextKey{}, (*state.APIKeyRecord)(nil))))
			return
		}
		if s.Store != nil {
			if rec, ok := s.Store.LookupAPIKey(token); ok {
				// rec is already loaded — stamp usage directly instead of
				// re-running the lookup inside ValidateAPIKey (saves a SELECT
				// on every authenticated request).
				s.Store.RecordAPIKeyUsage(rec)
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), apiKeyContextKey{}, rec)))
				return
			}
		}
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
	})
}

// APIKeyFromContext returns the API key record attached by Authenticate.
// A nil record (or absent context value) means full-access static token auth.
func APIKeyFromContext(r *http.Request) *state.APIKeyRecord {
	rec, _ := r.Context().Value(apiKeyContextKey{}).(*state.APIKeyRecord)
	return rec
}

// requireScope wraps a handler so it only serves requests whose API key
// grants the given permission scope. Static-token auth (nil record) bypasses
// scope checks.
func (s *Server) requireScope(scope string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if rec := APIKeyFromContext(r); rec != nil && !state.KeyAllows(rec, scope) {
			sendError(w, http.StatusForbidden, "INSUFFICIENT_SCOPE", "API key does not have permission for "+scope, nil)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// GetVersion returns canonical M3TAL Core version metadata.
func (s *Server) GetVersion(w http.ResponseWriter, r *http.Request) {
	sendSuccess(w, http.StatusOK, version.GetInfo(), nil)
}

// GetHealth returns host health status.
func (s *Server) GetHealth(w http.ResponseWriter, r *http.Request) {
	report := system.CheckHealth(s.Store)
	components := make(map[string]string, len(report.Components))
	for name, status := range report.Components {
		components[name] = string(status)
	}
	sendSuccess(w, http.StatusOK, models.Status{
		Status:     string(report.Status),
		Components: components,
		Details:    report.Details,
	}, nil)
}

// GetStats returns detailed host telemetry for the dashboard and system monitor.
func (s *Server) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := system.GetDetailedStats()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "STATS_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, stats, nil)
}

// GetProcesses returns running host processes.
func (s *Server) GetProcesses(w http.ResponseWriter, r *http.Request) {
	limit := 150
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	procs, err := system.GetProcesses(limit)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PROCESS_ERROR", err.Error(), nil)
		return
	}
	if procs == nil {
		procs = []models.ProcessInfo{}
	}
	sendSuccess(w, http.StatusOK, procs, nil)
}

// KillProcess terminates a host process by PID.
func (s *Server) KillProcess(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "pid")
	pid64, err := strconv.ParseInt(pidStr, 10, 32)
	if err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_PID", "Invalid process ID", nil)
		return
	}

	if reason := protectedKillReason(int32(pid64)); reason != "" {
		sendError(w, http.StatusForbidden, "PID_PROTECTED", reason, nil)
		return
	}

	if err := system.KillProcess(int32(pid64)); err != nil {
		sendError(w, http.StatusInternalServerError, "KILL_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"killed": true,
		"pid":    pid64,
	}, nil)
}

// protectedKillNames lists process comm names that must never be killed via
// the API, even by an authenticated system:write key. Killing PID 1 or the
// service manager would take down the host.
var protectedKillNames = map[string]bool{
	"systemd": true, "init": true, "sshd": true, "sudo": true,
	"dbus-daemon": true, "dbus-broker": true, "networkd-dispatcher": true,
	"systemd-logind": true, "systemd-resolved": true, "systemd-networkd": true,
	"systemd-timesyncd": true, "systemd-udevd": true, "cron": true,
	"crond": true, "atd": true, "rsyslogd": true, "login": true,
	"agetty": true, "getty": true, "m3tal-api": true,
}

// protectedKillReason returns a human-readable reason when pid refers to a
// process the API must not terminate, or "" when the kill is allowed.
func protectedKillReason(pid int32) string {
	if pid <= 1 {
		return "refusing to kill kernel/init process"
	}
	if int(pid) == os.Getpid() {
		return "refusing to kill the API server itself"
	}
	if comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); err == nil {
		name := strings.TrimSpace(string(comm))
		if protectedKillNames[name] {
			return fmt.Sprintf("refusing to kill protected system process %q", name)
		}
	}
	return ""
}

// GetSystemSpecs returns host hardware and OS specifications.
func (s *Server) GetSystemSpecs(w http.ResponseWriter, r *http.Request) {
	specs, err := system.GetSystemSpecs()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "SPECS_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, specs, nil)
}

// GetSystemNetwork returns host network interface telemetry.
func (s *Server) GetSystemNetwork(w http.ResponseWriter, r *http.Request) {
	interfaces, err := system.GetNetworkInterfaces()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "NETWORK_ERROR", err.Error(), nil)
		return
	}
	if interfaces == nil {
		interfaces = []models.NetworkInterfaceInfo{}
	}
	sendSuccess(w, http.StatusOK, interfaces, nil)
}
