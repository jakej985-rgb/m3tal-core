package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/health"
)

// RunAnomaly executes the anomaly detection loop
func RunAnomaly() {
	now := time.Now().Format("2006-01-02 15:04:05")
	base := health.GetControlPlaneDir()
	anomalyLogPath := filepath.Join(base, "logs", "anomaly.log")

	prov, err := docker.GetProvider()
	if err != nil {
		content := fmt.Sprintf("[%s] [ERROR] [anomaly] Docker provider unavailable: %v\n", now, err)
		_ = os.WriteFile(anomalyLogPath, []byte(content), 0644)
		return
	}

	list, err := prov.ListContainers()
	if err != nil {
		content := fmt.Sprintf("[%s] [ERROR] [anomaly] Failed to list containers: %v\n", now, err)
		_ = os.WriteFile(anomalyLogPath, []byte(content), 0644)
		return
	}

	anomaliesCount := 0
	var anomalyDetails []string

	for _, c := range list {
		// Check for non-running/exited/dead container states, or unhealthy status
		isExitedOrDead := c.State == "exited" || c.State == "dead"
		isUnhealthy := strings.Contains(strings.ToLower(c.Status), "unhealthy")

		if isExitedOrDead || isUnhealthy {
			anomaliesCount++
			name := "unknown"
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}
			anomalyDetails = append(anomalyDetails, fmt.Sprintf("container %s is %s (status: %s)", name, c.State, c.Status))
		}
	}

	var content string
	if anomaliesCount > 0 {
		content = fmt.Sprintf(
			"[%s] [WARN] [anomaly] Scanning for anomalies... %d detected.\n",
			now, anomaliesCount,
		)
		for _, detail := range anomalyDetails {
			content += fmt.Sprintf("[%s] [WARN] [anomaly] - %s\n", now, detail)
		}
	} else {
		content = fmt.Sprintf(
			"[%s] [INFO] [anomaly] Scanning for anomalies... 0 detected.\n",
			now,
		)
	}

	_ = os.WriteFile(anomalyLogPath, []byte(content), 0644)
}
