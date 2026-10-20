package config

import (
	"os"
	"path/filepath"
	"strings"
)

// StackInfo holds the file paths for a stack's compose and env template.
// The paths are absolute or relative to the Docker stack directory.
// In practice they are absolute paths beginning with /usr/share/m3tal/docker.
// The MergeEnv function later will use these paths.
//
// Example: stack "media" will have Compose="/usr/share/m3tal/docker/media-compose.yml"
// and Template="/usr/share/m3tal/docker/media.env.template".
//
// Public fields intentionally exported for tests.
// note: The struct can be extended to hold more metadata (e.g. stack ID).

type StackInfo struct {
	Compose  string
	Template string
}

// ScanStacks scans the directory containing stack definitions for files matching
// the expected naming pattern ("*-compose.yml") and verifies a matching
// "*.env.template" exists. The function returns a map from stack name to
// StackInfo or an error if the directory cannot be read.
//
// baseDir is the root directory containing stack files. If an empty string is
// passed, the function defaults to the system default directory
// "/usr/share/m3tal/docker".
func ScanStacks(baseDir string) (map[string]StackInfo, error) {
	if baseDir == "" {
		baseDir = "/usr/share/m3tal/docker"
	}
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	result := make(map[string]StackInfo)
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, "-compose.yml") {
			stack := strings.TrimSuffix(name, "-compose.yml")
			composePath := filepath.Join(baseDir, name)
			templatePath := filepath.Join(baseDir, stack+".env.template")
			if _, err := os.Stat(templatePath); os.IsNotExist(err) {
				// Skip stacks without a template, matching existing logic.
				continue
			}
			result[stack] = StackInfo{Compose: composePath, Template: templatePath}
		}
	}
	return result, nil
}
