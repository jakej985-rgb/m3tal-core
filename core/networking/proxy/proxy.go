package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/plugins"
	"github.com/jakej985-rgb/m3tal-core/core/state"
	"github.com/jakej985-rgb/m3tal-core/core/traefik"
	"github.com/jakej985-rgb/m3tal-core/pkg/system"
	"gopkg.in/yaml.v3"
)

type DiscoverableService struct {
	ContainerID   string   `json:"container_id"`
	ContainerName string   `json:"container_name"`
	Image         string   `json:"image"`
	State         string   `json:"state"`
	Ports         []int    `json:"ports"`
	Networks      []string `json:"networks"`
	Exposed       bool     `json:"exposed"`
	Domain        string   `json:"domain,omitempty"`
	SSL           bool     `json:"ssl,omitempty"`
	Middlewares   []string `json:"middlewares,omitempty"`
}

// Manager orchestrates proxy route configuration, container discovery, and SSL setup.
type Manager struct {
	Store *state.Store
}

// NewManager creates a new proxy Manager.
func NewManager(db *state.Store) *Manager {
	return &Manager{Store: db}
}

// DiscoverServices queries Docker containers and returns reverse proxy candidates.
func (m *Manager) DiscoverServices() ([]DiscoverableService, error) {
	provider, err := docker.GetProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get container provider: %w", err)
	}

	containersList, err := provider.ListContainers()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var dbRoutes []state.RouteRecord
	if m.Store != nil {
		dbRoutes, err = m.Store.ListRoutes()
		if err != nil {
			return nil, fmt.Errorf("failed to list stored routes: %w", err)
		}
	}

	exposedMap := make(map[string]state.RouteRecord)
	for _, r := range dbRoutes {
		exposedMap[r.Service] = r
	}

	var services []DiscoverableService
	for _, c := range containersList {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		if name == "" {
			continue
		}

		// Don't list system proxy elements as candidates
		if name == "traefik" || name == "cloudflared" || name == "traefik-manager" {
			continue
		}

		var ports []int
		for _, p := range c.Ports {
			if p.PrivatePort > 0 {
				ports = append(ports, p.PrivatePort)
			}
		}

		// Deduplicate ports
		portMap := make(map[int]bool)
		var uniquePorts []int
		for _, p := range ports {
			if !portMap[p] {
				portMap[p] = true
				uniquePorts = append(uniquePorts, p)
			}
		}

		exposed := false
		domain := ""
		ssl := false
		var mws []string

		// Check DB first
		if r, ok := exposedMap[name]; ok {
			exposed = true
			domain = r.Domain
			ssl = r.SSL
			if r.Middlewares != "" {
				mws = strings.Split(r.Middlewares, ",")
			}
		} else {
			// Fallback check labels
			if c.Labels["traefik.enable"] == "true" {
				exposed = true
				for k, v := range c.Labels {
					if strings.Contains(k, ".rule") && strings.Contains(v, "Host(") {
						domain = extractDomain(v)
					}
					if strings.Contains(k, ".tls") {
						ssl = true
					}
					if strings.Contains(k, ".middlewares") {
						mws = strings.Split(v, ",")
					}
				}
			}
		}

		services = append(services, DiscoverableService{
			ContainerID:   c.ID,
			ContainerName: name,
			Image:         c.Image,
			State:         c.State,
			Ports:         uniquePorts,
			Networks:      c.Networks,
			Exposed:       exposed,
			Domain:        domain,
			SSL:           ssl,
			Middlewares:   mws,
		})
	}

	return services, nil
}

// ExposeService registers a route mapping in DB and as a Route plugin.
func (m *Manager) ExposeService(service string, domain string, port int, ssl bool, middlewares []string) error {
	mwStr := strings.Join(middlewares, ",")
	entrypoints := "web"
	if ssl {
		entrypoints = "web,websecure"
	}

	if m.Store != nil {
		existing, err := m.Store.GetRouteByDomain(domain)
		if err != nil {
			return err
		}
		if existing != nil {
			_ = m.Store.DeleteRoute(existing.ID)
			_ = m.DeletePluginFile(existing.Service, existing.Domain)
		}

		_, err = m.Store.CreateRoute(service, domain, port, entrypoints, "", ssl, mwStr)
		if err != nil {
			return fmt.Errorf("failed to save route to database: %w", err)
		}
	}

	err := m.WriteRoutePluginFile(service, domain, port, ssl, middlewares)
	if err != nil {
		return fmt.Errorf("failed to write route plugin: %w", err)
	}

	return m.SyncPlugins()
}

