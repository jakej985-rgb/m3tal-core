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

// RunDecision executes the decision making loop
func RunDecision() {
	now := time.Now().Format("2006-01-02 15:04:05")
	base := health.GetControlPlaneDir()
	decisionLogPath := filepath.Join(base, "logs", "decision.log")

	prov, err := docker.GetProvider()
	if err != nil {
		content := fmt.Sprintf("[%s] [ERROR] [decision] Docker provider unavailable: %v\n", now, err)
		_ = os.WriteFile(decisionLogPath, []byte(content), 0644)
		return
	}

	list, err := prov.ListContainers()
	if err != nil {
		content := fmt.Sprintf("[%s] [ERROR] [decision] Failed to list containers: %v\n", now, err)
		_ = os.WriteFile(decisionLogPath, []byte(content), 0644)
		return
	}

	anomalyDetected := false
	for _, c := range list {
		if c.State == "exited" || c.State == "dead" || strings.Contains(strings.ToLower(c.Status), "unhealthy") {
			anomalyDetected = true
			break
		}
	}

	var content string
	if anomalyDetected {
		content = fmt.Sprintf("[%s] [WARN] [decision] Anomalies detected. Action required: triggering state reconciliation.\n", now)
	} else {
		content = fmt.Sprintf("[%s] [INFO] [decision] System state is stable. No action required.\n", now)
	}

	_ = os.WriteFile(decisionLogPath, []byte(content), 0644)
}
