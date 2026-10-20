package preflight

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jakej985-rgb/m3tal-core/pkg/system"
)

// CheckResult stores the outcome of a single pre-flight check.
type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "PASS", "WARN", "FAIL"
	Message string `json:"message"`
}

// RunAll executes the full suite of pre-flight checks and returns the results.
func RunAll(envPath string, baseStoragePath string) []CheckResult {
	var results []CheckResult

	results = append(results, checkDockerDaemon())
	results = append(results, checkDotEnv(envPath))
	results = append(results, checkStoragePath(baseStoragePath))
	results = append(results, checkPortAvailable(80, "Traefik HTTP"))
	results = append(results, checkPortAvailable(443, "Traefik HTTPS"))
	results = append(results, checkPortAvailable(8080, "Traefik Dashboard"))
	results = append(results, checkDockerGroup())
	results = append(results, checkDNSHosts())

	return results
}

func checkDockerDaemon() CheckResult {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "ps")
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return CheckResult{
				Name:    "Docker Daemon",
				Status:  "FAIL",
				Message: "Docker daemon did not respond within 3 seconds. Is Docker installed and running?",
			}
		}
		return CheckResult{
			Name:    "Docker Daemon",
			Status:  "FAIL",
			Message: fmt.Sprintf("Cannot reach Docker daemon: %v. Ensure 'docker ps' runs without sudo.", err),
		}
	}
	return CheckResult{
		Name:    "Docker Daemon",
		Status:  "PASS",
		Message: "Docker daemon is reachable and responding.",
	}
}

func checkDotEnv(envPath string) CheckResult {
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return CheckResult{
			Name:    ".env File",
			Status:  "FAIL",
			Message: fmt.Sprintf("No .env file found at %s. Run 'cp template.env .env' then edit it, or run './m3tal init'.", envPath),
		}
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		return CheckResult{
			Name:    ".env File",
			Status:  "FAIL",
			Message: fmt.Sprintf("Cannot read .env file at %s: %v", envPath, err),
		}
	}

	content := string(data)
	requiredVars := []string{"BASE_STORAGE_PATH", "DASHBOARD_SECRET", "API_TOKEN"}
	var missing []string
	for _, key := range requiredVars {
		if !containsKeyValue(content, key) {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return CheckResult{
			Name:    ".env File",
			Status:  "FAIL",
			Message: fmt.Sprintf("Missing required variables in .env: %s. Copy template.env to .env and fill in the values.", strings.Join(missing, ", ")),
		}
	}

	// Warn about default values
	var defaultsSet []string
	defaultChecks := map[string]string{
		"BASE_STORAGE_PATH": system.DataPath,
		"DASHBOARD_SECRET":  "change_me_immediately",
		"API_TOKEN":         "change_me_api_token",
		"ADMIN_PASSWORD":    "admin_pass",
	}
	for key, defaultValue := range defaultChecks {
		val := getValue(content, key)
		if val == defaultValue {
			defaultsSet = append(defaultsSet, key)
		}
	}

	if len(defaultsSet) > 0 {
		return CheckResult{
			Name:    ".env File",
			Status:  "WARN",
			Message: fmt.Sprintf("Found default/placeholder values for: %s. These should be changed for production use.", strings.Join(defaultsSet, ", ")),
		}
	}

	return CheckResult{
		Name:    ".env File",
		Status:  "PASS",
		Message: ".env file exists and contains all required variables.",
	}
}

func checkStoragePath(basePath string) CheckResult {
	if basePath == "" {
		return CheckResult{
			Name:    "Storage Path",
			Status:  "FAIL",
			Message: "BASE_STORAGE_PATH is empty. Set it in your .env file to an absolute or relative path.",
		}
	}

	// Resolve relative paths
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return CheckResult{
			Name:    "Storage Path",
			Status:  "FAIL",
			Message: fmt.Sprintf("Cannot resolve BASE_STORAGE_PATH '%s': %v", basePath, err),
		}
	}

	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return CheckResult{
			Name:    "Storage Path",
			Status:  "FAIL",
			Message: fmt.Sprintf("Directory does not exist: %s. Create it with: mkdir -p %s", absPath, absPath),
		}
	}
	if err != nil {
		return CheckResult{
			Name:    "Storage Path",
			Status:  "FAIL",
			Message: fmt.Sprintf("Cannot access storage path %s: %v", absPath, err),
		}
	}

	if !info.IsDir() {
		return CheckResult{
			Name:    "Storage Path",
			Status:  "FAIL",
			Message: fmt.Sprintf("Path exists but is not a directory: %s", absPath),
		}
	}

	// Test writability
	testFile := filepath.Join(absPath, ".m3tal-write-test")
	f, err := os.Create(testFile)
	if err != nil {
		return CheckResult{
			Name:    "Storage Path",
			Status:  "FAIL",
			Message: fmt.Sprintf("Storage path %s is NOT writable by the current user: %v", absPath, err),
		}
	}
	f.Close()
	os.Remove(testFile)

	return CheckResult{
		Name:    "Storage Path",
		Status:  "PASS",
		Message: fmt.Sprintf("Storage path exists and is writable: %s", absPath),
	}
}

