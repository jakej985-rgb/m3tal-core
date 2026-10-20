package output

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"github.com/jakej985-rgb/m3tal-core/pkg/registry"
)

// PrintContainersTable formats a list of containers as a table to standard output.
func PrintContainersTable(containers []models.Container) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "CONTAINER ID\tNAMES\tIMAGE\tSTATE\tSTATUS")
	fmt.Fprintln(w, "------------\t-----\t-----\t-----\t------")
	for _, c := range containers {
		id := c.ID
		if len(id) > 12 {
			id = id[:12]
		}
		names := strings.Join(c.Names, ",")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", id, names, c.Image, c.State, c.Status)
	}
	w.Flush()
}

// PrintStacksTable formats a list of stacks as a table to standard output.
func PrintStacksTable(stacks []models.Stack) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tPATH\tSERVICES")
	fmt.Fprintln(w, "----\t------\t----\t--------")
	for _, s := range stacks {
		svcs := strings.Join(s.Services, ",")
		if svcs == "" {
			svcs = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Status, s.ComposePath, svcs)
	}
	w.Flush()
}

// PrintStats prints system metrics in a styled dashboard layout.
func PrintStats(s *models.MetricsResponse) {
	fmt.Printf("💻 Hostname: %s\n", s.Hostname)
	uptimeDur := time.Duration(s.Uptime) * time.Second
	fmt.Printf("⏱️  Uptime:   %s\n", uptimeDur.String())
	fmt.Printf("🧠 Memory:   %.1f%%\n", s.MemoryUsage)
	fmt.Printf("💿 Disk:     %.1f%%\n", s.DiskUsage)
	fmt.Printf("🔥 CPU:      %.1f%%\n", s.CPUUsage)
}

// PrintDetailedStats prints comprehensive system telemetry including GPU, VRAM, Temps, and all Mounted & Network Drives.
func PrintDetailedStats(s *models.DetailedStats) {
	fmt.Printf("💻 Hostname: %s\n", s.Hostname)
	uptimeDur := time.Duration(s.Uptime) * time.Second
	days := int(uptimeDur.Hours()) / 24
	hours := int(uptimeDur.Hours()) % 24
	mins := int(uptimeDur.Minutes()) % 60
	if days > 0 {
		fmt.Printf("⏱️  Uptime:   %dd %dh %dm\n", days, hours, mins)
	} else {
		fmt.Printf("⏱️  Uptime:   %dh %dm\n", hours, mins)
	}

	// CPU
	if s.CPUTemp > 0 {
		tempF := s.CPUTemp*1.8 + 32.0
		fmt.Printf("🔥 CPU:      %.1f%%  (🌡️  %.1f°C / %.1f°F)\n", s.CPUUsage, s.CPUTemp, tempF)
	} else {
		fmt.Printf("🔥 CPU:      %.1f%%\n", s.CPUUsage)
	}

	// Memory
	memStr := fmt.Sprintf("%.1f%%  [%.1f / %.1f GB]", s.MemoryUsage, s.MemoryUsed, s.MemoryTotal)
	if s.MemoryFrequency != "" && s.MemoryFrequency != "Unknown" {
		memStr += fmt.Sprintf("  (%s)", s.MemoryFrequency)
	}
	fmt.Printf("🧠 Memory:   %s\n", memStr)

	// GPU & VRAM
	if s.GPUModel != "" && s.GPUModel != "No GPU Detected" {
		fmt.Printf("\n📟 GPU:      %s\n", s.GPUModel)
		if s.GPUTemp > 0 {
			gpuTempF := s.GPUTemp*1.8 + 32.0
			fmt.Printf("   • Core:   %.1f%%  (🌡️  %.1f°C / %.1f°F)\n", s.GPUUsage, s.GPUTemp, gpuTempF)
		} else {
			fmt.Printf("   • Core:   %.1f%%\n", s.GPUUsage)
		}
		if s.GPUMemTotal > 0 {
			vramPct := (s.GPUMemUsed / s.GPUMemTotal) * 100.0
			fmt.Printf("   • VRAM:   %.1f%%  [%.0f / %.0f MB]\n", vramPct, s.GPUMemUsed, s.GPUMemTotal)
		}
	}

	// Mounted Disks & Network Drives
	if len(s.DiskPartitions) > 0 {
		fmt.Println("\n💿 Storage & Mounted Drives:")
		for _, p := range s.DiskPartitions {
			isNet := strings.HasPrefix(p.Device, "//") || strings.HasPrefix(p.Device, "\\\\") || strings.Contains(p.Device, ":/") || p.FSType == "cifs" || strings.HasPrefix(p.FSType, "nfs")
			typeTag := ""
			if isNet {
				typeTag = "  [Network Share]"
			}

			// Render progress bar
			filled := int(p.UsedPercent / 10.0)
			if filled < 0 {
				filled = 0
			}
			if filled > 10 {
				filled = 10
			}
			bar := "[" + strings.Repeat("█", filled) + strings.Repeat("░", 10-filled) + "]"

			// Format capacity (GB or TB)
			formatCap := func(gb float64) string {
				if gb >= 1000.0 {
					return fmt.Sprintf("%.1f TB", gb/1024.0)
				}
				return fmt.Sprintf("%.1f GB", gb)
			}

			dispLabel := p.Label
			if dispLabel == "" || dispLabel == filepath.Base(p.Device) {
				dispLabel = p.Mountpoint
			} else {
				dispLabel = fmt.Sprintf("%s (%s)", dispLabel, p.Mountpoint)
			}

			fmt.Printf("   • %-30s %-6s %8s / %-8s  %s  %5.1f%%%s\n",
				dispLabel,
				p.FSType,
				formatCap(p.Used),
				formatCap(p.Total),
				bar,
				p.UsedPercent,
				typeTag,
			)
		}
	} else {
		fmt.Printf("💿 Disk:     %.1f%%  [%.1f / %.1f GB]\n", s.DiskUsage, s.DiskUsed, s.DiskTotal)
	}
}

