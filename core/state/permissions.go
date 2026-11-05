package state

import (
	"encoding/json"
	"strings"
)

// API key permission scopes. A key's Permissions list grants access to the
// matching route groups; "*" grants full access. Write scopes imply their
// read counterparts (see KeyAllows).
const (
	ScopeFull          = "*"
	ScopeSystemRead    = "system:read"
	ScopeSystemWrite   = "system:write"
	ScopePluginsRead   = "plugins:read"
	ScopePluginsWrite  = "plugins:write"
	ScopeSettingsWrite = "settings:write"
	ScopeKeysManage    = "keys:manage"
	ScopeMeshAccess    = "mesh:access"
)

// ValidAPIKeyPermissions is the catalog of scopes that may be assigned when
// creating an API key. ScopeFull ("*") is always permitted.
var ValidAPIKeyPermissions = map[string]bool{
	ScopeFull:          true,
	ScopeSystemRead:    true,
	ScopeSystemWrite:   true,
	ScopePluginsRead:   true,
	ScopePluginsWrite:  true,
	ScopeSettingsWrite: true,
	ScopeKeysManage:    true,
	ScopeMeshAccess:    true,
}

// Key directions.
const (
	DirectionLocal  = "local"  // Authenticates inbound access to this machine's API ("This PC" tab)
	DirectionRemote = "remote" // Credential this machine uses to reach other PCs/apps ("Other" tab)
)

// parsePermissions decodes the permissions column. Empty or unparseable
// values normalize to full access for backward compatibility with keys
// created before scoped permissions existed.
func parsePermissions(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{ScopeFull}
	}
	var perms []string
	if err := json.Unmarshal([]byte(raw), &perms); err != nil || len(perms) == 0 {
		return []string{ScopeFull}
	}
	return perms
}

// NormalizePermissions validates and normalizes a permission list. Unknown
// scopes are dropped; if nothing valid remains, full access is assumed only
// when the input was empty (callers should pass explicit scopes otherwise).
func NormalizePermissions(perms []string) []string {
	if len(perms) == 0 {
		return []string{ScopeFull}
	}
	out := make([]string, 0, len(perms))
	seen := make(map[string]bool)
	for _, p := range perms {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		if ValidAPIKeyPermissions[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{ScopeFull}
	}
	return out
}

// NormalizeDirection validates a direction value, defaulting to local.
func NormalizeDirection(d string) string {
	switch strings.TrimSpace(d) {
	case DirectionRemote:
		return DirectionRemote
	default:
		return DirectionLocal
	}
}

// KeyAllows reports whether a key record grants the given scope. The "*"
// scope grants everything; "<area>:write" implies "<area>:read"; an empty
// permission list is treated as full access for legacy compatibility.
func KeyAllows(rec *APIKeyRecord, scope string) bool {
	if rec == nil {
		return false
	}
	perms := rec.Permissions
	if len(perms) == 0 {
		perms = []string{ScopeFull}
	}
	for _, p := range perms {
		if p == ScopeFull || p == scope {
			return true
		}
	}
	// Write implies read within the same area (e.g. plugins:write -> plugins:read)
	if strings.HasSuffix(scope, ":read") {
		writeScope := strings.TrimSuffix(scope, ":read") + ":write"
		for _, p := range perms {
			if p == writeScope {
				return true
			}
		}
	}
	return false
}

// AllowsMeshAccess reports whether the key may be used for mesh/fleet
// telemetry federation. Accepts "*", "mesh:access", or any "mesh*" scope.
func AllowsMeshAccess(perms []string) bool {
	for _, p := range perms {
		if p == ScopeFull || p == ScopeMeshAccess || strings.HasPrefix(p, "mesh") {
			return true
		}
	}
	return false
}
