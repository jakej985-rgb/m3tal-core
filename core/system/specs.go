package system

import (
	"strings"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// GetSystemSpecs returns detailed hardware, operating system, and network specifications.
func GetSystemSpecs() (*models.SystemSpecs, error) {
	specs := &models.SystemSpecs{}

	// Host / OS info
	hostInfo, err := host.Info()
	if err == nil {
		specs.Hostname = hostInfo.Hostname
		specs.OS = hostInfo.OS
		specs.Platform = hostInfo.Platform
		specs.PlatformFamily = hostInfo.PlatformFamily
		specs.PlatformVersion = hostInfo.PlatformVersion
		specs.KernelVersion = hostInfo.KernelVersion
		specs.KernelArch = hostInfo.KernelArch
		specs.Virtualization = hostInfo.VirtualizationSystem
		if specs.Virtualization == "" {
			specs.Virtualization = "Bare Metal"
		}
		specs.Uptime = hostInfo.Uptime
	}

	// CPU info
	cpuInfos, err := cpu.Info()
	if err == nil && len(cpuInfos) > 0 {
		specs.CPUModel = cpuInfos[0].ModelName
		specs.CPUCores = cpuInfos[0].Cores
		specs.CPUMHz = cpuInfos[0].Mhz
	}
	threads, err := cpu.Counts(true)
	if err == nil {
		specs.CPUThreads = threads
	}

	// Memory & Swap
	vm, err := mem.VirtualMemory()
	if err == nil {
		specs.MemoryTotal = float64(vm.Total) / (1024 * 1024 * 1024)
		specs.MemoryUsed = float64(vm.Used) / (1024 * 1024 * 1024)
	}
	swap, err := mem.SwapMemory()
	if err == nil {
		specs.SwapTotal = float64(swap.Total) / (1024 * 1024 * 1024)
		specs.SwapUsed = float64(swap.Used) / (1024 * 1024 * 1024)
	}

	// GPU
	_, _, _, _, gpuModel := getGPUStats()
	specs.GPUModel = gpuModel
	if ifList, err := GetNetworkInterfaces(); err == nil {
		specs.NetworkInterfaces = ifList
	}

	return specs, nil
}

// GetNetworkInterfaces collects physical and virtual network interfaces and traffic counters.
func GetNetworkInterfaces() ([]models.NetworkInterfaceInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ioCounters, _ := net.IOCounters(true)
	ioMap := make(map[string]net.IOCountersStat)
	for _, io := range ioCounters {
		ioMap[io.Name] = io
	}

	ifList := make([]models.NetworkInterfaceInfo, 0, len(interfaces))
	for _, iface := range interfaces {
		// Ignore inactive loopback interface
		if iface.Name == "lo" {
			continue
		}

		isUp := false
		for _, flag := range iface.Flags {
			if strings.ToLower(flag) == "up" {
				isUp = true
				break
			}
		}

		ips := make([]string, 0, len(iface.Addrs))
		for _, addr := range iface.Addrs {
			ips = append(ips, addr.Addr)
		}

		var bytesSent, bytesRecv uint64
		if io, ok := ioMap[iface.Name]; ok {
			bytesSent = io.BytesSent
			bytesRecv = io.BytesRecv
		}

		ifList = append(ifList, models.NetworkInterfaceInfo{
			Name:        iface.Name,
			IPAddresses: ips,
			BytesSent:   bytesSent,
			BytesRecv:   bytesRecv,
			IsUp:        isUp,
		})
	}

	return ifList, nil
}