// UnexposeService deletes a route mapping from DB and plugin file.
func (m *Manager) UnexposeService(domain string) error {
	var serviceName string
	if m.Store != nil {
		r, err := m.Store.GetRouteByDomain(domain)
		if err != nil {
			return err
		}
		if r == nil {
			return fmt.Errorf("route for domain %s not found", domain)
		}
		serviceName = r.Service

		if err := m.Store.DeleteRoute(r.ID); err != nil {
			return err
		}
	} else {
		// Fallback parse from plugin files if no DB
		dirs := system.GetPluginDirs()
		reg, err := plugin.LoadAll(dirs...)
		if err == nil {
			for _, rp := range reg.Routes {
				if rp.Domain == domain {
					serviceName = rp.Service
					break
				}
			}
		}
	}

	if serviceName == "" {
		return fmt.Errorf("failed to determine service name for domain %s", domain)
	}

	_ = m.DeletePluginFile(serviceName, domain)

	return m.SyncPlugins()
}

// ConfigureSSL updates traefik.yml, routing-compose.yml, and deploys changes.
func (m *Manager) ConfigureSSL(email string) error {
	stackDir := system.GetStackDir()
	traefikYmlPath := filepath.Join(stackDir, "traefik.yml")

	// Read and parse traefik.yml
	data, err := os.ReadFile(traefikYmlPath)
	if err != nil {
		return fmt.Errorf("failed to read traefik.yml: %w", err)
	}

	var staticCfg TraefikStaticConfig
	if err := yaml.Unmarshal(data, &staticCfg); err != nil {
		return fmt.Errorf("failed to parse traefik.yml: %w", err)
	}

	// 1. Ensure EntryPoints
	if staticCfg.EntryPoints == nil {
		staticCfg.EntryPoints = make(map[string]*EntryPointConfig)
	}
	if _, ok := staticCfg.EntryPoints["web"]; !ok {
		staticCfg.EntryPoints["web"] = &EntryPointConfig{Address: ":80"}
	}
	staticCfg.EntryPoints["websecure"] = &EntryPointConfig{Address: ":443"}

	// 2. Configure Let's Encrypt certificatesResolvers
	if staticCfg.CertificatesResolvers == nil {
		staticCfg.CertificatesResolvers = make(map[string]*ResolverConfig)
	}
	staticCfg.CertificatesResolvers["letsencrypt"] = &ResolverConfig{
		ACME: &ACMEConfig{
			Email:   email,
			Storage: "/etc/traefik/acme.json",
			HTTPChallenge: &HTTPChallengeConfig{
				EntryPoint: "web",
			},
		},
	}

	// Write back traefik.yml
	updatedData, err := yaml.Marshal(&staticCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal traefik.yml: %w", err)
	}
	if err := os.WriteFile(traefikYmlPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write traefik.yml: %w", err)
	}

	// 3. Configure routing-compose.yml ports
	composePath := filepath.Join(stackDir, "routing-compose.yml")
	cf, err := traefik.ParseCompose(composePath)
	if err != nil {
		return fmt.Errorf("failed to parse routing-compose.yml: %w", err)
	}

	svc, ok := cf.Services["traefik"]
	if !ok {
		return fmt.Errorf("traefik service not found in routing-compose.yml")
	}

	has80 := false
	has443 := false
	for _, p := range svc.Ports {
		if strings.Contains(p, ":80") {
			has80 = true
		}
		if strings.Contains(p, ":443") {
			has443 = true
		}
	}

	if !has80 {
		svc.Ports = append(svc.Ports, "${TRAEFIK_WEB_PORT:-80}:80")
	}
	if !has443 {
		svc.Ports = append(svc.Ports, "${TRAEFIK_WEBHTTPS_PORT:-443}:443")
	}
	cf.Services["traefik"] = svc

	updatedCompose, err := yaml.Marshal(cf)
	if err != nil {
		return fmt.Errorf("failed to marshal routing-compose.yml: %w", err)
	}
	if err := os.WriteFile(composePath, updatedCompose, 0644); err != nil {
		return fmt.Errorf("failed to write routing-compose.yml: %w", err)
	}

	// 4. Redeploy routing stack
	_, err = docker.DeployStack(composePath, 0)
	if err != nil {
		return fmt.Errorf("failed to redeploy routing stack: %w", err)
	}

	return nil
}

