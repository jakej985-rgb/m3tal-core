package plugin

import (
	"fmt"
	"sort"
	"strings"
)

// Registry holds all loaded and typed plugins, indexed by kind.
type Registry struct {
	Routes      []RoutePlugin
	Stacks      []StackPlugin
	Middlewares []MiddlewarePlugin
	Services    []ServicePlugin
}

// BuildRegistry takes a list of validated plugins and constructs a typed Registry.
func BuildRegistry(plugins []Plugin) (*Registry, error) {
	r := &Registry{}

	for i := range plugins {
		p := &plugins[i]

		switch p.Kind {
		case KindRoute:
			rp, err := p.AsRoute()
			if err != nil {
				return nil, fmt.Errorf("plugin %q: %w", p.Metadata.Name, err)
			}
			r.Routes = append(r.Routes, *rp)

		case KindStack:
			sp, err := p.AsStack()
			if err != nil {
				return nil, fmt.Errorf("plugin %q: %w", p.Metadata.Name, err)
			}
			r.Stacks = append(r.Stacks, *sp)

		case KindMiddleware:
			mp, err := p.AsMiddleware()
			if err != nil {
				return nil, fmt.Errorf("plugin %q: %w", p.Metadata.Name, err)
			}
			r.Middlewares = append(r.Middlewares, *mp)

		case KindService:
			sp, err := p.AsService()
			if err != nil {
				return nil, fmt.Errorf("plugin %q: %w", p.Metadata.Name, err)
			}
			r.Services = append(r.Services, *sp)

		case KindTraefik:
			tp, err := p.AsTraefik()
			if err != nil {
				return nil, fmt.Errorf("plugin %q: %w", p.Metadata.Name, err)
			}
			// Map RouteSpecs to Registry.Routes
			for _, rs := range tp.Routes {
				routeName := rs.Name
				if routeName == "" {
					routeName = rs.Service
				}
				if routeName == "" {
					routeName = tp.Metadata.Name
				}
				r.Routes = append(r.Routes, RoutePlugin{
					Metadata: PluginMetadata{
						Name:        routeName,
						Description: tp.Metadata.Description,
						Version:     tp.Metadata.Version,
						Author:      tp.Metadata.Author,
					},
					Service:     rs.Service,
					Domain:      rs.Domain,
					Port:        rs.Port,
					Entrypoints: rs.Entrypoints,
					Network:     rs.Network,
					Middlewares: rs.Middlewares,
					SourcePath:  tp.SourcePath,
					Enabled:     tp.Enabled,
				})
			}
			// Map MiddlewareSpecs to Registry.Middlewares
			for _, ms := range tp.Middlewares {
				r.Middlewares = append(r.Middlewares, MiddlewarePlugin{
					Metadata: PluginMetadata{
						Name:        ms.Name,
						Description: tp.Metadata.Description,
						Version:     tp.Metadata.Version,
						Author:      tp.Metadata.Author,
					},
					Name:       ms.Name,
					Type:       ms.Type,
					Config:     ms.Config,
					SourcePath: tp.SourcePath,
					Enabled:    tp.Enabled,
				})
			}
		}
	}

	// Sort stacks by priority (lower = earlier)
	sort.Slice(r.Stacks, func(i, j int) bool {
		return r.Stacks[i].Priority < r.Stacks[j].Priority
	})

	// Sort routes, middlewares, and services by name for deterministic output
	sort.Slice(r.Routes, func(i, j int) bool {
		return r.Routes[i].Metadata.Name < r.Routes[j].Metadata.Name
	})
	sort.Slice(r.Middlewares, func(i, j int) bool {
		return r.Middlewares[i].Metadata.Name < r.Middlewares[j].Metadata.Name
	})
	sort.Slice(r.Services, func(i, j int) bool {
		return r.Services[i].Metadata.Name < r.Services[j].Metadata.Name
	})

	r.PopulateWarnings()
	return r, nil
}

// GetRoute returns a route plugin by metadata name, or nil if not found.
func (r *Registry) GetRoute(name string) *RoutePlugin {
	for i := range r.Routes {
		if r.Routes[i].Metadata.Name == name {
			return &r.Routes[i]
		}
	}
	return nil
}

// GetStack returns a stack plugin by metadata name, or nil if not found.
func (r *Registry) GetStack(name string) *StackPlugin {
	for i := range r.Stacks {
		if r.Stacks[i].Metadata.Name == name {
			return &r.Stacks[i]
		}
	}
	return nil
}

// GetMiddleware returns a middleware plugin by metadata name, or nil if not found.
func (r *Registry) GetMiddleware(name string) *MiddlewarePlugin {
	for i := range r.Middlewares {
		if r.Middlewares[i].Metadata.Name == name {
			return &r.Middlewares[i]
		}
	}
	return nil
}

// GetService returns a service plugin by metadata name, or nil if not found.
func (r *Registry) GetService(name string) *ServicePlugin {
	for i := range r.Services {
		if r.Services[i].Metadata.Name == name {
			return &r.Services[i]
		}
	}
	return nil
}

// ListRoutes returns all loaded route plugins.
func (r *Registry) ListRoutes() []RoutePlugin {
	if r.Routes == nil {
		return []RoutePlugin{}
	}
	return r.Routes
}

// ListStacks returns all loaded stack plugins, sorted by priority.
func (r *Registry) ListStacks() []StackPlugin {
	if r.Stacks == nil {
		return []StackPlugin{}
	}
	return r.Stacks
}

// ListMiddlewares returns all loaded middleware plugins.
func (r *Registry) ListMiddlewares() []MiddlewarePlugin {
	if r.Middlewares == nil {
		return []MiddlewarePlugin{}
	}
	return r.Middlewares
}

// ListServices returns all loaded service plugins.
func (r *Registry) ListServices() []ServicePlugin {
	if r.Services == nil {
		return []ServicePlugin{}
	}
	return r.Services
}

// Count returns the total number of plugins in the registry.
func (r *Registry) Count() int {
	return len(r.Routes) + len(r.Stacks) + len(r.Middlewares) + len(r.Services)
}

// Summary returns a human-readable summary of loaded plugins.
func (r *Registry) Summary() string {
	return fmt.Sprintf("%d plugins loaded (%d routes, %d stacks, %d middleware, %d services)",
		r.Count(), len(r.Routes), len(r.Stacks), len(r.Middlewares), len(r.Services))
}

// MatchService matches a service name/image/labels against Route plugins.
func (r *Registry) MatchService(serviceName string, image string, labels map[string]string) *RoutePlugin {
	// 1. Try matching by service name
	for i := range r.Routes {
		if strings.EqualFold(r.Routes[i].Service, serviceName) || strings.EqualFold(r.Routes[i].Metadata.Name, serviceName) {
			return &r.Routes[i]
		}
	}

	// 2. Try matching by image name
	if image != "" {
		parts := strings.Split(image, "/")
		lastPart := parts[len(parts)-1]
		imgName := strings.Split(lastPart, ":")[0]
		for i := range r.Routes {
			if strings.EqualFold(r.Routes[i].Service, imgName) || strings.EqualFold(r.Routes[i].Metadata.Name, imgName) {
				return &r.Routes[i]
			}
		}
	}

	// 3. Try matching by labels
	for k, v := range labels {
		for i := range r.Routes {
			if strings.Contains(k, r.Routes[i].Service) || strings.Contains(v, r.Routes[i].Service) {
				return &r.Routes[i]
			}
		}
	}

	return nil
}
