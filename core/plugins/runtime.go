package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// ExecuteHook runs a plugin lifecycle hook command using the system shell (bash or sh).
// It injects useful context environment variables including the M3TAL API URL and API token.
func ExecuteHook(ctx context.Context, hookCmd string, p *Plugin) error {
	if hookCmd == "" {
		return nil
	}

	var shell, flag string
	if _, err := os.Stat("/bin/bash"); err == nil {
		shell = "/bin/bash"
		flag = "-c"
	} else {
		shell = "/bin/sh"
		flag = "-c"
	}

	// Retrieve API credentials and state DB path to expose to the plugin
	apiURL := os.Getenv("M3TAL_API_URL")
	if apiURL == "" {
		port := os.Getenv("API_PORT")
		if port == "" {
			port = "5050"
		}
		apiURL = "http://localhost:" + port
	}

	apiToken := os.Getenv("M3TAL_API_TOKEN")
	if apiToken == "" {
		apiToken = os.Getenv("API_TOKEN")
	}

	stateDB := os.Getenv("M3TAL_STATE_DB")
	if stateDB == "" {
		// Use a local fallback if store package is not loaded,
		// or standard state db path.
		stateDB = "/var/lib/m3tal/state.db"
		if _, err := os.Stat(stateDB); os.IsNotExist(err) {
			stateDB = "./data/state.db"
		}
	}

	cmd := exec.CommandContext(ctx, shell, flag, hookCmd)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("M3TAL_PLUGIN_NAME=%s", p.GetName()),
		fmt.Sprintf("M3TAL_PLUGIN_KIND=%s", p.Kind),
		fmt.Sprintf("M3TAL_PLUGIN_PATH=%s", p.SourcePath),
		fmt.Sprintf("M3TAL_API_URL=%s", apiURL),
		fmt.Sprintf("M3TAL_API_TOKEN=%s", apiToken),
		fmt.Sprintf("M3TAL_STATE_DB=%s", stateDB),
	)

	// Suppress execution timeout issues or capture output for diagnostic clarity
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Track start time
	start := time.Now()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("hook %q failed after %v: %w", hookCmd, time.Since(start), err)
	}

	return nil
}
