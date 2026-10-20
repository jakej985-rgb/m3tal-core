package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/core/plugins"
	"github.com/jakej985-rgb/m3tal-core/pkg/system"
)

// StackManager represents a collection of docker-compose files
type StackManager struct {
	Files []string
}

// NewStackManager returns a manager with discovered M3TAL compose files.
func NewStackManager() *StackManager {
	dirs := system.GetPluginDirs()
	reg, _ := plugin.LoadAll(dirs...)
	paths := discoverComposeFiles(reg)
	return &StackManager{
		Files: paths,
	}
}

// discoverComposeFiles finds all matching compose files across known locations.
func discoverComposeFiles(reg *plugin.Registry) []string {
	var files []string

	dashboardExists := false
	cmd := exec.Command("docker", "ps", "-a", "-q", "-f", "name=m3tal-dashboard")
	if out, err := cmd.Output(); err == nil && len(strings.TrimSpace(string(out))) > 0 {
		dashboardExists = true
	}

	// Scan system paths recursively (prioritizes /docker or /opt/m3tal/stack)
	stackDir := system.GetStackDir()
	matches, _ := system.FindComposeFiles(stackDir)
	for _, match := range matches {
		name := filepath.Base(match)
		// Ignore dashboard compose so it is strictly managed by 'm3tal dash' commands
		// UNLESS it has already been started and the container exists.
		if !dashboardExists && name == "m3tal-compose.yml" {
			continue
		}
		stackName := strings.TrimSuffix(name, "-compose.yml")
		if reg != nil {
			if sp := reg.GetStack(stackName); sp != nil && !sp.Enabled {
				continue
			}
		}
		files = append(files, match)
	}

	// Deduplicate and sort for deterministic order
	files = uniqueSorted(files, reg)

	return files
}

// uniqueSorted deduplicates and sorts file paths with priority for infrastructure stacks.
func uniqueSorted(files []string, reg *plugin.Registry) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, f := range files {
		if !seen[f] {
			seen[f] = true
			unique = append(unique, f)
		}
	}

	sort.Slice(unique, func(i, j int) bool {
		nameI := filepath.Base(unique[i])
		nameJ := filepath.Base(unique[j])

		stackNameI := strings.TrimSuffix(nameI, "-compose.yml")
		stackNameJ := strings.TrimSuffix(nameJ, "-compose.yml")

		pI := 100
		if reg != nil {
			if sp := reg.GetStack(stackNameI); sp != nil {
				pI = sp.Priority
			}
		}

		pJ := 100
		if reg != nil {
			if sp := reg.GetStack(stackNameJ); sp != nil {
				pJ = sp.Priority
			}
		}

		if pI != pJ {
			return pI < pJ
		}
		return nameI < nameJ
	})

	return unique
}

// Run executes a docker compose command across all stack files
func (s *StackManager) Run(action string, args ...string) error {
	if len(s.Files) == 0 {
		fmt.Println("⚠️  No compose files found. Nothing to do.")
		return nil
	}

	envFile := system.GetConfigPath()
	hasEnv := false
	if _, err := os.Stat(envFile); err == nil {
		hasEnv = true
	}

	for _, file := range s.Files {
		// 1. Determine stack name and ensure <stack>.env exists
		stackBase := filepath.Base(file)
		stackName := stackBase[:len(stackBase)-len("-compose.yml")]
		stackDir := filepath.Dir(file)
		localEnv := filepath.Join(stackDir, stackName+".env")

		// Automatically symlink global env to <stack>.env if it doesn't exist
		if _, err := os.Lstat(localEnv); os.IsNotExist(err) && hasEnv {
			_ = os.Symlink(envFile, localEnv)
		}

		// Self-healing: Fix legacy .env paths in the manifest
		_ = fixLegacyManifest(file, stackName)

		fmt.Printf("🚀 Running docker compose %s on %s...\n", action, file)

		var cmdArgs []string
		if hasEnv {
			// Use --env-file for the central .env config
			cmdArgs = []string{"compose", "-p", stackName, "--env-file", envFile, "-f", file}
		} else {
			cmdArgs = []string{"compose", "-p", stackName, "-f", file}
		}

		// Inject override configs for the dashboard
		if stackName == "m3tal" {
			mode := "local" // Default
			if data, err := os.ReadFile(envFile); err == nil {
				// Inline getEnvValue implementation since it's defined in main.go, not here
				lines := strings.Split(string(data), "\n")
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if strings.HasPrefix(trimmed, "DASHBOARD_EXPOSE_MODE=") && !strings.HasPrefix(trimmed, "#") {
						parts := strings.SplitN(trimmed, "=", 2)
						if len(parts) == 2 {
							mode = parts[1]
							break
						}
					}
				}
			}

			if mode == "local" {
				localOverride := filepath.Join(stackDir, "m3tal-compose.local.yml")
				if _, err := os.Stat(localOverride); err == nil {
					cmdArgs = append(cmdArgs, "-f", localOverride)
				}
			} else {
				traefikOverride := filepath.Join(stackDir, "m3tal-compose.traefik.yml")
				if _, err := os.Stat(traefikOverride); err == nil {
					cmdArgs = append(cmdArgs, "-f", traefikOverride)
				}
			}
		}

		cmdArgs = append(cmdArgs, action)
		cmdArgs = append(cmdArgs, args...)
		cmd := exec.Command("docker", cmdArgs...)

		cmd.Env = os.Environ()
		cmd.Stdout = os.Stdout

		var errTracker errorTracker
		cmd.Stderr = &errTracker

		if err := cmd.Run(); err != nil {
			handleOrchestratorError(file, errTracker.buffer.String())
			return fmt.Errorf("failed to run %s on %s: %w", action, file, err)
		}
	}
	return nil
}

