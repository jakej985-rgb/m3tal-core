package system

import (
	"testing"
)

func TestMetricsHistory_Limit(t *testing.T) {
	history := &MetricsHistory{
		history: make([]*SystemStats, 0),
		limit:   3,
	}

	history.Add(&SystemStats{Hostname: "host-1"})
	history.Add(&SystemStats{Hostname: "host-2"})
	history.Add(&SystemStats{Hostname: "host-3"})

	list := history.Get()
	if len(list) != 3 {
		t.Fatalf("expected 3 items, got %d", len(list))
	}
	if list[0].Hostname != "host-1" || list[2].Hostname != "host-3" {
		t.Errorf("unexpected list content: %+v", list)
	}

	// Add 4th item, should slide out the first
	history.Add(&SystemStats{Hostname: "host-4"})

	list = history.Get()
	if len(list) != 3 {
		t.Fatalf("expected limit to keep 3 items, got %d", len(list))
	}
	if list[0].Hostname != "host-2" || list[2].Hostname != "host-4" {
		t.Errorf("expected host-1 to slide out, got: %+v", list)
	}
}

func TestGetDetailedStats(t *testing.T) {
	stats, err := GetDetailedStats()
	if err != nil {
		t.Fatalf("GetDetailedStats failed: %v", err)
	}
	t.Logf("Stats: CPU=%.1f%% (Temp=%.1fC), Mem=%.1f%% (%.1f/%.1fGB), Disk=%.1f%% (%.1f/%.1fGB), Partitions=%d, GPU=%s Util=%.1f%% Temp=%.1fC Mem=%.1f/%.1fMB",
		stats.CPUUsage, stats.CPUTemp, stats.MemoryUsage, stats.MemoryUsed, stats.MemoryTotal, stats.DiskUsage, stats.DiskUsed, stats.DiskTotal, len(stats.DiskPartitions), stats.GPUModel, stats.GPUUsage, stats.GPUTemp, stats.GPUMemUsed, stats.GPUMemTotal)
	for i, p := range stats.DiskPartitions {
		t.Logf("  Partition [%d]: %s (%s) mounted at %s - %.1f/%.1f GB (%.1f%%)", i, p.Device, p.FSType, p.Mountpoint, p.Used, p.Total, p.UsedPercent)
	}
}
