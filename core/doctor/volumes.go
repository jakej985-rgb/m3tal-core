package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type MountType = models.MountType
type MountResult = models.MountResult

const (
	MountTypeBind   = models.MountTypeBind
	MountTypeVolume = models.MountTypeVolume
	MountTypeTmpfs  = models.MountTypeTmpfs
)

// dockerMount is the minimal shape of a Docker inspect mount entry.
type dockerMount struct {
	Type        string `json:"Type"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Mode        string `json:"Mode"`
	RW          bool   `json:"RW"`
	Name        string `json:"Name"` // populated for named volumes
}

type dockerMountInspect struct {
	Name   string        `json:"Name"`
	Mounts []dockerMount `json:"Mounts"`
}

// ValidateMounts inspects all running containers and checks each mount for
// accessibility issues.
func ValidateMounts() ([]MountResult, error) {
	// Get list of all container IDs
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// List all container IDs
	out, err := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{.ID}}").Output()
	if err != nil {
		return nil, fmt.Errorf("cannot list containers: %w", err)
	}

	ids := strings.Fields(strings.TrimSpace(string(out)))
	if len(ids) == 0 {
		return nil, nil
	}

	// Batch inspect all containers
	args := append([]string{"inspect"}, ids...)
	inspectOut, err := exec.CommandContext(context.Background(), "docker", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect failed: %w", err)
	}

	var inspected []dockerMountInspect
	if err := json.Unmarshal(inspectOut, &inspected); err != nil {
		return nil, fmt.Errorf("cannot parse inspect output: %w", err)
	}

	var results []MountResult
	for _, c := range inspected {
		name := strings.TrimPrefix(c.Name, "/")
		for _, m := range c.Mounts {
			r := validateMount(name, m)
			results = append(results, r)
		}
	}

	return results, nil
}

func validateMount(containerName string, m dockerMount) MountResult {
	mt := MountType(strings.ToLower(m.Type))
	r := MountResult{
		Container: containerName,
		Source:    m.Source,
		Target:    m.Destination,
		Type:      mt,
		ReadOnly:  !m.RW,
		Severity:  SeverityPass,
	}

	switch mt {
	case MountTypeBind:
		validateBindMount(&r)
	case MountTypeVolume:
		validateNamedVolume(&r, m.Name)
	case MountTypeTmpfs:
		// tmpfs mounts have no host path — always OK
		r.Severity = SeverityPass
	default:
		r.Severity = SeverityPass
	}

	return r
}

// validateBindMount checks that the host path for a bind mount exists and is
// accessible (with correct read/write permissions).
func validateBindMount(r *MountResult) {
	if r.Source == "" {
		r.Severity = SeverityWarn
		r.Issue = "Bind mount has an empty source path"
		r.Fix = fmt.Sprintf("Inspect container %q and verify its compose file", r.Container)
		return
	}

	info, err := os.Stat(r.Source)
	if os.IsNotExist(err) {
		r.Severity = SeverityFail
		r.Issue = fmt.Sprintf("Host path does not exist: %s", r.Source)
		r.Fix = fmt.Sprintf("mkdir -p %s", r.Source)
		return
	}
	if err != nil {
		r.Severity = SeverityWarn
		r.Issue = fmt.Sprintf("Cannot stat host path %s: %v", r.Source, err)
		return
	}

	// Check read access
	if !isReadable(r.Source, info.IsDir()) {
		r.Severity = SeverityFail
		r.Issue = fmt.Sprintf("Host path not readable: %s", r.Source)
		r.Fix = fmt.Sprintf("sudo chmod a+r %s", r.Source)
		return
	}

	// Check write access for RW mounts
	if !r.ReadOnly && !isWritable(r.Source) {
		r.Severity = SeverityWarn
		r.Issue = fmt.Sprintf("Host path not writable (mount is RW): %s", r.Source)
		r.Fix = fmt.Sprintf("sudo chmod a+w %s", r.Source)
		return
	}
}

// validateNamedVolume checks that a named Docker volume still exists.
func validateNamedVolume(r *MountResult, volName string) {
	if volName == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "docker", "volume", "inspect", volName).Output()
	if err != nil {
		r.Severity = SeverityFail
		r.Issue = fmt.Sprintf("Named volume %q not found or dangling", volName)
		r.Fix = fmt.Sprintf("docker volume create %s", volName)
		return
	}
	// Parse mount path from volume inspect to verify host dir exists
	var vols []struct {
		Mountpoint string `json:"Mountpoint"`
	}
	if json.Unmarshal(out, &vols) == nil && len(vols) > 0 {
		mp := vols[0].Mountpoint
		if mp != "" {
			if _, statErr := os.Stat(mp); os.IsNotExist(statErr) {
				r.Severity = SeverityWarn
				r.Issue = fmt.Sprintf("Volume mountpoint missing on host: %s", mp)
				r.Fix = fmt.Sprintf("sudo mkdir -p %s", mp)
				return
			}
		}
	}
}

// isReadable returns true if the current process can read the path.
func isReadable(path string, isDir bool) bool {
	if isDir {
		f, err := os.Open(path)
		if err != nil {
			return false
		}
		f.Close()
		return true
	}
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// isWritable returns true if the current process can write to the path.
func isWritable(path string) bool {
	var st syscall.Stat_t
	if err := syscall.Stat(path, &st); err != nil {
		return false
	}
	// Check owner write bit; for a real check we'd also look at uid/gid
	mode := st.Mode
	return mode&0200 != 0 || mode&0020 != 0 || mode&0002 != 0
}
