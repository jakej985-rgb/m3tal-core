package registry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchIndex(t *testing.T) {
	mockIdx := Index{
		Stacks: []StackMetadata{
			{
				Name:        "sonarr",
				Category:    "media",
				Description: "Sonarr TV automation",
				Version:     "1.0.0",
				Requires:    []string{"docker"},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockIdx)
	}))
	defer server.Close()

	idx, err := FetchIndex(server.URL)
	if err != nil {
		t.Fatalf("failed to fetch index: %v", err)
	}

	if len(idx.Stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(idx.Stacks))
	}

	if idx.Stacks[0].Name != "sonarr" {
		t.Errorf("expected stack name 'sonarr', got %q", idx.Stacks[0].Name)
	}
}

func TestValidateRequirements(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "m3tal-registry-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	meta := &StackMetadata{
		Name:     "sonarr",
		Requires: []string{"docker", "traefik"},
	}

	warnings := ValidateRequirements(meta, tempDir)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}

	traefikFile := filepath.Join(tempDir, "traefik-compose.yml")
	if err := os.WriteFile(traefikFile, []byte("version: '3'"), 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}

	warnings = ValidateRequirements(meta, tempDir)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d: %v", len(warnings), warnings)
	}
}
