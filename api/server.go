package api

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	apiMiddleware "github.com/jakej985-rgb/m3tal-core/api/middleware"
	"github.com/jakej985-rgb/m3tal-core/core/state"
	"github.com/jakej985-rgb/m3tal-core/core/system"
	"github.com/jakej985-rgb/m3tal-core/gui"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	"github.com/jakej985-rgb/m3tal-core/pkg/plugins"
)

// GlobalPluginManager manages the external plugin supervisor.
var GlobalPluginManager = plugins.NewManager("v0.2.0")

func setupSighupHandler() {
	sigHupCh := make(chan os.Signal, 1)
	signal.Notify(sigHupCh, syscall.SIGHUP)
	go func() {
		for range sigHupCh {
			log.Println("🔄 Received SIGHUP: refreshing plugins...")
			_, _ = GlobalPluginManager.DiscoverAndAutoStart(context.Background())
		}
	}()
}

// RunServer automatically initializes the state store and starts the API daemon.
func RunServer(port string, token string) error {
	setupSighupHandler()
	dbPath := state.GetStatePath()
	db, err := state.Open(dbPath)
	if err != nil {
		log.Printf("⚠️  Could not open state database at %s: %v", dbPath, err)
		return StartServer(port, token)
	}
	defer db.Close()

	log.Printf("📦 State database: %s\n", dbPath)
	return StartServerWithStore(port, token, db)
}

// StartServer starts the API server on the specified port.
func StartServer(port string, token string) error {
	return StartServerWithStore(port, token, nil)
}

