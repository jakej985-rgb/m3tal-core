package vpn

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
)

// SyncForwardedPort fetches the active forwarded port from Gluetun and syncs it to dependent services.
func (m *Manager) SyncForwardedPort() (int, error) {
	port, err := m.GetForwardedPort()
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve forwarded port from Gluetun: %w", err)
	}

	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid forwarded port fetched: %d", port)
	}

	// Sync to qBittorrent
	qbErr := m.syncToQBittorrent(port)
	if qbErr != nil {
		// Log error but proceed as this is a non-blocking warning/fallback trigger
		fmt.Fprintf(os.Stderr, "⚠️  Failed to sync port to qBittorrent: %v\n", qbErr)
	}

	return port, nil
}

func (m *Manager) syncToQBittorrent(port int) error {
	// Try WebAPI first (running instance)
	apiErr := m.syncQBittorrentAPI(port)
	if apiErr == nil {
		return nil
	}

	// Fallback: direct configuration file modification
	confPath := "/mnt/config/qbittorrent/qBittorrent/qBittorrent.conf"
	// Also check local path just in case we are in development/sandbox
	if _, err := os.Stat("deploy/config/qbittorrent/qBittorrent/qBittorrent.conf"); err == nil {
		confPath = "deploy/config/qbittorrent/qBittorrent/qBittorrent.conf"
	}

	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		return fmt.Errorf("WebAPI failed (%v) and config file not found at %s", apiErr, confPath)
	}

	data, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("WebAPI failed (%v) and config file unreadable: %w", apiErr, err)
	}

	lines := strings.Split(string(data), "\n")
	portUpdated := false
	for i, line := range lines {
		if strings.HasPrefix(line, "Session\\Port=") {
			lines[i] = fmt.Sprintf("Session\\Port=%d", port)
			portUpdated = true
			break
		}
	}

	if !portUpdated {
		// Append to the Preferences section if we find it
		preferencesIndex := -1
		for i, line := range lines {
			if strings.TrimSpace(line) == "[Preferences]" {
				preferencesIndex = i
				break
			}
		}
		if preferencesIndex != -1 {
			// Insert after [Preferences] header
			lines = append(lines[:preferencesIndex+1], append([]string{fmt.Sprintf("Session\\Port=%d", port)}, lines[preferencesIndex+1:]...)...)
		} else {
			lines = append(lines, "[Preferences]", fmt.Sprintf("Session\\Port=%d", port))
		}
	}

	err = os.WriteFile(confPath, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		return fmt.Errorf("WebAPI failed (%v) and config file write failed: %w", apiErr, err)
	}

	// Restart qbittorrent container to apply config file changes
	_ = m.RestartContainerIfRunning("qbittorrent")

	return nil
}

func (m *Manager) syncQBittorrentAPI(port int) error {
	// Standard qBittorrent port is 8090, but check environment just in case
	portWebUI := "8090"
	apiURL := fmt.Sprintf("http://localhost:%s/api/v2", portWebUI)

	clientHTTP := &http.Client{Timeout: 3 * time.Second}

	// 1. Login
	loginURL := apiURL + "/auth/login"
	formData := url.Values{
		"username": {"admin"},
		"password": {"adminadmin"}, // standard default password
	}
	resp, err := clientHTTP.PostForm(loginURL, formData)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %s", resp.Status)
	}

	// Extract cookie
	var cookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "SID" {
			cookie = c
			break
		}
	}
	if cookie == nil {
		return fmt.Errorf("no SID cookie returned from login")
	}

	// 2. Set Preferences
	prefURL := apiURL + "/app/setPreferences"
	prefJSON := fmt.Sprintf(`{"listen_port": %d}`, port)
	req, err := http.NewRequest("POST", prefURL, bytes.NewBufferString(url.Values{"json": {prefJSON}}.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	respPref, err := clientHTTP.Do(req)
	if err != nil {
		return err
	}
	defer respPref.Body.Close()

	if respPref.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set preferences: status %s", respPref.Status)
	}

	return nil
}

// RestartContainerIfRunning restarts a container if it's currently running.
func (m *Manager) RestartContainerIfRunning(name string) error {
	ctx := context.Background()
	return m.cli.ContainerRestart(ctx, name, container.StopOptions{})
}
