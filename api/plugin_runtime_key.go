package api

import (
	"log"

	"github.com/jakej985-rgb/m3tal-core/core/state"
)

// pluginRuntimeKeyName identifies the dedicated API key provisioned for
// plugin subprocesses. It is deliberately scoped below full access: plugins
// can read system/plugin state and manage plugin lifecycle, but cannot mint
// API keys or kill host processes.
const pluginRuntimeKeyName = "m3tal-plugin-runtime"

var pluginRuntimeKeyScopes = []string{
	state.ScopeSystemRead,
	state.ScopePluginsRead,
	state.ScopePluginsWrite,
}

// pluginRuntimeAPIKey returns the token of the dedicated plugin-runtime API
// key, creating it on first use. Returns "" when the store is unavailable or
// provisioning fails, in which case plugins fall back to env/file resolution.
func pluginRuntimeAPIKey(db *state.Store) string {
	if db == nil {
		return ""
	}
	if keys, err := db.ListAPIKeys(); err == nil {
		for _, k := range keys {
			if k.Name == pluginRuntimeKeyName && k.Direction == state.DirectionLocal {
				return k.Token
			}
		}
	}
	rec, err := db.CreateAPIKey(pluginRuntimeKeyName, pluginRuntimeKeyScopes, state.DirectionLocal, "plugin-runtime", "")
	if err != nil {
		log.Printf("⚠️ Could not provision plugin runtime API key: %v", err)
		return ""
	}
	log.Printf("🔑 Provisioned scoped API key %q for plugin subprocesses", pluginRuntimeKeyName)
	return rec.Token
}
