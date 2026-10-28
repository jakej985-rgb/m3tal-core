package plugins

import (
	"encoding/json"
	"fmt"
)

// APIVersion defines current plugin specification schema version.
const (
	APIVersionV1Alpha1 = "m3tal.io/v1alpha1"
	KindPlugin         = "Plugin"
)

// Capability defines the functional roles a plugin can declare.
type Capability string

const (
	CapabilityCollector Capability = "collector"  // Periodically emits telemetry metrics & tables
	CapabilityUIView    Capability = "ui_view"    // Mounts dedicated view in Web GUI
	CapabilityUIWidget  Capability = "ui_widget"  // Exposes dashboard grid widgets
	CapabilityActions   Capability = "actions"    // Exposes executable actions/commands
	CapabilitySettings  Capability = "settings"   // Exposes configurable settings sections
	CapabilityExporter  Capability = "exporter"   // Streams or exports core telemetry
	CapabilityTUIWidget Capability = "tui_widget" // Deprecated: legacy terminal UI widget
)

// PluginManifest represents the static definition file (plugin.yaml) for an external plugin.
type PluginManifest struct {
	APIVersion string         `yaml:"apiVersion" json:"apiVersion"`
	Kind       string         `yaml:"kind" json:"kind"`
	Metadata   PluginMeta     `yaml:"metadata" json:"metadata"`
	Spec       PluginSpec     `yaml:"spec" json:"spec"`
	Config     map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
	Dir        string         `yaml:"-" json:"-"` // Filesystem directory containing the manifest
}

// PluginMeta contains descriptive metadata about the plugin.
type PluginMeta struct {
	Name        string `yaml:"name" json:"name"`
	Version     string `yaml:"version" json:"version"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Author      string `yaml:"author,omitempty" json:"author,omitempty"`
	Homepage    string `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	Icon        string `yaml:"icon,omitempty" json:"icon,omitempty"`
}

// PluginWidget declares an optional modular dashboard widget provided by the plugin.
type PluginWidget struct {
	ID              string `yaml:"id" json:"id"`
	Title           string `yaml:"title" json:"title"`
	Subtitle        string `yaml:"subtitle,omitempty" json:"subtitle,omitempty"`
	Size            []int  `yaml:"size,omitempty" json:"size,omitempty"` // [colSpan, rowSpan], e.g. [2, 1] or [1, 1]
	Type            string `yaml:"type,omitempty" json:"type,omitempty"` // "card" or "embed"
	URL             string `yaml:"url,omitempty" json:"url,omitempty"`
	RefreshInterval string `yaml:"refresh_interval,omitempty" json:"refresh_interval,omitempty"`
	Endpoint        string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
}

