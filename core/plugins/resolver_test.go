package plugin

import (
	"strings"
	"testing"
)

func TestResolveInstallOrder_ValidDAG(t *testing.T) {
	catalog := []CatalogItem{
		{
			Name: "A",
			Kind: "Route",
			Dependencies: []Dependency{
				{Name: "B", Kind: "Traefik", Required: true, AutoInstall: true},
				{Name: "C", Kind: "Middleware", Required: true, AutoInstall: true},
			},
		},
		{
			Name: "B",
			Kind: "Traefik",
			Dependencies: []Dependency{
				{Name: "D", Kind: "Middleware", Required: true, AutoInstall: true},
			},
		},
		{
			Name: "C",
			Kind: "Middleware",
			Dependencies: []Dependency{
				{Name: "D", Kind: "Middleware", Required: true, AutoInstall: true},
			},
		},
		{
			Name:         "D",
			Kind:         "Middleware",
			Dependencies: nil,
		},
	}

	installed := make(map[string]bool)
	target := catalog[0] // A

	order, err := ResolveInstallOrder(target, catalog, installed)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Correct topological order: D must be before B and C, and B & C must be before A.
	if len(order) != 4 {
		t.Fatalf("expected 4 items in resolve order, got: %d", len(order))
	}

	pos := make(map[string]int)
	for i, item := range order {
		pos[strings.ToLower(item.Name)] = i
	}

	if pos["d"] > pos["b"] {
		t.Error("expected D to be installed before B")
	}
	if pos["d"] > pos["c"] {
		t.Error("expected D to be installed before C")
	}
	if pos["b"] > pos["a"] {
		t.Error("expected B to be installed before A")
	}
	if pos["c"] > pos["a"] {
		t.Error("expected C to be installed before A")
	}
}

func TestResolveInstallOrder_Circular(t *testing.T) {
	catalog := []CatalogItem{
		{
			Name: "A",
			Kind: "Route",
			Dependencies: []Dependency{
				{Name: "B", Kind: "Traefik", Required: true},
			},
		},
		{
			Name: "B",
			Kind: "Traefik",
			Dependencies: []Dependency{
				{Name: "C", Kind: "Middleware", Required: true},
			},
		},
		{
			Name: "C",
			Kind: "Middleware",
			Dependencies: []Dependency{
				{Name: "A", Kind: "Route", Required: true},
			},
		},
	}

	installed := make(map[string]bool)
	target := catalog[0] // A

	_, err := ResolveInstallOrder(target, catalog, installed)
	if err == nil {
		t.Fatal("expected circular dependency error, got none")
	}

	if !strings.Contains(err.Error(), "circular dependency detected") {
		t.Errorf("expected circular dependency error message, got: %v", err)
	}
}

func TestResolveInstallOrder_MissingRequired(t *testing.T) {
	catalog := []CatalogItem{
		{
			Name: "A",
			Kind: "Route",
			Dependencies: []Dependency{
				{Name: "B", Kind: "Traefik", Required: true},
			},
		},
	}

	installed := make(map[string]bool)
	target := catalog[0] // A

	_, err := ResolveInstallOrder(target, catalog, installed)
	if err == nil {
		t.Fatal("expected missing required dependency error, got none")
	}

	if !strings.Contains(err.Error(), "missing required dependency") {
		t.Errorf("expected missing required dependency error, got: %v", err)
	}
}

func TestResolveInstallOrder_MissingOptional(t *testing.T) {
	catalog := []CatalogItem{
		{
			Name: "A",
			Kind: "Route",
			Dependencies: []Dependency{
				{Name: "B", Kind: "Traefik", Required: false},
			},
		},
	}

	installed := make(map[string]bool)
	target := catalog[0] // A

	order, err := ResolveInstallOrder(target, catalog, installed)
	if err != nil {
		t.Fatalf("expected no error for optional missing dependency, got: %v", err)
	}

	if len(order) != 1 || order[0].Name != "A" {
		t.Fatalf("expected only A to be resolved, got: %+v", order)
	}
}

func TestResolveInstallOrder_InstalledSkipped(t *testing.T) {
	catalog := []CatalogItem{
		{
			Name: "A",
			Kind: "Route",
			Dependencies: []Dependency{
				{Name: "B", Kind: "Traefik", Required: true},
			},
		},
		{
			Name:         "B",
			Kind:         "Traefik",
			Dependencies: nil,
		},
	}

	installed := map[string]bool{
		"b": true,
	}
	target := catalog[0] // A

	order, err := ResolveInstallOrder(target, catalog, installed)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(order) != 1 || order[0].Name != "A" {
		t.Fatalf("expected only A in resolution since B is already installed, got: %+v", order)
	}
}

func TestResolveInstallOrder_DependsOn(t *testing.T) {
	catalog := []CatalogItem{
		{
			Name:      "A",
			Kind:      "Route",
			DependsOn: []string{"B"},
		},
		{
			Name: "B",
			Kind: "Traefik",
		},
	}

	installed := make(map[string]bool)
	target := catalog[0] // A

	order, err := ResolveInstallOrder(target, catalog, installed)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("expected 2 items, got: %d", len(order))
	}
	if order[0].Name != "B" || order[1].Name != "A" {
		t.Errorf("expected B then A, got: %v then %v", order[0].Name, order[1].Name)
	}
}

func TestResolveInstallOrder_Capabilities(t *testing.T) {
	catalog := []CatalogItem{
		{
			Name:     "A",
			Kind:     "Route",
			Requires: []string{"identity-provider"},
		},
		{
			Name:     "B",
			Kind:     "Service",
			Provides: []string{"identity-provider"},
		},
		{
			Name:     "C",
			Kind:     "Service",
			Provides: []string{"identity-provider"},
		},
	}

	// 1. Not installed -> resolve B (first provider in catalog)
	installed := make(map[string]bool)
	target := catalog[0] // A

	order, err := ResolveInstallOrder(target, catalog, installed)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("expected 2 items, got: %d", len(order))
	}
	if order[0].Name != "B" || order[1].Name != "A" {
		t.Errorf("expected B then A, got: %s then %s", order[0].Name, order[1].Name)
	}

	// 2. C is installed -> A's requirement is satisfied, only A is resolved
	installed = map[string]bool{
		"c": true,
	}
	order2, err := ResolveInstallOrder(target, catalog, installed)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(order2) != 1 || order2[0].Name != "A" {
		t.Fatalf("expected only A since C is installed providing capability, got: %+v", order2)
	}
}
