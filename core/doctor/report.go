package doctor

import (
	"encoding/json"
	"fmt"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"os"
	"strings"
	"time"
)

type Section = models.Section
type Report = models.DoctorReport

const (
	SeverityPass = models.SeverityPass
	SeverityWarn = models.SeverityWarn
	SeverityFail = models.SeverityFail
)

// severityIcon returns an emoji for a given severity level.
func severityIcon(s Severity) string {
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

// GenerateReport aggregates all scan results into a single Report.
func GenerateReport(conts []ContainerResult, mounts []MountResult, ports []PortResult) Report {
	r := Report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Containers:  conts,
		Mounts:      mounts,
		Ports:       ports,
	}

	r.Sections = []Section{
		buildSection("Containers", severitiesFromContainers(conts)),
		buildSection("Mounts", severitiesFromMounts(mounts)),
		buildSection("Ports", severitiesFromPorts(ports)),
	}

	r.OverallSeverity = computeOverallSeverity(r.Sections)
	r.Recommendations = collectRecommendations(conts, mounts, ports)

	return r
}

func buildSection(name string, sevs []Severity) Section {
	s := Section{Name: name}
	for _, sv := range sevs {
		switch sv {
		case SeverityPass:
			s.PassCount++
		case SeverityWarn:
			s.WarnCount++
		case SeverityFail:
			s.FailCount++
		}
	}
	if s.FailCount > 0 {
		s.Severity = SeverityFail
	} else if s.WarnCount > 0 {
		s.Severity = SeverityWarn
	} else {
		s.Severity = SeverityPass
	}
	return s
}

func severitiesFromContainers(cs []ContainerResult) []Severity {
	var out []Severity
	for _, c := range cs {
		out = append(out, c.Severity)
	}
	return out
}

func severitiesFromMounts(ms []MountResult) []Severity {
	var out []Severity
	for _, m := range ms {
		out = append(out, m.Severity)
	}
	return out
}

func severitiesFromPorts(ps []PortResult) []Severity {
	var out []Severity
	for _, p := range ps {
		out = append(out, p.Severity)
	}
	return out
}

func computeOverallSeverity(sections []Section) Severity {
	worst := SeverityPass
	for _, s := range sections {
		if s.Severity == SeverityFail {
			return SeverityFail
		}
		if s.Severity == SeverityWarn {
			worst = SeverityWarn
		}
	}
	return worst
}

func collectRecommendations(conts []ContainerResult, mounts []MountResult, ports []PortResult) []string {
	seen := map[string]bool{}
	var recs []string
	add := func(s string) {
		if s != "" && !seen[s] {
			seen[s] = true
			recs = append(recs, s)
		}
	}
	for _, c := range conts {
		add(c.Recommendation)
	}
	for _, m := range mounts {
		if m.Fix != "" {
			add(fmt.Sprintf("[mount] %s: %s", m.Container, m.Fix))
		}
	}
	for _, p := range ports {
		if p.Note != "" {
			add("[port] " + p.Note)
		}
	}
	return recs
}

// PrintReport outputs a rich, human-readable health report to stdout.
func PrintReport(r Report) {
	overall := severityIcon(r.OverallSeverity)
	fmt.Printf("\n%s M3TAL Doctor Report — %s\n", overall, r.GeneratedAt)
	fmt.Println(strings.Repeat("═", 60))

	// Section summaries
	for _, s := range r.Sections {
		icon := severityIcon(s.Severity)
		fmt.Printf("  %s %-12s  pass=%-3d warn=%-3d fail=%d\n",
			icon, s.Name, s.PassCount, s.WarnCount, s.FailCount)
	}
	fmt.Println(strings.Repeat("─", 60))

	// Container details
	if len(r.Containers) > 0 {
		fmt.Println("\n📦 Containers")
		for _, c := range r.Containers {
			if c.Severity != SeverityPass {
				fmt.Printf("   %s\n", c.SummaryLine())
				if c.Recommendation != "" {
					fmt.Printf("      💡 %s\n", c.Recommendation)
				}
			}
		}
		// Show a condensed running-ok list
		ok := 0
		for _, c := range r.Containers {
			if c.Severity == SeverityPass {
				ok++
			}
		}
		if ok > 0 {
			fmt.Printf("   ✅ %d container(s) healthy\n", ok)
		}
	}

	// Mount details
	if len(r.Mounts) > 0 {
		hasMountIssues := false
		for _, m := range r.Mounts {
			if m.Severity != SeverityPass {
				hasMountIssues = true
				break
			}
		}
		if hasMountIssues {
			fmt.Println("\n📂 Mount Issues")
			for _, m := range r.Mounts {
				if m.Severity != SeverityPass {
					fmt.Printf("   %s\n", m.SummaryLine())
					if m.Fix != "" {
						fmt.Printf("      💡 %s\n", m.Fix)
					}
				}
			}
		}
	}

	// Port details
	if len(r.Ports) > 0 {
		hasPortIssues := false
		for _, p := range r.Ports {
			if p.Severity != SeverityPass {
				hasPortIssues = true
				break
			}
		}
		if hasPortIssues {
			fmt.Println("\n🔌 Port Conflicts")
			for _, p := range r.Ports {
				fmt.Printf("   %s\n", p.SummaryLine())
			}
		}
	}

	// Recommendations
	if len(r.Recommendations) > 0 {
		fmt.Println("\n💡 Recommendations")
		for i, rec := range r.Recommendations {
			fmt.Printf("   %d. %s\n", i+1, rec)
		}
	}

	// Footer
	fmt.Println()
	fmt.Println(strings.Repeat("═", 60))
	switch r.OverallSeverity {
	case SeverityPass:
		fmt.Println("  ✅ All checks passed — system is healthy.")
	case SeverityWarn:
		fmt.Println("  ⚠️  Warnings detected. Run 'm3tal doctor fix' to address them.")
	case SeverityFail:
		fmt.Println("  ❌ Issues detected. Run 'm3tal doctor fix --apply' to remediate.")
	}
	fmt.Println()
}

// WriteReportJSON marshals the report to a JSON file.
func WriteReportJSON(r Report, path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// PrintReportJSON prints the report as JSON to stdout.
func PrintReportJSON(r Report) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshalling report: %v\n", err)
		return
	}
	fmt.Println(string(data))
}
