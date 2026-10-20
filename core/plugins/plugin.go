package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Supported API versions and kinds.
const (
	APIVersion = "m3tal/v1"

	KindRoute      = "Route"
	KindStack      = "Stack"
	KindMiddleware = "Middleware"
	KindTraefik    = "Traefik"
	KindService    = "Service"
)

// Dependency represents a plugin dependency.
type Dependency struct {
	Name        string `yaml:"name" json:"name"`
	Kind        string `yaml:"kind" json:"kind"`
	Required    bool   `yaml:"required" json:"required"`
	AutoInstall bool   `yaml:"autoInstall" json:"autoInstall"`
}

// Plugin represents a loaded plugin manifest.
type Plugin struct {
	APIVersion   string         `yaml:"apiVersion" json:"apiVersion"`
	Kind         string         `yaml:"kind" json:"kind"`
	Name         string         `yaml:"name,omitempty" json:"name,omitempty"`
	Version      string         `yaml:"version,omitempty" json:"version,omitempty"`
	Author       string         `yaml:"author,omitempty" json:"author,omitempty"`
	Description  string         `yaml:"description,omitempty" json:"description,omitempty"`
	Category     string         `yaml:"category,omitempty" json:"category,omitempty"`
	Subcategory  string         `yaml:"subcategory,omitempty" json:"subcategory,omitempty"`
	Provider     string         `yaml:"provider,omitempty" json:"provider,omitempty"`
	Provides     []string       `yaml:"provides,omitempty" json:"provides,omitempty"`
	Requires     []string       `yaml:"requires,omitempty" json:"requires,omitempty"`
	DependsOn    []string       `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Dependencies []Dependency   `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Metadata     PluginMetadata `yaml:"metadata" json:"metadata"`
	Spec         map[string]any `yaml:"spec" json:"spec"`
	Hooks        *PluginHooks   `yaml:"hooks,omitempty" json:"hooks,omitempty"`

	// SourcePath is set by the loader to track where this plugin was loaded from.
	SourcePath string `yaml:"-" json:"-"`
	// Enabled indicates whether the plugin is active (no .disabled suffix).
	Enabled bool `yaml:"-" json:"enabled"`
	// Warnings contains any validation or dependency warnings.
	Warnings []string `yaml:"-" json:"warnings,omitempty"`
}

// GetName returns the plugin's name, checking the top-level name and nested metadata.name.
func (p *Plugin) GetName() string {
	if p.Name != "" {
		return p.Name
	}
	return p.Metadata.Name
}

// GetVersion returns the plugin's version.
func (p *Plugin) GetVersion() string {
	if p.Version != "" {
		return p.Version
	}
	return p.Metadata.Version
}

// GetAuthor returns the plugin's author.
func (p *Plugin) GetAuthor() string {
	if p.Author != "" {
		return p.Author
	}
	return p.Metadata.Author
}

// GetDescription returns the plugin's description.
func (p *Plugin) GetDescription() string {
	if p.Description != "" {
		return p.Description
	}
	return p.Metadata.Description
}

// PluginHooks defines scripts to run during various plugin lifecycle stages.
type PluginHooks struct {
	PreInstall    string `yaml:"pre-install,omitempty" json:"pre-install,omitempty"`
	PostInstall   string `yaml:"post-install,omitempty" json:"post-install,omitempty"`
	PreEnable     string `yaml:"pre-enable,omitempty" json:"pre-enable,omitempty"`
	PostEnable    string `yaml:"post-enable,omitempty" json:"post-enable,omitempty"`
	PreDisable    string `yaml:"pre-disable,omitempty" json:"pre-disable,omitempty"`
	PostDisable   string `yaml:"post-disable,omitempty" json:"post-disable,omitempty"`
	PreUninstall  string `yaml:"pre-uninstall,omitempty" json:"pre-uninstall,omitempty"`
	PostUninstall string `yaml:"post-uninstall,omitempty" json:"post-uninstall,omitempty"`
}

// PluginMetadata contains descriptive information about a plugin.
type PluginMetadata struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Version     string   `yaml:"version,omitempty" json:"version,omitempty"`
	Author      string   `yaml:"author,omitempty" json:"author,omitempty"`
	Provides    []string `yaml:"provides,omitempty" json:"provides,omitempty"`
	Requires    []string `yaml:"requires,omitempty" json:"requires,omitempty"`
	DependsOn   []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
}

