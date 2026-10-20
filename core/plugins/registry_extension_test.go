package plugin

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchService(t *testing.T) {
	reg := &Registry{
		Routes: []RoutePlugin{
			{
				Metadata: PluginMetadata{Name: "plex-route"},
				Service:  "plex",
				Domain:   "plex.example.com",
				Port:     32400,
				Enabled:  true,
			},
			{
				Metadata: PluginMetadata{Name: "jellyfin-route"},
				Service:  "jellyfin",
				Domain:   "jellyfin.example.com",
				Port:     8096,
				Enabled:  true,
			},
		},
	}

	// 1. Match by service name
	p := reg.MatchService("plex", "", nil)
	if p == nil || p.Metadata.Name != "plex-route" {
		t.Fatalf("expected plex-route, got %v", p)
	}

	// 2. Match by image name
	p = reg.MatchService("my-jellyfin", "jellyfin/jellyfin:latest", nil)
	if p == nil || p.Metadata.Name != "jellyfin-route" {
		t.Fatalf("expected jellyfin-route, got %v", p)
	}

	// 3. Match by labels
	p = reg.MatchService("unknown", "", map[string]string{"traefik.http.routers.plex.rule": "true"})
	if p == nil || p.Metadata.Name != "plex-route" {
		t.Fatalf("expected plex-route, got %v", p)
	}
}

