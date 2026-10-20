package traefik

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents a parsed docker-compose.yml
type ComposeFile struct {
	Services map[string]Service `yaml:"services"`
	Networks map[string]any     `yaml:"networks,omitempty"`
	Volumes  map[string]any     `yaml:"volumes,omitempty"`
}

// Service represents a single service in a compose file
type Service struct {
	Image         string      `yaml:"image"`
	Ports         []string    `yaml:"ports,omitempty"`
	Labels        yamlLabels  `yaml:"labels,omitempty"`
	Environment   yamlEnvList `yaml:"environment,omitempty"`
	Networks      yamlList    `yaml:"networks,omitempty"`
	VolumesRaw    []string    `yaml:"volumes,omitempty"`
	EnvFile       yamlList    `yaml:"env_file,omitempty"`
	ContainerName string      `yaml:"container_name,omitempty"`
	Restart       string      `yaml:"restart,omitempty"`
	Command       string      `yaml:"command,omitempty"`
}

// yamlLabels handles both map and list label formats in compose files.
// Docker compose supports:
//
//	labels:
//	  key: value          (map form)
//	labels:
//	  - "key=value"       (list form)
type yamlLabels struct {
	Values map[string]string
}

func (l *yamlLabels) UnmarshalYAML(value *yaml.Node) error {
	l.Values = make(map[string]string)

	switch value.Kind {
	case yaml.MappingNode:
		// map form: labels: { key: value }
		var m map[string]string
		if err := value.Decode(&m); err != nil {
			return err
		}
		l.Values = m

	case yaml.SequenceNode:
		// list form: labels: ["key=value"]
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		for _, item := range list {
			parts := strings.SplitN(item, "=", 2)
			if len(parts) == 2 {
				l.Values[parts[0]] = parts[1]
			} else {
				l.Values[item] = ""
			}
		}

	default:
		return fmt.Errorf("unsupported label format (kind=%d)", value.Kind)
	}

	return nil
}

// yamlList handles both string and list values for env_file, networks, etc.
type yamlList struct {
	Values []string
}

func (l *yamlList) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		l.Values = []string{value.Value}
	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		l.Values = list
	default:
		return fmt.Errorf("unsupported list format (kind=%d)", value.Kind)
	}
	return nil
}

// yamlEnvList handles environment as both list ["KEY=VAL"] and map {KEY: VAL}.
type yamlEnvList struct {
	Values map[string]string
}

func (e *yamlEnvList) UnmarshalYAML(value *yaml.Node) error {
	e.Values = make(map[string]string)

	switch value.Kind {
	case yaml.MappingNode:
		var m map[string]string
		if err := value.Decode(&m); err != nil {
			return err
		}
		e.Values = m

	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		for _, item := range list {
			parts := strings.SplitN(item, "=", 2)
			if len(parts) == 2 {
				e.Values[parts[0]] = parts[1]
			} else {
				e.Values[item] = ""
			}
		}

	default:
		return fmt.Errorf("unsupported environment format (kind=%d)", value.Kind)
	}

	return nil
}

// ParseCompose reads and parses a docker-compose.yml file.
func ParseCompose(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read compose file %s: %w", path, err)
	}
	return ParseComposeBytes(data)
}

// ParseComposeBytes parses compose YAML from raw bytes.
func ParseComposeBytes(data []byte) (*ComposeFile, error) {
	var cf ComposeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("invalid compose YAML: %w", err)
	}
	if cf.Services == nil {
		return nil, fmt.Errorf("compose file has no services defined")
	}
	return &cf, nil
}

// ServiceNames returns a sorted list of service names.
func (c *ComposeFile) ServiceNames() []string {
	names := make([]string, 0, len(c.Services))
	for name := range c.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetTraefikLabels extracts existing Traefik labels from a service.
// Returns nil if the service does not exist.
func (c *ComposeFile) GetTraefikLabels(service string) map[string]string {
	svc, ok := c.Services[service]
	if !ok {
		return nil
	}
	result := make(map[string]string)
	for k, v := range svc.Labels.Values {
		if strings.HasPrefix(k, "traefik.") {
			result[k] = v
		}
	}
	return result
}

// GetExposedPorts extracts host-side ports from a service's port mappings.
// It handles formats: "8080", "8080:80", "127.0.0.1:8080:80".
func (c *ComposeFile) GetExposedPorts(service string) []int {
	svc, ok := c.Services[service]
	if !ok {
		return nil
	}
	var ports []int
	for _, p := range svc.Ports {
		// Remove protocol suffix like /tcp, /udp
		p = strings.Split(p, "/")[0]
		// Remove quotes
		p = strings.Trim(p, "\"'")

		parts := strings.Split(p, ":")
		var hostPort string
		switch len(parts) {
		case 1:
			hostPort = parts[0]
		case 2:
			hostPort = parts[0]
		case 3:
			hostPort = parts[1] // ip:host:container
		default:
			continue
		}

		// Handle port ranges like "8000-8010"
		if strings.Contains(hostPort, "-") {
			rangeParts := strings.SplitN(hostPort, "-", 2)
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])
			if err1 == nil && err2 == nil {
				for i := start; i <= end; i++ {
					ports = append(ports, i)
				}
			}
			continue
		}

		port, err := strconv.Atoi(hostPort)
		if err == nil {
			ports = append(ports, port)
		}
	}
	return ports
}

// GetContainerPort extracts the internal container port from the first port mapping.
// Returns 0 if no ports are defined.
func (c *ComposeFile) GetContainerPort(service string) int {
	svc, ok := c.Services[service]
	if !ok || len(svc.Ports) == 0 {
		return 0
	}

	p := strings.Split(svc.Ports[0], "/")[0]
	p = strings.Trim(p, "\"'")
	parts := strings.Split(p, ":")

	var containerPort string
	switch len(parts) {
	case 1:
		containerPort = parts[0]
	case 2:
		containerPort = parts[1]
	case 3:
		containerPort = parts[2]
	default:
		return 0
	}

	port, err := strconv.Atoi(containerPort)
	if err != nil {
		return 0
	}
	return port
}

func (l yamlLabels) MarshalYAML() (interface{}, error) {
	return l.Values, nil
}

func (l yamlList) MarshalYAML() (interface{}, error) {
	return l.Values, nil
}

func (e yamlEnvList) MarshalYAML() (interface{}, error) {
	return e.Values, nil
}
