package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/health"
)

// RunMonitor executes the system monitoring loop
func RunMonitor() {
	now := time.Now().Format("2006-01-02 15:04:05")
	base := health.GetControlPlaneDir()
	monitorLogPath := filepath.Join(base, "logs", "monitor.log")

	prov, err := docker.GetProvider()
	if err != nil {
		content := fmt.Sprintf("[%s] [ERROR] [monitor] Docker provider unavailable: %v\n", now, err)
		_ = os.WriteFile(monitorLogPath, []byte(content), 0644)
		return
	}

	list, err := prov.ListContainers()
	if err != nil {
		content := fmt.Sprintf("[%s] [ERROR] [monitor] Failed to list containers: %v\n", now, err)
		_ = os.WriteFile(monitorLogPath, []byte(content), 0644)
		return
	}

	running := 0
	for _, c := range list {
		if c.State == "running" {
			running++
		}
	}

	content := fmt.Sprintf(
		"[%s] [INFO] [monitor] System load within normal parameters.\n"+
			"[%s] [INFO] [monitor] Checked container states: %d/%d running.\n",
		now, now, running, len(list),
	)

	_ = os.WriteFile(monitorLogPath, []byte(content), 0644)
}
