package api

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	apiMiddleware "github.com/jakej985-rgb/m3tal-core/api/middleware"
	"github.com/jakej985-rgb/m3tal-core/core/agents"
	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/events"
	"github.com/jakej985-rgb/m3tal-core/core/health"
	"github.com/jakej985-rgb/m3tal-core/core/queue"
	"github.com/jakej985-rgb/m3tal-core/core/state"
	"github.com/jakej985-rgb/m3tal-core/core/system"
	"github.com/jakej985-rgb/m3tal-core/gui"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
)

// GlobalQueueManager manages the background execution queue for the API.
var GlobalQueueManager *queue.Manager

// RunServer automatically initializes the state store and starts the API daemon.
func RunServer(port string, token string) error {
	dbPath := state.GetStatePath()
	db, err := state.Open(dbPath)
	if err != nil {
		log.Printf("⚠️  Could not open state database at %s: %v", dbPath, err)
		log.Println("⚠️  v2 engine endpoints will be disabled. Starting with v1 only.")
		return StartServer(port, token)
	}
	defer db.Close()

	log.Printf("📦 State database: %s\n", dbPath)
	return StartServerWithStore(port, token, db)
}

// StartServer starts the API server on the specified port.
// This is the legacy entrypoint that creates an in-memory server without a state.
func StartServer(port string, token string) error {
	return StartServerWithStore(port, token, nil)
}

