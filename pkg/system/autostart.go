package system

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// EnsureAPIRunning verifies that the API daemon is running at baseURL.
// If it is offline, it spawns a detached background instance of the API server daemon.
// It then polls the health endpoint (/api/v2/system/health) until healthy or maxWait is reached.
func EnsureAPIRunning(baseURL, token string, maxWait time.Duration) error {
	client := &http.Client{
		Timeout: 500 * time.Millisecond,
	}

	healthURL := fmt.Sprintf("%s/api/v2/system/health", baseURL)

	// Check if already running
	req, err := http.NewRequest("GET", healthURL, nil)
	if err == nil {
		if token != "" {
			req.Header.Set("X-API-Token", token)
		}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusUnauthorized {
				return nil
			}
		}
	}

	// Try spawning background API daemon
	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}

	cmdDaemon := exec.Command(exe, "api")
	cmdDaemon.Env = os.Environ()
	if token != "" {
		cmdDaemon.Env = append(cmdDaemon.Env, "API_TOKEN="+token)
	}
	cmdDaemon.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	logPath := filepath.Join(os.TempDir(), "m3tal-api.log")
	if lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644); err == nil {
		cmdDaemon.Stdout = lf
		cmdDaemon.Stderr = lf
	}

	if err := cmdDaemon.Start(); err != nil {
		return fmt.Errorf("failed to auto-start API server daemon: %w", err)
	}
	_ = cmdDaemon.Process.Release()

	// Poll until ready
	start := time.Now()
	for time.Since(start) < maxWait {
		time.Sleep(200 * time.Millisecond)
		req, err = http.NewRequest("GET", healthURL, nil)
		if err != nil {
			continue
		}
		if token != "" {
			req.Header.Set("X-API-Token", token)
		}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusUnauthorized {
				return nil
			}
		}
	}

	return fmt.Errorf("API daemon failed to start within %v", maxWait)
}
