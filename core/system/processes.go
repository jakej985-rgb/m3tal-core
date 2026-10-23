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

		cmdline, _ := p.Cmdline()
		if len(cmdline) > 200 {
			cmdline = cmdline[:200] + "..."
		}

		// Canonical M3TAL process naming: m3tal-<process>
		if name == "m3tal" || name == "m3tal-headless" || strings.Contains(cmdline, "m3tal") {
			cmdLower := strings.ToLower(cmdline)
			if strings.Contains(cmdLower, "sysmon") || strings.Contains(cmdLower, "monitor") {
				name = "m3tal-sysmon"
			} else if strings.Contains(cmdLower, "web") || strings.Contains(cmdLower, "headless") {
				name = "m3tal-web"
			} else if strings.Contains(cmdLower, "api") {
				name = "m3tal-api"
			} else if strings.Contains(cmdLower, "docker") {
				name = "m3tal-docker"
			} else if strings.Contains(cmdLower, "agent") || strings.Contains(cmdLower, "daemon") {
				name = "m3tal-core"
			} else if !strings.HasPrefix(name, "m3tal-") {
				name = "m3tal-core"
			}
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
			Cmdline:       cmdline,
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
