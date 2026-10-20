package state

import "github.com/jakej985-rgb/m3tal-core/pkg/models"

// SystemState represents the aggregated, single source of truth for platform state.
type SystemState struct {
	Containers []models.Container
	Metrics    models.MetricsResponse
	Health     string
}