// PrintRoutesTable displays a list of active reverse proxy routes.
func PrintRoutesTable(routes []models.RouteRecord) {
	if len(routes) == 0 {
		fmt.Println("No reverse proxy routes configured.")
		return
	}

	fmt.Printf("\n%-20s %-28s %-6s %-5s %-20s\n", "SERVICE", "DOMAIN", "PORT", "SSL", "MIDDLEWARES")
	fmt.Println(strings.Repeat("-", 84))
	for _, r := range routes {
		sslStr := "No"
		if r.SSL {
			sslStr = "Yes"
		}
		mwStr := r.Middlewares
		if mwStr == "" {
			mwStr = "-"
		}
		fmt.Printf("%-20s %-28s %-6d %-5s %-20s\n", r.Service, r.Domain, r.Port, sslStr, mwStr)
	}
	fmt.Println()
}

// PrintDiscoverTable prints discoverable candidate containers.
func PrintDiscoverTable(services []models.DiscoverableService) {
	if len(services) == 0 {
		fmt.Println("No services discovered.")
		return
	}

	fmt.Printf("\n%-20s %-24s %-8s %-8s %-20s\n", "CONTAINER", "IMAGE", "STATE", "EXPOSED", "PORTS")
	fmt.Println(strings.Repeat("-", 84))
	for _, s := range services {
		expStr := "No"
		if s.Exposed {
			expStr = "Yes (" + s.Domain + ")"
		}

		var portStrs []string
		for _, p := range s.Ports {
			portStrs = append(portStrs, strconv.Itoa(p))
		}
		portsJoin := strings.Join(portStrs, ",")
		if portsJoin == "" {
			portsJoin = "-"
		}

		fmt.Printf("%-20s %-24s %-8s %-8s %-20s\n", s.ContainerName, truncateString(s.Image, 24), s.State, expStr, portsJoin)
	}
	fmt.Println()
}

// PrintVPNStatus formats and prints the VPN status structure.
func PrintVPNStatus(status *models.VPNStatus) {
	fmt.Println("🌐 VPN Connection Status:")
	fmt.Println("----------------------------------------")
	if status.Connected {
		fmt.Println("🟢 Status:      Connected (Running)")
		fmt.Printf("🔒 Provider:    %s\n", status.Provider)
		fmt.Printf("🌍 Region:      %s\n", status.Region)
		fmt.Printf("📬 External IP: %s\n", status.ExternalIP)
		if status.ForwardedPort > 0 {
			fmt.Printf("🔌 Port:        %d (Forwarded)\n", status.ForwardedPort)
		}
	} else {
		fmt.Printf("🔴 Status:      Disconnected (%s)\n", status.StatusText)
	}
}

// PrintVPNLeakReport formats and prints the leak check report.
func PrintVPNLeakReport(report *models.VPNLeakReport) {
	fmt.Println("\n🛡️  VPN Leak Detection Report:")
	fmt.Println("----------------------------------------")
	fmt.Printf("🏠 Host Public IP: %s\n", report.HostIP)
	fmt.Printf("🔒 VPN Outbound IP: %s\n", report.VPNIP)

	if report.Leak {
		fmt.Println("🚨 RESULT: LEAK DETECTED! Your traffic is NOT protected!")
		if len(report.StoppedContainers) > 0 {
			fmt.Println("⚠️  Activating kill switch (stopping dependent containers)...")
			fmt.Printf("🛑 Successfully stopped containers: %s\n", strings.Join(report.StoppedContainers, ", "))
		} else {
			fmt.Println("✅ No active dependent containers found running.")
		}
	} else {
		fmt.Println("✅ RESULT: SAFE. Outbound IP is protected by VPN.")
	}
}

// PrintRegistrySearchTable formats registry search results.
func PrintRegistrySearchTable(stacks []registry.StackMetadata) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tCATEGORY\tVERSION\tDESCRIPTION")
	for _, s := range stacks {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Category, s.Version, s.Description)
	}
	w.Flush()
}

// FatalError prints the error to stderr and calls log.Fatalf.
func FatalError(err error) {
	log.Fatalf("❌ Error: %v", err)
}

// FatalErrorMsg prints a formatted error message and exits.
func FatalErrorMsg(msg string, args ...any) {
	log.Fatalf("❌ "+msg, args...)
}

// PrintError prints the error to stderr without exiting.
func PrintError(err error) {
	fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
}

func truncateString(s string, l int) string {
	if len(s) > l {
		return s[:l-3] + "..."
	}
	return s
}
