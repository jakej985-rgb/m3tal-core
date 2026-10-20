package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// parseAnyPlugin parses a plugin manifest file as YAML or JSON based on file extension.
func parseAnyPlugin(data []byte, path string) (*Plugin, error) {
	var p *Plugin
	var err error
	baseWithoutDisabled := strings.TrimSuffix(path, ".disabled")
	ext := strings.ToLower(filepath.Ext(baseWithoutDisabled))

	if ext == ".json" {
		p, err = ParsePluginJSON(data)
	} else {
		p, err = ParsePlugin(data)
	}
	if err != nil {
		return nil, err
	}
	p.SourcePath = path
	return p, nil
}

// ScanDir recursively scans a directory for plugin files (*.yml, *.yaml, *.json).
// Returns a slice of parsed Plugins. Invalid files are logged but skipped.
func ScanDir(dir string) ([]Plugin, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Directory doesn't exist yet — not an error
		}
		return nil, fmt.Errorf("cannot stat plugin directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	var plugins []Plugin
	var scanErrors []string

	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		base := filepath.Base(path)
		baseLower := strings.ToLower(base)
		if strings.HasSuffix(baseLower, "-compose.yml") || strings.HasSuffix(baseLower, "-compose.yaml") ||
			strings.HasSuffix(baseLower, "-compose.yml.disabled") || strings.HasSuffix(baseLower, "-compose.yaml.disabled") {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		enabled := true
		if ext == ".disabled" {
			enabled = false
			baseWithoutDisabled := strings.TrimSuffix(path, ".disabled")
			ext = strings.ToLower(filepath.Ext(baseWithoutDisabled))
		}
		if ext != ".yml" && ext != ".yaml" && ext != ".json" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			scanErrors = append(scanErrors, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		p, err := parseAnyPlugin(data, path)
		if err != nil {
			scanErrors = append(scanErrors, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		p.Enabled = enabled
		if err := p.Validate(); err != nil {
			scanErrors = append(scanErrors, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		plugins = append(plugins, *p)
		return nil
	})

	if err != nil {
		return plugins, fmt.Errorf("error walking plugin directory %s: %w", dir, err)
	}

	// Log scan errors to stderr but don't fail
	for _, e := range scanErrors {
		fmt.Fprintf(os.Stderr, "⚠️  Plugin scan warning: %s\n", e)
	}

	return plugins, nil
}

// LoadAll scans the given directories in order, loading and deduplicating plugins.
// Directories listed later take precedence (user overrides system).
// The standard order is: system dir first, user dir second.
func LoadAll(dirs ...string) (*Registry, error) {
	seen := make(map[string]*Plugin) // key: "kind/name"
	var allPlugins []Plugin

	for _, dir := range dirs {
		plugins, err := ScanDir(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to scan %s: %w", dir, err)
		}

		for i := range plugins {
			p := &plugins[i]
			key := p.Kind + "/" + p.GetName()
			seen[key] = p
		}
	}

	// Build deduplicated list from the seen map
	for _, p := range seen {
		allPlugins = append(allPlugins, *p)
	}

	return BuildRegistry(allPlugins)
}

// LoadFromPaths is a convenience that loads plugins from the system default paths.
// It uses GetPluginDirs from the system package via the provided dirs.
func LoadFromPaths(systemDir, userDir string) (*Registry, error) {
	return LoadAll(systemDir, userDir)
}

// EnablePlugin renames a plugin file ending in `.disabled` by removing the suffix.
func EnablePlugin(path string) (string, error) {
	if !strings.HasSuffix(path, ".disabled") {
		return path, fmt.Errorf("plugin at %s is already enabled (does not have .disabled suffix)", path)
	}

	// Run PreEnable hook if defined
	if data, err := os.ReadFile(path); err == nil {
		if p, parseErr := parseAnyPlugin(data, path); parseErr == nil {
			if p.Hooks != nil && p.Hooks.PreEnable != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_ = ExecuteHook(ctx, p.Hooks.PreEnable, p)
				cancel()
			}
		}
	}

	newPath := strings.TrimSuffix(path, ".disabled")
	if err := os.Rename(path, newPath); err != nil {
		return path, err
	}

	// Run PostEnable hook if defined
	if data, err := os.ReadFile(newPath); err == nil {
		if p, parseErr := parseAnyPlugin(data, newPath); parseErr == nil {
			if p.Hooks != nil && p.Hooks.PostEnable != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_ = ExecuteHook(ctx, p.Hooks.PostEnable, p)
				cancel()
			}
		}
	}

	return newPath, nil
}

// DisablePlugin renames a plugin file to append `.disabled`.
func DisablePlugin(path string) (string, error) {
	if strings.HasSuffix(path, ".disabled") {
		return path, fmt.Errorf("plugin at %s is already disabled", path)
	}

	// Run PreDisable hook if defined
	if data, err := os.ReadFile(path); err == nil {
		if p, parseErr := parseAnyPlugin(data, path); parseErr == nil {
			if p.Hooks != nil && p.Hooks.PreDisable != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_ = ExecuteHook(ctx, p.Hooks.PreDisable, p)
				cancel()
			}
		}
	}

	newPath := path + ".disabled"
	if err := os.Rename(path, newPath); err != nil {
		return path, err
	}

	// Run PostDisable hook if defined
	if data, err := os.ReadFile(newPath); err == nil {
		if p, parseErr := parseAnyPlugin(data, newPath); parseErr == nil {
			if p.Hooks != nil && p.Hooks.PostDisable != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_ = ExecuteHook(ctx, p.Hooks.PostDisable, p)
				cancel()
			}
		}
	}

	return newPath, nil
}

// LoadPlugin reads and parses a single plugin manifest from the filesystem.
func LoadPlugin(path string) (*Plugin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	enabled := true
	if ext == ".disabled" {
		enabled = false
	}

	p, err := parseAnyPlugin(data, path)
	if err != nil {
		return nil, err
	}
	p.Enabled = enabled
	return p, nil
}
