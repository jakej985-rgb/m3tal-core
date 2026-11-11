package api

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
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

// sessionCookieName carries the API credential for browser clients that
// cannot attach headers to every request (iframe subresources, WebSocket
// handshakes, plain navigation such as plugin .deb downloads). It is
// HttpOnly + SameSite=Strict; Secure is set when TLS is configured.
const sessionCookieName = "m3tal_session"

// cookieValuePrefix marks base64url-encoded cookie values so any accepted
// token (including ones with ',', ';', spaces, etc.) round-trips safely.
const cookieValuePrefix = "b64."

// encodeCookieToken encodes token into a standards-compliant cookie value.
func encodeCookieToken(token string) string {
	return cookieValuePrefix + base64.RawURLEncoding.EncodeToString([]byte(token))
}

// decodeCookieToken reverses encodeCookieToken; unprefixed values are
// returned as-is for compatibility with cookies set by earlier builds.
func decodeCookieToken(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, cookieValuePrefix) {
		if b, err := base64.RawURLEncoding.DecodeString(v[len(cookieValuePrefix):]); err == nil {
			return string(b)
		}
		return ""
	}
	return v
}

// sessionCookieSecure reports whether credential cookies get the Secure flag:
// when TLS is served directly, or when M3TAL_COOKIE_SECURE is enabled for
// deployments behind a TLS-terminating reverse proxy.
func sessionCookieSecure() bool {
	if os.Getenv("M3TAL_TLS_CERT") != "" && os.Getenv("M3TAL_TLS_KEY") != "" {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("M3TAL_COOKIE_SECURE"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// setSessionCookie refreshes the browser session cookie after a successful
// header/Bearer authentication.
func setSessionCookie(w http.ResponseWriter, token string) {
	if token == "" || len(token) > 512 {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    encodeCookieToken(token),
		Path:     "/",
		HttpOnly: true,
		Secure:   sessionCookieSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   12 * 3600,
	})
}

// requestUsedSessionCookie reports whether the request authenticated via the
// session cookie rather than an explicit header/Bearer credential.
func requestUsedSessionCookie(r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-API-Token")) != "" ||
		strings.HasPrefix(strings.ToLower(r.Header.Get("Authorization")), "bearer ") {
		return false
	}
	c, err := r.Cookie(sessionCookieName)
	return err == nil && strings.TrimSpace(c.Value) != ""
}

// ExtractToken extracts authentication token from header, Bearer auth, or the
// session cookie. URL query parameters are intentionally not accepted:
// credentials in URLs leak via logs, history, and Referer headers.
func (s *Server) ExtractToken(r *http.Request) string {
	token := strings.TrimSpace(r.Header.Get("X-API-Token"))
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			token = strings.TrimSpace(authHeader[7:])
		}
	}
	if token == "" {
		if c, err := r.Cookie(sessionCookieName); err == nil {
			token = decodeCookieToken(c.Value)
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
			if !requestUsedSessionCookie(r) {
				setSessionCookie(w, token)
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), apiKeyContextKey{}, (*state.APIKeyRecord)(nil))))
			return
		}
		if s.Store != nil {
			if rec, ok := s.Store.LookupAPIKey(token); ok {
				// rec is already loaded — stamp usage directly instead of
				// re-running the lookup inside ValidateAPIKey (saves a SELECT
				// on every authenticated request).
				s.Store.RecordAPIKeyUsage(rec)
				if !requestUsedSessionCookie(r) {
					setSessionCookie(w, token)
				}
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), apiKeyContextKey{}, rec)))
				return
			}
		}
		sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
	})
}

// pluginUICookieName carries the API credential for plugin UI subresource
// requests (CSS/JS inside the iframe) that cannot attach headers. It is scoped
// to Path=/plugins/ and HttpOnly.
const pluginUICookieName = "m3tal_ui_token"

// pluginUIAuth guards the plugin UI static routes. It accepts the same
// credentials as the API (header, Bearer, or the Path=/ session cookie sent
// automatically on same-origin iframe/subresource requests) plus the legacy
// Path=/plugins/ scoped cookie.
func (s *Server) pluginUIAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := s.ExtractToken(r)
		viaCookie := requestUsedSessionCookie(r)
		if token == "" {
			if c, err := r.Cookie(pluginUICookieName); err == nil {
				token = decodeCookieToken(c.Value)
				viaCookie = true
			}
		}
		if token == "" {
			sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
			return
		}
		// Static bootstrap token has full access; dynamic keys must carry
		// plugins:read, matching the plugin API route guards.
		staticOK := s.APIToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.APIToken)) == 1
		if !staticOK {
			var rec *state.APIKeyRecord
			ok := false
			if s.Store != nil {
				rec, ok = s.Store.LookupAPIKey(token)
			}
			if !ok {
				sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
				return
			}
			if !state.KeyAllows(rec, state.ScopePluginsRead) {
				sendError(w, http.StatusForbidden, "FORBIDDEN", "API key lacks plugins:read", nil)
				return
			}
		}
		if !viaCookie {
			http.SetCookie(w, &http.Cookie{
				Name:     pluginUICookieName,
				Value:    encodeCookieToken(token),
				Path:     "/plugins/",
				HttpOnly: true,
				Secure:   sessionCookieSecure(),
				SameSite: http.SameSiteStrictMode,
				MaxAge:   12 * 3600,
			})
		}
		next.ServeHTTP(w, r)
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
