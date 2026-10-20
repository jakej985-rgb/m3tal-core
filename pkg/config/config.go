package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Config represents M3TAL client/CLI configuration.
type Config struct {
	APIURL   string `json:"api_url"`
	APIToken string `json:"api_token"`
}

var (
	globalConfig Config
)

func init() {
	// Initialize default values
	globalConfig = Config{
		APIURL:   "http://localhost:5050",
		APIToken: os.Getenv("API_TOKEN"),
	}
	// Try to load ~/.m3tal/config or fallbacks
	_ = Load()
}

// Load loads the configuration from ~/.m3tal/config.
func Load() error {
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".m3tal", "config")
		if data, err := os.ReadFile(configPath); err == nil {
			var loaded Config
			if err := json.Unmarshal(data, &loaded); err == nil {
				if loaded.APIURL != "" {
					globalConfig.APIURL = loaded.APIURL
				}
				if loaded.APIToken != "" {
					globalConfig.APIToken = loaded.APIToken
				}
			}
		}
	}

	// Fallback to /etc/m3tal/.env for system-wide configuration
	if globalConfig.APIToken == "" {
		if data, err := os.ReadFile("/etc/m3tal/.env"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "#") || line == "" {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					val = strings.Trim(val, `"'`)
					if key == "API_TOKEN" && val != "" {
						globalConfig.APIToken = val
						break
					}
				}
			}
		}
	}

	return nil
}

// GetAPIURL returns the configured API base URL.
func GetAPIURL() string {
	return globalConfig.APIURL
}

// GetAPIToken returns the configured API authorization token.
func GetAPIToken() string {
	if globalConfig.APIToken != "" {
		return globalConfig.APIToken
	}
	return os.Getenv("API_TOKEN")
}