func TestGenerateTraefikConfig(t *testing.T) {
	reg := &Registry{
		Routes: []RoutePlugin{
			{
				Metadata:    PluginMetadata{Name: "plex-route"},
				Service:     "plex",
				Domain:      "plex.example.com",
				Port:        32400,
				Entrypoints: "web",
				Enabled:     true,
			},
		},
		Middlewares: []MiddlewarePlugin{
			{
				Metadata: PluginMetadata{Name: "auth-preset"},
				Name:     "m3tal-auth",
				Type:     "basicauth",
				Config: map[string]string{
					"usersFile": "/etc/traefik/users",
				},
				Enabled: true,
			},
		},
	}

	cfg, err := reg.GenerateTraefikConfig()
	if err != nil {
		t.Fatalf("failed to generate Traefik config: %v", err)
	}

	yamlStr := string(cfg)
	if !strings.Contains(yamlStr, "plex-route") {
		t.Errorf("expected plex-route in config, got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "m3tal-auth") {
		t.Errorf("expected m3tal-auth in config, got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "/etc/traefik/users") {
		t.Errorf("expected usersFile path in config, got:\n%s", yamlStr)
	}
}

func TestEnableDisablePlugin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "m3tal-plugin-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test-plugin.yml")
	err = os.WriteFile(filePath, []byte("apiVersion: m3tal/v1\nkind: Route\nmetadata:\n  name: test\nspec:\n  service: test\n  domain: test\n  port: 80"), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// 1. Disable
	newPath, err := DisablePlugin(filePath)
	if err != nil {
		t.Fatalf("failed to disable: %v", err)
	}
	if !strings.HasSuffix(newPath, ".disabled") {
		t.Errorf("expected .disabled suffix, got %s", newPath)
	}

	// 2. Enable
	restoredPath, err := EnablePlugin(newPath)
	if err != nil {
		t.Fatalf("failed to enable: %v", err)
	}
	if strings.HasSuffix(restoredPath, ".disabled") {
		t.Errorf("expected no .disabled suffix, got %s", restoredPath)
	}
}

func TestParameterize(t *testing.T) {
	tpl := "port: ${PORT}\nvolume: ${VOLUME}"
	vars := map[string]string{
		"PORT":   "8080",
		"VOLUME": "/mnt/data",
	}

	res := Parameterize(tpl, vars)
	expected := "port: 8080\nvolume: /mnt/data"
	if res != expected {
		t.Errorf("expected %q, got %q", expected, res)
	}
}

func TestCatalogListAndInstallUninstall(t *testing.T) {
	// Create mock registry
	reg := &Registry{
		Routes: []RoutePlugin{
			{
				Metadata: PluginMetadata{Name: "example-route"},
				Enabled:  true,
			},
		},
	}

	// 1. ListCatalog
	items := ListCatalog(reg)
	found := false
	for _, item := range items {
		if item.Name == "example-route" {
			found = true
			if !item.Installed {
				t.Error("expected example-route to be installed")
			}
			if item.Status != "enabled" {
				t.Errorf("expected example-route status to be enabled, got %s", item.Status)
			}
		}
	}
	if !found {
		t.Error("example-route not found in catalog listing")
	}

	// 2. Install and uninstall test using temporary directory
	tmpDir, err := os.MkdirTemp("", "m3tal-catalog-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test installing example-route
	err = InstallPlugin("example-route", "Route", tmpDir)
	if err != nil {
		t.Fatalf("failed to install example-route: %v", err)
	}

	// Check that manifest file was created
	pluginFile := filepath.Join(tmpDir, "routes", "example-route.yml")
	if _, err := os.Stat(pluginFile); os.IsNotExist(err) {
		t.Errorf("expected plugin file %s to exist", pluginFile)
	}

	// Load installed
	newReg, err := LoadAll(tmpDir)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	if len(newReg.Routes) != 1 || newReg.Routes[0].Metadata.Name != "example-app" {
		t.Errorf("expected loaded registry to contain example-app, got %+v", newReg.Routes)
	}

	// Test uninstalling
	err = UninstallPlugin("example-route", "Route", tmpDir, newReg)
	if err != nil {
		t.Fatalf("failed to uninstall example-route: %v", err)
	}

	// Check that file was deleted
	if _, err := os.Stat(pluginFile); !os.IsNotExist(err) {
		t.Errorf("expected plugin file %s to be deleted", pluginFile)
	}
}

func TestUnifiedTraefikPlugin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "m3tal-traefik-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	traefikDir := filepath.Join(tmpDir, "traefik")
	if err := os.MkdirAll(traefikDir, 0755); err != nil {
		t.Fatalf("failed to create traefik folder: %v", err)
	}

	manifestPath := filepath.Join(traefikDir, "gateway.yml")
	manifestContent := `
apiVersion: m3tal/v1
kind: Traefik
metadata:
  name: gateway
  description: Unified Traefik Test
  version: 1.0.0
  author: Test Team
spec:
  routes:
    - name: route-a
      service: service-a
      domain: a.example.local
      port: 8081
      entrypoints: web
      middlewares:
        - basic-auth-test
  middlewares:
    - name: basic-auth-test
      type: basicauth
      config:
        users: "admin:pass"
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// 1. Load the registry
	reg, err := LoadAll(tmpDir)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	if len(reg.Routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(reg.Routes))
	} else {
		route := reg.Routes[0]
		if route.Metadata.Name != "route-a" || route.Service != "service-a" || route.Port != 8081 {
			t.Errorf("route mismatch: %+v", route)
		}
	}

	if len(reg.Middlewares) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(reg.Middlewares))
	} else {
		mw := reg.Middlewares[0]
		if mw.Name != "basic-auth-test" || mw.Type != "basicauth" || mw.Config["users"] != "admin:pass" {
			t.Errorf("middleware mismatch: %+v", mw)
		}
	}

	// 2. Generate Traefik dynamic config
	cfg, err := reg.GenerateTraefikConfig()
	if err != nil {
		t.Fatalf("failed to generate Traefik dynamic config: %v", err)
	}
	yamlStr := string(cfg)
	if !strings.Contains(yamlStr, "route-a") || !strings.Contains(yamlStr, "basic-auth-test") {
		t.Errorf("expected route-a and basic-auth-test in dynamic config: %s", yamlStr)
	}

	// 3. Test Disable/Enable
	newPath, err := DisablePlugin(manifestPath)
	if err != nil {
		t.Fatalf("failed to disable: %v", err)
	}

	regDisabled, err := LoadAll(tmpDir)
	if err != nil {
		t.Fatalf("failed to reload: %v", err)
	}
	if len(regDisabled.Routes) != 1 || regDisabled.Routes[0].Enabled {
		t.Errorf("expected 1 disabled route, got %d (enabled: %v)", len(regDisabled.Routes), regDisabled.Routes[0].Enabled)
	}
	if len(regDisabled.Middlewares) != 1 || regDisabled.Middlewares[0].Enabled {
		t.Errorf("expected 1 disabled middleware, got %d (enabled: %v)", len(regDisabled.Middlewares), regDisabled.Middlewares[0].Enabled)
	}

	_, err = EnablePlugin(newPath)
	if err != nil {
		t.Fatalf("failed to enable: %v", err)
	}
}

func TestFetchCatalog(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{
				"name": "mock-plugin",
				"kind": "Route",
				"description": "Mock plugin",
				"version": "1.2.3",
				"author": "Mock Author",
				"url": "https://example.com/mock.yml"
			}
		]`))
	}))
	defer server.Close()

	// Backup original values
	origURL := CatalogURL
	origOverride := catalogCachePathOverride
	origDisableSig := DisableSignatureVerification
	defer func() {
		CatalogURL = origURL
		catalogCachePathOverride = origOverride
		DisableSignatureVerification = origDisableSig
	}()

	CatalogURL = server.URL + "/catalog.json"
	DisableSignatureVerification = true

	tmpDir, err := os.MkdirTemp("", "m3tal-catalog-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	catalogCachePathOverride = filepath.Join(tmpDir, "catalog.json")

	// 1. Fetch catalog: should download from server and update package Catalog variable
	catalog := FetchCatalog()
	if len(catalog) != 1 || catalog[0].Name != "mock-plugin" {
		t.Fatalf("expected 1 catalog item 'mock-plugin', got: %+v", catalog)
	}

	// Verify that the local cache file is written
	if _, err := os.Stat(catalogCachePathOverride); os.IsNotExist(err) {
		t.Fatal("expected catalog cache file to be written, but it wasn't")
	}

	// 2. Offline fallback test: change URL to invalid and verify it falls back to the cache
	CatalogURL = "http://invalid-url-that-fails.local"

	// Fetch again: should fall back to cache and return the same item
	catalogFallback := FetchCatalog()
	if len(catalogFallback) != 1 || catalogFallback[0].Name != "mock-plugin" {
		t.Fatalf("expected fallback to load 'mock-plugin' from cache, got: %+v", catalogFallback)
	}

	// 3. Complete failure fallback test: remove cache file and verify it falls back to default Catalog
	_ = os.Remove(catalogCachePathOverride)

	catalogDefault := FetchCatalog()
	// Should have at least one default plugin (like "m3tal")
	foundM3tal := false
	for _, item := range catalogDefault {
		if item.Name == "m3tal" {
			foundM3tal = true
			break
		}
	}
	if !foundM3tal {
		t.Fatal("expected fallback to default Catalog containing 'm3tal', but it wasn't found")
	}
}

