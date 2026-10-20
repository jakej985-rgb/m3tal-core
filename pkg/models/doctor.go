package models

import "fmt"

// Severity represents the outcome severity of a check.
type Severity string

const (
	SeverityPass Severity = "pass"
	SeverityWarn Severity = "warn"
	SeverityFail Severity = "fail"
)

// ContainerStatus classifies the overall health of a container.
type ContainerStatus string

const (
	StatusRunning    ContainerStatus = "running"
	StatusStopped    ContainerStatus = "stopped"
	StatusUnhealthy  ContainerStatus = "unhealthy"
	StatusRestarting ContainerStatus = "restarting"
	StatusCreated    ContainerStatus = "created"
	StatusPaused     ContainerStatus = "paused"
	StatusDead       ContainerStatus = "dead"
)

// ContainerResult holds the health scan result for a single container.
type ContainerResult struct {
	Name           string          `json:"name"`
	ID             string          `json:"id"`
	Image          string          `json:"image"`
	State          string          `json:"state"`
	Health         string          `json:"health"` // healthy / unhealthy / starting / none
	Restarts       int             `json:"restarts"`
	StartedAt      string          `json:"started_at"`
	FinishedAt     string          `json:"finished_at"`
	ExitCode       int             `json:"exit_code"`
	Status         ContainerStatus `json:"status"`
	Severity       Severity        `json:"severity"`
	Recommendation string          `json:"recommendation,omitempty"`
}

// MountType classifies a Docker mount.
type MountType string

const (
	MountTypeBind   MountType = "bind"
	MountTypeVolume MountType = "volume"
	MountTypeTmpfs  MountType = "tmpfs"
)

// MountResult captures the validation outcome for a single mount.
type MountResult struct {
	Container string    `json:"container"`
	Source    string    `json:"source"`
	Target    string    `json:"target"`
	Type      MountType `json:"type"`
	ReadOnly  bool      `json:"read_only"`
	Severity  Severity  `json:"severity"`
	Issue     string    `json:"issue,omitempty"`
	Fix       string    `json:"fix,omitempty"`
}

// PortResult captures the scan result for a single port.
type PortResult struct {
	Port       int      `json:"port"`
	InUse      bool     `json:"in_use"`
	OwnedBy    string   `json:"owned_by,omitempty"` // process name if detectable
	PID        int      `json:"pid,omitempty"`
	Conflict   bool     `json:"conflict"` // true when a service expects this port free
	Severity   Severity `json:"severity"`
	Suggestion int      `json:"suggestion,omitempty"` // next free port
	Note       string   `json:"note,omitempty"`
}

// Section groups checks of the same category.
type Section struct {
	Name      string   `json:"name"`
	Severity  Severity `json:"severity"`
	PassCount int      `json:"pass_count"`
	WarnCount int      `json:"warn_count"`
	FailCount int      `json:"fail_count"`
}

// DoctorReport is the aggregated output of all doctor scans.
type DoctorReport struct {
	GeneratedAt     string            `json:"generated_at"`
	OverallSeverity Severity          `json:"overall_severity"`
	Containers      []ContainerResult `json:"containers"`
	Mounts          []MountResult     `json:"mounts"`
	Ports           []PortResult      `json:"ports"`
	Sections        []Section         `json:"sections"`
	Recommendations []string          `json:"recommendations"`
}

// FixAction describes the type of remediation to apply.
type FixAction string

const (
	FixRestartContainer FixAction = "restart_container"
	FixChmodPath        FixAction = "chmod_path"
	FixMkdirPath        FixAction = "mkdir_path"
	FixSuggestPort      FixAction = "suggest_port"
)

// Fix represents a proposed (or applied) remediation action.
type Fix struct {
	Action      FixAction `json:"action"`
	Target      string    `json:"target"` // container name or path or port string
	Description string    `json:"description"`
	Command     string    `json:"command"` // human-readable command equivalent
	Applied     bool      `json:"applied"`
	Error       string    `json:"error,omitempty"`
}

// FixResult wraps a Fix with its execution outcome.
type FixResult struct {
	Fix
	Output string `json:"output,omitempty"`
}

// Icon returns an emoji for a given severity level.
func (s Severity) Icon() string {
	switch s {
	case SeverityPass:
		return "✅"
	case SeverityWarn:
		return "⚠️ "
	case SeverityFail:
		return "❌"
	default:
		return "❓"
	}
}

// SummaryLine returns a one-line status string for display.
func (r ContainerResult) SummaryLine() string {
	icon := r.Severity.Icon()
	health := ""
	if r.Health != "none" && r.Health != "" {
		health = fmt.Sprintf(" [health: %s]", r.Health)
	}
	restarts := ""
	if r.Restarts > 0 {
		restarts = fmt.Sprintf(" [restarts: %d]", r.Restarts)
	}
	return fmt.Sprintf("%s %-35s  state=%-12s%s%s", icon, r.Name, r.State, health, restarts)
}

// SummaryLine returns a one-line display string.
func (r MountResult) SummaryLine() string {
	icon := r.Severity.Icon()
	ro := ""
	if r.ReadOnly {
		ro = " [ro]"
	}
	issue := ""
	if r.Issue != "" {
		issue = "  ⤷ " + r.Issue
	}
	return fmt.Sprintf("%s [%s] %-25s → %s (%-5s)%s%s",
		icon, r.Container, r.Source, r.Target, string(r.Type), ro, issue)
}

// SummaryLine returns a one-line display string.
func (r PortResult) SummaryLine() string {
	icon := r.Severity.Icon()
	if r.InUse {
		return fmt.Sprintf("%s Port %-5d  IN USE by %-20s  → suggest port %d",
			icon, r.Port, fmt.Sprintf("%s (pid=%d)", r.OwnedBy, r.PID), r.Suggestion)
	}
	return fmt.Sprintf("%s Port %-5d  free", icon, r.Port)
}
