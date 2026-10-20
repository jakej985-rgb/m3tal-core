package orchestrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jakej985-rgb/m3tal-core/core/plugins"
)

func TestUniqueSorted(t *testing.T) {
	// Create mock registry
	plugins := []plugin.Plugin{
		{
			APIVersion: plugin.APIVersion,
			Kind:       plugin.KindStack,
			Metadata:   plugin.PluginMetadata{Name: "priority-five"},
			Spec:       map[string]any{"composePath": "priority-five-compose.yml", "priority": 5},
			Enabled:    true,
		},
		{
			APIVersion: plugin.APIVersion,
			Kind:       plugin.KindStack,
			Metadata:   plugin.PluginMetadata{Name: "priority-one"},
			Spec:       map[string]any{"composePath": "priority-one-compose.yml", "priority": 1},
			Enabled:    true,
		},
		{
			APIVersion: plugin.APIVersion,
			Kind:       plugin.KindStack,
			Metadata:   plugin.PluginMetadata{Name: "priority-ten-a"},
			Spec:       map[string]any{"composePath": "priority-ten-a-compose.yml", "priority": 10},
			Enabled:    true,
		},
		{
			APIVersion: plugin.APIVersion,
			Kind:       plugin.KindStack,
			Metadata:   plugin.PluginMetadata{Name: "priority-ten-b"},
			Spec:       map[string]any{"composePath": "priority-ten-b-compose.yml", "priority": 10},
			Enabled:    true,
		},
	}

	reg, err := plugin.BuildRegistry(plugins)
	if err != nil {
		t.Fatalf("BuildRegistry failed: %v", err)
	}

	files := []string{
		"/path/to/custom-compose.yml", // custom stack (no plugin, should get priority 100)
		"/path/to/priority-five-compose.yml",
		"/path/to/priority-ten-b-compose.yml",
		"/path/to/priority-one-compose.yml",
		"/path/to/priority-ten-a-compose.yml",
		"/path/to/priority-five-compose.yml", // duplicate
	}

	sorted := uniqueSorted(files, reg)

	// Check duplicates removed
	if len(sorted) != 5 {
		t.Fatalf("expected 5 files, got %d", len(sorted))
	}

	// Expected order:
	// 1. priority-one (priority 1)
	// 2. priority-five (priority 5)
	// 3. priority-ten-a (priority 10, alphabetical before b)
	// 4. priority-ten-b (priority 10, alphabetical after a)
	// 5. custom (priority 100)
	expected := []string{
		"priority-one-compose.yml",
		"priority-five-compose.yml",
		"priority-ten-a-compose.yml",
		"priority-ten-b-compose.yml",
		"custom-compose.yml",
	}

	for i, exp := range expected {
		base := filepath.Base(sorted[i])
		if base != exp {
			t.Errorf("expected sorted[%d] to be %q, got %q", i, exp, base)
		}
	}
}

func TestDiscoverComposeFiles(t *testing.T) {
	// Setup temporary stack directory
	tempDir, err := os.MkdirTemp("", "m3tal-test-stacks")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set environment override
	t.Setenv("M3TAL_STACK_DIR", tempDir)

	// Create dummy compose files in temp directory
	filesToCreate := []string{
		"active-one-compose.yml",
		"disabled-stack-compose.yml",
		"custom-stack-compose.yml",
	}

	for _, f := range filesToCreate {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte("version: '3'"), 0644); err != nil {
			t.Fatalf("failed to write dummy file %s: %v", f, err)
		}
	}

	// Construct registry with one active stack plugin, one disabled stack plugin
	plugins := []plugin.Plugin{
		{
			APIVersion: plugin.APIVersion,
			Kind:       plugin.KindStack,
			Metadata:   plugin.PluginMetadata{Name: "active-one"},
			Spec:       map[string]any{"composePath": "active-one-compose.yml", "priority": 10},
			Enabled:    true,
		},
		{
			APIVersion: plugin.APIVersion,
			Kind:       plugin.KindStack,
			Metadata:   plugin.PluginMetadata{Name: "disabled-stack"},
			Spec:       map[string]any{"composePath": "disabled-stack-compose.yml", "priority": 2},
			Enabled:    false,
		},
	}

	reg, err := plugin.BuildRegistry(plugins)
	if err != nil {
		t.Fatalf("failed to build registry: %v", err)
	}

	discovered := discoverComposeFiles(reg)

	// We expect active-one and custom-stack. disabled-stack should be filtered out.
	// Order should be: active-one (priority 10), custom-stack (priority 100)
	if len(discovered) != 2 {
		t.Fatalf("expected 2 discovered files, got %d: %v", len(discovered), discovered)
	}

	expected := []string{
		"active-one-compose.yml",
		"custom-stack-compose.yml",
	}

	for i, exp := range expected {
		base := filepath.Base(discovered[i])
		if base != exp {
			t.Errorf("expected discovered[%d] to be %q, got %q", i, exp, base)
		}
	}
}
