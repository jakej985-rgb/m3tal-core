package api

import (
	"context"
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

	GlobalPluginManager.StartAutoDiscovery(context.Background(), 5*time.Second)
	defer GlobalPluginManager.StopAll()

	log.Printf("🚀 M3TAL Core API starting on :%s...\n", port)

	r := chi.NewRouter()

	r.Use(apiMiddleware.Recoverer)
	r.Use(apiMiddleware.RequestLogger)
	r.Use(middleware.RealIP)

	// Embedded React GUI
	subFS, err := fs.Sub(gui.DistFS, "dist")
	if err == nil {
		fileServer(r, "/gui", http.FS(subFS))

		assetsFS, err := fs.Sub(subFS, "assets")
		if err == nil {
			fileServer(r, "/assets", http.FS(assetsFS))
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
		name := chi.URLParam(r, "name")
		if name == "" {
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
				"./plugins",
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

		subPath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
		if subPath == "" {
			subPath = "index.html"
		}

		filePath := filepath.Join(uiDir, subPath)
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

	r.Get("/plugins/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		http.Redirect(w, r, "/plugins/"+name+"/", http.StatusMovedPermanently)
	})
	r.Get("/plugins/{name}/*", servePluginUI)

	// Public version metadata
	r.Get("/version", srv.GetVersion)
	r.Get("/api/v2/version", srv.GetVersion)

	pluginH := NewPluginHandlers(GlobalPluginManager)

	r.Route("/api/v2", func(r chi.Router) {
		r.Use(srv.chiAuthMiddleware)

		// API Keys
		r.Get("/auth/keys", srv.ListAPIKeys)
		r.Post("/auth/keys", srv.CreateAPIKey)
		r.Delete("/auth/keys/{id}", srv.DeleteAPIKey)

		// Plugins
		r.Get("/plugins", pluginH.ListPlugins)
		r.Get("/plugins/catalog", pluginH.GetPluginCatalog)
		r.Get("/plugins/tabs", pluginH.ListTabs)
		r.Get("/plugins/{name}/changelog", pluginH.GetPluginChangelog)
		r.Get("/plugins/{name}/download", pluginH.DownloadPluginDeb)
		r.Post("/plugins/{name}/install", pluginH.InstallPlugin)
		r.Post("/plugins/{name}/uninstall", pluginH.UninstallPlugin)
		r.Post("/plugins/{name}/start", pluginH.StartPlugin)
		r.Post("/plugins/{name}/stop", pluginH.StopPlugin)
		r.Get("/plugins/{name}/collect", pluginH.CollectPlugin)
		r.Post("/plugins/{name}/action", pluginH.ExecuteAction)
		r.Get("/plugins/{name}/settings", pluginH.GetPluginSettings)
		r.Post("/plugins/{name}/settings", pluginH.SavePluginSettings)

		// Settings & repositories
		r.Get("/settings", pluginH.GetAllSettings)
		r.Post("/settings/core", pluginH.SaveCoreSettings)
		r.Get("/repositories", pluginH.GetRepositories)
		r.Post("/repositories", pluginH.SaveRepository)
		r.Post("/repositories/refresh", pluginH.RefreshRepositories)
		r.Delete("/repositories/{id}", pluginH.DeleteRepository)

		// Host telemetry and system monitor
		r.Get("/system/metrics", GetMetricsHistory)
		r.Get("/system/health", srv.GetHealth)
		r.Get("/system/stats", srv.GetStats)
		r.Get("/system/processes", srv.GetProcesses)
		r.Post("/system/processes/{pid}/kill", srv.KillProcess)
		r.Get("/system/specs", srv.GetSystemSpecs)
		r.Get("/system/network", srv.GetSystemNetwork)

		// WebSocket events
		r.Get("/ws/events", srv.GetWSEvents)
	})

	return http.ListenAndServe(":"+port, r)
}

// chiAuthMiddleware is a chi-compatible auth middleware.
func (s *Server) chiAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.ValidateToken(s.ExtractToken(r)) {
			sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
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
