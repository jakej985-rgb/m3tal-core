package system

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
)

// ListServices returns a list of systemd services and their status.
func ListServices() ([]models.ServiceStatus, error) {
	// 1. Get enabled/disabled state for all services
	enableMap := make(map[string]string)
	cmdFiles := exec.Command("systemctl", "list-unit-files", "--type=service", "--no-legend", "--no-pager")
	var outFiles bytes.Buffer
	cmdFiles.Stdout = &outFiles
	if err := cmdFiles.Run(); err == nil {
		scanner := bufio.NewScanner(&outFiles)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				enableMap[fields[0]] = fields[1]
			}
		}
	}

	// 2. Get active status for all services
	cmdUnits := exec.Command("systemctl", "list-units", "--type=service", "--all", "--no-legend", "--no-pager")
	var outUnits bytes.Buffer
	cmdUnits.Stdout = &outUnits
	if err := cmdUnits.Run(); err != nil {
		return nil, fmt.Errorf("failed to list systemd units: %w", err)
	}

	var services []models.ServiceStatus
	scanner := bufio.NewScanner(&outUnits)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Handle failed indicator dot "●"
		if strings.HasPrefix(line, "●") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "●"))
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		name := fields[0]
		loadState := fields[1]
		activeState := fields[2]
		subState := fields[3]

		// The rest of the line is the description
		desc := ""
		if len(fields) > 4 {
			desc = strings.Join(fields[4:], " ")
		}

		enabled := "unknown"
		if val, ok := enableMap[name]; ok {
			enabled = val
		}

		services = append(services, models.ServiceStatus{
			Name:        name,
			LoadState:   loadState,
			ActiveState: activeState,
			SubState:    subState,
			Description: desc,
			Enabled:     enabled,
		})
	}

	return services, nil
}

// ControlService manages a systemd service state.
func ControlService(name string, action string) error {
	// Validate action to prevent command injection
	switch action {
	case "start", "stop", "restart", "enable", "disable":
		// valid
	default:
		return fmt.Errorf("invalid service action: %s", action)
	}

	// Ensure the name has .service suffix if it doesn't already
	if !strings.HasSuffix(name, ".service") {
		name = name + ".service"
	}

	cmd := exec.Command("systemctl", action, name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run systemctl %s %s: %w", action, name, err)
	}
	return nil
}