// PluginSettingField defines a single configurable field for a plugin setting section.
type PluginSettingField struct {
	Key         string   `yaml:"key" json:"key"`
	Label       string   `yaml:"label" json:"label"`
	Type        string   `yaml:"type" json:"type"` // "text", "password", "number", "boolean", "select"
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Default     any      `yaml:"default,omitempty" json:"default,omitempty"`
	Options     []string `yaml:"options,omitempty" json:"options,omitempty"` // For "select"
	Placeholder string   `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
}

// PluginSettingSection defines a group of settings provided by a plugin.
type PluginSettingSection struct {
	ID          string               `yaml:"id" json:"id"`
	Plugin      string               `yaml:"plugin,omitempty" json:"plugin,omitempty"`
	Title       string               `yaml:"title" json:"title"`
	Description string               `yaml:"description,omitempty" json:"description,omitempty"`
	Icon        string               `yaml:"icon,omitempty" json:"icon,omitempty"`
	Fields      []PluginSettingField `yaml:"fields,omitempty" json:"fields,omitempty"`
}

// PluginSpec defines the execution model and capabilities for the plugin.
type PluginSpec struct {
	Command      string                 `yaml:"command" json:"command"`
	Args         []string               `yaml:"args,omitempty" json:"args,omitempty"`
	ProcessName  string                 `yaml:"process_name,omitempty" json:"process_name,omitempty"`
	Capabilities []Capability           `yaml:"capabilities" json:"capabilities"`
	PollInterval string                 `yaml:"poll_interval,omitempty" json:"poll_interval,omitempty"`
	UI           PluginUISpec           `yaml:"ui,omitempty" json:"ui,omitempty"`
	Widgets      []PluginWidget         `yaml:"widgets,omitempty" json:"widgets,omitempty"`
	Settings     []PluginSettingSection `yaml:"settings,omitempty" json:"settings,omitempty"`
}

// PluginUISpec declares static web UI configuration in plugin.yaml.
type PluginUISpec struct {
	Entry         string   `yaml:"entry,omitempty" json:"entry,omitempty"`
	Route         string   `yaml:"route,omitempty" json:"route,omitempty"`
	Category      string   `yaml:"category,omitempty" json:"category,omitempty"`
	CategoryTitle string   `yaml:"category_title,omitempty" json:"category_title,omitempty"`
	TabID         string   `yaml:"tab_id,omitempty" json:"tab_id,omitempty"`
	Title         string   `yaml:"title,omitempty" json:"title,omitempty"`
	Sidebar       bool     `yaml:"sidebar,omitempty" json:"sidebar,omitempty"`
	Order         int      `yaml:"order,omitempty" json:"order,omitempty"`
	Roles         []string `yaml:"roles,omitempty" json:"roles,omitempty"`
}

// HasCapability checks if the plugin declared a given capability.
func (m *PluginManifest) HasCapability(cap Capability) bool {
	for _, c := range m.Spec.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// --- JSON-RPC 2.0 Protocol Envelope ---

// RPCRequest is a standard JSON-RPC 2.0 request envelope.
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RPCResponse is a standard JSON-RPC 2.0 response envelope.
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError holds error code and message.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error [%d]: %s", e.Code, e.Message)
}

// --- Method Parameters & Results ---

// Method names supported by the M3TAL Core Plugin Protocol.
const (
	MethodInit    = "m3tal.init"
	MethodCollect = "m3tal.collect"
	MethodRender  = "m3tal.render"
	MethodAction  = "m3tal.action"
)

// InitParams passed from Core to Plugin during startup.
type InitParams struct {
	CoreVersion string         `json:"core_version"`
	Config      map[string]any `json:"config,omitempty"`
}

// InitResult returned by Plugin in response to m3tal.init.
type InitResult struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Capabilities []Capability           `json:"capabilities"`
	Tabs         []PluginTabSpec        `json:"tabs,omitempty"`
	Actions      []ActionSpec           `json:"actions,omitempty"`
	Widgets      []PluginWidget         `json:"widgets,omitempty"`
	Settings     []PluginSettingSection `json:"settings,omitempty"`
}

// PluginTabSpec defines a UI tab/viewport provided by a plugin.
type PluginTabSpec struct {
	ID            string       `json:"id"`
	Title         string       `json:"title"`
	Shortcut      string       `json:"shortcut,omitempty"`
	Description   string       `json:"description,omitempty"`
	Order         int          `json:"order,omitempty"`
	Actions       []ActionSpec `json:"actions,omitempty"`
	Plugin        string       `json:"plugin,omitempty"`
	URL           string       `json:"url,omitempty"`
	Category      string       `json:"category,omitempty"`
	CategoryTitle string       `json:"category_title,omitempty"`
}

// ActionSpec defines an executable action (e.g. restart service, flush cache).
type ActionSpec struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Keybind     string `json:"keybind,omitempty"`
	Description string `json:"description,omitempty"`
	Confirm     bool   `json:"confirm,omitempty"`
}

// CollectParams passed to plugin on collection tick.
type CollectParams struct {
	Timestamp int64 `json:"timestamp"`
}

// Metric represents a single measured data point.
type Metric struct {
	Name     string  `json:"name"`
	Value    float64 `json:"value"`
	Unit     string  `json:"unit,omitempty"`
	Category string  `json:"category,omitempty"`
	Status   string  `json:"status,omitempty"` // "ok", "warning", "critical"
}

// TableData represents structured tabular data for display in TUI or GUI.
type TableData struct {
	ID      string     `json:"id"`
	Title   string     `json:"title,omitempty"`
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// CollectResult returned by plugin on m3tal.collect.
type CollectResult struct {
	Status  string      `json:"status"` // "ok", "degraded", "error"
	Metrics []Metric    `json:"metrics,omitempty"`
	Tables  []TableData `json:"tables,omitempty"`
	Message string      `json:"message,omitempty"`
}

// RenderParams request TUI or GUI render output.
type RenderParams struct {
	TabID  string `json:"tab_id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Theme  string `json:"theme,omitempty"` // e.g. "dark", "matrix", "dracula"
}

// RenderResult returns the rendered content.
type RenderResult struct {
	Format  string `json:"format"` // "ansi", "plain", "json_tree"
	Content string `json:"content"`
}

// ActionParams request plugin to execute an action.
type ActionParams struct {
	ActionID string         `json:"action_id"`
	Target   string         `json:"target,omitempty"`
	Args     map[string]any `json:"args,omitempty"`
}

// ActionResult returns the outcome of an action while preserving
// plugin-specific fields in Payload.
type ActionResult struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Error   string         `json:"error,omitempty"`
	Data    any            `json:"data,omitempty"`
	Payload map[string]any `json:"payload,omitempty"`
}

// UnmarshalJSON preserves action-specific fields returned by plugins.
func (r *ActionResult) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	type actionResultAlias ActionResult
	var standard actionResultAlias
	if err := json.Unmarshal(data, &standard); err != nil {
		return err
	}

	r.Success = standard.Success
	r.Message = standard.Message
	r.Error = standard.Error
	r.Data = standard.Data
	r.Payload = make(map[string]any)

	if rawPayload, ok := fields["payload"]; ok {
		if err := json.Unmarshal(rawPayload, &r.Payload); err != nil {
			return err
		}
	}
	if r.Payload == nil {
		r.Payload = make(map[string]any)
	}

	for key, raw := range fields {
		switch key {
		case "success", "message", "error", "data", "payload":
			continue
		}

		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		r.Payload[key] = value
	}

	if len(r.Payload) == 0 {
		r.Payload = nil
	}
	return nil
}
