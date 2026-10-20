package output

import (
	"fmt"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
)

// SeverityIcon returns an emoji for a given severity level.
func SeverityIcon(s models.Severity) string {
	switch s {
	case models.SeverityPass:
		return "✅"
	case models.SeverityWarn:
		return "⚠️ "
	case models.SeverityFail:
		return "❌"
	default:
		return "❓"
	}
}

// ContainerSummaryLine returns a formatted status string for a container check.
func ContainerSummaryLine(r models.ContainerResult) string {
	icon := SeverityIcon(r.Severity)
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

// MountSummaryLine returns a formatted status string for a mount check.
func MountSummaryLine(r models.MountResult) string {
	icon := SeverityIcon(r.Severity)
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

// PortSummaryLine returns a formatted status string for a port check.
func PortSummaryLine(r models.PortResult) string {
	icon := SeverityIcon(r.Severity)
	if r.InUse {
		return fmt.Sprintf("%s Port %-5d  IN USE by %-20s  → suggest port %d",
			icon, r.Port, fmt.Sprintf("%s (pid=%d)", r.OwnedBy, r.PID), r.Suggestion)
	}
	return fmt.Sprintf("%s Port %-5d  free", icon, r.Port)
}

// PrintFixes pretty-prints the proposed fixes (dry-run view).
func PrintFixes(fixes []models.Fix) {
	if len(fixes) == 0 {
		fmt.Println("✅ No fixes required — everything looks healthy!")
		return
	}
	fmt.Printf("\n🔧 Proposed Fixes (%d action(s))\n", len(fixes))
	fmt.Println("══════════════════════════════════════════")
	for i, f := range fixes {
		fmt.Printf("  [%d] %s\n", i+1, f.Description)
		fmt.Printf("      cmd: %s\n\n", f.Command)
	}
	fmt.Println("══════════════════════════════════════════")
	fmt.Println("  Run with --apply to execute these fixes.")
	fmt.Println()
}

// PrintFixResults prints the outcome of applied fixes.
func PrintFixResults(results []models.FixResult) {
	fmt.Printf("\n⚙️  Fix Results (%d action(s))\n", len(results))
	fmt.Println("══════════════════════════════════════════")
	for _, r := range results {
		icon := "✅"
		status := "applied"
		if !r.Applied {
			if r.Action == models.FixSuggestPort {
				icon = "💡"
				status = "advisory"
			} else {
				icon = "❌"
				status = "failed"
			}
		}
		fmt.Printf("  %s [%s] %s\n", icon, status, r.Description)
		if r.Error != "" {
			fmt.Printf("         error: %s\n", r.Error)
		}
		if r.Output != "" {
			fmt.Printf("         note:  %s\n", r.Output)
		}
	}
	fmt.Println()
}

// PrintReport outputs a rich, human-readable health report to stdout.
func PrintReport(r models.DoctorReport) {
	overall := SeverityIcon(r.OverallSeverity)
	fmt.Printf("\n%s M3TAL Doctor Report — %s\n", overall, r.GeneratedAt)
	fmt.Println(strings.Repeat("═", 60))

	// Section summaries
	for _, s := range r.Sections {
		icon := SeverityIcon(s.Severity)
		fmt.Printf("  %s %-12s  pass=%-3d warn=%-3d fail=%d\n",
			icon, s.Name, s.PassCount, s.WarnCount, s.FailCount)
	}
	fmt.Println(strings.Repeat("─", 60))

	// Container details
	if len(r.Containers) > 0 {
		fmt.Println("\n📦 Containers")
		for _, c := range r.Containers {
			if c.Severity != models.SeverityPass {
				fmt.Printf("   %s\n", ContainerSummaryLine(c))
				if c.Recommendation != "" {
					fmt.Printf("      💡 %s\n", c.Recommendation)
				}
			}
		}
		// Show a condensed running-ok list
		ok := 0
		for _, c := range r.Containers {
			if c.Severity == models.SeverityPass {
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
			if m.Severity != models.SeverityPass {
				hasMountIssues = true
				break
			}
		}
		if hasMountIssues {
			fmt.Println("\n📂 Mount Issues")
			for _, m := range r.Mounts {
				if m.Severity != models.SeverityPass {
					fmt.Printf("   %s\n", MountSummaryLine(m))
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
			if p.Severity != models.SeverityPass {
				hasPortIssues = true
				break
			}
		}
		if hasPortIssues {
			fmt.Println("\n🔌 Port Conflicts")
			for _, p := range r.Ports {
				fmt.Printf("   %s\n", PortSummaryLine(p))
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
	case models.SeverityPass:
		fmt.Println("  ✅ All checks passed — system is healthy.")
	case models.SeverityWarn:
		fmt.Println("  ⚠️  Warnings detected. Run 'm3tal doctor fix' to address them.")
	case models.SeverityFail:
		fmt.Println("  ❌ Issues detected. Run 'm3tal doctor fix --apply' to remediate.")
	}
	fmt.Println()
}

// PrintPreflightResults outputs the check results in a human-readable format.
func PrintPreflightResults(results []models.CheckResult) {
	allPassed := true
	hasWarnings := false

	fmt.Println("\n🔍 M3TAL Pre-Flight Check")
	fmt.Println("==========================")
	for _, r := range results {
		var icon string
		switch r.Status {
		case "PASS":
			icon = "✅"
		case "WARN":
			icon = "⚠️"
			hasWarnings = true
		case "FAIL":
			icon = "❌"
			allPassed = false
		default:
			icon = "❓"
		}
		fmt.Printf(" %s %s\n", icon, r.Name)
		fmt.Printf("    %s\n", r.Message)
	}

	fmt.Println("==========================")
	if allPassed && !hasWarnings {
		fmt.Println("✅ All checks passed! System is ready.")
	} else if allPassed && hasWarnings {
		fmt.Println("✅ All checks passed (with warnings). Review the warnings above.")
	} else {
		fmt.Println("❌ Some checks FAILED. Review the errors above before proceeding.")
	}
	fmt.Println()
}
