package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StackMetadata represents the metadata for a prebuilt stack.
type StackMetadata struct {
	Name        string         `json:"name"`
	Category    string         `json:"category"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	Requires    []string       `json:"requires"`
	Optional    []string       `json:"optional"`
	Ports       []int          `json:"ports"`
	Resources   StackResources `json:"resources"`
	DownloadURL string         `json:"download_url"`
	ManifestURL string         `json:"manifest_url"`
}

// StackResources represents the resource constraints for the stack.
type StackResources struct {
	Memory string `json:"memory"`
	CPU    any    `json:"cpu"` // float or string
}

// Index represents the registry index.json.
type Index struct {
	Stacks []StackMetadata `json:"stacks"`
}

// GetRegistryURL returns the default registry endpoint, supporting env override.
func GetRegistryURL() string {
	url := os.Getenv("M3TAL_STACKS_REGISTRY")
	if url == "" {
		url = "https://jakej985-rgb.github.io/m3tal-stacks/index.json"
	}
	return url
}

// FetchIndex retrieves the stacks index.json from the registry.
func FetchIndex(registryURL string) (*Index, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(registryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry response: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(body, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse registry index: %w", err)
	}

	return &idx, nil
}

// DownloadStack downloads the stack's compose and metadata to the destination directory.
func DownloadStack(name string, registryURL string, destDir string) error {
	idx, err := FetchIndex(registryURL)
	if err != nil {
		return err
	}

	var target *StackMetadata
	for i := range idx.Stacks {
		if strings.EqualFold(idx.Stacks[i].Name, name) {
			target = &idx.Stacks[i]
			break
		}
	}

	if target == nil {
		return fmt.Errorf("stack %q not found in registry", name)
	}

	downloadURL := target.DownloadURL
	if downloadURL == "" {
		return fmt.Errorf("download URL for stack %q is empty", name)
	}

	// If using local registry, redirect download URLs to the local host server
	if strings.Contains(registryURL, "localhost:") || strings.Contains(registryURL, "127.0.0.1:") {
		prodBase := "https://raw.githubusercontent.com/jakej985-rgb/m3tal-stacks/main/"
		localBase := registryURL[:strings.LastIndex(registryURL, "/")+1]
		if strings.HasPrefix(downloadURL, prodBase) {
			downloadURL = localBase + strings.TrimPrefix(downloadURL, prodBase)
		}
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// 1. Download docker-compose.yml
	composePath := filepath.Join(destDir, fmt.Sprintf("%s-compose.yml", strings.ToLower(target.Name)))
	if err := downloadFile(downloadURL, composePath); err != nil {
		return fmt.Errorf("failed to download compose file: %w", err)
	}

	// 2. Save stack.yml info as JSON metadata
	metaPath := filepath.Join(destDir, fmt.Sprintf("%s.stack.json", strings.ToLower(target.Name)))
	metaBytes, err := json.MarshalIndent(target, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize stack metadata: %w", err)
	}
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		return fmt.Errorf("failed to write stack metadata: %w", err)
	}

	return nil
}

// ValidateRequirements checks if system/stack pre-requisites are satisfied.
func ValidateRequirements(meta *StackMetadata, destDir string) []string {
	var warnings []string
	for _, req := range meta.Requires {
		reqLower := strings.ToLower(req)
		if reqLower == "docker" {
			continue
		}

		// Check if required stack's compose file exists in destDir
		composeFile := filepath.Join(destDir, fmt.Sprintf("%s-compose.yml", reqLower))
		if _, err := os.Stat(composeFile); os.IsNotExist(err) {
			warnings = append(warnings, fmt.Sprintf("Requires stack %q to be installed first", req))
		}
	}
	return warnings
}

func downloadFile(url, destPath string) error {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
