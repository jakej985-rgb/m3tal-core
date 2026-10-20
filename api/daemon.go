package api

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jakej985-rgb/m3tal-core/core/agents"
	"github.com/jakej985-rgb/m3tal-core/core/health"
)

// RunAgentsDaemon starts the background M3TAL Agents Loop.
func RunAgentsDaemon() {
	fmt.Println("Starting M3TAL Agents Daemon loop...")
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Initial execution
	executeAgentsStep()

	for range ticker.C {
		executeAgentsStep()
	}
}

func executeAgentsStep() {
	health.EnsureControlPlaneDirs()
	base := health.GetControlPlaneDir()

	// Run the real agents
	agents.RunMonitor()
	agents.RunAnomaly()
	agents.RunDecision()
	agents.ReconcileAll()

	// Read logs written by agents
	monitorLog, _ := os.ReadFile(filepath.Join(base, "logs", "monitor.log"))
	anomalyLog, _ := os.ReadFile(filepath.Join(base, "logs", "anomaly.log"))
	decisionLog, _ := os.ReadFile(filepath.Join(base, "logs", "decision.log"))
	reconcileLog, _ := os.ReadFile(filepath.Join(base, "logs", "reconcile.log"))

	// Write to/append to agents.log
	var f *os.File
	var err error
	if os.Geteuid() == 0 {
		agentsLogPath := "/var/log/m3tal/agents.log"
		f, err = os.OpenFile(agentsLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		err = os.ErrPermission
	}
	if err != nil {
		// Fallback to local logs directory if not root or /var/log/m3tal is not writable
		fallbackPath := filepath.Join(base, "logs", "agents.log")
		f, err = os.OpenFile(fallbackPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	}
	if err == nil {
		defer f.Close()
		logLines := string(monitorLog) + string(anomalyLog) + string(decisionLog) + string(reconcileLog)
		_, _ = f.WriteString(logLines)
	}
}