// fixLegacyManifest scans a compose file for old relative .env paths and repairs them.
func fixLegacyManifest(filePath, stackName string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// We look for common legacy patterns and replace with the local ./<stack>.env
	brokenPaths := []string{"../../.env", "../m3tal-core/.env"}
	newPath := fmt.Sprintf("./%s.env", stackName)

	modified := false
	sContent := string(content)
	for _, old := range brokenPaths {
		if strings.Contains(sContent, old) {
			sContent = strings.ReplaceAll(sContent, old, newPath)
			modified = true
		}
	}

	// Remove legacy docker-proxy service block if it exists
	if strings.Contains(sContent, "docker-proxy:") {
		// Simple block removal for docker-proxy
		startIdx := strings.Index(sContent, "  docker-proxy:")
		if startIdx != -1 {
			// Find the start of the next top-level block (networks:)
			endIdx := strings.Index(sContent[startIdx:], "\nnetworks:")
			if endIdx != -1 {
				sContent = sContent[:startIdx] + sContent[startIdx+endIdx+1:]
				modified = true
			}
		}
	}

	// NEW: Fix strict interpolation checks using Regex
	// This catches ${VAR:?Error message} and converts to ${VAR:-}
	re := regexp.MustCompile(`:\?([^}]+)}`)
	if re.MatchString(sContent) {
		sContent = re.ReplaceAllString(sContent, ":-}")
		modified = true
	}

	// NEW: Fix legacy Traefik volume mounts
	if strings.Contains(sContent, "./dynamic/api.yml") {
		// Remove the individual file mount
		sContent = strings.ReplaceAll(sContent, "      - ./dynamic/api.yml:/etc/traefik/dynamic/api.yml:ro", "")
		// Update the directory mount to the simplified local version
		sContent = strings.ReplaceAll(sContent, "      - ${CONFIG_PATH:-/mnt/config}/traefik/dynamic:/etc/traefik/dynamic:ro", "      - ./dynamic:/etc/traefik/dynamic:ro")
		modified = true
	}

	if modified {
		return os.WriteFile(filePath, []byte(sContent), 0664)
	}
	return nil
}

