package compose

import (
	"fmt"
	"strconv"
	"strings"
)

// Lint scans the configuration for errors and warnings.
func Lint(cfg *ComposeConfig) []LintIssue {
	var issues []LintIssue

	// Map to keep track of used top-level volumes
	usedVolumes := make(map[string]bool)

	for name, svc := range cfg.Services {
		if svc == nil {
			continue
		}

		// 1. Check missing image/build (Error)
		if svc.Image == "" && svc.Build == nil {
			issues = append(issues, LintIssue{
				Severity: SeverityError,
				Message:  "Service must define either 'image' or 'build' block",
				Service:  name,
				Rule:     "missing_image_or_build",
			})
		}

		// 2. Check missing restart policy (Warning)
		if svc.Restart == "" {
			issues = append(issues, LintIssue{
				Severity: SeverityWarning,
				Message:  "Service is missing a restart policy (recommended: 'unless-stopped')",
				Service:  name,
				Rule:     "missing_restart_policy",
			})
		} else {
			validRestarts := map[string]bool{
				"no":             true,
				"always":         true,
				"on-failure":     true,
				"unless-stopped": true,
			}
			if !validRestarts[svc.Restart] {
				issues = append(issues, LintIssue{
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("Service uses non-standard restart policy: %q (should be 'no', 'always', 'on-failure', or 'unless-stopped')", svc.Restart),
					Service:  name,
					Rule:     "invalid_restart_policy",
				})
			}
		}

		// 3. Check invalid ports (Error)
		for _, portEntry := range svc.Ports {
			if err := validatePort(portEntry); err != nil {
				issues = append(issues, LintIssue{
					Severity: SeverityError,
					Message:  fmt.Sprintf("Invalid port mapping: %v", err),
					Service:  name,
					Rule:     "invalid_port_format",
				})
			}
		}

		// Collect used volumes
		for _, volEntry := range svc.Volumes {
			src := extractVolumeSource(volEntry)
			if src != "" {
				usedVolumes[src] = true
			}
		}
	}

	// 4. Check unused top-level volumes (Warning)
	for volName := range cfg.Volumes {
		if !usedVolumes[volName] {
			issues = append(issues, LintIssue{
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Named volume %q is declared but not used by any service", volName),
				Rule:     "unused_volume",
			})
		}
	}

	return issues
}

func validatePort(portEntry any) error {
	switch p := portEntry.(type) {
	case int:
		if p < 1 || p > 65535 {
			return fmt.Errorf("port number %d must be between 1 and 65535", p)
		}
		return nil
	case string:
		return validatePortString(p)
	case map[string]any:
		return validatePortMap(p)
	case map[any]any:
		// Convert to map[string]any
		m := make(map[string]any)
		for k, v := range p {
			if strK, ok := k.(string); ok {
				m[strK] = v
			}
		}
		return validatePortMap(m)
	default:
		return fmt.Errorf("unknown port entry type: %T", portEntry)
	}
}

func validatePortMap(m map[string]any) error {
	// Look for target and published fields
	target, okTarget := m["target"]
	if !okTarget {
		return fmt.Errorf("missing 'target' field in port mapping map")
	}

	targetVal, err := getPortInt(target)
	if err != nil {
		return fmt.Errorf("invalid target port: %w", err)
	}
	if targetVal < 1 || targetVal > 65535 {
		return fmt.Errorf("target port %d must be between 1 and 65535", targetVal)
	}

	if published, okPublished := m["published"]; okPublished {
		publishedVal, err := getPortInt(published)
		if err != nil {
			return fmt.Errorf("invalid published port: %w", err)
		}
		if publishedVal < 1 || publishedVal > 65535 {
			return fmt.Errorf("published port %d must be between 1 and 65535", publishedVal)
		}
	}

	return nil
}

func getPortInt(val any) (int, error) {
	switch v := val.(type) {
	case int:
		return v, nil
	case string:
		// If it's a variable reference, return a placeholder
		if strings.Contains(v, "$") || strings.Contains(v, "{") || strings.Contains(v, "}") {
			return 80, nil
		}
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("unsupported type %T", val)
	}
}

func validatePortString(portStr string) error {
	// If it contains variables, skip static validation
	if strings.Contains(portStr, "$") || strings.Contains(portStr, "{") || strings.Contains(portStr, "}") {
		return nil
	}

	// Remove protocol suffix (e.g. /tcp or /udp)
	parts := strings.Split(portStr, "/")
	mainPart := parts[0]
	if len(parts) > 1 {
		proto := strings.ToLower(parts[1])
		if proto != "tcp" && proto != "udp" {
			return fmt.Errorf("invalid protocol %q", proto)
		}
	}

	// Split by ":"
	colParts := strings.Split(mainPart, ":")
	if len(colParts) > 3 {
		return fmt.Errorf("too many colons in port mapping %q", portStr)
	}

	// The container port is the last part
	containerPortPart := colParts[len(colParts)-1]
	if err := validatePortRangeOrNumber(containerPortPart); err != nil {
		return fmt.Errorf("invalid container port %q: %w", containerPortPart, err)
	}

	// If there's a host port
	if len(colParts) > 1 {
		hostPortPart := colParts[len(colParts)-2]
		if err := validatePortRangeOrNumber(hostPortPart); err != nil {
			return fmt.Errorf("invalid host port %q: %w", hostPortPart, err)
		}
	}

	return nil
}

func validatePortRangeOrNumber(part string) error {
	if strings.Contains(part, "-") {
		rangeParts := strings.Split(part, "-")
		if len(rangeParts) != 2 {
			return fmt.Errorf("invalid port range %q", part)
		}
		p1, err1 := strconv.Atoi(rangeParts[0])
		p2, err2 := strconv.Atoi(rangeParts[1])
		if err1 != nil || err2 != nil || p1 < 1 || p1 > 65535 || p2 < 1 || p2 > 65535 || p1 > p2 {
			return fmt.Errorf("invalid port numbers in range %q", part)
		}
		return nil
	}

	p, err := strconv.Atoi(part)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %q", part)
	}
	return nil
}

func extractVolumeSource(vol any) string {
	switch v := vol.(type) {
	case string:
		parts := strings.Split(v, ":")
		if len(parts) > 0 {
			return parts[0]
		}
	case map[string]any:
		if src, ok := v["source"].(string); ok {
			return src
		}
	case map[any]any:
		if src, ok := v["source"].(string); ok {
			return src
		}
	}
	return ""
}
