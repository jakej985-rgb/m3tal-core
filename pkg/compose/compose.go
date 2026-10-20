package compose

// ComposeConfig represents the root structure of a Docker Compose file.
type ComposeConfig struct {
	Version  string                    `yaml:"version,omitempty"`
	Services map[string]*ServiceConfig `yaml:"services"`
	Volumes  map[string]any            `yaml:"volumes,omitempty"`
	Networks map[string]any            `yaml:"networks,omitempty"`
}

// ServiceConfig represents the configuration of a single service.
type ServiceConfig struct {
	Image       string `yaml:"image,omitempty"`
	Build       any    `yaml:"build,omitempty"`
	Ports       []any  `yaml:"ports,omitempty"`
	Volumes     []any  `yaml:"volumes,omitempty"`
	Networks    any    `yaml:"networks,omitempty"`
	Restart     string `yaml:"restart,omitempty"`
	Environment any    `yaml:"environment,omitempty"`
	EnvFile     any    `yaml:"env_file,omitempty"`
	Labels      any    `yaml:"labels,omitempty"`
	DependsOn   any    `yaml:"depends_on,omitempty"`
	NetworkMode string `yaml:"network_mode,omitempty"`
}

// Severity represents the issue severity level.
type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARNING"
)

// LintIssue represents a single linting or validation issue.
type LintIssue struct {
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Service  string   `json:"service,omitempty"`
	Rule     string   `json:"rule"`
}
