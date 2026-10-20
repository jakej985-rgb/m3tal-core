package compose

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
)

// Parse parses docker compose YAML data into ComposeConfig.
func Parse(yamlData []byte) (*ComposeConfig, error) {
	var cfg ComposeConfig
	if err := yaml.Unmarshal(yamlData, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML syntax: %w", err)
	}

	// Validate basic structural requirements
	if len(cfg.Services) == 0 {
		return nil, errors.New("missing or empty 'services' section")
	}

	return &cfg, nil
}
