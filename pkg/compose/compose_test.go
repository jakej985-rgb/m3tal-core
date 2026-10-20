package compose

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	// 1. Valid parse
	validYAML := `
version: '3.8'
services:
  web:
    image: nginx:latest
`
	cfg, err := Parse([]byte(validYAML))
	if err != nil {
		t.Fatalf("expected no parse error, got: %v", err)
	}
	if cfg.Services["web"] == nil || cfg.Services["web"].Image != "nginx:latest" {
		t.Fatalf("parsed service config incorrect: %+v", cfg.Services["web"])
	}

	// 2. Invalid YAML syntax
	invalidYAML := `
version: '3.8'
services:
  web:
    image: [invalid
`
	_, err = Parse([]byte(invalidYAML))
	if err == nil {
		t.Fatal("expected error on invalid YAML syntax, got nil")
	}

	// 3. Missing services section
	noServicesYAML := `
version: '3.8'
volumes:
  db_data:
`
	_, err = Parse([]byte(noServicesYAML))
	if err == nil {
		t.Fatal("expected error on missing services section, got nil")
	}
}

func TestLint(t *testing.T) {
	yamlInput := `
services:
  web:
    image: nginx:latest
    restart: unless-stopped
    ports:
      - "80:80"
      - "invalid-port-mapping"
      - "${DYNAMIC_PORT}:8080"
    volumes:
      - web_data:/usr/share/nginx/html

  db:
    restart: "" # warning: missing restart policy
    # error: missing both image and build

volumes:
  web_data:
  unused_data: # warning: unused volume
`
	cfg, err := Parse([]byte(yamlInput))
	if err != nil {
		t.Fatalf("failed to parse input: %v", err)
	}

	issues := Lint(cfg)

	// We expect issues:
	// - Error on db service: missing image or build
	// - Warning on db service: missing restart policy
	// - Error on web service: invalid-port-mapping
	// - Warning: unused_data volume unused
	// Note: ${DYNAMIC_PORT}:8080 should NOT trigger an issue because it contains $ / { }

	errCount := 0
	warnCount := 0
	foundMissingImage := false
	foundMissingRestart := false
	foundInvalidPort := false
	foundUnusedVolume := false

	for _, issue := range issues {
		if issue.Severity == SeverityError {
			errCount++
		} else if issue.Severity == SeverityWarning {
			warnCount++
		}

		switch issue.Rule {
		case "missing_image_or_build":
			if issue.Service == "db" {
				foundMissingImage = true
			}
		case "missing_restart_policy":
			if issue.Service == "db" {
				foundMissingRestart = true
			}
		case "invalid_port_format":
			if issue.Service == "web" && strings.Contains(issue.Message, "invalid-port-mapping") {
				foundInvalidPort = true
			}
		case "unused_volume":
			if strings.Contains(issue.Message, "unused_data") {
				foundUnusedVolume = true
			}
		}
	}

	if !foundMissingImage {
		t.Error("expected missing_image_or_build issue on 'db'")
	}
	if !foundMissingRestart {
		t.Error("expected missing_restart_policy issue on 'db'")
	}
	if !foundInvalidPort {
		t.Error("expected invalid_port_format issue on 'web'")
	}
	if !foundUnusedVolume {
		t.Error("expected unused_volume issue on 'unused_data'")
	}
}

func TestAutoFix(t *testing.T) {
	yamlInput := `
services:
  web:
    image: nginx:latest
    ports:
      - 80
`
	fixed, fixes, err := AutoFix([]byte(yamlInput))
	if err != nil {
		t.Fatalf("expected no AutoFix error, got: %v", err)
	}

	if len(fixes) != 2 {
		t.Errorf("expected 2 fixes applied, got %d: %v", len(fixes), fixes)
	}

	fixedStr := string(fixed)
	if !strings.Contains(fixedStr, "restart: unless-stopped") {
		t.Error("expected fixed YAML to contain restart: unless-stopped")
	}
	if !strings.Contains(fixedStr, "- \"80\"") && !strings.Contains(fixedStr, "- '80'") {
		t.Error("expected ports to be normalized to string format")
	}
}

func TestGenerate(t *testing.T) {
	// Test generating webapp
	params := map[string]string{
		"APP_NAME": "my-custom-app",
		"APP_PORT": "9000",
		"DB_USER":  "dbadmin",
		"DB_NAME":  "maindb",
	}

	yamlData, err := Generate("webapp", params)
	if err != nil {
		t.Fatalf("expected no Generate error, got: %v", err)
	}

	yamlStr := string(yamlData)
	if !strings.Contains(yamlStr, "my-custom-app:") {
		t.Error("expected APP_NAME replacement to be applied")
	}
	if !strings.Contains(yamlStr, "\"9000:80\"") {
		t.Error("expected APP_PORT replacement to be applied")
	}
	if !strings.Contains(yamlStr, "POSTGRES_USER=dbadmin") {
		t.Error("expected DB_USER replacement to be applied")
	}
	// Verify default fallback for APP_IMAGE (which we didn't specify in params)
	if !strings.Contains(yamlStr, "image: nginx:latest") {
		t.Error("expected default fallback for APP_IMAGE to be nginx:latest")
	}
}