// StartServerWithStore starts the API server with an optional SQLite store.
func StartServerWithStore(port string, token string, db *state.Store) error {
	log.Printf("[api-daemon] StartServerWithStore starting on port %s (SQLite store active: %v)\n", port, db != nil)
	system.StartMetricsAggregator()
	srv := NewServerWithStore(token, db)

	if db != nil {
		// Plugins receive a dedicated scoped API key instead of the primary
		// full-access credential.
		plugins.SetAPIKeyProvider(func() string { return pluginRuntimeAPIKey(db) })
	}

	GlobalPluginManager.StartAutoDiscovery(context.Background(), 5*time.Second)
	defer GlobalPluginManager.StopAll()

	log.Printf("🚀 M3TAL Core API starting on :%s...\n", port)

	r := chi.NewRouter()

	r.Use(apiMiddleware.Recoverer)
	r.Use(apiMiddleware.RequestLogger)
	r.Use(middleware.RealIP)
	// Compress JSON API payloads — the dashboard polls several endpoints every
	// few seconds and responses (process lists, stats history, catalogs) are
	// highly compressible (~5-10x smaller on the wire).
	r.Use(middleware.Compress(5, "application/json"))

	// Embedded React GUI
	subFS, err := fs.Sub(gui.DistFS, "dist")
	if err == nil {
		fileServer(r, "/gui", http.FS(subFS))

		assetsFS, err := fs.Sub(subFS, "assets")
		if err == nil {
			// Vite emits content-hashed filenames (index-<hash>.js), so these
			// assets are safe to cache immutably — eliminates revalidation
			// requests on every page load.
			assetCache := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
					next.ServeHTTP(w, r)
				})
			}
			fileServer(r.With(assetCache), "/assets", http.FS(assetsFS))
		}

		redirectRoot := func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/gui/", http.StatusFound)
		}
		r.Get("/", redirectRoot)
		r.Head("/", redirectRoot)
	} else {
		log.Printf("⚠️ Failed to load embedded React GUI: %v", err)
	}

	servePluginUI := func(w http.ResponseWriter, r *http.Request) {
		name, err := sanitizePluginName(chi.URLParam(r, "name"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		candidates := []string{
			name,
			strings.TrimPrefix(name, "m3tal-"),
			"m3tal-" + strings.TrimPrefix(name, "m3tal-"),
		}

		var targetDir string
		if GlobalPluginManager != nil {
			for _, c := range candidates {
				if p := GlobalPluginManager.GetPlugin(c); p != nil && p.Manifest.Dir != "" {
					targetDir = p.Manifest.Dir
					break
				}
			}
		}

		if targetDir == "" {
			home, _ := os.UserHomeDir()
			baseDirs := []string{
				"/home/m3tal/apps/m3tal-plugins",
				"/usr/lib/m3tal/plugins",
				filepath.Join(home, ".config", "m3tal", "plugins"),
			}
			for _, base := range baseDirs {
				for _, c := range candidates {
					testPath := filepath.Join(base, c)
					if info, err := os.Stat(testPath); err == nil && info.IsDir() {
						targetDir = testPath
						break
					}
				}
				if targetDir != "" {
					break
				}
			}
		}

		if targetDir == "" {
			http.NotFound(w, r)
			return
		}

		uiDir := filepath.Join(targetDir, "ui", "dist")
		if _, err := os.Stat(uiDir); os.IsNotExist(err) {
			uiDir = filepath.Join(targetDir, "ui")
			if _, err := os.Stat(uiDir); os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
		}

		subPath := "index.html"
		if raw := strings.TrimPrefix(chi.URLParam(r, "*"), "/"); raw != "" {
			var err error
			subPath, err = validatePluginUISubPath(raw)
			if err != nil {
				http.NotFound(w, r)
				return
			}
		}

		filePath := filepath.Join(uiDir, subPath)
		if rel, err := filepath.Rel(uiDir, filePath); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			http.NotFound(w, r)
			return
		}
		info, err := os.Stat(filePath)
		if err == nil && info.IsDir() {
			filePath = filepath.Join(filePath, "index.html")
			_, err = os.Stat(filePath)
		}

		if err != nil {
			fallback := filepath.Join(uiDir, "index.html")
			if _, err2 := os.Stat(fallback); err2 == nil {
				http.ServeFile(w, r, fallback)
				return
			}
			http.NotFound(w, r)
			return
		}

		http.ServeFile(w, r, filePath)
	}

	// Plugin UI static assets require the same credential as the API. The
	// bundled GUI loads them in same-origin iframes; the Path=/ session
	// cookie (set by Authenticate on the first header-authenticated request)
	// covers document and subresource loads that cannot carry headers.
	pluginUI := r.With(srv.pluginUIAuth)
	pluginUI.Get("/plugins/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if _, err := sanitizePluginName(name); err != nil {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/plugins/"+name+"/", http.StatusMovedPermanently)
	})
	pluginUI.Get("/plugins/{name}/*", servePluginUI)

	// Public version metadata
	r.Get("/version", srv.GetVersion)
	r.Get("/api/v2/version", srv.GetVersion)

	pluginH := NewPluginHandlers(GlobalPluginManager)

	r.Route("/api/v2", func(r chi.Router) {
		r.Use(srv.Authenticate)

		// API Keys
		r.Get("/auth/keys", srv.requireScope(state.ScopeKeysManage, srv.ListAPIKeys))
		r.Post("/auth/keys", srv.requireScope(state.ScopeKeysManage, srv.CreateAPIKey))
		r.Delete("/auth/keys/{id}", srv.requireScope(state.ScopeKeysManage, srv.DeleteAPIKey))

		// Plugins
		r.Get("/plugins", srv.requireScope(state.ScopePluginsRead, pluginH.ListPlugins))
		r.Get("/plugins/catalog", srv.requireScope(state.ScopePluginsRead, pluginH.GetPluginCatalog))
		r.Get("/plugins/tabs", srv.requireScope(state.ScopePluginsRead, pluginH.ListTabs))
		r.Get("/plugins/{name}/changelog", srv.requireScope(state.ScopePluginsRead, pluginH.GetPluginChangelog))
		r.Get("/plugins/{name}/download", srv.requireScope(state.ScopePluginsRead, pluginH.DownloadPluginDeb))
		r.Post("/plugins/{name}/install", srv.requireScope(state.ScopePluginsWrite, pluginH.InstallPlugin))
		r.Post("/plugins/{name}/uninstall", srv.requireScope(state.ScopePluginsWrite, pluginH.UninstallPlugin))
		r.Post("/plugins/{name}/start", srv.requireScope(state.ScopePluginsWrite, pluginH.StartPlugin))
		r.Post("/plugins/{name}/stop", srv.requireScope(state.ScopePluginsWrite, pluginH.StopPlugin))
		r.Get("/plugins/{name}/collect", srv.requireScope(state.ScopePluginsRead, pluginH.CollectPlugin))
		r.Post("/plugins/{name}/action", srv.requireScope(state.ScopePluginsWrite, pluginH.ExecuteAction))
		r.Get("/plugins/{name}/settings", srv.requireScope(state.ScopePluginsRead, pluginH.GetPluginSettings))
		r.Post("/plugins/{name}/settings", srv.requireScope(state.ScopePluginsWrite, pluginH.SavePluginSettings))

		// Settings & repositories
		r.Get("/settings", srv.requireScope(state.ScopeSystemRead, pluginH.GetAllSettings))
		r.Post("/settings/core", srv.requireScope(state.ScopeSettingsWrite, pluginH.SaveCoreSettings))
		r.Get("/repositories", srv.requireScope(state.ScopeSystemRead, pluginH.GetRepositories))
		r.Post("/repositories", srv.requireScope(state.ScopeSettingsWrite, pluginH.SaveRepository))
		r.Post("/repositories/refresh", srv.requireScope(state.ScopeSettingsWrite, pluginH.RefreshRepositories))
		r.Delete("/repositories/{id}", srv.requireScope(state.ScopeSettingsWrite, pluginH.DeleteRepository))

		// Host telemetry and system monitor
		r.Get("/system/metrics", srv.requireScope(state.ScopeSystemRead, GetMetricsHistory))
		r.Get("/system/health", srv.requireScope(state.ScopeSystemRead, srv.GetHealth))
		r.Get("/system/stats", srv.requireScope(state.ScopeSystemRead, srv.GetStats))
		r.Get("/system/processes", srv.requireScope(state.ScopeSystemRead, srv.GetProcesses))
		r.Post("/system/processes/{pid}/kill", srv.requireScope(state.ScopeSystemWrite, srv.KillProcess))
		r.Get("/system/specs", srv.requireScope(state.ScopeSystemRead, srv.GetSystemSpecs))
		r.Get("/system/network", srv.requireScope(state.ScopeSystemRead, srv.GetSystemNetwork))

		// WebSocket events
		r.Get("/ws/events", srv.requireScope(state.ScopeSystemRead, srv.GetWSEvents))
	})

	httpSrv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	certFile := os.Getenv("M3TAL_TLS_CERT")
	keyFile := os.Getenv("M3TAL_TLS_KEY")
	if (certFile == "") != (keyFile == "") {
		return fmt.Errorf("M3TAL_TLS_CERT and M3TAL_TLS_KEY must be set together")
	}
	if certFile != "" {
		log.Printf("🔒 TLS enabled for API listener (cert: %s)", certFile)
		return httpSrv.ListenAndServeTLS(certFile, keyFile)
	}
	log.Printf("⚠️  API listener on :%s is plaintext HTTP — authentication material traverses the network unencrypted. Set M3TAL_TLS_CERT/M3TAL_TLS_KEY or front the API with a TLS-terminating proxy. See docs/SECURITY.md.", port)
	return httpSrv.ListenAndServe()
}

// GetMetricsHistory returns the sliding window history of system metrics.
// GET /api/v2/system/metrics
func GetMetricsHistory(w http.ResponseWriter, r *http.Request) {
	history := system.GlobalMetricsHistory.Get()
	typedHistory := make([]models.MetricsResponse, len(history))
	for i, h := range history {
		typedHistory[i] = models.MetricsResponse{
			CPUUsage:    h.CPUUsage,
			MemoryUsage: h.MemoryUsage,
			DiskUsage:   h.DiskUsage,
			Uptime:      h.Uptime,
			Hostname:    h.Hostname,
		}
	}
	sendSuccess(w, http.StatusOK, typedHistory, nil)
}

// fileServer conveniently sets up a http.FileServer for a chi router.
func fileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	handler := func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	}
	r.Get(path, handler)
	r.Head(path, handler)
}
