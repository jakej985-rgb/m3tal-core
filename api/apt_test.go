package api

import (
	"context"
	"testing"
	"time"
)

func TestParseAptPolicy(t *testing.T) {
	rawOutput := `m3tal-plugin-docker-monitor:
  Installed: 0.6.0
  Candidate: 0.6.0
  Version table:
 *** 0.6.0 500
        500 https://jakej985-rgb.github.io/m3tal-apt-key stable/main amd64 Packages
        100 /var/lib/dpkg/status
     0.5.0 500
        500 https://jakej985-rgb.github.io/m3tal-apt-key stable/main amd64 Packages
m3tal-plugin-compose:
  Installed: (none)
  Candidate: 0.3.0
  Version table:
     0.3.0 500
        500 https://jakej985-rgb.github.io/m3tal-apt-key stable/main amd64 Packages
m3tal-plugin-unreleased:
  Installed: (none)
  Candidate: (none)
  Version table:
`

	parsed := ParseAptPolicy(rawOutput)

	docker, ok := parsed["m3tal-plugin-docker-monitor"]
	if !ok {
		t.Fatalf("expected m3tal-plugin-docker-monitor in parsed map")
	}
	if docker.InstalledVersion != "0.6.0" {
		t.Errorf("expected installed version 0.6.0, got %s", docker.InstalledVersion)
	}
	if docker.CandidateVersion != "0.6.0" {
		t.Errorf("expected candidate version 0.6.0, got %s", docker.CandidateVersion)
	}
	if docker.Status != "installed" {
		t.Errorf("expected status installed, got %s", docker.Status)
	}
	if docker.RepositoryURL != "https://jakej985-rgb.github.io/m3tal-apt-key" {
		t.Errorf("expected repository URL https://jakej985-rgb.github.io/m3tal-apt-key, got %s", docker.RepositoryURL)
	}

	compose, ok := parsed["m3tal-plugin-compose"]
	if !ok {
		t.Fatalf("expected m3tal-plugin-compose in parsed map")
	}
	if compose.InstalledVersion != "" {
		t.Errorf("expected empty installed version for compose, got %s", compose.InstalledVersion)
	}
	if compose.CandidateVersion != "0.3.0" {
		t.Errorf("expected candidate version 0.3.0, got %s", compose.CandidateVersion)
	}
	if compose.Status != "available" {
		t.Errorf("expected status available, got %s", compose.Status)
	}

	unreleased, ok := parsed["m3tal-plugin-unreleased"]
	if !ok {
		t.Fatalf("expected m3tal-plugin-unreleased in parsed map")
	}
	if unreleased.CandidateVersion != "" || unreleased.InstalledVersion != "" {
		t.Errorf("expected empty versions for unreleased, got inst=%s cand=%s", unreleased.InstalledVersion, unreleased.CandidateVersion)
	}
	if unreleased.Status != "not_found" {
		t.Errorf("expected status not_found, got %s", unreleased.Status)
	}
}

func TestQueryAptCandidatesReal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pkgs := []string{"m3tal-plugin-docker-monitor", "m3tal-plugin-compose"}
	res, err := QueryAptCandidates(ctx, pkgs, true)
	if err != nil {
		t.Skipf("apt-cache not available on this environment: %v", err)
	}

	if doc, ok := res["m3tal-plugin-docker-monitor"]; ok {
		if doc.CandidateVersion == "" {
			t.Errorf("expected non-empty candidate version for docker-monitor")
		}
	}
}
