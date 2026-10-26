package api

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// AptPackageCandidate represents version telemetry queried from APT cache.
type AptPackageCandidate struct {
	PackageName      string `json:"package_name"`
	InstalledVersion string `json:"installed_version"`
	CandidateVersion string `json:"candidate_version"`
	RepositoryURL    string `json:"repository_url,omitempty"`
	Status           string `json:"status"` // "installed", "available", "not_found"
}

type aptCacheEntry struct {
	data      map[string]AptPackageCandidate
	timestamp time.Time
}

var (
	aptMu         sync.RWMutex
	aptCache      = make(map[string]AptPackageCandidate)
	aptLastUpdate time.Time
	cacheTTL      = 60 * time.Second
)

// InvalidateAptCache clears the cached APT policy results.
func InvalidateAptCache() {
	aptMu.Lock()
	defer aptMu.Unlock()
	aptCache = make(map[string]AptPackageCandidate)
	aptLastUpdate = time.Time{}
}

// ParseAptPolicy parses the stdout of `apt-cache policy [packages...]`.
func ParseAptPolicy(rawOutput string) map[string]AptPackageCandidate {
	results := make(map[string]AptPackageCandidate)
	scanner := bufio.NewScanner(strings.NewReader(rawOutput))

	var currentPkg string
	var currentCandidate AptPackageCandidate

	finishCurrent := func() {
		if currentPkg != "" {
			if currentCandidate.CandidateVersion == "(none)" {
				currentCandidate.CandidateVersion = ""
			}
			if currentCandidate.InstalledVersion == "(none)" {
				currentCandidate.InstalledVersion = ""
			}
			if currentCandidate.CandidateVersion != "" {
				currentCandidate.Status = "available"
				if currentCandidate.InstalledVersion != "" && currentCandidate.InstalledVersion == currentCandidate.CandidateVersion {
					currentCandidate.Status = "installed"
				}
			} else if currentCandidate.InstalledVersion != "" {
				currentCandidate.Status = "installed"
			} else {
				currentCandidate.Status = "not_found"
			}
			currentCandidate.PackageName = currentPkg
			results[currentPkg] = currentCandidate
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check for package header e.g. "m3tal-plugin-docker-monitor:"
		if strings.HasSuffix(line, ":") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			finishCurrent()
			currentPkg = strings.TrimSuffix(trimmed, ":")
			currentCandidate = AptPackageCandidate{
				PackageName: currentPkg,
			}
			continue
		}

		if currentPkg == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "Installed:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				currentCandidate.InstalledVersion = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(trimmed, "Candidate:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				currentCandidate.CandidateVersion = strings.TrimSpace(parts[1])
			}
		} else if currentCandidate.RepositoryURL == "" && (strings.Contains(trimmed, "http://") || strings.Contains(trimmed, "https://")) {
			// Extract repository url from version table entry e.g. "500 https://jakej985-rgb.github.io/m3tal-apt-key stable/main amd64 Packages"
			fields := strings.Fields(trimmed)
			for _, f := range fields {
				if strings.HasPrefix(f, "http://") || strings.HasPrefix(f, "https://") {
					currentCandidate.RepositoryURL = f
					break
				}
			}
		}
	}

	finishCurrent()
	return results
}

// QueryAptCandidates returns candidate versions for the given packages using apt-cache policy.
func QueryAptCandidates(ctx context.Context, packageNames []string, forceRefresh bool) (map[string]AptPackageCandidate, error) {
	aptMu.RLock()
	if !forceRefresh && time.Since(aptLastUpdate) < cacheTTL && len(aptCache) > 0 {
		allCached := true
		for _, name := range packageNames {
			if _, exists := aptCache[name]; !exists {
				allCached = false
				break
			}
		}
		if allCached {
			ret := make(map[string]AptPackageCandidate, len(packageNames))
			for _, name := range packageNames {
				ret[name] = aptCache[name]
			}
			aptMu.RUnlock()
			return ret, nil
		}
	}
	aptMu.RUnlock()

	if len(packageNames) == 0 {
		return make(map[string]AptPackageCandidate), nil
	}

	// Filter package names to prevent argument injection
	validNames := make([]string, 0, len(packageNames))
	for _, name := range packageNames {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" && !strings.ContainsAny(trimmed, " \t\n;&|><$") {
			validNames = append(validNames, trimmed)
		}
	}

	if len(validNames) == 0 {
		return make(map[string]AptPackageCandidate), nil
	}

	args := append([]string{"policy"}, validNames...)
	cmd := exec.CommandContext(ctx, "apt-cache", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil && stdout.Len() == 0 {
		return nil, err
	}

	parsed := ParseAptPolicy(stdout.String())

	aptMu.Lock()
	for k, v := range parsed {
		aptCache[k] = v
	}
	aptLastUpdate = time.Now()
	aptMu.Unlock()

	return parsed, nil
}

// DiscoverAptPlugins searches APT cache for all registered m3tal-plugin-* packages.
func DiscoverAptPlugins(ctx context.Context, forceRefresh bool) ([]AptPackageCandidate, error) {
	cmd := exec.CommandContext(ctx, "apt-cache", "search", "^m3tal-plugin-")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()

	var pkgs []string
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) > 0 {
			pkgs = append(pkgs, parts[0])
		}
	}

	if len(pkgs) == 0 {
		return []AptPackageCandidate{}, nil
	}

	candidates, err := QueryAptCandidates(ctx, pkgs, forceRefresh)
	if err != nil {
		return nil, err
	}

	results := make([]AptPackageCandidate, 0, len(pkgs))
	for _, p := range pkgs {
		if c, found := candidates[p]; found {
			results = append(results, c)
		}
	}
	return results, nil
}
