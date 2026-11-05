package models

// SystemInfo represents basic system information.
type SystemInfo struct {
	Hostname string `json:"hostname"`
	Uptime   uint64 `json:"uptime"`
}

// DetailedStats represents extended system metrics for dashboard/tray.
type DetailedStats struct {
	CPUUsage          float64                `json:"cpu_usage"`
	CPUTemp           float64                `json:"cpu_temp"`
	MemoryUsage       float64                `json:"memory_usage"`
	MemoryUsed        float64                `json:"memory_used"`  // GB
	MemoryTotal       float64                `json:"memory_total"` // GB
	MemoryFrequency   string                 `json:"memory_frequency"`
	DiskUsage         float64                `json:"disk_usage"`
	DiskUsed          float64                `json:"disk_used"`  // GB
	DiskTotal         float64                `json:"disk_total"` // GB
	DiskPartitions    []DiskPartition        `json:"disk_partitions"`
	GPUUsage          float64                `json:"gpu_usage"`
	GPUTemp           float64                `json:"gpu_temp"`
	GPUMemUsed        float64                `json:"gpu_mem_used"`  // MB
	GPUMemTotal       float64                `json:"gpu_mem_total"` // MB
	GPUModel          string                 `json:"gpu_model"`
	NetworkInterfaces []NetworkInterfaceInfo `json:"network_interfaces,omitempty"`
	Uptime            uint64                 `json:"uptime"`
	Hostname          string                 `json:"hostname"`
}

// DiskPartition represents a storage partition or mount point.
type DiskPartition struct {
	Device      string  `json:"device"`
	DiskName    string  `json:"disk_name,omitempty"`
	DiskModel   string  `json:"disk_model,omitempty"`
	DriveType   string  `json:"drive_type,omitempty"` // "system", "usb", "network", "internal"
	Label       string  `json:"label"`
	Mountpoint  string  `json:"mountpoint"`
	FSType      string  `json:"fstype"`
	Total       float64 `json:"total"`        // GB
	Used        float64 `json:"used"`         // GB
	Free        float64 `json:"free"`         // GB
	UsedPercent float64 `json:"used_percent"` // %
}

// ProcessInfo represents a running process on the host.
type ProcessInfo struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	Username      string  `json:"username"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float32 `json:"memory_percent"`
	MemoryRSS     uint64  `json:"memory_rss"` // bytes
	Status        string  `json:"status"`
	Cmdline       string  `json:"cmdline"`
	CreateTime    int64   `json:"create_time"`
}

// NetworkInterfaceInfo represents a network interface and its counters.
type NetworkInterfaceInfo struct {
	Name        string   `json:"name"`
	IPAddresses []string `json:"ip_addresses"`
	BytesSent   uint64   `json:"bytes_sent"`
	BytesRecv   uint64   `json:"bytes_recv"`
	IsUp        bool     `json:"is_up"`
}

// SystemSpecs represents comprehensive host hardware and OS profile.
type SystemSpecs struct {
	Hostname          string                 `json:"hostname"`
	OS                string                 `json:"os"`
	Platform          string                 `json:"platform"`
	PlatformFamily    string                 `json:"platform_family"`
	PlatformVersion   string                 `json:"platform_version"`
	KernelVersion     string                 `json:"kernel_version"`
	KernelArch        string                 `json:"kernel_arch"`
	Virtualization    string                 `json:"virtualization"`
	Uptime            uint64                 `json:"uptime"`
	CPUModel          string                 `json:"cpu_model"`
	CPUCores          int32                  `json:"cpu_cores"`
	CPUThreads        int                    `json:"cpu_threads"`
	CPUMHz            float64                `json:"cpu_mhz"`
	MemoryTotal       float64                `json:"memory_total"` // GB
	MemoryUsed        float64                `json:"memory_used"`  // GB
	SwapTotal         float64                `json:"swap_total"`   // GB
	SwapUsed          float64                `json:"swap_used"`    // GB
	GPUModel          string                 `json:"gpu_model"`
	NetworkInterfaces []NetworkInterfaceInfo `json:"network_interfaces"`
}