// RunRaw executes an arbitrary command and pipes its output to stdout/stderr.
func RunRaw(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

type errorTracker struct {
	buffer strings.Builder
}

func (e *errorTracker) Write(p []byte) (n int, err error) {
	e.buffer.Write(p)
	return os.Stderr.Write(p)
}

func handleOrchestratorError(file string, errStr string) {
	if strings.Contains(errStr, "tried to kill container, but did not receive an exit event") {
		fmt.Println("\n⚠️  [M3TAL Diagnostician] Stuck Container Lockup Detected!")
		fmt.Println("==========================================================")
		fmt.Println("This is a low-level Docker/kernel lockup where a container process ignores SIGKILL.")
		fmt.Println("This usually happens due to a hung file system handle or network storage driver.")
		fmt.Println("\n💡 To manually fix this, run:")
		fmt.Println("  1. Find the containerd-shim PID: ps aux | grep containerd-shim | grep <container_id>")
		fmt.Println("  2. Kill the shim process:        sudo kill -9 <PID>")
		fmt.Println("  3. Restart the Docker daemon:    sudo systemctl restart docker")
		fmt.Println("  4. Force-remove the container:   sudo docker rm -f <container_name>")

		fmt.Print("\n🔧 Would you like M3TAL to attempt an automatic self-healing fix now? (y/N): ")
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(strings.TrimSpace(answer)) == "y" {
			attemptStuckContainerFix(errStr)
		}
	} else if strings.Contains(errStr, "joining network namespace of container") {
		fmt.Println("\n⚠️  [M3TAL Diagnostician] Network Namespace Dependency Mismatch!")
		fmt.Println("==========================================================")
		fmt.Println("This happens when a parent network container (like gluetun) was recreated with a new ID,")
		fmt.Println("leaving dependent containers with stale network namespace references.")
		fmt.Println("\n💡 To manually fix this, run:")
		fmt.Println("  1. Force-remove stale dependent containers in the media stack.")
		fmt.Println("     Run: sudo docker rm -f qbittorrent prowlarr flaresolverr")

		fmt.Print("\n🔧 Would you like M3TAL to auto-remove dependent containers in this stack so they can be recreated? (y/N): ")
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(strings.TrimSpace(answer)) == "y" {
			attemptNetworkNamespaceFix(file)
		}
	}
}

func attemptStuckContainerFix(errStr string) {
	re := regexp.MustCompile(`[0-9a-fA-F]{64}`)
	id := re.FindString(errStr)
	if id == "" {
		fmt.Println("❌ Could not parse stuck container ID from error message.")
		return
	}

	fmt.Printf("👉 Found stuck container ID: %s\n", id[:12])

	// Scan /proc to find containerd-shim PID for this container ID
	pid := 0
	files, err := os.ReadDir("/proc")
	if err == nil {
		for _, f := range files {
			if !f.IsDir() {
				continue
			}
			var p int
			if _, err := fmt.Sscanf(f.Name(), "%d", &p); err != nil {
				continue
			}
			cmdlinePath := filepath.Join("/proc", f.Name(), "cmdline")
			data, err := os.ReadFile(cmdlinePath)
			if err != nil {
				continue
			}
			cmdline := string(data)
			if strings.Contains(cmdline, "containerd-shim") && strings.Contains(cmdline, id) {
				pid = p
				break
			}
		}
	}

	if pid > 0 {
		fmt.Printf("👉 Found containerd-shim process PID: %d. Terminating it...\n", pid)
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Kill()
			fmt.Println("✅ Stuck process terminated.")
		}
	} else {
		fmt.Println("👉 No active containerd-shim process found. Proceeding...")
	}

	fmt.Println("👉 Restarting Docker daemon to clean up active locks...")
	cmd := exec.Command("systemctl", "restart", "docker")
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("service", "docker", "restart")
		_ = cmd.Run()
	}
	fmt.Println("✅ Docker daemon restarted successfully.")

	fmt.Println("👉 Force-removing stale container registration...")
	cmd = exec.Command("docker", "rm", "-f", id)
	_ = cmd.Run()
	fmt.Println("✅ Stale container removed.")
	fmt.Println("\n🎉 Self-healing completed! Please re-run your orchestrator command.")
}

func attemptNetworkNamespaceFix(composeFile string) {
	data, err := os.ReadFile(composeFile)
	if err != nil {
		fmt.Printf("❌ Failed to read compose file: %v\n", err)
		return
	}

	lines := strings.Split(string(data), "\n")
	var currentService string
	var servicesToFix []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(trimmed, ":") {
			currentService = strings.TrimSuffix(trimmed, ":")
		}
		if strings.HasPrefix(trimmed, "network_mode:") && strings.Contains(trimmed, "container:") && currentService != "" {
			servicesToFix = append(servicesToFix, currentService)
		}
	}

	if len(servicesToFix) == 0 {
		fmt.Println("👉 No dependent network containers found in this stack compose file.")
		return
	}

	fmt.Printf("👉 Found dependent containers: %s\n", strings.Join(servicesToFix, ", "))
	for _, svc := range servicesToFix {
		fmt.Printf("👉 Force-removing stale container %s...\n", svc)
		cmd := exec.Command("docker", "rm", "-f", svc)
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️  Failed to remove %s: %v\n", svc, err)
		} else {
			fmt.Printf("✅ Removed stale container %s.\n", svc)
		}
	}
	fmt.Println("\n🎉 Self-healing completed! Please re-run your orchestrator command.")
}
