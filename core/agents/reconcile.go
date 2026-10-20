package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/health"
	"github.com/jakej985-rgb/m3tal-core/core/state"
)

// RunReconcile executes the state reconciliation loop and returns a list of actions taken.
func RunReconcile() ([]string, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	base := health.GetControlPlaneDir()
	reconcileLogPath := filepath.Join(base, "logs", "reconcile.log")

	var actions []string

	// 1. Open the SQLite database
	dbPath := state.GetStatePath()
	db, err := state.Open(dbPath)
	if err != nil {
		errStr := fmt.Sprintf("[%s] [ERROR] [reconcile] Cannot open state DB at %s: %v\n", now, dbPath, err)
		_ = os.WriteFile(reconcileLogPath, []byte(errStr), 0644)
		return nil, err
	}
	defer db.Close()

	// 2. List all desired stacks
	dbStacks, err := db.ListStacks()
	if err != nil {
		errStr := fmt.Sprintf("[%s] [ERROR] [reconcile] Cannot list stacks from DB: %v\n", now, err)
		_ = os.WriteFile(reconcileLogPath, []byte(errStr), 0644)
		return nil, err
	}

	// 3. Get actual container statuses
	prov, err := docker.GetProvider()
	if err != nil {
		errStr := fmt.Sprintf("[%s] [ERROR] [reconcile] Docker provider unavailable: %v\n", now, err)
		_ = os.WriteFile(reconcileLogPath, []byte(errStr), 0644)
		return nil, err
	}

	containersList, err := prov.ListContainers()
	if err != nil {
		errStr := fmt.Sprintf("[%s] [ERROR] [reconcile] Failed to list containers: %v\n", now, err)
		_ = os.WriteFile(reconcileLogPath, []byte(errStr), 0644)
		return nil, err
	}

	// 4. Compare desired vs actual state per stack
	driftCount := 0
	for _, dbStack := range dbStacks {
		// Only reconcile if stack has a valid compose path and is expected to be running or stopped
		if dbStack.ComposePath == "" {
			continue
		}

		// Find actual containers belonging to this stack
		var stackContainers []docker.ContainerInfo
		for _, c := range containersList {
			proj := c.Labels["com.docker.compose.project"]
			if proj == "" {
				name := ""
				if len(c.Names) > 0 {
					name = strings.TrimPrefix(c.Names[0], "/")
				}
				if strings.HasPrefix(strings.ToLower(name), strings.ToLower(dbStack.Name)+"-") {
					stackContainers = append(stackContainers, c)
				}
			} else if strings.EqualFold(proj, dbStack.Name) {
				stackContainers = append(stackContainers, c)
			}
		}

		if dbStack.Status == "running" {
			// Expected to be running, verify if all are running
			hasDrift := false
			reason := ""
			if len(stackContainers) == 0 {
				hasDrift = true
				reason = "no containers found"
			} else {
				for _, sc := range stackContainers {
					if sc.State != "running" {
						hasDrift = true
						reason = fmt.Sprintf("container %s is %s", sc.Names[0], sc.State)
						break
					}
				}
			}

			if hasDrift {
				driftCount++
				actionMsg := fmt.Sprintf("Drift detected on stack %s (%s). Recreating/deploying...", dbStack.Name, reason)
				actions = append(actions, actionMsg)

				// Trigger deployment to reconcile
				_, deployErr := docker.DeployStack(dbStack.ComposePath, 0)
				if deployErr != nil {
					_ = db.UpdateStackStatus(dbStack.Name, "failed")
					actions = append(actions, fmt.Sprintf("Failed to reconcile stack %s: %v", dbStack.Name, deployErr))
				} else {
					_ = db.UpdateStackStatus(dbStack.Name, "running")
					actions = append(actions, fmt.Sprintf("Successfully reconciled stack %s to running.", dbStack.Name))
				}
			}
		} else if dbStack.Status == "stopped" {
			// Expected to be stopped, verify if any are running
			hasRunningContainers := false
			for _, sc := range stackContainers {
				if sc.State == "running" {
					hasRunningContainers = true
					break
				}
			}

			if hasRunningContainers {
				driftCount++
				actionMsg := fmt.Sprintf("Drift detected on stack %s (should be stopped but has running containers). Stopping...", dbStack.Name)
				actions = append(actions, actionMsg)

				// Trigger stop to reconcile
				_, stopErr := docker.StopStack(dbStack.ComposePath, 0)
				if stopErr != nil {
					_ = db.UpdateStackStatus(dbStack.Name, "failed")
					actions = append(actions, fmt.Sprintf("Failed to stop/reconcile stack %s: %v", dbStack.Name, stopErr))
				} else {
					_ = db.UpdateStackStatus(dbStack.Name, "stopped")
					actions = append(actions, fmt.Sprintf("Successfully reconciled stack %s to stopped.", dbStack.Name))
				}
			}
		}
	}

	// 5. Write log details
	var logContent string
	if len(actions) > 0 {
		logContent = fmt.Sprintf("[%s] [INFO] [reconcile] Reconciliation complete. Found %d drifts. Actions:\n", now, driftCount)
		for _, action := range actions {
			logContent += fmt.Sprintf("[%s] [INFO] [reconcile] - %s\n", now, action)
		}
	} else {
		logContent = fmt.Sprintf("[%s] [INFO] [reconcile] System state matches target configuration.\n", now)
	}

	_ = os.WriteFile(reconcileLogPath, []byte(logContent), 0644)
	return actions, nil
}

// Start runs a periodic background state reconciliation ticker.
func Start() {
	for {
		reconcileAll()
		time.Sleep(10 * time.Second)
	}
}

// reconcileAll executes a single reconciliation pass.
func reconcileAll() {
	_, _ = RunReconcile()
}

// ReconcileAll is an exported wrapper for reconcileAll.
func ReconcileAll() {
	reconcileAll()
}
