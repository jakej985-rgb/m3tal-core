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
