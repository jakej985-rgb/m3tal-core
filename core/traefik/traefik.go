package traefik

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// RouteInput is the core input for generating Traefik labels.
type RouteInput struct {
	Service     string `json:"service"`
	Domain      string `json:"domain"`
	Port        int    `json:"port"`
	Entrypoints string `json:"entrypoints,omitempty"` // default: "web"
	Network     string `json:"network,omitempty"`     // default: "proxy"
}

// MiddlewareInput describes a Traefik middleware configuration.
type MiddlewareInput struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`   // "basicauth", "ratelimit", "stripprefix", "headers"
	Config map[string]string `json:"config"` // type-specific key/value pairs
}

// GenerateRouteLabels produces the standard set of Traefik Docker labels
// for a given route input. This is the core differentiator — one clean
// opinionated way to route a service.
func GenerateRouteLabels(input RouteInput) map[string]string {
	entrypoints := input.Entrypoints
	if entrypoints == "" {
		entrypoints = "web"
	}
	network := input.Network
	if network == "" {
		network = "proxy"
	}

	svc := sanitizeName(input.Service)

	labels := map[string]string{
		"traefik.enable": "true",
		fmt.Sprintf("traefik.http.routers.%s.rule", svc):                      fmt.Sprintf("Host(`%s`)", input.Domain),
		fmt.Sprintf("traefik.http.routers.%s.entrypoints", svc):               entrypoints,
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", svc): fmt.Sprintf("%d", input.Port),
		"traefik.docker.network":                                              network,
	}

	return labels
}

// GenerateMiddlewareLabels produces Traefik middleware labels based on type.
//
// Supported types:
//   - "basicauth": requires "users" in config (htpasswd format)
//   - "ratelimit": requires "average" and optionally "burst"
//   - "stripprefix": requires "prefixes" (comma-separated)
//   - "headers": supports "stsSeconds", "browserXssFilter", etc.
func GenerateMiddlewareLabels(input MiddlewareInput) map[string]string {
	name := sanitizeName(input.Name)
	labels := make(map[string]string)

	prefix := fmt.Sprintf("traefik.http.middlewares.%s", name)

	switch input.Type {
	case "basicauth":
		if users, ok := input.Config["users"]; ok {
			labels[prefix+".basicauth.users"] = users
		}
		if usersFile, ok := input.Config["usersFile"]; ok {
			labels[prefix+".basicauth.usersfile"] = usersFile
		}

	case "ratelimit":
		if avg, ok := input.Config["average"]; ok {
			labels[prefix+".ratelimit.average"] = avg
		}
		if burst, ok := input.Config["burst"]; ok {
			labels[prefix+".ratelimit.burst"] = burst
		}

	case "stripprefix":
		if prefixes, ok := input.Config["prefixes"]; ok {
			labels[prefix+".stripprefix.prefixes"] = prefixes
		}

	case "headers":
		for k, v := range input.Config {
			labels[fmt.Sprintf("%s.headers.%s", prefix, k)] = v
		}
	}

	return labels
}

// InjectLabels reads a compose file, merges the given labels into the
// specified service's label block, and writes the result back to disk.
// Existing labels with the same key are overwritten.
func InjectLabels(composePath string, service string, labels map[string]string) error {
	cf, err := ParseCompose(composePath)
	if err != nil {
		return err
	}

	svc, ok := cf.Services[service]
	if !ok {
		return fmt.Errorf("service %q not found in %s", service, composePath)
	}

	// Initialize labels if nil
	if svc.Labels.Values == nil {
		svc.Labels.Values = make(map[string]string)
	}

	// Merge new labels (overwrite existing keys)
	for k, v := range labels {
		svc.Labels.Values[k] = v
	}

	cf.Services[service] = svc

	// Marshal back to YAML
	data, err := yaml.Marshal(cf)
	if err != nil {
		return fmt.Errorf("failed to marshal compose file: %w", err)
	}

	return os.WriteFile(composePath, data, 0644)
}

// AttachMiddleware links a middleware to an existing router in labels.
// It appends the middleware name to the router's middlewares list.
func AttachMiddleware(labels map[string]string, service string, middlewareName string) map[string]string {
	svc := sanitizeName(service)
	key := fmt.Sprintf("traefik.http.routers.%s.middlewares", svc)

	existing, ok := labels[key]
	if ok && existing != "" {
		// Append if not already present
		if !strings.Contains(existing, middlewareName) {
			labels[key] = existing + "," + middlewareName
		}
	} else {
		labels[key] = middlewareName
	}

	return labels
}

// sanitizeName converts a service name to a valid Traefik identifier.
// Replaces non-alphanumeric characters with hyphens.
func sanitizeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
