package plugin

import (
	"fmt"
	"strings"
)

// ResolveInstallOrder resolves the install order for a target catalog item.
// It checks the available catalog items, avoids circular dependencies, detects missing dependencies,
// and returns a list of CatalogItems sorted in topological order (dependencies first) that need to be installed.
func ResolveInstallOrder(target CatalogItem, catalog []CatalogItem, installed map[string]bool) ([]CatalogItem, error) {
	// Create lookup map for catalog items by name
	catalogMap := make(map[string]CatalogItem)
	for _, item := range catalog {
		key := strings.ToLower(item.Name)
		catalogMap[key] = item
	}

	var order []CatalogItem
	visited := make(map[string]int) // 0 = unvisited, 1 = visiting, 2 = visited
	var dfs func(item CatalogItem) error

	dfs = func(item CatalogItem) error {
		nameLower := strings.ToLower(item.Name)
		if visited[nameLower] == 2 {
			return nil
		}
		if visited[nameLower] == 1 {
			// Found cycle
			return fmt.Errorf("circular dependency detected: %s", item.Name)
		}

		visited[nameLower] = 1

		// 1. Resolve Dependencies
		for _, dep := range item.Dependencies {
			depLower := strings.ToLower(dep.Name)
			// Skip if already installed
			if installed[depLower] {
				continue
			}

			// Look up dependency in available catalog
			depItem, found := catalogMap[depLower]
			if !found {
				if dep.Required {
					return fmt.Errorf("missing required dependency %q (kind: %s) for plugin %q", dep.Name, dep.Kind, item.Name)
				}
				// Skip non-required missing dependencies
				continue
			}

			if err := dfs(depItem); err != nil {
				if strings.Contains(err.Error(), "circular dependency detected") {
					return fmt.Errorf("%s -> %s", err.Error(), item.Name)
				}
				return err
			}
		}

		// 2. Resolve DependsOn
		for _, depName := range item.DependsOn {
			depLower := strings.ToLower(depName)
			// Skip if already installed
			if installed[depLower] {
				continue
			}

			// Look up dependency in available catalog
			depItem, found := catalogMap[depLower]
			if !found {
				return fmt.Errorf("missing required dependency %q (depends_on) for plugin %q", depName, item.Name)
			}

			if err := dfs(depItem); err != nil {
				if strings.Contains(err.Error(), "circular dependency detected") {
					return fmt.Errorf("%s -> %s", err.Error(), item.Name)
				}
				return err
			}
		}

		// 3. Resolve Requires (capabilities)
		for _, req := range item.Requires {
			// Check if already satisfied by installed catalog items, or by items in the current resolution order
			satisfied := false
			for _, c := range catalog {
				cLower := strings.ToLower(c.Name)
				if installed[cLower] || visited[cLower] > 0 {
					for _, prov := range c.Provides {
						if strings.EqualFold(prov, req) {
							satisfied = true
							break
						}
					}
				}
				if satisfied {
					break
				}
			}
			if satisfied {
				continue
			}

			// Find a catalog item that provides the required capability
			var providerItem *CatalogItem
			for _, c := range catalog {
				for _, prov := range c.Provides {
					if strings.EqualFold(prov, req) {
						providerItem = &c
						break
					}
				}
				if providerItem != nil {
					break
				}
			}

			if providerItem == nil {
				return fmt.Errorf("missing provider for required capability %q for plugin %q", req, item.Name)
			}

			if err := dfs(*providerItem); err != nil {
				if strings.Contains(err.Error(), "circular dependency detected") {
					return fmt.Errorf("%s -> %s", err.Error(), item.Name)
				}
				return err
			}
		}

		visited[nameLower] = 2
		order = append(order, item)
		return nil
	}

	// Start resolution from target
	if err := dfs(target); err != nil {
		return nil, err
	}

	return order, nil
}