// StartServerWithStore starts the API server with an optional SQLite store
// for the v2 engine endpoints.
func StartServerWithStore(port string, token string, db *state.Store) error {
	log.Printf("[api-daemon] StartServerWithStore starting on port %s (SQLite store active: %v)\n", port, db != nil)
	system.StartMetricsAggregator()
	srv := NewServer(token)

	// Initialize global queue manager
	GlobalQueueManager = queue.NewManager(2, 100)
	defer GlobalQueueManager.Close()

	// Start background Docker events forwarder
	go func() {
		mgr, err := docker.GetProvider()
		if err != nil {
			log.Printf("⚠️ Docker provider unavailable for forwarder: %v", err)
			return
		}

		ctx := context.Background()
		eventCh, err := mgr.SubscribeEvents(ctx)
		if err != nil {
			log.Printf("⚠️ Failed to subscribe to container events: %v", err)
			return
		}

		log.Println("🔌 Docker events forwarder started")
		for ev := range eventCh {
			var eventType string
			switch ev.Action {
			case "start":
				eventType = "container.started"
			case "stop", "die", "kill":
				eventType = "container.stopped"
			}

			if eventType != "" {
				events.GlobalEventBus.Publish(eventType, map[string]string{
					"container": ev.ContainerName,
				})
			}
		}
	}()

	log.Printf("🚀 M3TAL API Interface starting on :%s...\n", port)

	r := chi.NewRouter()

	// Global middleware
	r.Use(apiMiddleware.Recoverer)
	r.Use(apiMiddleware.RequestLogger)
	r.Use(middleware.RealIP)

	// Serve static files for the React GUI from embedded FS
	subFS, err := fs.Sub(gui.DistFS, "dist")
	if err == nil {
		fileServer(r, "/gui", http.FS(subFS))
		redirectRoot := func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/gui/", http.StatusFound)
		}
		r.Get("/", redirectRoot)
		r.Head("/", redirectRoot)
	} else {
		log.Printf("⚠️ Failed to load embedded React GUI: %v", err)
	}

	// Serve the system tray popup UI
	r.Get("/tray", srv.GetTrayPage)

	// ─── v1 Legacy Endpoints (backward compatible) ───
	r.Group(func(r chi.Router) {
		r.Use(srv.chiAuthMiddleware)

		// Original endpoints
		r.Get("/health", srv.GetHealth)
		r.Get("/services", srv.GetServices)
		r.Get("/stack", srv.GetStack)
		r.Get("/config", srv.GetConfig)

		// /api/* prefixed duplicates
		r.Get("/api/health", srv.GetHealth)
		r.Get("/api/services", srv.GetServices)
		r.Get("/api/stack", srv.GetStack)
		r.Get("/api/config", srv.GetConfig)
	})

	// ─── v2 Engine Endpoints ───
	if db != nil {
		routeH := &RouteHandlers{Store: db}
		stackH := &StackHandlers{Store: db}
		mwH := &MiddlewareHandlers{Store: db}
		pluginH := NewPluginHandlers(db)
		composeH := &ComposeHandlers{Store: db}
		vpnH := NewVPNHandlers()
		proxyH := NewProxyHandlers(db)

		r.Route("/api/v2", func(r chi.Router) {
			r.Use(srv.chiAuthMiddleware)

			// Routes
			r.Get("/routes", routeH.ListRoutes)
			r.Post("/routes", routeH.CreateRoute)
			r.Delete("/routes/{id}", routeH.DeleteRoute)
			r.Post("/routes/preview", routeH.GetRouteLabels)

			// Proxy
			r.Get("/proxy/discover", proxyH.DiscoverServices)
			r.Post("/proxy/expose", proxyH.ExposeService)
			r.Post("/proxy/unexpose", proxyH.UnexposeService)
			r.Post("/proxy/secure", proxyH.ConfigureSSL)

			// Stacks
			r.Get("/stacks", stackH.ListStacks)
			r.Get("/stacks/{name}", stackH.GetStackByName)
			r.Post("/stacks/load", stackH.LoadStack)
			r.Post("/stacks/deploy", stackH.DeployStack)
			r.Post("/stacks/scan", stackH.ScanStacks)
			r.Post("/stacks/{name}/up", stackH.DeployStackByName)
			r.Post("/stacks/{name}/down", stackH.StopStackByName)
			r.Post("/stacks/pull", stackH.PullStacks)
			r.Get("/stacks/{name}/logs", stackH.GetStackLogs)

			// Services (pass-through to Docker provider)
			r.Get("/services", srv.GetServices)
			r.Get("/containers/{name}/logs", srv.GetContainerLogs)

			// Middleware
			r.Get("/middleware", mwH.ListMiddleware)
			r.Post("/middleware", mwH.CreateMiddleware)

			// Plugins
			r.Get("/plugins", pluginH.ListPlugins)
			r.Get("/plugins/catalog", pluginH.ListCatalog)
			r.Get("/plugins/{kind}", pluginH.ListPluginsByKind)
			r.Post("/plugins/reload", pluginH.Reload)
			r.Post("/plugins/enable", pluginH.Enable)
			r.Post("/plugins/disable", pluginH.Disable)
			r.Post("/plugins/sync", pluginH.Sync)
			r.Post("/plugins/install", pluginH.Install)
			r.Post("/plugins/uninstall", pluginH.Uninstall)
			r.Post("/plugins/validate", pluginH.Validate)
			r.Post("/plugins/match", pluginH.Match)
			r.Post("/plugins/install-stack", pluginH.InstallStackPlugin)

			// Compose
			r.Post("/compose/validate", composeH.ValidateCompose)
			r.Post("/compose/fix", composeH.FixCompose)
			r.Get("/compose/templates", composeH.ListTemplates)
			r.Post("/compose/generate", composeH.GenerateTemplate)
			r.Post("/compose/save", composeH.SaveCompose)

			// VPN
			r.Get("/vpn/status", vpnH.GetStatus)
			r.Post("/vpn/control", vpnH.ControlVPN)
			r.Post("/vpn/region", vpnH.SwitchRegion)
			r.Post("/vpn/sync-port", vpnH.SyncPort)
			r.Get("/vpn/check-leak", vpnH.CheckLeak)

			// Containers
			r.Post("/containers/{name}/{action}", srv.HandleContainerActionV2)

			// System Observability
			r.Get("/system/metrics", GetMetricsHistory)
			r.Get("/system/health", GetSystemHealth(db))
			r.Get("/system/stats", srv.GetStats)
			r.Get("/tray/stats", srv.GetTrayStats)
			r.Get("/tray/containers", srv.GetTrayContainers)
			r.Get("/system/doctor", srv.GetDoctor)
			r.Get("/doctor/containers", srv.GetDoctorContainers)
			r.Get("/doctor/mounts", srv.GetDoctorMounts)
			r.Get("/doctor/ports", srv.GetDoctorPorts)
			r.Post("/doctor/fix", srv.HandleDoctorFix)
			r.Get("/doctor/report", srv.GetDoctorReport)
			r.Post("/auth/dashpass", srv.HandleDashpass)
			r.Post("/stacks/init", srv.HandleInit)
			r.Get("/config/cloudflared", srv.GetCloudflaredConfig)
			r.Post("/config/cloudflared", srv.UpdateCloudflaredConfig)
			r.Get("/config/env", srv.GetEnvConfig)
			r.Post("/config/env", srv.UpdateEnvConfig)

			// System Services, Storage, Cron, and Updates
			r.Get("/system/services", srv.GetSystemServices)
			r.Post("/system/services/{name}/{action}", srv.ControlSystemService)
			r.Get("/system/storage", srv.GetSystemStorage)
			r.Get("/system/cron", srv.GetCronJobs)
			r.Get("/system/updates", srv.GetSystemUpdates)
			r.Post("/system/updates/apply", srv.ApplySystemUpdates)

			// Docker Controls
			r.Get("/docker/images", srv.GetDockerImages)
			r.Post("/docker/images/prune", srv.PruneDockerImages)
			r.Get("/docker/volumes", srv.GetDockerVolumes)
			r.Post("/docker/volumes/prune", srv.PruneDockerVolumes)
			r.Get("/docker/networks", srv.GetDockerNetworks)
			r.Post("/docker/networks/prune", srv.PruneDockerNetworks)

			// WebSockets
			r.Get("/ws/events", srv.GetWSEvents)
			r.Get("/ws/logs/{name}", srv.GetWSLogs)

			// Queue/AI
			r.Get("/queue", ListQueue)
			r.Get("/ai/queue", ListQueue)
			r.Post("/queue/cancel", CancelQueueJob)
			r.Post("/queue/{id}/cancel", CancelQueueJob)
			r.Get("/ai/models", ListAIModels)
			r.Post("/ai/run", RunAIInference)
			r.Post("/system/reconcile", HandleReconcile)
		})

		log.Println("✅ v2 engine endpoints enabled (SQLite store active)")
	} else {
		// Plugin endpoints work without a store
		pluginH := NewPluginHandlers(nil)
		composeH := &ComposeHandlers{Store: nil}
		vpnH := NewVPNHandlers()
		proxyH := NewProxyHandlers(nil)
		r.Route("/api/v2", func(r chi.Router) {
			r.Use(srv.chiAuthMiddleware)
			// Plugins
			r.Get("/plugins", pluginH.ListPlugins)
			r.Get("/plugins/catalog", pluginH.ListCatalog)
			r.Get("/plugins/{kind}", pluginH.ListPluginsByKind)
			r.Post("/plugins/reload", pluginH.Reload)
			r.Post("/plugins/enable", pluginH.Enable)
			r.Post("/plugins/disable", pluginH.Disable)
			r.Post("/plugins/sync", pluginH.Sync)
			r.Post("/plugins/install", pluginH.Install)
			r.Post("/plugins/uninstall", pluginH.Uninstall)
			r.Post("/plugins/validate", pluginH.Validate)
			r.Post("/plugins/match", pluginH.Match)
			r.Post("/plugins/install-stack", pluginH.InstallStackPlugin)

			// Proxy
			r.Get("/proxy/discover", proxyH.DiscoverServices)
			r.Post("/proxy/expose", proxyH.ExposeService)
			r.Post("/proxy/unexpose", proxyH.UnexposeService)
			r.Post("/proxy/secure", proxyH.ConfigureSSL)

			// Compose
			r.Post("/compose/validate", composeH.ValidateCompose)
			r.Post("/compose/fix", composeH.FixCompose)
			r.Get("/compose/templates", composeH.ListTemplates)
			r.Post("/compose/generate", composeH.GenerateTemplate)
			r.Post("/compose/save", composeH.SaveCompose)

			// VPN
			r.Get("/vpn/status", vpnH.GetStatus)
			r.Post("/vpn/control", vpnH.ControlVPN)
			r.Post("/vpn/region", vpnH.SwitchRegion)
			r.Post("/vpn/sync-port", vpnH.SyncPort)
			r.Get("/vpn/check-leak", vpnH.CheckLeak)

			// Containers
			r.Post("/containers/{name}/{action}", srv.HandleContainerActionV2)

			// System Observability
			r.Get("/system/metrics", GetMetricsHistory)
			r.Get("/system/health", GetSystemHealth(nil))
			r.Get("/system/stats", srv.GetStats)
			r.Get("/tray/stats", srv.GetTrayStats)
			r.Get("/tray/containers", srv.GetTrayContainers)
			r.Get("/system/doctor", srv.GetDoctor)
			r.Get("/doctor/containers", srv.GetDoctorContainers)
			r.Get("/doctor/mounts", srv.GetDoctorMounts)
			r.Get("/doctor/ports", srv.GetDoctorPorts)
			r.Post("/doctor/fix", srv.HandleDoctorFix)
			r.Get("/doctor/report", srv.GetDoctorReport)
			r.Post("/auth/dashpass", srv.HandleDashpass)
			r.Post("/stacks/init", srv.HandleInit)
			r.Get("/config/cloudflared", srv.GetCloudflaredConfig)
			r.Post("/config/cloudflared", srv.UpdateCloudflaredConfig)
			r.Get("/config/env", srv.GetEnvConfig)
			r.Post("/config/env", srv.UpdateEnvConfig)

			// System Services, Storage, Cron, and Updates
			r.Get("/system/services", srv.GetSystemServices)
			r.Post("/system/services/{name}/{action}", srv.ControlSystemService)
			r.Get("/system/storage", srv.GetSystemStorage)
			r.Get("/system/cron", srv.GetCronJobs)
			r.Get("/system/updates", srv.GetSystemUpdates)
			r.Post("/system/updates/apply", srv.ApplySystemUpdates)

			// Docker Controls
			r.Get("/docker/images", srv.GetDockerImages)
			r.Post("/docker/images/prune", srv.PruneDockerImages)
			r.Get("/docker/volumes", srv.GetDockerVolumes)
			r.Post("/docker/volumes/prune", srv.PruneDockerVolumes)
			r.Get("/docker/networks", srv.GetDockerNetworks)
			r.Post("/docker/networks/prune", srv.PruneDockerNetworks)

			// WebSockets
			r.Get("/ws/events", srv.GetWSEvents)
			r.Get("/ws/logs/{name}", srv.GetWSLogs)

			// Queue/AI
			r.Get("/queue", ListQueue)
			r.Get("/ai/queue", ListQueue)
			r.Post("/queue/cancel", CancelQueueJob)
			r.Post("/queue/{id}/cancel", CancelQueueJob)
			r.Get("/ai/models", ListAIModels)
			r.Post("/ai/run", RunAIInference)
			r.Post("/system/reconcile", HandleReconcile)
		})

		log.Println("⚠️  v2 engine endpoints disabled (no store configured), plugin endpoints still available")
	}

	return http.ListenAndServe(":"+port, r)
}

