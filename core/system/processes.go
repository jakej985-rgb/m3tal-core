package system

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"github.com/shirou/gopsutil/v3/process"
)

// GetProcesses returns a snapshot of running processes on the system.
func GetProcesses(limit int) ([]models.ProcessInfo, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	results := make([]models.ProcessInfo, 0, len(procs))

	for _, p := range procs {
		pid := p.Pid
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}

		cpuPercent, _ := p.CPUPercent()
		memPercent, _ := p.MemoryPercent()
		memInfo, _ := p.MemoryInfo()
		var rss uint64
		if memInfo != nil {
			rss = memInfo.RSS
		}

		username, _ := p.Username()
		statusList, _ := p.Status()
		status := "running"
		if len(statusList) > 0 {
			status = statusList[0]
		}

		rawCmdline, _ := p.Cmdline()
		name = ResolveProcessName(name, rawCmdline)

		displayCmdline := rawCmdline
		if len(displayCmdline) > 200 {
			displayCmdline = displayCmdline[:200] + "..."
		}

		createTime, _ := p.CreateTime()

		results = append(results, models.ProcessInfo{
			PID:           pid,
			Name:          name,
			Username:      username,
			CPUPercent:    cpuPercent,
			MemoryPercent: memPercent,
			MemoryRSS:     rss,
			Status:        strings.ToLower(status),
			Cmdline:       displayCmdline,
			CreateTime:    createTime,
		})
	}

	// Sort by CPU usage descending by default
	sort.Slice(results, func(i, j int) bool {
		if results[i].CPUPercent == results[j].CPUPercent {
			return results[i].MemoryRSS > results[j].MemoryRSS
		}
		return results[i].CPUPercent > results[j].CPUPercent
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// KillProcess terminates a process by its PID.
func KillProcess(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("process with PID %d not found: %w", pid, err)
	}
	return p.Kill()
}

// ResolveProcessName maps system process names and command-line arguments
// to canonical M3TAL process titles (e.g. m3tal-chrome-zygote (SB), m3tal-chrome-render (#5), m3tal-docker).
func ResolveProcessName(name, cmdline string) string {
	cmdLower := strings.ToLower(cmdline)

	// Canonical M3TAL process naming: m3tal-<process>
	if name == "m3tal" || strings.HasPrefix(name, "m3tal-") {
		if strings.Contains(cmdLower, "sysmon") {
			return "m3tal-sysmon"
		} else if strings.Contains(cmdLower, "docker-daemon") || strings.Contains(cmdLower, "docker") {
			return "m3tal-docker"
		} else if strings.Contains(cmdLower, "api") {
			return "m3tal-api"
		} else if strings.Contains(cmdLower, "headless") || strings.Contains(cmdLower, "web") {
			return "m3tal-web"
		} else if strings.Contains(cmdLower, "ui") {
			return "m3tal-gui"
		} else if strings.Contains(cmdLower, "daemon") {
			return "m3tal-core"
		} else if name == "m3tal" {
			return "m3tal-core"
		}
	} else if (name == "chrome" || name == "google-chrome" || name == "chromium" || name == "brave" || name == "chrome_crashpad") &&
		strings.Contains(cmdline, "--user-data-dir=") && strings.Contains(cmdline, "m3tal/browser-profile") {
		// Chrome subprocesses belonging specifically to M3TAL Desktop App Window
		if strings.Contains(cmdLower, "--type=renderer") {
			clientID := extractRendererClientID(cmdline)
			if clientID != "" {
				return fmt.Sprintf("m3tal-chrome-render (#%s)", clientID)
			}
			return "m3tal-chrome-render"
		} else if strings.Contains(cmdLower, "--type=gpu-process") {
			return "m3tal-chrome-gpu"
		} else if strings.Contains(cmdLower, "--type=utility") {
			if strings.Contains(cmdLower, "network.mojom") {
				return "m3tal-chrome-net"
			} else if strings.Contains(cmdLower, "storage.mojom") {
				return "m3tal-chrome-storage"
			} else if strings.Contains(cmdLower, "audio.mojom") {
				return "m3tal-chrome-audio"
			}
			return "m3tal-chrome-utility"
		} else if strings.Contains(cmdLower, "--type=zygote") {
			if strings.Contains(cmdLower, "--no-zygote-sandbox") {
				return "m3tal-chrome-zygote"
			}
			return "m3tal-chrome-zygote (SB)"
		} else if strings.Contains(cmdLower, "--type=broker") {
			return "m3tal-chrome-broker"
		} else if strings.Contains(cmdLower, "crashpad-handler") || strings.Contains(cmdLower, "chrome_crashpad") {
			return "m3tal-chrome-crashpad"
		} else if strings.Contains(cmdLower, "--app=") {
			return "m3tal-chrome-gui"
		}
		return "m3tal-chrome"
	}

	return name
}

func extractRendererClientID(cmdline string) string {
	const prefix = "--renderer-client-id="
	idx := strings.Index(cmdline, prefix)
	if idx == -1 {
		return ""
	}
	start := idx + len(prefix)
	end := strings.IndexAny(cmdline[start:], " \t\r\n\"'")
	if end == -1 {
		return cmdline[start:]
	}
	return cmdline[start : start+end]
}
