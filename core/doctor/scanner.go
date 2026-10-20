// Package doctor provides the M3TAL self-healing and diagnostic subsystem.
// It covers container health scanning, volume/mount validation, port conflict
// detection, automated remediation, and health report generation.
package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
)

type ContainerStatus = models.ContainerStatus
type ContainerResult = models.ContainerResult

const (
	StatusRunning    = models.StatusRunning
	StatusStopped    = models.StatusStopped
	StatusUnhealthy  = models.StatusUnhealthy
	StatusRestarting = models.StatusRestarting
	StatusCreated    = models.StatusCreated
	StatusPaused     = models.StatusPaused
	StatusDead       = models.StatusDead
)

// dockerInspectState mirrors the subset of Docker's inspect output we need.
type dockerInspectState struct {
	Status     string `json:"Status"`
	Running    bool   `json:"Running"`
	Paused     bool   `json:"Paused"`
	Restarting bool   `json:"Restarting"`
	Dead       bool   `json:"Dead"`
	ExitCode   int    `json:"ExitCode"`
	StartedAt  string `json:"StartedAt"`
	FinishedAt string `json:"FinishedAt"`
	Health     *struct {
		Status string `json:"Status"`
	} `json:"Health"`
}

type dockerInspectHostConfig struct {
	RestartPolicy struct {
		MaximumRetryCount int `json:"MaximumRetryCount"`
	} `json:"RestartPolicy"`
}

type dockerInspectResult struct {
	Id           string                  `json:"Id"`
	Name         string                  `json:"Name"`
	State        dockerInspectState      `json:"State"`
	HostConfig   dockerInspectHostConfig `json:"HostConfig"`
	RestartCount int                     `json:"RestartCount"`
	Config       struct {
		Image string `json:"Image"`
	} `json:"Config"`
}

// ScanContainers queries Docker for all containers and returns detailed health
// results. It uses the Docker API for the list and docker inspect for details.
func ScanContainers() ([]ContainerResult, error) {
	mgr, err := docker.GetProvider()
	if err != nil {
		return nil, fmt.Errorf("cannot connect to Docker: %w", err)
	}

	list, err := mgr.ListContainers()
	if err != nil {
		return nil, fmt.Errorf("cannot list containers: %w", err)
	}

	var results []ContainerResult
	for _, c := range list {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		res := ContainerResult{
			Name:  name,
			ID:    c.ID,
			Image: c.Image,
			State: c.State,
		}

		// Enrich with docker inspect
		inspectData, inspectErr := runDockerInspect(c.ID)
		if inspectErr == nil {
			applyInspectData(&res, inspectData)
		} else {
			// Fallback: derive from basic container info
			res.Status = deriveStatus(c.State)
			res.Health = "none"
		}

		res.Severity = computeContainerSeverity(res)
		res.Recommendation = buildContainerRecommendation(res)

		results = append(results, res)
	}

	return results, nil
}

// runDockerInspect shells out to `docker inspect` to get rich container data.
// Using exec rather than the SDK ContainerInspect avoids another API type import.
func runDockerInspect(id string) (*dockerInspectResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "docker", "inspect", id).Output()
	if err != nil {
		return nil, err
	}

	var results []dockerInspectResult
	if err := json.Unmarshal(out, &results); err != nil || len(results) == 0 {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}
	return &results[0], nil
}

func applyInspectData(res *ContainerResult, d *dockerInspectResult) {
	res.State = d.State.Status
	res.Restarts = d.RestartCount
	res.ExitCode = d.State.ExitCode

	// Normalise timestamps
	if t, err := time.Parse(time.RFC3339Nano, d.State.StartedAt); err == nil {
		res.StartedAt = t.UTC().Format(time.RFC3339)
	}
	if t, err := time.Parse(time.RFC3339Nano, d.State.FinishedAt); err == nil && !t.IsZero() {
		res.FinishedAt = t.UTC().Format(time.RFC3339)
	}

	if d.State.Health != nil {
		res.Health = d.State.Health.Status
	} else {
		res.Health = "none"
	}

	res.Status = deriveStatus(d.State.Status)
}

func deriveStatus(state string) ContainerStatus {
	switch strings.ToLower(state) {
	case "running":
		return StatusRunning
	case "exited", "stopped":
		return StatusStopped
	case "restarting":
		return StatusRestarting
	case "paused":
		return StatusPaused
	case "dead":
		return StatusDead
	case "created":
		return StatusCreated
	default:
		return ContainerStatus(state)
	}
}

func computeContainerSeverity(r ContainerResult) Severity {
	switch r.Status {
	case StatusDead:
		return SeverityFail
	case StatusStopped:
		if r.ExitCode != 0 {
			return SeverityFail
		}
		return SeverityWarn
	case StatusRestarting:
		return SeverityWarn
	case StatusUnhealthy:
		return SeverityFail
	}
	if strings.EqualFold(r.Health, "unhealthy") {
		return SeverityFail
	}
	if r.Restarts >= 5 {
		return SeverityWarn
	}
	return SeverityPass
}

func buildContainerRecommendation(r ContainerResult) string {
	switch r.Status {
	case StatusStopped:
		if r.ExitCode != 0 {
			return fmt.Sprintf("Container exited with code %d. Run: docker logs %s", r.ExitCode, r.Name)
		}
		return fmt.Sprintf("Container is stopped. Run: m3tal doctor fix --name %s", r.Name)
	case StatusRestarting:
		return fmt.Sprintf("Container is crash-looping. Run: docker logs %s", r.Name)
	case StatusDead:
		return fmt.Sprintf("Container is dead. Remove and recreate: docker rm %s && m3tal up", r.Name)
	}
	if strings.EqualFold(r.Health, "unhealthy") {
		return fmt.Sprintf("Healthcheck failing. Inspect logs: docker logs --tail 50 %s", r.Name)
	}
	if r.Restarts >= 5 {
		return fmt.Sprintf("High restart count (%d). Check logs: docker logs --tail 50 %s", r.Restarts, r.Name)
	}
	return ""
}
