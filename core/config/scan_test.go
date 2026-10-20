package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanStacks(t *testing.T) {
	// Create temporary directory to simulate /usr/share/m3tal/docker
	tmpDir, err := os.MkdirTemp("", "m3tal-docker-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	// Clean up after test
	defer os.RemoveAll(tmpDir)

	// Setup stack files
	files := []struct {
		name    string
		content string
	}{
		{"media-compose.yml", "version: 3"},
		{"media.env.template", "MEDIA_VAR=foo"},
		{"m3tal-compose.yml", "version: 3"},
		{"m3tal.env.template", "M3TAL_VAR=bar"},
		// This stack should be skipped because template missing
		{"spotify-compose.yml", "version: 3"},
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f.name), []byte(f.content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", f.name, err)
		}
	}

	result, err := ScanStacks(tmpDir)
	if err != nil {
		t.Fatalf("ScanStacks returned error: %v", err)
	}

	expected := map[string]StackInfo{
		"media": {Compose: filepath.Join(tmpDir, "media-compose.yml"), Template: filepath.Join(tmpDir, "media.env.template")},
		"m3tal": {Compose: filepath.Join(tmpDir, "m3tal-compose.yml"), Template: filepath.Join(tmpDir, "m3tal.env.template")},
	}

	if len(result) != len(expected) {
		t.Fatalf("Expected %d stacks, got %d", len(expected), len(result))
	}
	for key, exp := range expected {
		got, ok := result[key]
		if !ok {
			t.Fatalf("Expected stack %s missing", key)
		}
		if got.Compose != exp.Compose || got.Template != exp.Template {
			t.Fatalf("Stack %s mismatch. Expected %+v, got %+v", key, exp, got)
		}
	}
}