func checkPortAvailable(port int, service string) CheckResult {
	host := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return CheckResult{
			Name:    fmt.Sprintf("Port %d (%s)", port, service),
			Status:  "PASS",
			Message: fmt.Sprintf("Port %d is available.", port),
		}
	}
	conn.Close()
	return CheckResult{
		Name:    fmt.Sprintf("Port %d (%s)", port, service),
		Status:  "FAIL",
		Message: fmt.Sprintf("Port %d is already in use by another process. Check with: ss -tlnp | grep ':%d '", port, port),
	}
}

func checkDockerGroup() CheckResult {
	// Root always has access
	if os.Geteuid() == 0 {
		return CheckResult{
			Name:    "Docker Group",
			Status:  "PASS",
			Message: "Running as root — Docker access is available.",
		}
	}

	groupsOutput, err := exec.Command("groups").Output()
	if err != nil {
		return CheckResult{
			Name:    "Docker Group",
			Status:  "WARN",
			Message: fmt.Sprintf("Cannot determine group membership: %v", err),
		}
	}

	if strings.Contains(string(groupsOutput), "docker") {
		return CheckResult{
			Name:    "Docker Group",
			Status:  "PASS",
			Message: "User is in the docker group.",
		}
	}

	return CheckResult{
		Name:    "Docker Group",
		Status:  "WARN",
		Message: "User is NOT in the docker group. Run: sudo usermod -aG docker $USER, then log out and back in.",
	}
}

func checkDNSHosts() CheckResult {
	data, err := os.ReadFile("/etc/hosts")
	if err != nil {
		return CheckResult{
			Name:    "DNS /etc/hosts",
			Status:  "WARN",
			Message: fmt.Sprintf("Cannot read /etc/hosts: %v", err),
		}
	}

	content := string(data)
	expected := []string{"m3tal.localhost", "api.localhost", "traefik.localhost"}
	var missing []string
	for _, host := range expected {
		if !strings.Contains(content, host) {
			missing = append(missing, host)
		}
	}

	if len(missing) > 0 {
		return CheckResult{
			Name:    "DNS /etc/hosts",
			Status:  "WARN",
			Message: fmt.Sprintf("Missing host entries in /etc/hosts: %s. Add: '127.0.0.1 m3tal.localhost api.localhost traefik.localhost'", strings.Join(missing, ", ")),
		}
	}

	return CheckResult{
		Name:    "DNS /etc/hosts",
		Status:  "PASS",
		Message: "All required host entries found in /etc/hosts.",
	}
}

// ValidateStoragePath is a lightweight version of checkStoragePath intended
// for use during `init` — returns error string or empty string on success.
func ValidateStoragePath(basePath string) error {
	if basePath == "" {
		return fmt.Errorf("BASE_STORAGE_PATH is empty. Set it in your .env file")
	}

	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("cannot resolve path '%s': %w", basePath, err)
	}

	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s\nCreate it with: mkdir -p %s", absPath, absPath)
	}
	if err != nil {
		return fmt.Errorf("cannot access '%s': %w", absPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", absPath)
	}

	// Test writability
	testFile := filepath.Join(absPath, ".m3tal-write-test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("path '%s' is NOT writable: %w", absPath, err)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}

// --- helpers ---

func containsKeyValue(content, key string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") && !strings.HasPrefix(trimmed, "#") {
			return true
		}
	}
	return false
}

func getValue(content, key string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") && !strings.HasPrefix(trimmed, "#") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}
	return ""
}

// PrintResults outputs the check results in a human-readable format.
func PrintResults(results []CheckResult) {
	allPassed := true
	hasWarnings := false

	fmt.Println("\n🔍 M3TAL Pre-Flight Check")
	fmt.Println("==========================")
	for _, r := range results {
		var icon string
		switch r.Status {
		case "PASS":
			icon = "✅"
		case "WARN":
			icon = "⚠️"
			hasWarnings = true
		case "FAIL":
			icon = "❌"
			allPassed = false
		default:
			icon = "❓"
		}
		fmt.Printf(" %s %s\n", icon, r.Name)
		fmt.Printf("    %s\n", r.Message)
	}

	fmt.Println("==========================")
	if allPassed && !hasWarnings {
		fmt.Println("✅ All checks passed! System is ready.")
	} else if allPassed && hasWarnings {
		fmt.Println("✅ All checks passed (with warnings). Review the warnings above.")
	} else {
		fmt.Println("❌ Some checks FAILED. Review the errors above before proceeding.")
	}
	fmt.Println()
}