// WriteRoutePluginFile generates the plugin route manifest file.
func (m *Manager) WriteRoutePluginFile(service string, domain string, port int, ssl bool, middlewares []string) error {
	userDir := system.GetUserPluginsDir()

	routesDir := filepath.Join(userDir, "routes")
	if err := os.MkdirAll(routesDir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s-%s.yml", service, sanitizeDomainFilename(domain))
	filePath := filepath.Join(routesDir, filename)

	type RouteMetadata struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description,omitempty"`
	}
	type RouteSpecData struct {
		Service     string   `yaml:"service"`
		Domain      string   `yaml:"domain"`
		Port        int      `yaml:"port"`
		Entrypoints string   `yaml:"entrypoints,omitempty"`
		Network     string   `yaml:"network,omitempty"`
		Middlewares []string `yaml:"middlewares,omitempty"`
		SSL         bool     `yaml:"ssl,omitempty"`
	}
	type RoutePluginData struct {
		APIVersion string        `yaml:"apiVersion"`
		Kind       string        `yaml:"kind"`
		Metadata   RouteMetadata `yaml:"metadata"`
		Spec       RouteSpecData `yaml:"spec"`
	}

	ep := "web"
	if ssl {
		ep = "web,websecure"
	}

	pData := RoutePluginData{
		APIVersion: "m3tal/v1",
		Kind:       "Route",
		Metadata: RouteMetadata{
			Name:        fmt.Sprintf("%s-%s", service, sanitizeDomainFilename(domain)),
			Description: fmt.Sprintf("Exposed route for %s on %s", service, domain),
		},
		Spec: RouteSpecData{
			Service:     service,
			Domain:      domain,
			Port:        port,
			Entrypoints: ep,
			Network:     "proxy",
			Middlewares: middlewares,
			SSL:         ssl,
		},
	}

	data, err := yaml.Marshal(pData)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// DeletePluginFile deletes the plugin route manifest file.
func (m *Manager) DeletePluginFile(service string, domain string) error {
	userDir := system.GetUserPluginsDir()

	filename := fmt.Sprintf("%s-%s.yml", service, sanitizeDomainFilename(domain))
	filePath := filepath.Join(userDir, "routes", filename)
	return os.Remove(filePath)
}

// SyncPlugins compiles all route and middleware plugins to generate dynamic config.
func (m *Manager) SyncPlugins() error {
	dirs := system.GetPluginDirs()
	reg, err := plugin.LoadAll(dirs...)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	configData, err := reg.GenerateTraefikConfig()
	if err != nil {
		return fmt.Errorf("failed to generate Traefik config: %w", err)
	}

	stackDir := system.GetStackDir()
	dynamicDir := filepath.Join(stackDir, "dynamic")
	if err := os.MkdirAll(dynamicDir, 0755); err != nil {
		return fmt.Errorf("failed to create dynamic directory: %w", err)
	}

	outputPath := filepath.Join(dynamicDir, "m3tal-plugins.yml")
	if err := os.WriteFile(outputPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func sanitizeDomainFilename(domain string) string {
	var b strings.Builder
	for _, r := range domain {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

func extractDomain(rule string) string {
	re := regexp.MustCompile(`Host\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	matches := re.FindStringSubmatch(rule)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// Traefik Static Configuration Structures for traefik.yml parsing
type TraefikStaticConfig struct {
	EntryPoints           map[string]*EntryPointConfig `yaml:"entryPoints"`
	API                   *APIConfig                   `yaml:"api,omitempty"`
	Providers             *ProvidersConfig             `yaml:"providers"`
	CertificatesResolvers map[string]*ResolverConfig   `yaml:"certificatesResolvers,omitempty"`
}

type EntryPointConfig struct {
	Address string `yaml:"address"`
}

type APIConfig struct {
	Dashboard bool `yaml:"dashboard"`
	Insecure  bool `yaml:"insecure"`
}

type ProvidersConfig struct {
	Docker *DockerProviderConfig `yaml:"docker,omitempty"`
	File   *FileProviderConfig   `yaml:"file,omitempty"`
}

type DockerProviderConfig struct {
	ExposedByDefault bool   `yaml:"exposedByDefault"`
	Network          string `yaml:"network,omitempty"`
}

type FileProviderConfig struct {
	Directory string `yaml:"directory"`
	Watch     bool   `yaml:"watch"`
}

type ResolverConfig struct {
	ACME *ACMEConfig `yaml:"acme"`
}

type ACMEConfig struct {
	Email         string               `yaml:"email"`
	Storage       string               `yaml:"storage"`
	HTTPChallenge *HTTPChallengeConfig `yaml:"httpChallenge,omitempty"`
}

type HTTPChallengeConfig struct {
	EntryPoint string `yaml:"entryPoint"`
}
