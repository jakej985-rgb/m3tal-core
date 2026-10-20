package system

import (
	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/state"
	"github.com/shirou/gopsutil/v3/disk"
)

type ComponentStatus string

const (
	StatusHealthy   ComponentStatus = "healthy"
	StatusUnhealthy ComponentStatus = "unhealthy"
)

// HealthReport represents the aggregated status details of system components.
type HealthReport struct {
	Status     ComponentStatus            `json:"status"`
	Components map[string]ComponentStatus `json:"components"`
	Details    map[string]string          `json:"details,omitempty"`
}

// CheckHealth queries database, docker, and disk metrics to construct a HealthReport.
func CheckHealth(db *state.Store) HealthReport {
	report := HealthReport{
		Status:     StatusHealthy,
		Components: make(map[string]ComponentStatus),
		Details:    make(map[string]string),
	}

	// 1. Check database
	if db != nil {
		err := db.Ping()
		if err != nil {
			report.Components["database"] = StatusUnhealthy
			report.Details["database"] = err.Error()
			report.Status = StatusUnhealthy
		} else {
			report.Components["database"] = StatusHealthy
		}
	} else {
		report.Components["database"] = StatusUnhealthy
		report.Details["database"] = "no active database store"
		report.Status = StatusUnhealthy
	}

	// 2. Check Docker provider
	prov, err := docker.GetProvider()
	if err != nil {
		report.Components["docker"] = StatusUnhealthy
		report.Details["docker"] = err.Error()
		report.Status = StatusUnhealthy
	} else {
		_, err := prov.ListContainers()
		if err != nil {
			report.Components["docker"] = StatusUnhealthy
			report.Details["docker"] = "connection failed: " + err.Error()
			report.Status = StatusUnhealthy
		} else {
			report.Components["docker"] = StatusHealthy
		}
	}

	// 3. Check Disk usage
	usage, err := disk.Usage("/")
	if err != nil {
		report.Components["disk"] = StatusUnhealthy
		report.Details["disk"] = err.Error()
		report.Status = StatusUnhealthy
	} else {
		if usage.UsedPercent >= 95.0 {
			report.Components["disk"] = StatusUnhealthy
			report.Details["disk"] = "low disk space: > 95% full"
			report.Status = StatusUnhealthy
		} else {
			report.Components["disk"] = StatusHealthy
		}
	}

	return report
}
