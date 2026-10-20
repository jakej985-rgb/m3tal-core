package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/pkg/system"
	"github.com/shirou/gopsutil/v3/disk"
)

type SystemHealthState struct {
	LastSeenHealthy string `json:"last_seen_healthy"`
	LastFailure     string `json:"last_failure"`
	Status          string `json:"status"` // Healthy, Degraded, Unhealthy
}

type ContainerHealthInfo struct {
	Name     string `json:"name"`
	State    string `json:"state"`
	Healthy  bool   `json:"healthy"`
	Critical bool   `json:"critical"`
}

type DockerHealthState struct {
	Status              string                `json:"status"`
	TotalContainers     int                   `json:"total_containers"`
	RunningContainers   int                   `json:"running_containers"`
	UnhealthyContainers int                   `json:"unhealthy_containers"`
	Containers          []ContainerHealthInfo `json:"containers"`
}

type AgentsHealthState struct {
	Status        string `json:"status"`
	DaemonRunning bool   `json:"daemon_running"`
	LastActivity  string `json:"last_activity"`
}

type DiskHealthState struct {
	Status      string  `json:"status"`
	UsedPercent float64 `json:"used_percent"`
	Path        string  `json:"path"`
}

type UnifiedHealthRegistry struct {
	System SystemHealthState `json:"system"`
	Docker DockerHealthState `json:"docker"`
	Agents AgentsHealthState `json:"agents"`
	Disk   DiskHealthState   `json:"disk"`
}

// GetControlPlaneDir returns the data path
func GetControlPlaneDir() string {
	sysPath := system.DataPath
	if info, err := os.Stat(sysPath); err == nil && info.IsDir() {
		return sysPath
	}
	return "./data"
}

func EnsureControlPlaneDirs() {
	base := GetControlPlaneDir()
	_ = os.MkdirAll(filepath.Join(base, "state"), 0755)
	_ = os.MkdirAll(filepath.Join(base, "logs"), 0755)
}

func GetAgentLogAge(logName string) time.Duration {
	base := GetControlPlaneDir()
	path := filepath.Join(base, "logs", logName)
	info, err := os.Stat(path)
	if err != nil {
		return 999 * time.Hour
	}
	return time.Since(info.ModTime())
}

func ReadSystemHealthJSON() *UnifiedHealthRegistry {
	path := filepath.Join(GetControlPlaneDir(), "state", "system.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var registry UnifiedHealthRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil
	}
	return &registry
}

func GetDockerHealthState() DockerHealthState {
	var state DockerHealthState
	state.Status = "🟢"

	mgr, err := docker.GetProvider()
	if err != nil {
		state.Status = "🔴"
		return state
	}

	list, err := mgr.ListContainers()
	if err != nil {
		state.Status = "🔴"
		return state
	}

	state.TotalContainers = len(list)
	criticalServices := []string{"radarr", "sonarr", "qbittorrent"}

	for _, c := range list {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		isCritical := false
		for _, cs := range criticalServices {
			if strings.Contains(strings.ToLower(name), cs) {
				isCritical = true
				break
			}
		}

		isRunning := strings.EqualFold(c.State, "running")
		isHealthy := !strings.Contains(strings.ToLower(c.Status), "unhealthy")

		if isRunning {
			state.RunningContainers++
		}
		if !isHealthy {
			state.UnhealthyContainers++
		}

		cHealth := ContainerHealthInfo{
			Name:     name,
			State:    c.State,
			Healthy:  isHealthy,
			Critical: isCritical,
		}
		state.Containers = append(state.Containers, cHealth)

		if isCritical && (!isRunning || !isHealthy) {
			state.Status = "🔴"
		}
	}

	if state.Status != "🔴" {
		if state.RunningContainers < state.TotalContainers || state.UnhealthyContainers > 0 {
			state.Status = "🟡"
		}
	}

	prev := ReadSystemHealthJSON()
	if prev != nil && prev.Docker.TotalContainers > 0 {
		if state.TotalContainers < prev.Docker.TotalContainers {
			if state.Status != "🔴" {
				state.Status = "🟡"
			}
		}
	}

	return state
}