// RoutePlugin is the typed representation of a Route plugin spec.
type RoutePlugin struct {
	Metadata     PluginMetadata `yaml:"metadata" json:"metadata"`
	Service      string         `yaml:"service" json:"service"`
	Domain       string         `yaml:"domain" json:"domain"`
	Port         int            `yaml:"port" json:"port"`
	Entrypoints  string         `yaml:"entrypoints,omitempty" json:"entrypoints,omitempty"`
	Network      string         `yaml:"network,omitempty" json:"network,omitempty"`
	Middlewares  []string       `yaml:"middlewares,omitempty" json:"middlewares,omitempty"`
	SSL          bool           `yaml:"ssl,omitempty" json:"ssl,omitempty"`
	Category     string         `yaml:"category,omitempty" json:"category,omitempty"`
	Subcategory  string         `yaml:"subcategory,omitempty" json:"subcategory,omitempty"`
	Provider     string         `yaml:"provider,omitempty" json:"provider,omitempty"`
	Provides     []string       `yaml:"provides,omitempty" json:"provides,omitempty"`
	Requires     []string       `yaml:"requires,omitempty" json:"requires,omitempty"`
	DependsOn    []string       `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Dependencies []Dependency   `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	SourcePath   string         `yaml:"-" json:"-"`
	Enabled      bool           `yaml:"-" json:"enabled"`
	Warnings     []string       `yaml:"-" json:"warnings,omitempty"`
}

// StackPlugin is the typed representation of a Stack plugin spec.
type StackPlugin struct {
	Metadata     PluginMetadata `yaml:"metadata" json:"metadata"`
	ComposePath  string         `yaml:"composePath" json:"composePath"`
	EnvTemplate  string         `yaml:"envTemplate,omitempty" json:"envTemplate,omitempty"`
	Priority     int            `yaml:"priority,omitempty" json:"priority,omitempty"`
	Category     string         `yaml:"category,omitempty" json:"category,omitempty"`
	Subcategory  string         `yaml:"subcategory,omitempty" json:"subcategory,omitempty"`
	Provider     string         `yaml:"provider,omitempty" json:"provider,omitempty"`
	Provides     []string       `yaml:"provides,omitempty" json:"provides,omitempty"`
	Requires     []string       `yaml:"requires,omitempty" json:"requires,omitempty"`
	DependsOn    []string       `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Dependencies []Dependency   `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	SourcePath   string         `yaml:"-" json:"-"`
	Enabled      bool           `yaml:"-" json:"enabled"`
	Warnings     []string       `yaml:"-" json:"warnings,omitempty"`
}

// MiddlewarePlugin is the typed representation of a Middleware plugin spec.
type MiddlewarePlugin struct {
	Metadata     PluginMetadata    `yaml:"metadata" json:"metadata"`
	Name         string            `yaml:"name" json:"name"`
	Type         string            `yaml:"type" json:"type"`
	Config       map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
	Category     string            `yaml:"category,omitempty" json:"category,omitempty"`
	Subcategory  string            `yaml:"subcategory,omitempty" json:"subcategory,omitempty"`
	Provider     string            `yaml:"provider,omitempty" json:"provider,omitempty"`
	Provides     []string          `yaml:"provides,omitempty" json:"provides,omitempty"`
	Requires     []string          `yaml:"requires,omitempty" json:"requires,omitempty"`
	DependsOn    []string          `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Dependencies []Dependency      `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	SourcePath   string            `yaml:"-" json:"-"`
	Enabled      bool              `yaml:"-" json:"enabled"`
	Warnings     []string          `yaml:"-" json:"warnings,omitempty"`
}

