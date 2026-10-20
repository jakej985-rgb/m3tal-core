package system

import (
	"sync"
	"time"
)

// MetricsHistory represents a thread-safe sliding history of system metrics.
type MetricsHistory struct {
	mu      sync.RWMutex
	history []*SystemStats
	limit   int
}

// GlobalMetricsHistory stores historical measurements for the API metrics history endpoint.
var GlobalMetricsHistory = &MetricsHistory{
	history: make([]*SystemStats, 0),
	limit:   60, // Keep last 5 minutes of history (60 * 5s)
}

// Add appends a new metric measurement, sliding the queue if capacity is reached.
func (h *MetricsHistory) Add(stats *SystemStats) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.history = append(h.history, stats)
	if len(h.history) > h.limit {
		h.history = h.history[len(h.history)-h.limit:]
	}
}

// Get returns a copy of all stored metric history measurements.
func (h *MetricsHistory) Get() []*SystemStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]*SystemStats, len(h.history))
	copy(list, h.history)
	return list
}

// StartMetricsAggregator initializes the periodic background task collecting stats.
func StartMetricsAggregator() {
	// Add initial measurement immediately
	if stats, err := GetStats(); err == nil && stats != nil {
		GlobalMetricsHistory.Add(stats)
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats, err := GetStats()
			if err == nil && stats != nil {
				GlobalMetricsHistory.Add(stats)
			}
		}
	}()
}
