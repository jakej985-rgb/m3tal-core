package proxy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jakej985-rgb/m3tal-core/pkg/system"
)

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		rule string
		want string
	}{
		{"Host(`app.localhost`)", "app.localhost"},
		{"Host(`my-app.domain.com`)", "my-app.domain.com"},
		{"Host(`sub.domain.local`) && Path(`/api`)", "sub.domain.local"},
		{"InvalidRule", ""},
	}

	for _, tt := range tests {
		got := extractDomain(tt.rule)
		if got != tt.want {
			t.Errorf("extractDomain(%q) = %q, want %q", tt.rule, got, tt.want)
		}
	}
}

func TestSanitizeDomainFilename(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{"app.localhost", "app_localhost"},
		{"my-app.domain.com", "my-app_domain_com"},
		{"test-service", "test-service"},
	}

	for _, tt := range tests {
		got := sanitizeDomainFilename(tt.domain)
		if got != tt.want {
			t.Errorf("sanitizeDomainFilename(%q) = %q, want %q", tt.domain, got, tt.want)
		}
	}
}

func TestWriteAndDeletePluginFile(t *testing.T) {
	// Override UserPluginsDir for testing
	tmpDir := t.TempDir()
	os.Setenv("M3TAL_PLUGINS_DIR", tmpDir)
	defer os.Unsetenv("M3TAL_PLUGINS_DIR")

	// Set deploy/plugins check fallback by checking the directory
	// In the real code it checks os.Stat("deploy/plugins")
	// Since we are running in workspace root, it might resolve deploy/plugins
	// So we handle checking both routes paths

	mgr := &Manager{}

	err := mgr.WriteRoutePluginFile("test-service", "test.domain.local", 8080, true, []string{"mw1", "mw2"})
	if err != nil {
		t.Fatalf("WriteRoutePluginFile failed: %v", err)
	}

	// Verify file is created in whichever directory was resolved
	userDir := system.GetUserPluginsDir()
	expectedFile := filepath.Join(userDir, "routes", "test-service-test_domain_local.yml")

	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", expectedFile)
	}

	// Read content and check YAML structure
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "service: test-service") {
		t.Errorf("expected file to specify service: test-service, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "domain: test.domain.local") {
		t.Errorf("expected file to specify domain, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "ssl: true") {
		t.Errorf("expected file to specify ssl: true, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "- mw1") {
		t.Errorf("expected file to contain middleware mw1, got:\n%s", contentStr)
	}

	// Delete file
	err = mgr.DeletePluginFile("test-service", "test.domain.local")
	if err != nil {
		t.Fatalf("DeletePluginFile failed: %v", err)
	}

	if _, err := os.Stat(expectedFile); !os.IsNotExist(err) {
		t.Errorf("expected file %s to be deleted", expectedFile)
	}
}
