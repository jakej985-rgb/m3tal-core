package system

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
)

// CheckUpdates runs apt dry-run to discover packages that need updating.
func CheckUpdates() (*models.SystemUpdates, error) {
	// Verify if apt-get is available
	if _, err := exec.LookPath("apt-get"); err != nil {
		return &models.SystemUpdates{HasUpdates: false, Count: 0, UpdatesList: []string{}}, nil
	}

	// Run dry-run package upgrade check
	cmd := exec.Command("apt-get", "-s", "upgrade")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		// If apt-get fails (e.g. locks or other errors), return empty list gracefully
		return &models.SystemUpdates{HasUpdates: false, Count: 0, UpdatesList: []string{}}, nil
	}

	var updates []string
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Inst ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				pkgName := fields[1]
				// Include version details if present
				verInfo := ""
				if len(fields) >= 3 {
					verInfo = " " + fields[2]
				}
				updates = append(updates, pkgName+verInfo)
			}
		}
	}

	return &models.SystemUpdates{
		HasUpdates:  len(updates) > 0,
		Count:       len(updates),
		UpdatesList: updates,
	}, nil
}

// TriggerUpdates performs the actual apt package update and upgrade.
func TriggerUpdates() error {
	// We run apt-get update first
	cmdUpdate := exec.Command("apt-get", "update")
	_ = cmdUpdate.Run()

	// Then run apt-get upgrade non-interactively
	cmdUpgrade := exec.Command("apt-get", "upgrade", "-y")
	return cmdUpgrade.Run()
}