func GetAgentsHealthState() AgentsHealthState {
	var state AgentsHealthState
	daemonRunning := false
	cmd := exec.Command("pgrep", "-f", "m3tal daemon")
	if err := cmd.Run(); err == nil {
		daemonRunning = true
	}
	state.DaemonRunning = daemonRunning

	monitorAge := GetAgentLogAge("monitor.log")
	anomalyAge := GetAgentLogAge("anomaly.log")

	var lastActivity time.Time
	base := GetControlPlaneDir()
	mPath := filepath.Join(base, "logs", "monitor.log")
	if info, err := os.Stat(mPath); err == nil {
		lastActivity = info.ModTime()
	}
	aPath := filepath.Join(base, "logs", "anomaly.log")
	if info, err := os.Stat(aPath); err == nil {
		if info.ModTime().After(lastActivity) {
			lastActivity = info.ModTime()
		}
	}

	if !lastActivity.IsZero() {
		state.LastActivity = lastActivity.UTC().Format(time.RFC3339)
	}

	if !daemonRunning {
		state.Status = "🔴"
	} else if monitorAge > 100*time.Hour || anomalyAge > 100*time.Hour {
		state.Status = "🟢"
	} else if monitorAge < 2*time.Minute && anomalyAge < 2*time.Minute {
		state.Status = "🟢"
	} else if monitorAge < 10*time.Minute && anomalyAge < 10*time.Minute {
		state.Status = "🟡"
	} else {
		state.Status = "🔴"
	}

	return state
}

func GetDiskHealthState() DiskHealthState {
	var state DiskHealthState
	state.Path = "/"
	usage, err := disk.Usage("/")
	if err != nil {
		state.Status = "🔴"
		return state
	}
	state.UsedPercent = usage.UsedPercent
	if usage.UsedPercent > 90 {
		state.Status = "🔴"
	} else if usage.UsedPercent > 75 {
		state.Status = "🟡"
	} else {
		state.Status = "🟢"
	}
	return state
}

func GetSystemHealthState(docker DockerHealthState, agents AgentsHealthState, disk DiskHealthState) SystemHealthState {
	var state SystemHealthState

	prev := ReadSystemHealthJSON()
	if prev != nil {
		state.LastSeenHealthy = prev.System.LastSeenHealthy
		state.LastFailure = prev.System.LastFailure
	}

	if docker.Status == "🔴" || agents.Status == "🔴" || disk.Status == "🔴" {
		state.Status = "🔴"
		state.LastFailure = time.Now().UTC().Format(time.RFC3339)
	} else if docker.Status == "🟡" || agents.Status == "🟡" || disk.Status == "🟡" {
		state.Status = "🟡"
	} else {
		state.Status = "🟢"
		state.LastSeenHealthy = time.Now().UTC().Format(time.RFC3339)
	}

	return state
}

func WriteStatusShellScript() {
	base := GetControlPlaneDir()
	libDir := filepath.Join(base, "lib")
	_ = os.MkdirAll(libDir, 0755)

	scriptContent := `#!/bin/bash

get_disk_status() {
  usage=$(df / | awk 'NR==2 {print $5}' | tr -d '%')

  if [ "$usage" -gt 90 ]; then
    echo "🔴 ${usage}% used"
  elif [ "$usage" -gt 75 ]; then
    echo "🟡 ${usage}% used"
  else
    echo "🟢 ${usage}% used"
  fi
}

get_docker_status() {
  total=$(docker ps -a -q | wc -l)
  running=$(docker ps -q | wc -l)

  if [ "$running" -eq "$total" ]; then
    echo "🟢 $running/$total running"
  else
    echo "🟡 $running/$total running"
  fi
}

render_header() {
  echo "═══════════════════════════════════════"
  echo "SYSTEM:   🟢 Healthy"
  echo "DOCKER:   $(get_docker_status)"
  echo "AGENTS:   🟡 checking..."
  echo "DISK:     $(get_disk_status)"
  echo "═══════════════════════════════════════"
}

render_header
`
	scriptPath := filepath.Join(libDir, "status.sh")
	_ = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
}

func UpdateAndSaveHealthRegistry() UnifiedHealthRegistry {
	EnsureControlPlaneDirs()
	WriteStatusShellScript()

	var registry UnifiedHealthRegistry
	registry.Docker = GetDockerHealthState()
	registry.Agents = GetAgentsHealthState()
	registry.Disk = GetDiskHealthState()
	registry.System = GetSystemHealthState(registry.Docker, registry.Agents, registry.Disk)

	path := filepath.Join(GetControlPlaneDir(), "state", "system.json")
	data, err := json.MarshalIndent(registry, "", "  ")
	if err == nil {
		_ = os.WriteFile(path, data, 0644)
	}

	return registry
}

// ServiceHealth represents the health of a specific service
type ServiceHealth struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// CheckService checks the health of an HTTP endpoint
func CheckService(name string, url string) ServiceHealth {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return ServiceHealth{
			Name:   name,
			Status: "down",
			Error:  err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return ServiceHealth{
			Name:   name,
			Status: "up",
		}
	}

	return ServiceHealth{
		Name:   name,
		Status: "degraded",
		Error:  fmt.Sprintf("HTTP %d", resp.StatusCode),
	}
}