// TraefikPlugin is the typed representation of a Traefik plugin spec.
type TraefikPlugin struct {
	Metadata     PluginMetadata   `yaml:"metadata" json:"metadata"`
	Routes       []RouteSpec      `yaml:"routes,omitempty" json:"routes,omitempty"`
	Middlewares  []MiddlewareSpec `yaml:"middlewares,omitempty" json:"middlewares,omitempty"`
	Category     string           `yaml:"category,omitempty" json:"category,omitempty"`
	Subcategory  string           `yaml:"subcategory,omitempty" json:"subcategory,omitempty"`
	Provider     string           `yaml:"provider,omitempty" json:"provider,omitempty"`
	Provides     []string         `yaml:"provides,omitempty" json:"provides,omitempty"`
	Requires     []string         `yaml:"requires,omitempty" json:"requires,omitempty"`
	DependsOn    []string         `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Dependencies []Dependency     `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	SourcePath   string           `yaml:"-" json:"-"`
	Enabled      bool             `yaml:"-" json:"enabled"`
	Warnings     []string         `yaml:"-" json:"warnings,omitempty"`
}

// ServicePlugin is the typed representation of a Service plugin spec.
type ServicePlugin struct {
	Metadata     PluginMetadata `yaml:"metadata" json:"metadata"`
	Image        string         `yaml:"image,omitempty" json:"image,omitempty"`
	Ports        []string       `yaml:"ports,omitempty" json:"ports,omitempty"`
	Env          []string       `yaml:"env,omitempty" json:"env,omitempty"`
	Volumes      []string       `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Labels       []string       `yaml:"labels,omitempty" json:"labels,omitempty"`
	Category     string         `yaml:"category,omitempty" json:"category,omitempty"`
	Subcategory  string         `yaml:"subcategory,omitempty" json:"subcategory,omitempty"`
	Provider     string         `yaml:"provider,omitempty" json:"provider,omitempty"`
	Provides     []string       `yaml:"provides,omitempty" json:"provides,omitempty"`
	Requires     []string       `yaml:"requires,omitempty" json:"requires,omitempty"`
	DependsOn    []string       `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Dependencies []Dependency   `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	SourcePath   string         `yaml:"-" json:"-"`
	Enabled      bool           `yaml:"-" json:"enabled"`
	Warnings     []string       `yaml:"-" json:"warnings,omitempty"`
}

// RouteSpec represents a route configuration inside a Traefik plugin.
type RouteSpec struct {
	Name        string   `yaml:"name" json:"name"`
	Service     string   `yaml:"service" json:"service"`
	Domain      string   `yaml:"domain" json:"domain"`
	Port        int      `yaml:"port" json:"port"`
	Entrypoints string   `yaml:"entrypoints,omitempty" json:"entrypoints,omitempty"`
	Network     string   `yaml:"network,omitempty" json:"network,omitempty"`
	Middlewares []string `yaml:"middlewares,omitempty" json:"middlewares,omitempty"`
	SSL         bool     `yaml:"ssl,omitempty" json:"ssl,omitempty"`
}

// MiddlewareSpec represents a middleware configuration inside a Traefik plugin.
type MiddlewareSpec struct {
	Name   string            `yaml:"name" json:"name"`
	Type   string            `yaml:"type" json:"type"`
	Config map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
}

// ValidCategories is the set of allowed categories.
var ValidCategories = map[string]bool{
	"system":     true,
	"network":    true,
	"ai":         true,
	"dev":        true,
	"monitoring": true,
	"security":   true,
	"storage":    true,
	"automation": true,
}

// ValidProviders is the set of allowed providers.
var ValidProviders = map[string]bool{
	"traefik": true,
	"docker":  true,
	"caddy":   true,
	"nginx":   true,
	"native":  true,
}

// Validate checks a raw Plugin for structural correctness.
func (p *Plugin) Validate() error {
	if p.APIVersion != APIVersion {
		return fmt.Errorf("unsupported apiVersion %q (expected %q)", p.APIVersion, APIVersion)
	}

	switch p.Kind {
	case KindRoute, KindStack, KindMiddleware, KindTraefik, KindService:
		// valid
	default:
		return fmt.Errorf("unsupported kind %q (expected one of: %s)",
			p.Kind, strings.Join([]string{KindRoute, KindStack, KindMiddleware, KindTraefik, KindService}, ", "))
	}

	name := p.GetName()
	if name == "" {
		return fmt.Errorf("metadata.name or top-level name is required")
	}

	category := strings.ToLower(p.Category)
	if category != "" && !ValidCategories[category] {
		return fmt.Errorf("plugin %q: invalid category %q (must be one of: system, network, ai, dev, monitoring, security, storage, automation)", name, p.Category)
	}

	provider := strings.ToLower(p.Provider)
	if provider != "" && !ValidProviders[provider] {
		return fmt.Errorf("plugin %q: invalid provider %q (must be one of: traefik, docker, caddy, nginx, native)", name, p.Provider)
	}

	for i, dep := range p.Dependencies {
		if dep.Name == "" {
			return fmt.Errorf("plugin %q: dependency[%d] name is required", name, i)
		}
		if dep.Kind == "" {
			return fmt.Errorf("plugin %q: dependency[%d] kind is required", name, i)
		}
	}

	return p.validateSpec()
}

// validateSpec performs kind-specific spec checks.
func (p *Plugin) validateSpec() error {
	if p.Spec == nil {
		return fmt.Errorf("spec is required")
	}

	switch p.Kind {
	case KindService:
		// Service validation (optional spec validation)
	case KindRoute:
		for _, required := range []string{"service", "domain", "port"} {
			if _, ok := p.Spec[required]; !ok {
				return fmt.Errorf("Route plugin %q: spec.%s is required", p.Metadata.Name, required)
			}
		}
	case KindStack:
		if _, ok := p.Spec["composePath"]; !ok {
			return fmt.Errorf("Stack plugin %q: spec.composePath is required", p.Metadata.Name)
		}
	case KindMiddleware:
		for _, required := range []string{"name", "type"} {
			if _, ok := p.Spec[required]; !ok {
				return fmt.Errorf("Middleware plugin %q: spec.%s is required", p.Metadata.Name, required)
			}
		}
	case KindTraefik:
		if routes, ok := p.Spec["routes"]; ok {
			routesList, ok := routes.([]any)
			if !ok {
				return fmt.Errorf("Traefik plugin %q: spec.routes must be a list", p.Metadata.Name)
			}
			for i, r := range routesList {
				rMap, ok := r.(map[string]any)
				if !ok {
					// Also support map[any]any in case yaml unmarshals it that way
					if rMapAny, ok := r.(map[any]any); ok {
						rMap = make(map[string]any)
						for k, v := range rMapAny {
							rMap[fmt.Sprintf("%v", k)] = v
						}
					} else {
						return fmt.Errorf("Traefik plugin %q: spec.routes[%d] must be a map", p.Metadata.Name, i)
					}
				}
				for _, required := range []string{"service", "domain", "port"} {
					if _, ok := rMap[required]; !ok {
						return fmt.Errorf("Traefik plugin %q routes[%d]: %s is required", p.Metadata.Name, i, required)
					}
				}
			}
		}
		if middlewares, ok := p.Spec["middlewares"]; ok {
			middlewaresList, ok := middlewares.([]any)
			if !ok {
				return fmt.Errorf("Traefik plugin %q: spec.middlewares must be a list", p.Metadata.Name)
			}
			for i, m := range middlewaresList {
				mMap, ok := m.(map[string]any)
				if !ok {
					if mMapAny, ok := m.(map[any]any); ok {
						mMap = make(map[string]any)
						for k, v := range mMapAny {
							mMap[fmt.Sprintf("%v", k)] = v
						}
					} else {
						return fmt.Errorf("Traefik plugin %q: spec.middlewares[%d] must be a map", p.Metadata.Name, i)
					}
				}
				for _, required := range []string{"name", "type"} {
					if _, ok := mMap[required]; !ok {
						return fmt.Errorf("Traefik plugin %q middlewares[%d]: %s is required", p.Metadata.Name, i, required)
					}
				}
			}
		}
	}

	return nil
}

// AsRoute converts a validated Route plugin to its typed struct.
func (p *Plugin) AsRoute() (*RoutePlugin, error) {
	if p.Kind != KindRoute {
		return nil, fmt.Errorf("cannot convert %s plugin to RoutePlugin", p.Kind)
	}

	// Re-marshal spec to YAML then decode into typed struct.
	data, err := yaml.Marshal(p.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %w", err)
	}

	var rp RoutePlugin
	if err := yaml.Unmarshal(data, &rp); err != nil {
		return nil, fmt.Errorf("failed to decode Route spec: %w", err)
	}
	rp.Metadata.Name = p.GetName()
	rp.Metadata.Description = p.GetDescription()
	rp.Metadata.Version = p.GetVersion()
	rp.Metadata.Author = p.GetAuthor()
	if p.Category != "" {
		rp.Category = p.Category
	}
	if p.Subcategory != "" {
		rp.Subcategory = p.Subcategory
	}
	if p.Provider != "" {
		rp.Provider = p.Provider
	}
	if len(p.Provides) > 0 {
		rp.Provides = p.Provides
	} else {
		rp.Provides = p.Metadata.Provides
	}
	if len(p.Requires) > 0 {
		rp.Requires = p.Requires
	} else {
		rp.Requires = p.Metadata.Requires
	}
	if len(p.DependsOn) > 0 {
		rp.DependsOn = p.DependsOn
	} else {
		rp.DependsOn = p.Metadata.DependsOn
	}
	if len(p.Dependencies) > 0 {
		rp.Dependencies = p.Dependencies
	}
	rp.SourcePath = p.SourcePath
	rp.Enabled = p.Enabled
	rp.Warnings = p.Warnings
	return &rp, nil
}

// AsStack converts a validated Stack plugin to its typed struct.
func (p *Plugin) AsStack() (*StackPlugin, error) {
	if p.Kind != KindStack {
		return nil, fmt.Errorf("cannot convert %s plugin to StackPlugin", p.Kind)
	}

	data, err := yaml.Marshal(p.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %w", err)
	}

	var sp StackPlugin
	if err := yaml.Unmarshal(data, &sp); err != nil {
		return nil, fmt.Errorf("failed to decode Stack spec: %w", err)
	}
	sp.Metadata.Name = p.GetName()
	sp.Metadata.Description = p.GetDescription()
	sp.Metadata.Version = p.GetVersion()
	sp.Metadata.Author = p.GetAuthor()
	if p.Category != "" {
		sp.Category = p.Category
	}
	if p.Subcategory != "" {
		sp.Subcategory = p.Subcategory
	}
	if p.Provider != "" {
		sp.Provider = p.Provider
	}
	if len(p.Provides) > 0 {
		sp.Provides = p.Provides
	} else {
		sp.Provides = p.Metadata.Provides
	}
	if len(p.Requires) > 0 {
		sp.Requires = p.Requires
	} else {
		sp.Requires = p.Metadata.Requires
	}
	if len(p.DependsOn) > 0 {
		sp.DependsOn = p.DependsOn
	} else {
		sp.DependsOn = p.Metadata.DependsOn
	}
	if len(p.Dependencies) > 0 {
		sp.Dependencies = p.Dependencies
	}
	sp.SourcePath = p.SourcePath
	sp.Enabled = p.Enabled
	sp.Warnings = p.Warnings
	return &sp, nil
}

// AsMiddleware converts a validated Middleware plugin to its typed struct.
func (p *Plugin) AsMiddleware() (*MiddlewarePlugin, error) {
	if p.Kind != KindMiddleware {
		return nil, fmt.Errorf("cannot convert %s plugin to MiddlewarePlugin", p.Kind)
	}

	data, err := yaml.Marshal(p.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %w", err)
	}

	var mp MiddlewarePlugin
	if err := yaml.Unmarshal(data, &mp); err != nil {
		return nil, fmt.Errorf("failed to decode Middleware spec: %w", err)
	}
	mp.Metadata.Name = p.GetName()
	mp.Metadata.Description = p.GetDescription()
	mp.Metadata.Version = p.GetVersion()
	mp.Metadata.Author = p.GetAuthor()
	if p.Category != "" {
		mp.Category = p.Category
	}
	if p.Subcategory != "" {
		mp.Subcategory = p.Subcategory
	}
	if p.Provider != "" {
		mp.Provider = p.Provider
	}
	if len(p.Provides) > 0 {
		mp.Provides = p.Provides
	} else {
		mp.Provides = p.Metadata.Provides
	}
	if len(p.Requires) > 0 {
		mp.Requires = p.Requires
	} else {
		mp.Requires = p.Metadata.Requires
	}
	if len(p.DependsOn) > 0 {
		mp.DependsOn = p.DependsOn
	} else {
		mp.DependsOn = p.Metadata.DependsOn
	}
	if len(p.Dependencies) > 0 {
		mp.Dependencies = p.Dependencies
	}
	mp.SourcePath = p.SourcePath
	mp.Enabled = p.Enabled
	mp.Warnings = p.Warnings
	return &mp, nil
}

// AsTraefik converts a validated Traefik plugin to its typed struct.
func (p *Plugin) AsTraefik() (*TraefikPlugin, error) {
	if p.Kind != KindTraefik {
		return nil, fmt.Errorf("cannot convert %s plugin to TraefikPlugin", p.Kind)
	}

	data, err := yaml.Marshal(p.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %w", err)
	}

	var tp TraefikPlugin
	if err := yaml.Unmarshal(data, &tp); err != nil {
		return nil, fmt.Errorf("failed to decode Traefik spec: %w", err)
	}
	tp.Metadata.Name = p.GetName()
	tp.Metadata.Description = p.GetDescription()
	tp.Metadata.Version = p.GetVersion()
	tp.Metadata.Author = p.GetAuthor()
	if p.Category != "" {
		tp.Category = p.Category
	}
	if p.Subcategory != "" {
		tp.Subcategory = p.Subcategory
	}
	if p.Provider != "" {
		tp.Provider = p.Provider
	}
	if len(p.Provides) > 0 {
		tp.Provides = p.Provides
	} else {
		tp.Provides = p.Metadata.Provides
	}
	if len(p.Requires) > 0 {
		tp.Requires = p.Requires
	} else {
		tp.Requires = p.Metadata.Requires
	}
	if len(p.DependsOn) > 0 {
		tp.DependsOn = p.DependsOn
	} else {
		tp.DependsOn = p.Metadata.DependsOn
	}
	if len(p.Dependencies) > 0 {
		tp.Dependencies = p.Dependencies
	}
	tp.SourcePath = p.SourcePath
	tp.Enabled = p.Enabled
	tp.Warnings = p.Warnings
	return &tp, nil
}

// AsService converts a validated Service plugin to its typed struct.
func (p *Plugin) AsService() (*ServicePlugin, error) {
	if p.Kind != KindService {
		return nil, fmt.Errorf("cannot convert %s plugin to ServicePlugin", p.Kind)
	}

	data, err := yaml.Marshal(p.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %w", err)
	}

	var sp ServicePlugin
	if err := yaml.Unmarshal(data, &sp); err != nil {
		return nil, fmt.Errorf("failed to decode Service spec: %w", err)
	}
	sp.Metadata.Name = p.GetName()
	sp.Metadata.Description = p.GetDescription()
	sp.Metadata.Version = p.GetVersion()
	sp.Metadata.Author = p.GetAuthor()
	if p.Category != "" {
		sp.Category = p.Category
	}
	if p.Subcategory != "" {
		sp.Subcategory = p.Subcategory
	}
	if p.Provider != "" {
		sp.Provider = p.Provider
	}
	if len(p.Provides) > 0 {
		sp.Provides = p.Provides
	} else {
		sp.Provides = p.Metadata.Provides
	}
	if len(p.Requires) > 0 {
		sp.Requires = p.Requires
	} else {
		sp.Requires = p.Metadata.Requires
	}
	if len(p.DependsOn) > 0 {
		sp.DependsOn = p.DependsOn
	} else {
		sp.DependsOn = p.Metadata.DependsOn
	}
	if len(p.Dependencies) > 0 {
		sp.Dependencies = p.Dependencies
	}
	sp.SourcePath = p.SourcePath
	sp.Enabled = p.Enabled
	sp.Warnings = p.Warnings
	return &sp, nil
}

// ParsePlugin parses raw YAML bytes into a Plugin.
func ParsePlugin(data []byte) (*Plugin, error) {
	var p Plugin
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("invalid plugin YAML: %w", err)
	}
	return &p, nil
}

// ParsePluginJSON parses raw JSON bytes into a Plugin.
func ParsePluginJSON(data []byte) (*Plugin, error) {
	var p Plugin
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("invalid plugin JSON: %w", err)
	}
	return &p, nil
}
