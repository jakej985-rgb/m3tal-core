package system

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemStats represents system metrics
type SystemStats struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	Uptime      uint64  `json:"uptime"`
	Hostname    string  `json:"hostname"`
}

// GetStats collects system metrics (basic)
func GetStats() (*SystemStats, error) {
	stats := &SystemStats{}

	// CPU
	percentages, err := cpu.Percent(time.Second, false)
	if err == nil && len(percentages) > 0 {
		stats.CPUUsage = percentages[0]
	}

	// Memory
	vm, err := mem.VirtualMemory()
	if err == nil {
		stats.MemoryUsage = vm.UsedPercent
	}

	// Disk (root)
	usage, err := disk.Usage("/")
	if err == nil {
		stats.DiskUsage = usage.UsedPercent
	}

	// Host info
	info, err := host.Info()
	if err == nil {
		stats.Uptime = info.Uptime
		stats.Hostname = info.Hostname
	}

	return stats, nil
}

// GetDetailedStats collects rich metrics including GPU, temp, and details
func GetDetailedStats() (*models.DetailedStats, error) {
	stats := &models.DetailedStats{}

	// CPU Usage
	percentages, err := cpu.Percent(200*time.Millisecond, false)
	if err == nil && len(percentages) > 0 {
		stats.CPUUsage = percentages[0]
	}

	// CPU Temp
	stats.CPUTemp = getCPUTemp()

	// Memory
	vm, err := mem.VirtualMemory()
	if err == nil {
		stats.MemoryUsage = vm.UsedPercent
		stats.MemoryUsed = float64(vm.Used) / (1024 * 1024 * 1024)
		stats.MemoryTotal = float64(vm.Total) / (1024 * 1024 * 1024)
	}

	// Memory Frequency
	stats.MemoryFrequency = getMemoryFrequency()

	// Disk
	usage, err := disk.Usage("/")
	if err == nil {
		stats.DiskUsage = usage.UsedPercent
		stats.DiskUsed = float64(usage.Used) / (1024 * 1024 * 1024)
		stats.DiskTotal = float64(usage.Total) / (1024 * 1024 * 1024)
	}

	// Disk Partitions
	if partitions, err := GetDiskPartitions(); err == nil {
		stats.DiskPartitions = partitions
	}

	// Host Info
	info, err := host.Info()
	if err == nil {
		stats.Uptime = info.Uptime
		stats.Hostname = info.Hostname
	}

	// GPU Stats (nvidia-smi fallback) — served from a short-lived cache so
	// polling endpoints don't spawn a subprocess per request.
	gpu := getGPUStatsCached()
	stats.GPUUsage = gpu.usage
	stats.GPUTemp = gpu.temp
	stats.GPUMemUsed = gpu.memUsed
	stats.GPUMemTotal = gpu.memTotal
	stats.GPUModel = gpu.model

	// Network Interfaces & Throughput
	if ifList, err := GetNetworkInterfaces(); err == nil {
		stats.NetworkInterfaces = ifList
	}

	return stats, nil
}

func getCPUTemp() float64 {
	temps, err := host.SensorsTemperatures()
	if err == nil {
		for _, t := range temps {
			name := strings.ToLower(t.SensorKey)
			if strings.Contains(name, "core") || strings.Contains(name, "cpu") || strings.Contains(name, "temp1") {
				return t.Temperature
			}
		}
	}
	return 0
}

// getGPUStatsCached wraps getGPUStats with a 5s TTL. nvidia-smi costs
// ~50-300ms per spawn; dashboard polling hits this every few seconds, so a
// cache aligned with the poll cadence removes nearly every subprocess call
// while keeping readings fresh enough for telemetry.
var (
	gpuStatsMu    sync.Mutex
	gpuStatsCache gpuStatsResult
	gpuStatsAt    time.Time
)

const gpuStatsTTL = 5 * time.Second

type gpuStatsResult struct {
	usage, temp, memUsed, memTotal float64
	model                          string
}

func getGPUStatsCached() gpuStatsResult {
	gpuStatsMu.Lock()
	defer gpuStatsMu.Unlock()
	if !gpuStatsAt.IsZero() && time.Since(gpuStatsAt) < gpuStatsTTL {
		return gpuStatsCache
	}
	u, t, mu, mt, model := getGPUStats()
	gpuStatsCache = gpuStatsResult{usage: u, temp: t, memUsed: mu, memTotal: mt, model: model}
	gpuStatsAt = time.Now()
	return gpuStatsCache
}

func getGPUStats() (usage float64, temp float64, memUsed float64, memTotal float64, model string) {
	out, err := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu,temperature.gpu,memory.used,memory.total,name", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return 0, 0, 0, 0, "No GPU Detected"
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) >= 5 {
		fmt.Sscanf(parts[0], "%f", &usage)
		fmt.Sscanf(parts[1], "%f", &temp)
		fmt.Sscanf(parts[2], "%f", &memUsed)
		fmt.Sscanf(parts[3], "%f", &memTotal)
		model = strings.TrimSpace(parts[4])
	}
	return
}

// memoryFrequency probes `inxi -m` once — RAM speed cannot change at runtime,
// so the subprocess result is cached for the process lifetime.
var memoryFrequency = sync.OnceValue(func() string {
	out, err := exec.Command("/usr/bin/inxi", "-m").Output()
	if err != nil {
		return "Unknown"
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "speed:") {
			parts := strings.Split(line, "speed:")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown"
})

func getMemoryFrequency() string {
	return memoryFrequency()
}
