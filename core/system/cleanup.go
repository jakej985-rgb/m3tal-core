package system

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"github.com/shirou/gopsutil/v3/disk"
)

// FormatBytes converts a raw byte count into a human-readable string.
func FormatBytes(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// CleanSystem executes safe host cleanup routines across package managers (APT, Flatpak, Snap)
// and temporary/cache directories.
func CleanSystem() (*models.SystemCleanupResult, error) {
	var beforeFree uint64
	usageBefore, err := disk.Usage("/")
	if err == nil && usageBefore != nil {
		beforeFree = usageBefore.Free
	}

	var itemsCleaned []string
	var details []string

	// 1. APT cleanup
	if _, err := exec.LookPath("apt-get"); err == nil {
		// Clean cached .deb archives
		cmdClean := exec.Command("apt-get", "clean")
		_ = cmdClean.Run()

		// Remove obsolete packages from archive cache
		cmdAutoClean := exec.Command("apt-get", "autoclean", "-y")
		_ = cmdAutoClean.Run()

		// Autoremove orphan dependencies
		cmdAutoRemove := exec.Command("apt-get", "autoremove", "-y")
		if out, err := cmdAutoRemove.CombinedOutput(); err == nil {
			details = append(details, "APT: clean, autoclean, and autoremove finished successfully.")
		} else {
			details = append(details, fmt.Sprintf("APT autoremove completed with notice: %s", strings.TrimSpace(string(out))))
		}
		itemsCleaned = append(itemsCleaned, "APT package cache & unused dependencies")
	}

	// 2. Flatpak cleanup
	if _, err := exec.LookPath("flatpak"); err == nil {
		cmdFlatpak := exec.Command("flatpak", "uninstall", "--unused", "-y")
		if out, err := cmdFlatpak.CombinedOutput(); err == nil {
			details = append(details, "Flatpak: unused runtimes uninstalled.")
			itemsCleaned = append(itemsCleaned, "Flatpak unused runtimes")
		} else {
			details = append(details, fmt.Sprintf("Flatpak: %s", strings.TrimSpace(string(out))))
		}
	}

	// 3. Snap cleanup (remove disabled / old revisions)
	if _, err := exec.LookPath("snap"); err == nil {
		cmdList := exec.Command("snap", "list", "--all")
		if out, err := cmdList.Output(); err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(out))
			removedSnaps := 0
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.Contains(line, "disabled") {
					fields := strings.Fields(line)
					if len(fields) >= 3 {
						snapName := fields[0]
						snapRev := fields[2]
						cmdRemove := exec.Command("snap", "remove", snapName, fmt.Sprintf("--revision=%s", snapRev))
						if cmdRemove.Run() == nil {
							removedSnaps++
						}
					}
				}
			}
			if removedSnaps > 0 {
				details = append(details, fmt.Sprintf("Snap: removed %d disabled revision(s).", removedSnaps))
				itemsCleaned = append(itemsCleaned, fmt.Sprintf("Snap disabled revisions (%d removed)", removedSnaps))
			}
		}
	}

	// 4. Systemd journal vacuum (retain 3 days)
	if _, err := exec.LookPath("journalctl"); err == nil {
		cmdJournal := exec.Command("journalctl", "--vacuum-time=3d")
		if out, err := cmdJournal.CombinedOutput(); err == nil {
			details = append(details, "Journalctl: vacuumed older logs (>3 days).")
			itemsCleaned = append(itemsCleaned, "Systemd journal vacuum (3d)")
		} else {
			details = append(details, fmt.Sprintf("Journalctl notice: %s", strings.TrimSpace(string(out))))
		}
	}

	// 5. Temporary & thumbnail caches
	tempCleaned := cleanTempAndCaches()
	if tempCleaned > 0 {
		details = append(details, fmt.Sprintf("Caches: cleaned %d thumbnail and temporary cache files.", tempCleaned))
		itemsCleaned = append(itemsCleaned, "User thumbnail cache & temporary files")
	}

	// Measure freed space
	var freedBytes int64
	usageAfter, err := disk.Usage("/")
	if err == nil && usageAfter != nil && beforeFree > 0 {
		if usageAfter.Free > beforeFree {
			freedBytes = int64(usageAfter.Free - beforeFree)
		}
	}

	return &models.SystemCleanupResult{
		FreedBytes:   freedBytes,
		FreedHuman:   FormatBytes(freedBytes),
		ItemsCleaned: itemsCleaned,
		Details:      strings.Join(details, "\n"),
	}, nil
}

// cleanTempAndCaches removes obsolete thumbnail files and safe temp files older than 48 hours.
func cleanTempAndCaches() int {
	cleaned := 0

	// 1. User thumbnail caches (~/.cache/thumbnails)
	if homeDir, err := os.UserHomeDir(); err == nil {
		thumbDir := filepath.Join(homeDir, ".cache", "thumbnails")
		if entries, err := os.ReadDir(thumbDir); err == nil {
			for _, entry := range entries {
				p := filepath.Join(thumbDir, entry.Name())
				if err := os.RemoveAll(p); err == nil {
					cleaned++
				}
			}
		}
	}

	// 2. Safe removal of temporary files older than 48h in /tmp
	cutoff := time.Now().Add(-48 * time.Hour)
	if entries, err := os.ReadDir("/tmp"); err == nil {
		for _, entry := range entries {
			// Do not attempt to delete active socket files or system lock files
			name := entry.Name()
			if strings.HasPrefix(name, ".") || strings.Contains(name, "socket") || strings.Contains(name, "lock") {
				continue
			}
			info, err := entry.Info()
			if err == nil && info.ModTime().Before(cutoff) {
				p := filepath.Join("/tmp", name)
				if err := os.Remove(p); err == nil {
					cleaned++
				}
			}
		}
	}

	return cleaned
}