// chiAuthMiddleware is a chi-compatible version of AuthMiddleware.
func (s *Server) chiAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-API-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token != s.APIToken {
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

func GetSystemHealth(db *state.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reg := health.UpdateAndSaveHealthRegistry()
		status := http.StatusOK
		if reg.System.Status == "🔴" {
			status = http.StatusServiceUnavailable
		}

		componentsMap := map[string]string{
			"system": reg.System.Status,
			"docker": reg.Docker.Status,
			"agents": reg.Agents.Status,
			"disk":   reg.Disk.Status,
		}
		if db != nil {
			if err := db.Ping(); err != nil {
				componentsMap["database"] = "🔴"
			} else {
				componentsMap["database"] = "🟢"
			}
		}

		details := map[string]string{
			"last_seen_healthy": reg.System.LastSeenHealthy,
			"last_failure":      reg.System.LastFailure,
			"docker_running":    strconv.Itoa(reg.Docker.RunningContainers),
			"docker_total":      strconv.Itoa(reg.Docker.TotalContainers),
			"disk_used_percent": fmt.Sprintf("%.1f", reg.Disk.UsedPercent),
		}

		sendSuccess(w, status, models.Status{
			Status:     reg.System.Status,
			Components: componentsMap,
			Details:    details,
		}, nil)
	}
}

// fileServer conveniently sets up a http.FileServer for a chi router.
func fileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
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

// HandleReconcile triggers the state reconciliation loop and returns the actions taken.
// POST /api/v2/system/reconcile
func HandleReconcile(w http.ResponseWriter, r *http.Request) {
	actions, err := agents.RunReconcile()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "RECONCILE_FAILED", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, map[string]any{
		"reconciled": true,
		"actions":    actions,
	}, nil)
}
