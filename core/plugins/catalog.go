package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jakej985-rgb/m3tal-core/pkg/system"
)

// CatalogItem represents a plugin available in the remote official catalog.
type CatalogItem struct {
	Name         string       `json:"name"`
	Kind         string       `json:"kind"`
	Description  string       `json:"description"`
	Version      string       `json:"version"`
	Author       string       `json:"author"`
	URL          string       `json:"url"`
	ComposeURL   string       `json:"composeUrl,omitempty"`
	Category     string       `json:"category,omitempty"`
	Subcategory  string       `json:"subcategory,omitempty"`
	Provider     string       `json:"provider,omitempty"`
	Provides     []string     `json:"provides,omitempty"`
	Requires     []string     `json:"requires,omitempty"`
	DependsOn    []string     `json:"depends_on,omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
}

// CatalogItemStatus wraps CatalogItem with its local installation state.
type CatalogItemStatus struct {
	CatalogItem
	Installed bool   `json:"installed"`
	Status    string `json:"status"` // "enabled", "disabled", "not_installed"
}

// Catalog is the static index of official M3TAL plugins available for download.
var Catalog = []CatalogItem{
	{
		Name:        "m3tal",
		Kind:        "Stack",
		Description: "M3TAL Core dashboard and system services stack",
		Version:     "1.0.0",
		Author:      "M3TAL Team",
		URL:         "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/plugins/stacks/m3tal.yml",
		ComposeURL:  "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/stack/m3tal-compose.yml",
	},
	{
		Name:        "ai",
		Kind:        "Stack",
		Description: "Modular AI Addon (Ollama + env profiles)",
		Version:     "1.0.0",
		Author:      "M3TAL Team",
		URL:         "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/plugins/stacks/ai.yml",
		ComposeURL:  "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/stack/ai-compose.yml",
	},
	{
		Name:        "maintenance",
		Kind:        "Stack",
		Description: "System maintenance and monitoring tools",
		Version:     "1.0.0",
		Author:      "M3TAL Team",
		URL:         "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/plugins/stacks/maintenance.yml",
		ComposeURL:  "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/stack/maintenance-compose.yml",
	},
	{
		Name:        "network",
		Kind:        "Stack",
		Description: "Core network services and reverse proxy setup",
		Version:     "1.0.0",
		Author:      "M3TAL Team",
		URL:         "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/plugins/stacks/network.yml",
		ComposeURL:  "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/stack/network-compose.yml",
	},
	{
		Name:        "routing",
		Kind:        "Stack",
		Description: "Traefik reverse proxy and SSL termination",
		Version:     "1.0.0",
		Author:      "M3TAL Team",
		URL:         "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/plugins/stacks/routing.yml",
		ComposeURL:  "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/stack/routing-compose.yml",
	},
	{
		Name:        "basicauth",
		Kind:        "Middleware",
		Description: "Basic authentication preset for routes",
		Version:     "1.0.0",
		Author:      "M3TAL Team",
		URL:         "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/plugins/middleware/basicauth.yml",
	},
	{
		Name:        "ratelimit",
		Kind:        "Middleware",
		Description: "Rate limiting configuration for endpoints",
		Version:     "1.0.0",
		Author:      "M3TAL Team",
		URL:         "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/plugins/middleware/ratelimit.yml",
	},
	{
		Name:        "security-headers",
		Kind:        "Middleware",
		Description: "Hardened security headers for HTTP services",
		Version:     "1.0.0",
		Author:      "M3TAL Team",
		URL:         "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/plugins/middleware/security-headers.yml",
	},
	{
		Name:        "example-route",
		Kind:        "Route",
		Description: "Example route plugin — customize for your service",
		Version:     "1.0.0",
		Author:      "M3TAL Team",
		URL:         "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/plugins/routes/example-route.yml",
	},
	{
		Name:        "traefik-gateway",
		Kind:        "Traefik",
		Description: "Unified Traefik configuration with default routing and basic authentication preset",
		Version:     "1.0.0",
		Author:      "M3TAL Team",
		URL:         "https://raw.githubusercontent.com/jakej985-rgb/m3tal-core/main/deploy/plugins/traefik/traefik-gateway.yml",
	},
}

// GetPluginBaseName extracts the clean name of the plugin from its filepath.
// It also supports .json files.
func GetPluginBaseName(path string) string {
	if path == "" {
		return ""
	}
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".disabled")
	base = strings.TrimSuffix(base, ".yml")
	base = strings.TrimSuffix(base, ".yaml")
	base = strings.TrimSuffix(base, ".json")
	return base
}

// MatchesPluginName checks if a plugin matches the target catalog name.
func MatchesPluginName(sourcePath, metadataName, targetName string) bool {
	if strings.EqualFold(metadataName, targetName) {
		return true
	}
	base := GetPluginBaseName(sourcePath)
	return base != "" && strings.EqualFold(base, targetName)
}

// CatalogURL is the remote URL of the official M3TAL plugin catalog.
var CatalogURL = "https://jakej985-rgb.github.io/m3tal-plugin-page/catalog.json"

// DisableSignatureVerification allows tests to bypass GPG signature checks.
var DisableSignatureVerification = false

// catalogCachePathOverride allows tests to redirect the catalog cache file.
var catalogCachePathOverride = ""

// bootstrapCatalog preserves the initial hardcoded list to allow restoring on complete failures.
var bootstrapCatalog []CatalogItem

func init() {
	bootstrapCatalog = make([]CatalogItem, len(Catalog))
	copy(bootstrapCatalog, Catalog)
}

// getCatalogCachePath returns the file path where the catalog should be cached.
func getCatalogCachePath() string {
	if catalogCachePathOverride != "" {
		return catalogCachePathOverride
	}
	if _, err := os.Stat("deploy/plugins"); err == nil {
		return "deploy/plugins/catalog.json"
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".cache", "m3tal", "catalog.json")
	}
	return filepath.Join(os.TempDir(), "m3tal-catalog.json")
}

// VerifyCatalogSignature verifies that the catalog file is signed by a trusted GPG keyring.
func VerifyCatalogSignature(catalogPath, sigPath string) error {
	if DisableSignatureVerification {
		return nil
	}

	// Look for GPG keyrings in standard system paths
	keyrings := []string{
		"/etc/apt/trusted.gpg",
		"/usr/share/keyrings/m3tal-archive-keyring.gpg",
		"/etc/apt/trusted.gpg.d/m3tal.gpg",
		"/etc/apt/trusted.gpg.d/m3tal-archive-keyring.gpg",
		"/etc/m3tal/public.key",
	}

	var activeKeyring string
	for _, k := range keyrings {
		if _, err := os.Stat(k); err == nil {
			activeKeyring = k
			break
		}
	}

	if activeKeyring == "" {
		// Non-blocking warning for development / custom environments
		fmt.Fprintln(os.Stderr, "⚠️  Catalog signature verification skipped (no trusted keyring found in standard paths)")
		return nil
	}

	// Verify using gpgv
	cmd := exec.Command("gpgv", "--keyring", activeKeyring, sigPath, catalogPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gpgv signature validation failed: %w", err)
	}

	return nil
}

// FetchCatalog updates the global Catalog slice from the remote repository or local cache.
func FetchCatalog() []CatalogItem {
	cachePath := getCatalogCachePath()
	sigCachePath := cachePath + ".sig"

	// Try fetching catalog and signature from remote URL
	client := &http.Client{Timeout: 3 * time.Second}

	// Fetch catalog.json
	resp, err := client.Get(CatalogURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			data, err := io.ReadAll(resp.Body)
			if err == nil {
				// Fetch catalog.json.sig
				respSig, errSig := client.Get(CatalogURL + ".sig")
				if errSig == nil {
					defer respSig.Body.Close()
					if respSig.StatusCode == http.StatusOK {
						sigData, errSigRead := io.ReadAll(respSig.Body)
						if errSigRead == nil {
							// Write both to temporary files first to verify
							tempDir := os.TempDir()
							tempCatalog := filepath.Join(tempDir, "m3tal-catalog-temp.json")
							tempSig := filepath.Join(tempDir, "m3tal-catalog-temp.json.sig")

							if errWrite := os.WriteFile(tempCatalog, data, 0644); errWrite == nil {
								if errSigWrite := os.WriteFile(tempSig, sigData, 0644); errSigWrite == nil {
									// Verify the GPG signature
									if errVerify := VerifyCatalogSignature(tempCatalog, tempSig); errVerify == nil {
										// Signature verified! Save both to local cache
										_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
										_ = os.WriteFile(cachePath, data, 0644)
										_ = os.WriteFile(sigCachePath, sigData, 0644)

										var remoteCatalog []CatalogItem
										if errUnmarshal := json.Unmarshal(data, &remoteCatalog); errUnmarshal == nil && len(remoteCatalog) > 0 {
											Catalog = remoteCatalog
											return Catalog
										}
									} else {
										fmt.Fprintf(os.Stderr, "❌ Catalog signature verification failed: %v\n", errVerify)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Fallback to local cache, but also verify the cached signature if possible
	if data, err := os.ReadFile(cachePath); err == nil {
		if _, errSig := os.Stat(sigCachePath); errSig == nil {
			if errVerify := VerifyCatalogSignature(cachePath, sigCachePath); errVerify == nil {
				var cachedCatalog []CatalogItem
				if errUnmarshal := json.Unmarshal(data, &cachedCatalog); errUnmarshal == nil && len(cachedCatalog) > 0 {
					Catalog = cachedCatalog
					return Catalog
				}
			} else {
				fmt.Fprintf(os.Stderr, "⚠️  Cached catalog signature verification failed: %v\n", errVerify)
			}
		} else {
			// If cached sig is missing but catalog is there, parse it with warning (developer/custom mode fallback)
			var cachedCatalog []CatalogItem
			if errUnmarshal := json.Unmarshal(data, &cachedCatalog); errUnmarshal == nil && len(cachedCatalog) > 0 {
				Catalog = cachedCatalog
				return Catalog
			}
		}
	}

	// Fallback to bootstrap default
	Catalog = bootstrapCatalog
	return Catalog
}

// ListCatalog correlates remote catalog items with local loaded plugins.
func ListCatalog(reg *Registry) []CatalogItemStatus {
	var list []CatalogItemStatus

	catalog := FetchCatalog()
	for _, item := range catalog {
		status := CatalogItemStatus{
			CatalogItem: item,
			Installed:   false,
			Status:      "not_installed",
		}

		switch item.Kind {
		case KindRoute:
			for _, r := range reg.Routes {
				if MatchesPluginName(r.SourcePath, r.Metadata.Name, item.Name) {
					status.Installed = true
					if r.Enabled {
						status.Status = "enabled"
					} else {
						status.Status = "disabled"
					}
					break
				}
			}
		case KindStack:
			for _, s := range reg.Stacks {
				if MatchesPluginName(s.SourcePath, s.Metadata.Name, item.Name) {
					status.Installed = true
					if s.Enabled {
						status.Status = "enabled"
					} else {
						status.Status = "disabled"
					}
					break
				}
			}
		case KindMiddleware:
			for _, m := range reg.Middlewares {
				if MatchesPluginName(m.SourcePath, m.Metadata.Name, item.Name) {
					status.Installed = true
					if m.Enabled {
						status.Status = "enabled"
					} else {
						status.Status = "disabled"
					}
					break
				}
			}
		case KindService:
			for _, s := range reg.Services {
				if MatchesPluginName(s.SourcePath, s.Metadata.Name, item.Name) {
					status.Installed = true
					if s.Enabled {
						status.Status = "enabled"
					} else {
						status.Status = "disabled"
					}
					break
				}
			}
		case KindTraefik:
			for _, r := range reg.Routes {
				if MatchesPluginName(r.SourcePath, r.Metadata.Name, item.Name) {
					status.Installed = true
					if r.Enabled {
						status.Status = "enabled"
					} else {
						status.Status = "disabled"
					}
					break
				}
			}
			if !status.Installed {
				for _, m := range reg.Middlewares {
					if MatchesPluginName(m.SourcePath, m.Metadata.Name, item.Name) {
						status.Installed = true
						if m.Enabled {
							status.Status = "enabled"
						} else {
							status.Status = "disabled"
						}
						break
					}
				}
			}
		}

		list = append(list, status)
	}

	return list
}

// downloadFile fetches a URL and writes it locally.
func downloadFile(url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(dest), err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http get failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	return nil
}

// installSinglePlugin downloads a single plugin's definition and optional compose files.
func installSinglePlugin(item CatalogItem, userPluginsDir string) error {
	var subfolder string
	switch item.Kind {
	case KindRoute:
		subfolder = "routes"
	case KindStack:
		subfolder = "stacks"
	case KindMiddleware:
		subfolder = "middleware"
	case KindTraefik:
		subfolder = "traefik"
	case KindService:
		subfolder = "services"
	default:
		return fmt.Errorf("unsupported plugin kind: %s", item.Kind)
	}

	// We default to .yml for catalog items unless the catalog URL specifically ends in .json
	ext := ".yml"
	if strings.HasSuffix(strings.ToLower(item.URL), ".json") {
		ext = ".json"
	}

	pluginDest := filepath.Join(userPluginsDir, subfolder, item.Name+ext)
	err := downloadFile(item.URL, pluginDest)
	if err != nil {
		return fmt.Errorf("failed to download plugin manifest: %w", err)
	}

	// Read and parse downloaded manifest to run PreInstall hook
	var parsedPlugin *Plugin
	if data, err := os.ReadFile(pluginDest); err == nil {
		if p, parseErr := parseAnyPlugin(data, pluginDest); parseErr == nil {
			parsedPlugin = p
			if p.Hooks != nil && p.Hooks.PreInstall != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_ = ExecuteHook(ctx, p.Hooks.PreInstall, p)
				cancel()
			}
		}
	}

	// Download compose file if it is a Stack plugin and has a ComposeURL
	if item.Kind == KindStack && item.ComposeURL != "" {
		composeFilename := filepath.Base(item.ComposeURL)
		composeDest := filepath.Join(userPluginsDir, subfolder, composeFilename)
		err = downloadFile(item.ComposeURL, composeDest)
		if err != nil {
			_ = os.Remove(pluginDest) // clean up manifest
			return fmt.Errorf("failed to download associated compose file %s: %w", composeFilename, err)
		}
	}

	// Run PostInstall hook
	if parsedPlugin != nil && parsedPlugin.Hooks != nil && parsedPlugin.Hooks.PostInstall != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_ = ExecuteHook(ctx, parsedPlugin.Hooks.PostInstall, parsedPlugin)
		cancel()
	}

	return nil
}

// InstallPlugin downloads a plugin's definition and optional compose files, resolving dependencies.
func InstallPlugin(name, kind, userPluginsDir string) error {
	catalog := FetchCatalog()

	// Find the target item in the catalog to verify it exists
	var targetItem *CatalogItem
	for i := range catalog {
		if strings.EqualFold(catalog[i].Name, name) && strings.EqualFold(catalog[i].Kind, kind) {
			targetItem = &catalog[i]
			break
		}
	}
	if targetItem == nil {
		return fmt.Errorf("plugin %q (kind: %s) not found in catalog", name, kind)
	}

	// Build map of currently installed plugins to avoid re-installing
	dirs := []string{userPluginsDir}
	if userPluginsDir != system.SystemPluginsDir {
		dirs = append(dirs, system.SystemPluginsDir)
	}
	reg, err := LoadAll(dirs...)
	installed := make(map[string]bool)
	if err == nil && reg != nil {
		for _, r := range reg.Routes {
			installed[strings.ToLower(r.Metadata.Name)] = true
		}
		for _, s := range reg.Stacks {
			installed[strings.ToLower(s.Metadata.Name)] = true
		}
		for _, m := range reg.Middlewares {
			installed[strings.ToLower(m.Metadata.Name)] = true
		}
		for _, s := range reg.Services {
			installed[strings.ToLower(s.Metadata.Name)] = true
		}
	}

	// Resolve dependency install order
	plan, err := ResolveInstallOrder(*targetItem, catalog, installed)
	if err != nil {
		return err
	}

	// Install all resolved dependencies first
	for _, item := range plan {
		if strings.EqualFold(item.Name, name) && strings.EqualFold(item.Kind, kind) {
			continue
		}

		if installed[strings.ToLower(item.Name)] {
			continue
		}

		// Check if the dependency is marked as autoInstall in the plan
		isAutoInstall := false
		for _, planItem := range plan {
			for _, dep := range planItem.Dependencies {
				if strings.EqualFold(dep.Name, item.Name) && dep.AutoInstall {
					isAutoInstall = true
					break
				}
			}
		}

		if isAutoInstall {
			fmt.Printf("📥 Auto-installing dependency %s %q...\n", item.Kind, item.Name)
			err = installSinglePlugin(item, userPluginsDir)
			if err != nil {
				return fmt.Errorf("failed to auto-install dependency %q: %w", item.Name, err)
			}
			installed[strings.ToLower(item.Name)] = true
		} else {
			return fmt.Errorf("missing required dependency %q (kind: %s) which is not marked as auto-installable", item.Name, item.Kind)
		}
	}

	// Finally, install the target item itself
	return installSinglePlugin(*targetItem, userPluginsDir)
}

// UninstallPlugin deletes a user-installed plugin's files.
func UninstallPlugin(name, kind, userPluginsDir string, reg *Registry) error {
	var sourcePath string
	var composePathRel string

	switch kind {
	case KindRoute:
		for i := range reg.Routes {
			if MatchesPluginName(reg.Routes[i].SourcePath, reg.Routes[i].Metadata.Name, name) {
				sourcePath = reg.Routes[i].SourcePath
				break
			}
		}
	case KindStack:
		for i := range reg.Stacks {
			if MatchesPluginName(reg.Stacks[i].SourcePath, reg.Stacks[i].Metadata.Name, name) {
				sourcePath = reg.Stacks[i].SourcePath
				composePathRel = reg.Stacks[i].ComposePath
				break
			}
		}
	case KindMiddleware:
		for i := range reg.Middlewares {
			if MatchesPluginName(reg.Middlewares[i].SourcePath, reg.Middlewares[i].Metadata.Name, name) {
				sourcePath = reg.Middlewares[i].SourcePath
				break
			}
		}
	case KindService:
		for i := range reg.Services {
			if MatchesPluginName(reg.Services[i].SourcePath, reg.Services[i].Metadata.Name, name) {
				sourcePath = reg.Services[i].SourcePath
				break
			}
		}
	case KindTraefik:
		for i := range reg.Routes {
			if MatchesPluginName(reg.Routes[i].SourcePath, reg.Routes[i].Metadata.Name, name) {
				if strings.Contains(reg.Routes[i].SourcePath, "/traefik/") {
					sourcePath = reg.Routes[i].SourcePath
					break
				}
			}
		}
		if sourcePath == "" {
			for i := range reg.Middlewares {
				if MatchesPluginName(reg.Middlewares[i].SourcePath, reg.Middlewares[i].Metadata.Name, name) {
					if strings.Contains(reg.Middlewares[i].SourcePath, "/traefik/") {
						sourcePath = reg.Middlewares[i].SourcePath
						break
					}
				}
			}
		}
	}

	if sourcePath == "" {
		return fmt.Errorf("plugin %q (kind: %s) is not currently installed", name, kind)
	}

	// Security check: Only allow deleting files inside userPluginsDir
	absUserDir, err := filepath.Abs(userPluginsDir)
	if err != nil {
		return fmt.Errorf("failed to resolve user plugins directory path: %w", err)
	}

	absSourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to resolve plugin source path: %w", err)
	}

	// Allow development/local paths if we are running in local/dev directory
	// Normally SystemPluginsDir is "/opt/m3tal/plugins", and userPluginsDir is "/etc/m3tal/plugins"
	// But in development, they might use "deploy/plugins"
	isDeveloperPath := false
	if _, err := os.Stat("deploy/plugins"); err == nil {
		absDevDir, err := filepath.Abs("deploy/plugins")
		if err == nil && strings.HasPrefix(absSourcePath, absDevDir) {
			isDeveloperPath = true
		}
	}

	if !strings.HasPrefix(absSourcePath, absUserDir) && !isDeveloperPath {
		return fmt.Errorf("security violation: cannot uninstall system plugin at %s", sourcePath)
	}

	// Load and parse manifest first to run PreUninstall hook
	var parsedPlugin *Plugin
	if data, err := os.ReadFile(absSourcePath); err == nil {
		if p, parseErr := parseAnyPlugin(data, absSourcePath); parseErr == nil {
			parsedPlugin = p
			if p.Hooks != nil && p.Hooks.PreUninstall != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_ = ExecuteHook(ctx, p.Hooks.PreUninstall, p)
				cancel()
			}
		}
	}

	// Delete main manifest
	err = os.Remove(absSourcePath)
	if err != nil {
		return fmt.Errorf("failed to remove plugin manifest: %w", err)
	}

	// If Stack plugin, also clean up the compose file
	if kind == KindStack && composePathRel != "" && !filepath.IsAbs(composePathRel) {
		composeFile := filepath.Join(filepath.Dir(absSourcePath), composePathRel)
		if _, err := os.Stat(composeFile); err == nil {
			_ = os.Remove(composeFile)
		}
	}

	// Run PostUninstall hook
	if parsedPlugin != nil && parsedPlugin.Hooks != nil && parsedPlugin.Hooks.PostUninstall != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_ = ExecuteHook(ctx, parsedPlugin.Hooks.PostUninstall, parsedPlugin)
		cancel()
	}

	return nil
}
