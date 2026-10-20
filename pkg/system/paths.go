package system

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	// ConfigPath is the main environment configuration file
	ConfigPath = "/etc/m3tal/.env"

	// StackPath is where the Docker Compose files are stored
	StackPath = "/opt/m3tal/stack"

	// DataPath is the root for persistent application data
	DataPath = "/var/lib/m3tal"

	// UserfacingStackPath is the symlink used for user interaction
	UserfacingStackPath = "/docker"

	// DashPath is where the dashboard files are stored
	DashPath = "/opt/m3tal/dash"

	// StacksDir is where individual stack env files live
	StacksDir = "/etc/m3tal/stacks"

	// SystemPluginsDir is where built-in plugin definitions are installed (via DEB)
	SystemPluginsDir = "/opt/m3tal/plugins"

	// UserPluginsDir is where user-customized plugin definitions live
	UserPluginsDir = "/etc/m3tal/plugins"
)

// PluginSubdirs lists the standard plugin category subdirectories.
var PluginSubdirs = []string{"routes", "stacks", "middleware", "providers", "ai"}

// GetUserPluginsDir returns the active user plugins directory path, supporting overrides.
func GetUserPluginsDir() string {
	if p := os.Getenv("M3TAL_PLUGINS_DIR"); p != "" {
		return p
	}
	if _, err := os.Stat("deploy/plugins"); err == nil {
		return "deploy/plugins"
	}
	return UserPluginsDir
}

// GetPluginDirs returns the ordered list of plugin directories to scan.
// System dir is first, user dir is second (user takes precedence in dedup).
func GetPluginDirs() []string {
	userDir := GetUserPluginsDir()
	dirs := []string{SystemPluginsDir, userDir}

	if userDir != "deploy/plugins" {
		if _, err := os.Stat("deploy/plugins"); err == nil {
			dirs = append([]string{"deploy/plugins"}, dirs...)
		}
	}

	return dirs
}

// GetConfigPath returns the active configuration path, supporting overrides
func GetConfigPath() string {
	if p := os.Getenv("M3TAL_CONFIG"); p != "" {
		return p
	}
	// Check system path first
	if _, err := os.Stat(ConfigPath); err == nil {
		return ConfigPath
	}
	// Fallback to local .env
	return ".env"
}

// GetStackDir returns the directory containing stack compose files
func GetStackDir() string {
	if p := os.Getenv("M3TAL_STACK_DIR"); p != "" {
		return p
	}
	// Check user-facing path first (as per UX rule)
	if _, err := os.Stat(UserfacingStackPath); err == nil {
		return UserfacingStackPath
	}
	// Then system path
	if _, err := os.Stat(StackPath); err == nil {
		return StackPath
	}
	// Fallback to local deploy/stack
	if _, err := os.Stat("deploy/stack"); err == nil {
		return "deploy/stack"
	}
	return "."
}

// IsDashInstalled checks if the dashboard is present on the system
func IsDashInstalled() bool {
	// The plan says check in /opt/m3tal/dash/dash-compose.yml
	// But user corrected: "the compose files will be in /docker"
	// So we check /docker/dash/dash-compose.yml which is /opt/m3tal/stack/dash/dash-compose.yml
	path := "/docker/dash/dash-compose.yml"
	if _, err := os.Stat(path); err == nil {
		return true
	}
	// Also check the real path just in case symlink isn't there yet
	realPath := StackPath + "/dash/dash-compose.yml"
	if _, err := os.Stat(realPath); err == nil {
		return true
	}
	return false
}

// FindComposeFiles recursively searches the root directory for *-compose.yml files, resolving symlinks.
func FindComposeFiles(root string) ([]string, error) {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		resolved = root
	}
	return findComposeFilesRecursive(root, root, resolved, make(map[string]bool))
}

func findComposeFilesRecursive(rootDisplayPath string, currentDisplayPath string, resolvedPath string, visited map[string]bool) ([]string, error) {
	if visited[resolvedPath] {
		return nil, nil
	}
	visited[resolvedPath] = true

	var files []string
	err := filepath.WalkDir(resolvedPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}

		// Handle symlinks to directories encountered during walk
		if d.Type()&os.ModeSymlink != 0 {
			resolvedSymlink, err := filepath.EvalSymlinks(path)
			if err == nil {
				info, err := os.Stat(resolvedSymlink)
				if err == nil && info.IsDir() {
					subDisplayPath := filepath.Join(currentDisplayPath, strings.TrimPrefix(path, resolvedPath))
					subFiles, err := findComposeFilesRecursive(rootDisplayPath, subDisplayPath, resolvedSymlink, visited)
					if err == nil {
						files = append(files, subFiles...)
					}
					return nil
				}
			}
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".github" || name == "node_modules" || name == "data" || name == "db" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(d.Name(), "-compose.yml") {
			relPath := strings.TrimPrefix(path, resolvedPath)
			displayPath := filepath.Join(currentDisplayPath, relPath)
			log.Printf("[DEBUG] Discovered compose file: %s\n", displayPath)
			files = append(files, displayPath)
		}
		return nil
	})

	return files, err
}
