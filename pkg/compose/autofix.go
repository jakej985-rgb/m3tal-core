package compose

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// AutoFix parses the YAML, fixes common issues (restart policies, port normalization), and returns the fixed content.
func AutoFix(yamlData []byte) ([]byte, []string, error) {
	var data map[string]any
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var appliedFixes []string

	servicesSection, ok := data["services"]
	if !ok || servicesSection == nil {
		return nil, nil, fmt.Errorf("missing services section")
	}

	servicesMap, ok := servicesSection.(map[string]any)
	if !ok {
		// Try map[any]any
		servicesMapAny, okAny := servicesSection.(map[any]any)
		if !okAny {
			return nil, nil, fmt.Errorf("services section is not a map")
		}
		// Convert
		servicesMap = make(map[string]any)
		for k, v := range servicesMapAny {
			if strK, okStr := k.(string); okStr {
				servicesMap[strK] = v
			}
		}
		data["services"] = servicesMap
	}

	for svcName, svcVal := range servicesMap {
		svcMap, ok := svcVal.(map[string]any)
		if !ok {
			svcMapAny, okAny := svcVal.(map[any]any)
			if !okAny {
				continue
			}
			svcMap = make(map[string]any)
			for k, v := range svcMapAny {
				if strK, okStr := k.(string); okStr {
					svcMap[strK] = v
				}
			}
			servicesMap[svcName] = svcMap
		}

		// 1. Fix missing/empty restart policy
		restartRaw, okRestart := svcMap["restart"]
		if !okRestart || restartRaw == nil || restartRaw == "" {
			svcMap["restart"] = "unless-stopped"
			appliedFixes = append(appliedFixes, fmt.Sprintf("Added 'restart: unless-stopped' to service %q", svcName))
		}

		// 2. Normalize ports
		portsRaw, okPorts := svcMap["ports"]
		if okPorts && portsRaw != nil {
			if portsList, okList := portsRaw.([]any); okList {
				var normalizedPorts []any
				modifiedPorts := false
				for _, p := range portsList {
					switch val := p.(type) {
					case int:
						// Convert numeric port to string format
						strVal := fmt.Sprintf("%d", val)
						normalizedPorts = append(normalizedPorts, strVal)
						modifiedPorts = true
					case string:
						cleaned := strings.TrimSpace(val)
						if cleaned != val {
							normalizedPorts = append(normalizedPorts, cleaned)
							modifiedPorts = true
						} else {
							normalizedPorts = append(normalizedPorts, val)
						}
					default:
						normalizedPorts = append(normalizedPorts, p)
					}
				}
				if modifiedPorts {
					svcMap["ports"] = normalizedPorts
					appliedFixes = append(appliedFixes, fmt.Sprintf("Normalized port formats for service %q", svcName))
				}
			}
		}
	}

	fixedYAML, err := yaml.Marshal(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal fixed YAML: %w", err)
	}

	return fixedYAML, appliedFixes, nil
}