func TestRegistry_PopulateWarnings(t *testing.T) {
	reg := &Registry{
		Routes: []RoutePlugin{
			{
				Metadata:  PluginMetadata{Name: "route-a"},
				DependsOn: []string{"stack-b"},
				Requires:  []string{"capability-c"},
				Enabled:   true,
			},
		},
		Stacks: []StackPlugin{
			{
				Metadata: PluginMetadata{Name: "stack-b"},
				Enabled:  false, // disabled provider!
			},
		},
	}

	reg.PopulateWarnings()

	if len(reg.Routes) != 1 {
		t.Fatal("expected 1 route plugin")
	}

	warnings := reg.Routes[0].Warnings
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d: %v", len(warnings), warnings)
	}

	hasDependencyWarning := false
	hasCapabilityWarning := false
	for _, w := range warnings {
		if strings.Contains(w, "Dependency plugin stack-b is disabled") {
			hasDependencyWarning = true
		}
		if strings.Contains(w, "Missing provider for capability: capability-c") {
			hasCapabilityWarning = true
		}
	}

	if !hasDependencyWarning {
		t.Errorf("expected dependency warning for disabled stack-b, got warnings: %v", warnings)
	}
	if !hasCapabilityWarning {
		t.Errorf("expected capability warning for capability-c, got warnings: %v", warnings)
	}
}
