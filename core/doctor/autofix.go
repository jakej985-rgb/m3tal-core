package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
)

type FixAction = models.FixAction
type Fix = models.Fix
type FixResult = models.FixResult

const (
	FixRestartContainer = models.FixRestartContainer
	FixChmodPath        = models.FixChmodPath
	FixMkdirPath        = models.FixMkdirPath
	FixSuggestPort      = models.FixSuggestPort
)

// BuildFixes derives a list of proposed fixes from scan results.
func BuildFixes(containers []ContainerResult, mounts []MountResult, ports []PortResult) []Fix {
	var fixes []Fix

	for _, c := range containers {
		if c.Severity == SeverityFail || c.Severity == SeverityWarn {
			switch c.Status {
			case StatusStopped, StatusRestarting, StatusUnhealthy:
				fixes = append(fixes, Fix{
					Action:      FixRestartContainer,
					Target:      c.Name,
					Description: fmt.Sprintf("Restart container %q (state: %s)", c.Name, c.State),
					Command:     fmt.Sprintf("docker restart %s", c.Name),
				})
			case StatusDead:
				fixes = append(fixes, Fix{
					Action:      FixRestartContainer,
					Target:      c.Name,
					Description: fmt.Sprintf("Start dead container %q", c.Name),
					Command:     fmt.Sprintf("docker start %s", c.Name),
				})
			}
		}
	}

	for _, m := range mounts {
		if m.Severity == SeverityFail || m.Severity == SeverityWarn {
			if m.Type == MountTypeBind && m.Source != "" {
				if _, err := os.Stat(m.Source); os.IsNotExist(err) {
					fixes = append(fixes, Fix{
						Action:      FixMkdirPath,
						Target:      m.Source,
						Description: fmt.Sprintf("Create missing bind-mount path %s (container: %s)", m.Source, m.Container),
						Command:     fmt.Sprintf("mkdir -p %s", m.Source),
					})
				} else if strings.Contains(m.Issue, "not writable") {
					fixes = append(fixes, Fix{
						Action:      FixChmodPath,
						Target:      m.Source,
						Description: fmt.Sprintf("Fix write permissions on %s (container: %s)", m.Source, m.Container),
						Command:     fmt.Sprintf("chmod a+w %s", m.Source),
					})
				} else if strings.Contains(m.Issue, "not readable") {
					fixes = append(fixes, Fix{
						Action:      FixChmodPath,
						Target:      m.Source,
						Description: fmt.Sprintf("Fix read permissions on %s (container: %s)", m.Source, m.Container),
						Command:     fmt.Sprintf("chmod a+r %s", m.Source),
					})
				}
			}
		}
	}

	for _, p := range ports {
		if p.Conflict && p.Suggestion > 0 {
			fixes = append(fixes, Fix{
				Action:      FixSuggestPort,
				Target:      fmt.Sprintf("%d", p.Port),
				Description: fmt.Sprintf("Port %d is in use by %q — change to %d in your .env", p.Port, p.OwnedBy, p.Suggestion),
				Command:     fmt.Sprintf("m3tal config set PORT_%d %d", p.Port, p.Suggestion),
			})
		}
	}

	return fixes
}

// PrintFixes pretty-prints the proposed fixes (dry-run view).
func PrintFixes(fixes []Fix) {
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

// ApplyFixes executes each fix and returns the results.
// containerFixes that need `docker start/restart` use the containers.Provider
// when possible, falling back to exec.
func ApplyFixes(fixes []Fix) []FixResult {
	mgr, _ := docker.GetProvider()

	var results []FixResult
	for _, f := range fixes {
		res := FixResult{Fix: f}
		var err error

		switch f.Action {
		case FixRestartContainer:
			if mgr != nil {
				err = mgr.RestartContainer(f.Target)
			} else {
				err = runShell("docker", "restart", f.Target)
			}
			if err != nil {
				// Fallback to docker start for dead containers
				if mgr != nil {
					err = mgr.StartContainer(f.Target)
				} else {
					err = runShell("docker", "start", f.Target)
				}
			}

		case FixMkdirPath:
			err = os.MkdirAll(f.Target, 0755)

		case FixChmodPath:
			err = runShell("chmod", "a+rw", f.Target)

		case FixSuggestPort:
			// Port fixes are advisory only — we just print the suggestion
			res.Output = f.Description
			res.Applied = false
			results = append(results, res)
			continue
		}

		if err != nil {
			res.Error = err.Error()
			res.Applied = false
		} else {
			res.Applied = true
		}
		results = append(results, res)
	}
	return results
}

// PrintFixResults prints the outcome of applied fixes.
func PrintFixResults(results []FixResult) {
	fmt.Printf("\n⚙️  Fix Results (%d action(s))\n", len(results))
	fmt.Println("══════════════════════════════════════════")
	for _, r := range results {
		icon := "✅"
		status := "applied"
		if !r.Applied {
			if r.Action == FixSuggestPort {
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

func runShell(name string, args ...string) error {
	cmd := exec.Command(name, args...) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
