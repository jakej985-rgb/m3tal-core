package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/core/state"
)

// StateManager coordinates plugin configuration and enablement state.
// It bridges the gap between declarative filesystem (.disabled) status and persistent SQLite status.
type StateManager struct {
	db *state.Store
}

// NewStateManager creates a new StateManager.
func NewStateManager(db *state.Store) *StateManager {
	return &StateManager{db: db}
}

// GetPluginState retrieves the status of a plugin, prioritizing SQLite records if available.
func (m *StateManager) GetPluginState(p *Plugin) (bool, string, error) {
	if m.db != nil {
		rec, err := m.db.GetPluginState(p.GetName())
		if err != nil {
			return p.Enabled, "{}", err
		}
		if rec != nil {
			return rec.Enabled, rec.Config, nil
		}
	}
	// Fallback to filesystem detection
	configJSON := "{}"
	if p.Spec != nil {
		if data, err := json.Marshal(p.Spec); err == nil {
			configJSON = string(data)
		}
	}
	return p.Enabled, configJSON, nil
}

// SetPluginEnabled enables or disables a plugin, updating both the filesystem and the database.
func (m *StateManager) SetPluginEnabled(p *Plugin, enabled bool) error {
	// 1. Filesystem update
	var newPath string
	var err error
	if enabled {
		if strings.HasSuffix(p.SourcePath, ".disabled") {
			newPath, err = EnablePlugin(p.SourcePath)
		} else {
			newPath = p.SourcePath
		}
	} else {
		if !strings.HasSuffix(p.SourcePath, ".disabled") {
			newPath, err = DisablePlugin(p.SourcePath)
		} else {
			newPath = p.SourcePath
		}
	}

	if err != nil {
		return fmt.Errorf("filesystem state transition failed: %w", err)
	}

	p.SourcePath = newPath
	p.Enabled = enabled

	// 2. Database update
	if m.db != nil {
		configJSON := "{}"
		if p.Spec != nil {
			if data, err := json.Marshal(p.Spec); err == nil {
				configJSON = string(data)
			}
		}
		err = m.db.SetPluginState(p.GetName(), p.Kind, enabled, configJSON)
		if err != nil {
			return fmt.Errorf("database plugin state save failed: %w", err)
		}
	}

	return nil
}

// UninstallPlugin removes the plugin manifest from filesystem and its record from the database.
func (m *StateManager) UninstallPlugin(p *Plugin, userPluginsDir string, reg *Registry) error {
	// 1. Filesystem uninstall
	err := UninstallPlugin(p.GetName(), p.Kind, userPluginsDir, reg)
	if err != nil {
		return err
	}

	// 2. Database cleanup
	if m.db != nil {
		_ = m.db.DeletePluginState(p.GetName())
	}

	return nil
}
