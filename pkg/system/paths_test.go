package system

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestFindComposeFiles_Symlinks(t *testing.T) {
	// 1. Setup temporary test directory structure
	tmpDir, err := os.MkdirTemp("", "m3tal-test-*")
	if err != nil {
		t.Fatalf("failed to create temp root: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	root := filepath.Join(tmpDir, "root")
	err = os.Mkdir(root, 0755)
	if err != nil {
		t.Fatalf("failed to create root: %v", err)
	}

	// Create a standard nested compose file
	nestedDir := filepath.Join(root, "nested")
	err = os.Mkdir(nestedDir, 0755)
	if err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	nestedCompose := filepath.Join(nestedDir, "nested-compose.yml")
	err = os.WriteFile(nestedCompose, []byte("version: '3'"), 0644)
	if err != nil {
		t.Fatalf("failed to write nested compose: %v", err)
	}

	// Create an external directory with a compose file
	externalDir := filepath.Join(tmpDir, "external")
	err = os.Mkdir(externalDir, 0755)
	if err != nil {
		t.Fatalf("failed to create external dir: %v", err)
	}
	externalCompose := filepath.Join(externalDir, "external-compose.yml")
	err = os.WriteFile(externalCompose, []byte("version: '3'"), 0644)
	if err != nil {
		t.Fatalf("failed to write external compose: %v", err)
	}

	// Symlink externalDir into root
	symlinkPath := filepath.Join(root, "linked-stack")
	err = os.Symlink(externalDir, symlinkPath)
	if err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// 2. Execute scanner
	matches, err := FindComposeFiles(root)
	if err != nil {
		t.Fatalf("FindComposeFiles failed: %v", err)
	}

	// 3. Assert results
	expected := []string{
		filepath.Join(root, "nested", "nested-compose.yml"),
		filepath.Join(root, "linked-stack", "external-compose.yml"),
	}

	sort.Strings(matches)
	sort.Strings(expected)

	if len(matches) != len(expected) {
		t.Fatalf("expected %d matches, got %d: %+v", len(expected), len(matches), matches)
	}

	for i := range expected {
		if matches[i] != expected[i] {
			t.Errorf("expected match %q, got %q", expected[i], matches[i])
		}
	}
}
