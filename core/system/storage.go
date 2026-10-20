package system

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"github.com/shirou/gopsutil/v3/disk"
)

// getDeviceLabel resolves the volume label for a device by scanning /dev/disk/by-label symlinks.
// If no label is found, it returns the base name of the device (e.g., sda4).
func getDeviceLabel(devicePath string) string {
	absDev, err := filepath.Abs(devicePath)
	if err != nil {
		absDev = devicePath
	}
	files, err := os.ReadDir("/dev/disk/by-label")
	if err == nil {
		for _, file := range files {
			linkPath := filepath.Join("/dev/disk/by-label", file.Name())
			target, err := os.Readlink(linkPath)
			if err == nil {
				resolvedTarget := filepath.Clean(filepath.Join("/dev/disk/by-label", target))
				if resolvedTarget == absDev {
					return file.Name()
				}
			}
		}
	}
	return filepath.Base(devicePath)
}

// GetDiskPartitions returns detailed disk usage for all mounted partitions (physical and network shares).
func GetDiskPartitions() ([]models.DiskPartition, error) {
	partitions, err := disk.Partitions(true) // all partitions including network mounts
	if err != nil {
		return nil, err
	}

	var list []models.DiskPartition
	seenMounts := make(map[string]bool)

	for _, p := range partitions {
		// Skip virtual mounts that look physical
		if strings.HasPrefix(p.Mountpoint, "/var/lib/docker") ||
			strings.HasPrefix(p.Mountpoint, "/run") ||
			seenMounts[p.Mountpoint] {
			continue
		}

		// Check if it is a real physical or network device
		isReal := (strings.HasPrefix(p.Device, "/dev/") && !strings.HasPrefix(p.Device, "/dev/loop")) ||
			strings.HasPrefix(p.Device, "//") ||
			strings.Contains(p.Device, ":/") ||
			p.Fstype == "cifs" || strings.HasPrefix(p.Fstype, "nfs")

		if !isReal {
			continue
		}

		seenMounts[p.Mountpoint] = true

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		// Filter out tiny or pseudo file systems
		if usage.Total < 100*1024*1024 { // < 100 MB
			continue
		}

		totalGB := float64(usage.Total) / (1024 * 1024 * 1024)
		usedGB := float64(usage.Used) / (1024 * 1024 * 1024)
		freeGB := float64(usage.Free) / (1024 * 1024 * 1024)

		list = append(list, models.DiskPartition{
			Device:      p.Device,
			Label:       getDeviceLabel(p.Device),
			Mountpoint:  p.Mountpoint,
			FSType:      p.Fstype,
			Total:       totalGB,
			Used:        usedGB,
			Free:        freeGB,
			UsedPercent: usage.UsedPercent,
		})
	}

	return list, nil
}

// GetSambaShares parses /etc/samba/smb.conf to list Samba network share definitions.
func GetSambaShares() ([]models.SambaShare, error) {
	file, err := os.Open("/etc/samba/smb.conf")
	if err != nil {
		if os.IsNotExist(err) {
			return []models.SambaShare{}, nil // No Samba config, return empty gracefully
		}
		return nil, err
	}
	defer file.Close()

	var shares []models.SambaShare
	var currentShare *models.SambaShare

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Check for section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sectionName := strings.Trim(line, "[]")
			if strings.ToLower(sectionName) == "global" || strings.ToLower(sectionName) == "printers" || strings.ToLower(sectionName) == "homes" {
				if currentShare != nil {
					shares = append(shares, *currentShare)
					currentShare = nil
				}
				continue
			}

			if currentShare != nil {
				shares = append(shares, *currentShare)
			}
			currentShare = &models.SambaShare{
				Name:     sectionName,
				ReadOnly: true, // Samba default
			}
			continue
		}

		if currentShare == nil {
			continue
		}

		// Parse key-value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch key {
		case "path":
			currentShare.Path = val
		case "comment":
			currentShare.Comment = val
		case "read only":
			currentShare.ReadOnly = (strings.ToLower(val) == "yes" || strings.ToLower(val) == "true" || val == "1")
		case "writable", "writeable":
			currentShare.ReadOnly = !(strings.ToLower(val) == "yes" || strings.ToLower(val) == "true" || val == "1")
		case "guest ok", "public":
			currentShare.GuestOk = (strings.ToLower(val) == "yes" || strings.ToLower(val) == "true" || val == "1")
		}
	}

	if currentShare != nil {
		shares = append(shares, *currentShare)
	}

	return shares, nil
}
