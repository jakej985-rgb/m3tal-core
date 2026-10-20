package docker

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/core/traefik"
)

// ValidationError describes a single validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// reservedPorts that should never be used as service ports.
var reservedPorts = map[int]string{
	22:   "SSH",
	2375: "Docker socket (unencrypted)",
	2376: "Docker socket (TLS)",
	5432: "PostgreSQL",
	3306: "MySQL",
}

// domainRegex validates hostname format.
var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// ValidateRoute checks a RouteInput for correctness.
func ValidateRoute(r traefik.RouteInput) []ValidationError {
	var errs []ValidationError

	// Service name
	if r.Service == "" {
		errs = append(errs, ValidationError{Field: "service", Message: "service name is required"})
	} else if strings.ContainsAny(r.Service, " \t\n") {
		errs = append(errs, ValidationError{Field: "service", Message: "service name must not contain whitespace"})
	}

	// Domain
	errs = append(errs, ValidateDomain(r.Domain)...)

	// Port
	errs = append(errs, ValidatePort(r.Port)...)

	return errs
}

// ValidateDomain checks that a domain string is well-formed.
func ValidateDomain(domain string) []ValidationError {
	var errs []ValidationError

	if domain == "" {
		errs = append(errs, ValidationError{Field: "domain", Message: "domain is required"})
		return errs
	}

	if !domainRegex.MatchString(domain) {
		errs = append(errs, ValidationError{Field: "domain", Message: fmt.Sprintf("invalid domain format: %q", domain)})
	}

	return errs
}

// ValidatePort checks that a port number is valid and not reserved.
func ValidatePort(port int) []ValidationError {
	var errs []ValidationError

	if port < 1 || port > 65535 {
		errs = append(errs, ValidationError{
			Field:   "port",
			Message: fmt.Sprintf("port must be between 1 and 65535, got %d", port),
		})
		return errs
	}

	if svc, reserved := reservedPorts[port]; reserved {
		errs = append(errs, ValidationError{
			Field:   "port",
			Message: fmt.Sprintf("port %d is reserved for %s", port, svc),
		})
	}

	return errs
}

// ValidateDomainUnique checks that a domain is not already in use.
func ValidateDomainUnique(domain string, existing []traefik.RouteInput) []ValidationError {
	var errs []ValidationError
	for _, r := range existing {
		if strings.EqualFold(r.Domain, domain) {
			errs = append(errs, ValidationError{
				Field:   "domain",
				Message: fmt.Sprintf("domain %q is already routed to service %q", domain, r.Service),
			})
			break
		}
	}
	return errs
}

// ValidateCompose checks a parsed compose file for common issues.
func ValidateCompose(cf *traefik.ComposeFile) []ValidationError {
	var errs []ValidationError

	if len(cf.Services) == 0 {
		errs = append(errs, ValidationError{
			Field:   "services",
			Message: "compose file contains no services",
		})
		return errs
	}

	// Track domains and ports for duplicate detection
	seenDomains := make(map[string]string) // domain -> service
	seenPorts := make(map[int]string)      // port -> service

	for name, svc := range cf.Services {
		// Check for missing image
		if svc.Image == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("services.%s.image", name),
				Message: "service has no image defined",
			})
		}

		// Check for port conflicts within the compose file
		for _, p := range cf.GetExposedPorts(name) {
			if other, dup := seenPorts[p]; dup {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("services.%s.ports", name),
					Message: fmt.Sprintf("port %d conflicts with service %q", p, other),
				})
			} else {
				seenPorts[p] = name
			}
		}

		// Check for duplicate Traefik domains
		for k, v := range svc.Labels.Values {
			if strings.Contains(k, ".rule") && strings.Contains(v, "Host(") {
				domain := extractDomainFromRule(v)
				if domain != "" {
					if other, dup := seenDomains[domain]; dup {
						errs = append(errs, ValidationError{
							Field:   fmt.Sprintf("services.%s.labels", name),
							Message: fmt.Sprintf("domain %q conflicts with service %q", domain, other),
						})
					} else {
						seenDomains[domain] = name
					}
				}
			}
		}
	}

	return errs
}

// ValidatePortAvailable checks if a port is currently in use on the host.
func ValidatePortAvailable(port int) *ValidationError {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return &ValidationError{
			Field:   "port",
			Message: fmt.Sprintf("port %d is already in use on the host", port),
		}
	}
	ln.Close()
	return nil
}

// extractDomainFromRule pulls the domain from a Traefik Host() rule.
// e.g. "Host(`app.local`)" -> "app.local"
func extractDomainFromRule(rule string) string {
	re := regexp.MustCompile(`Host\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	matches := re.FindStringSubmatch(rule)
	if len(matches) >= 2 {
		// Strip ${VAR:-default} patterns to get base domain
		domain := matches[1]
		// Handle cases like "app.${DOMAIN:-localhost}"
		if strings.Contains(domain, "${") {
			// Can't validate template domains, skip
			return ""
		}
		return domain
	}
	return ""
}
