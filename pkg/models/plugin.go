package models

// Dependency represents a plugin dependency.
type Dependency struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Required    bool   `json:"required"`
	AutoInstall bool   `json:"autoInstall"`
}

// PluginMetadata contains descriptive information about a plugin.
type PluginMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version,omitempty"`
	Author      string   `json:"author,omitempty"`
	Provides    []string `json:"provides,omitempty"`
	Requires    []string `json:"requires,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
}

// Plugin represents a loaded plugin manifest returned by the API.
type Plugin struct {
	APIVersion   string         `json:"apiVersion"`
	Kind         string         `json:"kind"`
	Name         string         `json:"name,omitempty"`
	Version      string         `json:"version,omitempty"`
	Author       string         `json:"author,omitempty"`
	Description  string         `json:"description,omitempty"`
	Category     string         `json:"category,omitempty"`
	Subcategory  string         `json:"subcategory,omitempty"`
	Provider     string         `json:"provider,omitempty"`
	Provides     []string       `json:"provides,omitempty"`
	Requires     []string       `json:"requires,omitempty"`
	DependsOn    []string       `json:"depends_on,omitempty"`
	Dependencies []Dependency   `json:"dependencies,omitempty"`
	Metadata     PluginMetadata `json:"metadata"`
	Enabled      bool           `json:"enabled"`
	Warnings     []string       `json:"warnings,omitempty"`

	// Spec fields returned by the API
	Service  string            `json:"service,omitempty"`
	Domain   string            `json:"domain,omitempty"`
	Port     int               `json:"port,omitempty"`
	Priority int               `json:"priority,omitempty"`
	Type     string            `json:"type,omitempty"`
	Config   map[string]string `json:"config,omitempty"`
}

// PluginsResponse represents the response body of GET /api/v2/plugins.
type PluginsResponse struct {
	Summary    any      `json:"summary"`
	Routes     []Plugin `json:"routes"`
	Stacks     []Plugin `json:"stacks"`
	Middleware []Plugin `json:"middleware"`
}

// CatalogItem represents a plugin available in the remote official catalog.
type CatalogItem struct {
	Name         string       `json:"name"`
	Kind         string       `json:"kind"`
	Description  string       `json:"description"`
	Version      string       `json:"version"`
	Author       string       `json:"author"`
	URL          string       `json:"url"`
	ComposeURL   string       `json:"composeUrl,omitempty"`
	Category     string       `json:"category,omitempty"`
	Subcategory  string       `json:"subcategory,omitempty"`
	Provider     string       `json:"provider,omitempty"`
	Provides     []string     `json:"provides,omitempty"`
	Requires     []string     `json:"requires,omitempty"`
	DependsOn    []string     `json:"depends_on,omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
}

// CatalogItemStatus wraps CatalogItem with its local installation state.
type CatalogItemStatus struct {
	CatalogItem
	Installed bool   `json:"installed"`
	Status    string `json:"status"` // "enabled", "disabled", "not_installed"
}
