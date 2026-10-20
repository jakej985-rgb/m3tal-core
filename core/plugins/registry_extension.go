package plugin

import (
	"fmt"
	"strings"
)

// PluginHeader represents generic plugin metadata for dependency validation.
type PluginHeader struct {
	Name         string       `json:"name"`
	Kind         string       `json:"kind"`
	Enabled      bool         `json:"enabled"`
	Provides     []string     `json:"provides,omitempty"`
	Requires     []string     `json:"requires,omitempty"`
	DependsOn    []string     `json:"depends_on,omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
}

// AllHeaders returns a unified list of PluginHeader for all plugins in the registry.
func (r *Registry) AllHeaders() []PluginHeader {
	var headers []PluginHeader
	for _, p := range r.Routes {
		headers = append(headers, PluginHeader{
			Name:         p.Metadata.Name,
			Kind:         KindRoute,
			Enabled:      p.Enabled,
			Provides:     p.Provides,
			Requires:     p.Requires,
			DependsOn:    p.DependsOn,
			Dependencies: p.Dependencies,
		})
	}
	for _, p := range r.Stacks {
		headers = append(headers, PluginHeader{
			Name:         p.Metadata.Name,
			Kind:         KindStack,
			Enabled:      p.Enabled,
			Provides:     p.Provides,
			Requires:     p.Requires,
			DependsOn:    p.DependsOn,
			Dependencies: p.Dependencies,
		})
	}
	for _, p := range r.Middlewares {
		headers = append(headers, PluginHeader{
			Name:         p.Metadata.Name,
			Kind:         KindMiddleware,
			Enabled:      p.Enabled,
			Provides:     p.Provides,
			Requires:     p.Requires,
			DependsOn:    p.DependsOn,
			Dependencies: p.Dependencies,
		})
	}
	for _, p := range r.Services {
		headers = append(headers, PluginHeader{
			Name:         p.Metadata.Name,
			Kind:         KindService,
			Enabled:      p.Enabled,
			Provides:     p.Provides,
			Requires:     p.Requires,
			DependsOn:    p.DependsOn,
			Dependencies: p.Dependencies,
		})
	}
	return headers
}

// GetWarningsForHeader returns a list of dependency warning messages for a given PluginHeader.
func (r *Registry) GetWarningsForHeader(h PluginHeader) []string {
	var warnings []string
	headers := r.AllHeaders()

	// 1. Check DependsOn
	for _, depName := range h.DependsOn {
		found := false
		enabled := false
		for _, other := range headers {
			if strings.EqualFold(other.Name, depName) {
				found = true
				if other.Enabled {
					enabled = true
					break
				}
			}
		}
		if !found {
			warnings = append(warnings, fmt.Sprintf("Missing dependency plugin: %s", depName))
		} else if !enabled {
			warnings = append(warnings, fmt.Sprintf("Dependency plugin %s is disabled", depName))
		}
	}

	// 2. Check Requires
	for _, req := range h.Requires {
		provided := false
		disabledProvider := ""
		for _, other := range headers {
			for _, prov := range other.Provides {
				if strings.EqualFold(prov, req) {
					if other.Enabled {
						provided = true
						break
					} else {
						disabledProvider = other.Name
					}
				}
			}
			if provided {
				break
			}
		}
		if !provided {
			if disabledProvider != "" {
				warnings = append(warnings, fmt.Sprintf("Capability provider %s for %s is disabled", disabledProvider, req))
			} else {
				warnings = append(warnings, fmt.Sprintf("Missing provider for capability: %s", req))
			}
		}
	}

	// 3. Check Dependencies
	for _, dep := range h.Dependencies {
		if dep.Required {
			found := false
			enabled := false
			for _, other := range headers {
				if strings.EqualFold(other.Name, dep.Name) && strings.EqualFold(other.Kind, dep.Kind) {
					found = true
					if other.Enabled {
						enabled = true
						break
					}
				}
			}
			if !found {
				warnings = append(warnings, fmt.Sprintf("Missing required dependency: %s (%s)", dep.Name, dep.Kind))
			} else if !enabled {
				warnings = append(warnings, fmt.Sprintf("Required dependency %s (%s) is disabled", dep.Name, dep.Kind))
			}
		}
	}

	return warnings
}

// PopulateWarnings iterates through all plugins and populates their Warnings arrays.
func (r *Registry) PopulateWarnings() {
	headers := r.AllHeaders()
	warningsMap := make(map[string][]string) // key: kind/name
	for _, h := range headers {
		warningsMap[strings.ToLower(h.Kind+"/"+h.Name)] = r.GetWarningsForHeader(h)
	}

	for i := range r.Routes {
		r.Routes[i].Warnings = warningsMap[strings.ToLower(KindRoute+"/"+r.Routes[i].Metadata.Name)]
	}
	for i := range r.Stacks {
		r.Stacks[i].Warnings = warningsMap[strings.ToLower(KindStack+"/"+r.Stacks[i].Metadata.Name)]
	}
	for i := range r.Middlewares {
		r.Middlewares[i].Warnings = warningsMap[strings.ToLower(KindMiddleware+"/"+r.Middlewares[i].Metadata.Name)]
	}
	for i := range r.Services {
		r.Services[i].Warnings = warningsMap[strings.ToLower(KindService+"/"+r.Services[i].Metadata.Name)]
	}
}
