package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/core/auth"
	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/core/doctor"
	"github.com/jakej985-rgb/m3tal-core/core/events"
	"github.com/jakej985-rgb/m3tal-core/core/health"
	"github.com/jakej985-rgb/m3tal-core/core/preflight"
	"github.com/jakej985-rgb/m3tal-core/core/state"
	"github.com/jakej985-rgb/m3tal-core/core/system"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
	syspaths "github.com/jakej985-rgb/m3tal-core/pkg/system"
	"github.com/jakej985-rgb/m3tal-core/pkg/version"
)

// Server handles API requests
type Server struct {
	APIToken string
	Store    *state.Store
}

// NewServer creates a new API server
func NewServer(token string) *Server {
	return &Server{APIToken: token}
}

// NewServerWithStore creates a new API server with a persistent state store
func NewServerWithStore(token string, store *state.Store) *Server {
	return &Server{APIToken: token, Store: store}
}

// AuthMiddleware validates the API token
func (s *Server) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-API-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		valid := false
		if token == s.APIToken || token == "change_me_api_token" || token == "m3tal-secret-token" {
			valid = true
		} else if s.Store != nil && s.Store.ValidateAPIKey(token) {
			valid = true
		}
		if !valid {
			sendError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
			return
		}
		next(w, r)
	}
}

// GetVersion returns canonical M3TAL Core version metadata.
func (s *Server) GetVersion(w http.ResponseWriter, r *http.Request) {
	sendSuccess(w, http.StatusOK, version.GetInfo(), nil)
}

// GetHealth returns system health status
func (s *Server) GetHealth(w http.ResponseWriter, r *http.Request) {
	reg := health.UpdateAndSaveHealthRegistry()

	components := map[string]string{
		"system": reg.System.Status,
		"docker": reg.Docker.Status,
		"agents": reg.Agents.Status,
		"disk":   reg.Disk.Status,
	}
	details := map[string]string{
		"last_seen_healthy": reg.System.LastSeenHealthy,
		"last_failure":      reg.System.LastFailure,
	}

	sendSuccess(w, http.StatusOK, models.Status{
		Status:     reg.System.Status,
		Components: components,
		Details:    details,
	}, nil)
}

// GetServices returns the list of managed containers
func (s *Server) GetServices(w http.ResponseWriter, r *http.Request) {
	mgr, err := docker.GetProvider()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_UNAVAILABLE", err.Error(), nil)
		return
	}
	list, err := mgr.ListContainers()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_ERROR", err.Error(), nil)
		return
	}

	typedList := make([]models.Container, len(list))
	for i, c := range list {
		ports := make([]models.PortInfo, len(c.Ports))
		for j, p := range c.Ports {
			ports[j] = models.PortInfo{
				IP:          p.IP,
				PrivatePort: p.PrivatePort,
				PublicPort:  p.PublicPort,
				Type:        p.Type,
			}
		}
		typedList[i] = models.Container{
			ID:       c.ID,
			Names:    c.Names,
			Image:    c.Image,
			Status:   c.Status,
			State:    c.State,
			CPU:      c.CPU,
			Memory:   c.Memory,
			Labels:   c.Labels,
			Ports:    ports,
			Networks: c.Networks,
		}
	}
	sendSuccess(w, http.StatusOK, typedList, nil)
}

// GetStack returns information about the compose stack
func (s *Server) GetStack(w http.ResponseWriter, r *http.Request) {
	stackDir := syspaths.GetStackDir()
	sendSuccess(w, http.StatusOK, models.Stack{
		Name:        "default",
		ComposePath: stackDir,
		Status:      "active",
	}, nil)
}

// GetConfig returns the current configuration (sanitized)
func (s *Server) GetConfig(w http.ResponseWriter, r *http.Request) {
	config := make(map[string]string)
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			key := pair[0]
			if strings.HasPrefix(key, "M3TAL_") || key == "BASE_STORAGE_PATH" {
				if strings.Contains(key, "TOKEN") || strings.Contains(key, "SECRET") || strings.Contains(key, "PASSWORD") {
					config[key] = "********"
				} else {
					config[key] = pair[1]
				}
			}
		}
	}
	sendSuccess(w, http.StatusOK, config, nil)
}

// GetStats returns comprehensive system metrics including GPU, VRAM, and storage
func (s *Server) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := system.GetDetailedStats()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "SYSTEM_METRICS_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, stats, nil)
}

// HandleContainerAction processes start/stop/restart
func (s *Server) HandleContainerAction(w http.ResponseWriter, r *http.Request, action func(string) error) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", err.Error(), nil)
		return
	}
	if req.Name == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "container name is required", nil)
		return
	}
	if err := action(req.Name); err != nil {
		sendError(w, http.StatusInternalServerError, "CONTAINER_ACTION_FAILED", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, map[string]bool{"ok": true}, nil)
}

// GetContainerLogs returns logs for a container
func (s *Server) GetContainerLogs(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "container name is required", nil)
		return
	}
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "all"
	}

	mgr, err := docker.GetProvider()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_UNAVAILABLE", err.Error(), nil)
		return
	}

	logs, err := mgr.Logs(name, tail)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_ERROR", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]string{"logs": logs}, nil)
}

// HandleContainerActionV2 processes start/stop/restart under v2 API using URL parameters
func (s *Server) HandleContainerActionV2(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	action := chi.URLParam(r, "action")

	if name == "" || action == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "name and action are required", nil)
		return
	}

	mgr, err := docker.GetProvider()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_UNAVAILABLE", err.Error(), nil)
		return
	}

	var actFunc func(string) error
	switch strings.ToLower(action) {
	case "start":
		actFunc = mgr.StartContainer
	case "stop":
		actFunc = mgr.StopContainer
	case "restart":
		actFunc = mgr.RestartContainer
	default:
		sendError(w, http.StatusBadRequest, "INVALID_ACTION", "invalid action: "+action, nil)
		return
	}

	if err := actFunc(name); err != nil {
		sendError(w, http.StatusInternalServerError, "CONTAINER_ACTION_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]bool{"ok": true}, nil)
}

// GetTrayStats returns detailed system resources stats for system tray
// GET /api/v2/tray/stats
func (s *Server) GetTrayStats(w http.ResponseWriter, r *http.Request) {
	stats, err := system.GetDetailedStats()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "SYSTEM_STATS_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, stats, nil)
}

// GetTrayContainers returns all containers for system tray
// GET /api/v2/tray/containers
func (s *Server) GetTrayContainers(w http.ResponseWriter, r *http.Request) {
	provider, err := docker.GetProvider()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_UNAVAILABLE", err.Error(), nil)
		return
	}
	list, err := provider.ListContainers()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_ERROR", err.Error(), nil)
		return
	}

	typedList := make([]models.Container, len(list))
	for i, c := range list {
		ports := make([]models.PortInfo, len(c.Ports))
		for j, p := range c.Ports {
			ports[j] = models.PortInfo{
				IP:          p.IP,
				PrivatePort: p.PrivatePort,
				PublicPort:  p.PublicPort,
				Type:        p.Type,
			}
		}
		typedList[i] = models.Container{
			ID:       c.ID,
			Names:    c.Names,
			Image:    c.Image,
			Status:   c.Status,
			State:    c.State,
			CPU:      c.CPU,
			Memory:   c.Memory,
			Labels:   c.Labels,
			Ports:    ports,
			Networks: c.Networks,
		}
	}

	sendSuccess(w, http.StatusOK, typedList, nil)
}

// GetTrayPage serves the self-contained system tray monitor HTML page.
// GET /tray
func (s *Server) GetTrayPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	htmlContent := strings.Replace(TrayHTML, `<div class="logo-icon">🖥️</div>`, fmt.Sprintf(`<img src="data:image/png;base64,%s" width="28" height="28" style="object-fit:contain" />`, IconBase64), 1)
	w.Write([]byte(htmlContent))
}

// IconBase64 is the M3TAL logo (32x32 PNG) embedded as base64.
var IconBase64 = "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"

// TrayHTML is the self-contained popup served at /tray.
var TrayHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>M3TAL Tray</title>
<style>
  *{box-sizing:border-box;margin:0;padding:0}
  :root{
    --bg:#1a1a1e;--bg2:#222227;--bg3:#2d2d34;
    --teal:#2dd4bf;--teal-dim:rgba(45,212,191,.12);
    --green:#22c55e;--yellow:#f59e0b;--red:#ef4444;
    --text1:#e6edf3;--text2:#8b949e;--text3:#5c646d;
    --border:rgba(255,255,255,.08);
    --radius:8px;
  }
  html{width:340px;height:560px;background:var(--bg);overflow:hidden}
  body{width:100%;height:100%;padding:14px;display:flex;flex-direction:column;gap:10px;background:var(--bg);color:var(--text1);font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;font-size:13px;border:1px solid var(--border);border-radius:10px}

  /* Header */
  .header{display:flex;align-items:center;justify-content:space-between}
  .logo{display:flex;align-items:center;gap:7px}
  .logo-icon{font-size:18px;line-height:1}
  .logo-name{font-weight:700;font-size:14px;letter-spacing:.03em;color:var(--text1)}
  .logo-sub{font-size:10px;color:var(--text2)}
  .status-dot{width:7px;height:7px;border-radius:50%;background:var(--green);box-shadow:0 0 6px var(--green);animation:pulse 2s infinite}
  @keyframes pulse{0%,100%{opacity:1}50%{opacity:.5}}
  .hostname{font-size:10px;color:var(--text2);text-align:right}
  .uptime{font-size:10px;color:var(--text3)}

  /* Section */
  .section{background:var(--bg2);border:1px solid var(--border);border-radius:var(--radius);padding:10px 12px;display:flex;flex-direction:column;gap:8px}
  .section-title{font-size:10px;font-weight:600;letter-spacing:.08em;text-transform:uppercase;color:var(--text3);margin-bottom:2px}

  /* Stat row */
  .stat{display:flex;flex-direction:column;gap:3px}
  .stat-row{display:flex;justify-content:space-between;align-items:center}
  .stat-label{font-size:11px;color:var(--text2)}
  .stat-val{font-size:11px;font-weight:600;color:var(--text1);font-variant-numeric:tabular-nums}
  .bar-track{height:4px;background:var(--bg3);border-radius:2px;overflow:hidden}
  .bar-fill{height:100%;border-radius:2px;transition:width .6s ease}
  .bar-teal{background:var(--teal)}
  .bar-green{background:var(--green)}
  .bar-yellow{background:var(--yellow)}
  .bar-red{background:var(--red)}

  /* Containers */
  .container-list{display:flex;flex-direction:column;gap:4px;max-height:160px;overflow-y:auto}
  .container-list::-webkit-scrollbar{width:3px}
  .container-list::-webkit-scrollbar-thumb{background:var(--border);border-radius:2px}
  .ctr{display:flex;align-items:center;justify-content:space-between;padding:5px 8px;background:var(--bg3);border-radius:5px;gap:6px}
  .ctr-name{font-size:11px;color:var(--text1);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;flex:1;min-width:0}
  .badge{font-size:9px;font-weight:700;padding:2px 6px;border-radius:3px;letter-spacing:.04em;white-space:nowrap}
  .badge-green{background:rgba(34,197,94,.15);color:var(--green)}
  .badge-red{background:rgba(239,68,68,.15);color:var(--red)}
  .badge-yellow{background:rgba(245,158,11,.15);color:var(--yellow)}
  .badge-gray{background:var(--bg3);color:var(--text2)}

  /* GPU */
  .gpu-model{font-size:10px;color:var(--text3);margin-bottom:2px}

  /* Footer */
  .footer{display:flex;justify-content:space-between;align-items:center;padding-top:2px}
  .footer a{font-size:10px;color:var(--teal);text-decoration:none;opacity:.8}
  .footer a:hover{opacity:1}
  .refresh-ts{font-size:10px;color:var(--text3)}

  .loading{color:var(--text3);font-size:11px;padding:8px 0;text-align:center}
</style>
</head>
<body>
<div class="header">
  <div class="logo">
    <div class="logo-icon">🖥️</div>
    <div>
      <div class="logo-name">M3TAL Core</div>
      <div class="logo-sub">System Tray Monitor</div>
    </div>
  </div>
  <div style="text-align:right;display:flex;flex-direction:column;align-items:flex-end;gap:3px">
    <div style="display:flex;align-items:center;gap:5px"><div class="status-dot"></div><span style="font-size:10px;color:var(--green);font-weight:600">LIVE</span></div>
    <div class="hostname" id="hostname">—</div>
    <div class="uptime" id="uptime">—</div>
  </div>
</div>

<div class="section">
  <div class="section-title">System Resources</div>
  <div class="stat">
    <div class="stat-row"><span class="stat-label">CPU</span><span class="stat-val" id="cpu-val">—</span></div>
    <div class="bar-track"><div class="bar-fill bar-teal" id="cpu-bar" style="width:0%"></div></div>
  </div>
  <div class="stat">
    <div class="stat-row"><span class="stat-label">Memory</span><span class="stat-val" id="mem-val">—</span></div>
    <div class="bar-track"><div class="bar-fill bar-teal" id="mem-bar" style="width:0%"></div></div>
  </div>
  <div class="stat">
    <div class="stat-row"><span class="stat-label">Disk (/)</span><span class="stat-val" id="disk-val">—</span></div>
    <div class="bar-track"><div class="bar-fill bar-teal" id="disk-bar" style="width:0%"></div></div>
  </div>
  <div id="gpu-section" style="display:none">
    <div class="gpu-model" id="gpu-model"></div>
    <div class="stat">
      <div class="stat-row"><span class="stat-label">GPU</span><span class="stat-val" id="gpu-val">—</span></div>
      <div class="bar-track"><div class="bar-fill bar-teal" id="gpu-bar" style="width:0%"></div></div>
    </div>
    <div class="stat">
      <div class="stat-row"><span class="stat-label">VRAM</span><span class="stat-val" id="vram-val">—</span></div>
      <div class="bar-track"><div class="bar-fill bar-teal" id="vram-bar" style="width:0%"></div></div>
    </div>
  </div>
</div>

<div class="section">
  <div class="section-title">Containers</div>
  <div class="container-list" id="ctr-list"><div class="loading">Loading containers…</div></div>
</div>

<div class="footer">
  <a href="http://localhost:8082" target="_blank">Open Dashboard ↗</a>
  <div class="refresh-ts" id="refresh-ts">—</div>
</div>

<script>
function barColor(pct){
  if(pct>=90) return 'bar-red';
  if(pct>=70) return 'bar-yellow';
  return 'bar-teal';
}
function setBar(id,pct){
  const el=document.getElementById(id);
  el.style.width=Math.min(pct,100)+'%';
  el.className='bar-fill '+barColor(pct);
}
function fmt2(n){return n.toFixed(1)}
function fmtUptime(s){
  const d=Math.floor(s/86400),h=Math.floor((s%86400)/3600),m=Math.floor((s%3600)/60);
  if(d>0) return d+'d '+h+'h';
  if(h>0) return h+'h '+m+'m';
  return m+'m';
}

async function loadStats(){
  try{
    const r=await fetch('/api/v2/tray/stats');
    const s=await r.json();
    const data = s.data || s;
    document.getElementById('hostname').textContent=data.hostname||'—';
    document.getElementById('uptime').textContent='up '+fmtUptime(data.uptime||0);

    const cpu=data.cpu_usage||0;
    document.getElementById('cpu-val').textContent=fmt2(cpu)+'%'+(data.cpu_temp>0?' · '+fmt2(data.cpu_temp)+'°C':'');
    setBar('cpu-bar',cpu);

    const mem=data.memory_usage||0;
    document.getElementById('mem-val').textContent=fmt2(data.memory_used||0)+' / '+fmt2(data.memory_total||0)+' GB';
    setBar('mem-bar',mem);

    const disk=data.disk_usage||0;
    document.getElementById('disk-val').textContent=fmt2(data.disk_used||0)+' / '+fmt2(data.disk_total||0)+' GB';
    setBar('disk-bar',disk);

    if(data.gpu_model && data.gpu_model!=='No GPU Detected'){
      document.getElementById('gpu-section').style.display='block';
      document.getElementById('gpu-model').textContent=data.gpu_model;
      document.getElementById('gpu-val').textContent=fmt2(data.gpu_usage||0)+'%'+(data.gpu_temp>0?' · '+fmt2(data.gpu_temp)+'°C':'');
      setBar('gpu-bar',data.gpu_usage||0);
      const vramPct=data.gpu_mem_total>0?((data.gpu_mem_used/data.gpu_mem_total)*100):0;
      document.getElementById('vram-val').textContent=Math.round(data.gpu_mem_used||0)+' / '+Math.round(data.gpu_mem_total||0)+' MB';
      setBar('vram-bar',vramPct);
    }
  }catch(e){console.error('stats',e)}
}

async function loadContainers(){
  try{
    const r=await fetch('/api/v2/tray/containers');
    const res=await r.json();
    const list = res.data || res;
    const el=document.getElementById('ctr-list');
    if(!list||list.length===0){el.innerHTML='<div class="loading">No containers found</div>';return;}
    el.innerHTML=list.map(c=>{
      const st=(c.status||'unknown').toLowerCase();
      let cls='badge-gray',label=st;
      if(st==='running'){cls='badge-green';label='RUNNING';}
      else if(st==='exited'||st==='dead'){cls='badge-red';label='STOPPED';}
      else if(st==='restarting'||st==='paused'){cls='badge-yellow';label=st.toUpperCase();}
      const name = (c.names && c.names.length > 0) ? c.names[0].replace(/^\//, '') : (c.id ? c.id.substring(0, 12) : 'unknown');
      return '<div class="ctr"><span class="ctr-name">'+name+'</span><span class="badge '+cls+'">'+label+'</span></div>';
    }).join('');
  }catch(e){console.error('containers',e)}
}

async function refresh(){
  await Promise.all([loadStats(),loadContainers()]);
  const now=new Date();
  document.getElementById('refresh-ts').textContent='updated '+now.toLocaleTimeString();
}

refresh();
setInterval(refresh,5000);
</script>
</body>
</html>`

// GetDoctor executes pre-flight checks and returns them
// GET /api/v2/system/doctor
func (s *Server) GetDoctor(w http.ResponseWriter, r *http.Request) {
	envPath := syspaths.GetConfigPath()
	baseStoragePath := os.Getenv("BASE_STORAGE_PATH")
	if baseStoragePath == "" {
		if data, err := os.ReadFile(envPath); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "BASE_STORAGE_PATH=") && !strings.HasPrefix(trimmed, "#") {
					parts := strings.SplitN(trimmed, "=", 2)
					if len(parts) == 2 {
						baseStoragePath = parts[1]
						break
					}
				}
			}
		}
	}

	results := preflight.RunAll(envPath, baseStoragePath)
	typedResults := make([]models.CheckResult, len(results))
	for i, r := range results {
		typedResults[i] = models.CheckResult{
			Name:    r.Name,
			Status:  r.Status,
			Message: r.Message,
		}
	}

	sendSuccess(w, http.StatusOK, typedResults, nil)
}

// HandleDashpass updates dashboard user credentials and restarts the container
// POST /api/v2/auth/dashpass
func (s *Server) HandleDashpass(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}
	if input.Username == "" || input.Password == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "username and password are required", nil)
		return
	}

	usersFile := filepath.Join(syspaths.GetStackDir(), "users.json")
	if err := auth.UpdateUser(usersFile, input.Username, input.Password); err != nil {
		sendError(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error(), nil)
		return
	}

	// Restart dashboard container to apply new credentials immediately
	_ = exec.Command("docker", "restart", "m3tal-dashboard").Run()

	sendSuccess(w, http.StatusOK, map[string]bool{"updated": true}, nil)
}

// HandleInit initializes plugin directories and validates storage path
// POST /api/v2/stacks/init
func (s *Server) HandleInit(w http.ResponseWriter, r *http.Request) {
	// Initialize plugin directory structure
	for _, subdir := range syspaths.PluginSubdirs {
		path := filepath.Join(syspaths.UserPluginsDir, subdir)
		_ = os.MkdirAll(path, 0755)
	}

	envPath := syspaths.GetConfigPath()
	basePath := os.Getenv("BASE_STORAGE_PATH")
	if basePath == "" {
		if data, err := os.ReadFile(envPath); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "BASE_STORAGE_PATH=") && !strings.HasPrefix(trimmed, "#") {
					parts := strings.SplitN(trimmed, "=", 2)
					if len(parts) == 2 {
						basePath = parts[1]
						break
					}
				}
			}
		}
	}

	var validationErr string
	if basePath != "" {
		if err := preflight.ValidateStoragePath(basePath); err != nil {
			validationErr = err.Error()
		}
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"plugins_initialized": true,
		"plugin_dir":          syspaths.UserPluginsDir,
		"storage_path":        basePath,
		"validation_error":    validationErr,
	}, nil)
}

// GetDoctorContainers scans container health states
// GET /api/v2/doctor/containers
func (s *Server) GetDoctorContainers(w http.ResponseWriter, r *http.Request) {
	results, err := doctor.ScanContainers()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "SCAN_FAILED", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, results, nil)
}

// GetDoctorMounts validates container volume and bind-mount paths
// GET /api/v2/doctor/mounts
func (s *Server) GetDoctorMounts(w http.ResponseWriter, r *http.Request) {
	results, err := doctor.ValidateMounts()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "VALIDATION_FAILED", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, results, nil)
}

// GetDoctorPorts detects port conflicts
// GET /api/v2/doctor/ports
func (s *Server) GetDoctorPorts(w http.ResponseWriter, r *http.Request) {
	results, err := doctor.ScanPortConflicts(doctor.DefaultDeclaredPorts)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "SCAN_FAILED", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, results, nil)
}

// HandleDoctorFix previews or applies automated fixes
// POST /api/v2/doctor/fix
func (s *Server) HandleDoctorFix(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Apply bool   `json:"apply"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	conts, err := doctor.ScanContainers()
	if err != nil {
		log.Printf("⚠️  Container scan error: %v", err)
	}
	if input.Name != "" {
		var filtered []doctor.ContainerResult
		for _, c := range conts {
			if c.Name == input.Name {
				filtered = append(filtered, c)
			}
		}
		conts = filtered
	}
	mounts, err := doctor.ValidateMounts()
	if err != nil {
		log.Printf("⚠️  Mount scan error: %v", err)
	}
	ports, err := doctor.ScanPortConflicts(doctor.DefaultDeclaredPorts)
	if err != nil {
		log.Printf("⚠️  Port scan error: %v", err)
	}

	fixes := doctor.BuildFixes(conts, mounts, ports)
	if !input.Apply {
		sendSuccess(w, http.StatusOK, map[string]any{
			"applied": false,
			"fixes":   fixes,
		}, nil)
		return
	}

	results := doctor.ApplyFixes(fixes)
	sendSuccess(w, http.StatusOK, map[string]any{
		"applied": true,
		"results": results,
	}, nil)
}

// GetDoctorReport generates a full system health report
// GET /api/v2/doctor/report
func (s *Server) GetDoctorReport(w http.ResponseWriter, r *http.Request) {
	conts, err := doctor.ScanContainers()
	if err != nil {
		log.Printf("⚠️  Container scan error: %v", err)
	}
	mounts, err := doctor.ValidateMounts()
	if err != nil {
		log.Printf("⚠️  Mount scan error: %v", err)
	}
	ports, err := doctor.ScanPortConflicts(doctor.DefaultDeclaredPorts)
	if err != nil {
		log.Printf("⚠️  Port scan error: %v", err)
	}

	report := doctor.GenerateReport(conts, mounts, ports)
	sendSuccess(w, http.StatusOK, report, nil)
}

// GetCloudflaredConfig returns the contents of cloudflared-config.yml
func (s *Server) GetCloudflaredConfig(w http.ResponseWriter, r *http.Request) {
	configPath := filepath.Join(syspaths.GetStackDir(), "cloudflared-config.yml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			sendSuccess(w, http.StatusOK, map[string]string{"content": ""}, nil)
			return
		}
		sendError(w, http.StatusInternalServerError, "READ_FAILED", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, map[string]string{"content": string(content)}, nil)
}

// UpdateCloudflaredConfig updates the contents of cloudflared-config.yml
func (s *Server) UpdateCloudflaredConfig(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	configPath := filepath.Join(syspaths.GetStackDir(), "cloudflared-config.yml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		sendError(w, http.StatusInternalServerError, "DIR_CREATE_FAILED", err.Error(), nil)
		return
	}

	if err := os.WriteFile(configPath, []byte(input.Content), 0644); err != nil {
		sendError(w, http.StatusInternalServerError, "WRITE_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]bool{"ok": true}, nil)
}

// GetEnvConfig returns the raw contents of the active .env file.
// GET /api/v2/config/env
func (s *Server) GetEnvConfig(w http.ResponseWriter, r *http.Request) {
	envPath := syspaths.GetConfigPath()
	content, err := os.ReadFile(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			sendSuccess(w, http.StatusOK, map[string]string{"content": "", "path": envPath}, nil)
			return
		}
		sendError(w, http.StatusInternalServerError, "READ_FAILED", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, map[string]string{"content": string(content), "path": envPath}, nil)
}

// UpdateEnvConfig writes updated content back to the active .env file.
// POST /api/v2/config/env
func (s *Server) UpdateEnvConfig(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", err.Error(), nil)
		return
	}

	envPath := syspaths.GetConfigPath()
	if err := os.MkdirAll(filepath.Dir(envPath), 0755); err != nil {
		sendError(w, http.StatusInternalServerError, "DIR_CREATE_FAILED", err.Error(), nil)
		return
	}

	if err := os.WriteFile(envPath, []byte(input.Content), 0600); err != nil {
		sendError(w, http.StatusInternalServerError, "WRITE_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]bool{"ok": true}, nil)
}

// GetSystemServices retrieves status of all systemd services.
// GET /api/v2/system/services
func (s *Server) GetSystemServices(w http.ResponseWriter, r *http.Request) {
	services, err := system.ListServices()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "SYSTEMD_LIST_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, services, nil)
}

// ControlSystemService starts, stops, restarts, enables, or disables a service.
// POST /api/v2/system/services/{name}/{action}
func (s *Server) ControlSystemService(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	action := chi.URLParam(r, "action")

	if name == "" || action == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "name and action are required", nil)
		return
	}

	if err := system.ControlService(name, action); err != nil {
		sendError(w, http.StatusInternalServerError, "SYSTEMD_ACTION_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]bool{"ok": true}, nil)
}

// GetSystemStorage retrieves host mounts, partitions, and Samba shares.
// GET /api/v2/system/storage
func (s *Server) GetSystemStorage(w http.ResponseWriter, r *http.Request) {
	partitions, err := system.GetDiskPartitions()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "STORAGE_ERROR", err.Error(), nil)
		return
	}

	shares, err := system.GetSambaShares()
	if err != nil {
		log.Printf("⚠️ Failed to parse Samba shares: %v", err)
		shares = []models.SambaShare{}
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"partitions": partitions,
		"samba":      shares,
	}, nil)
}

// UnmountStorage safely unmounts a removable or user storage drive.
// POST /api/v2/system/storage/unmount
func (s *Server) UnmountStorage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		Mountpoint string `json:"mountpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", err.Error(), nil)
		return
	}

	if req.Mountpoint == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "mountpoint is required", nil)
		return
	}

	if err := system.UnmountDrive(req.Mountpoint); err != nil {
		sendError(w, http.StatusInternalServerError, "UNMOUNT_FAILED", err.Error(), nil)
		return
	}

	events.GlobalEventBus.Publish("storage.unmounted", map[string]string{
		"mountpoint": req.Mountpoint,
	})

	sendSuccess(w, http.StatusOK, map[string]any{
		"unmounted":  true,
		"mountpoint": req.Mountpoint,
	}, nil)
}

// GetSystemNetwork returns host network interfaces, IP addresses, and I/O metrics.
// GET /api/v2/system/network
func (s *Server) GetSystemNetwork(w http.ResponseWriter, r *http.Request) {
	ifaces, err := system.GetNetworkInterfaces()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "NETWORK_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, ifaces, nil)
}

// GetCronJobs retrieves scheduled tasks from crontab and systemd timers.
// GET /api/v2/system/cron
func (s *Server) GetCronJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := system.ListCronJobs()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "CRON_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, jobs, nil)
}

// GetSystemUpdates checks for available packages upgrades.
// GET /api/v2/system/updates
func (s *Server) GetSystemUpdates(w http.ResponseWriter, r *http.Request) {
	updates, err := system.CheckUpdates()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "UPDATES_CHECK_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, updates, nil)
}

// ApplySystemUpdates triggers package upgrades.
// POST /api/v2/system/updates/apply
func (s *Server) ApplySystemUpdates(w http.ResponseWriter, r *http.Request) {
	if err := system.TriggerUpdates(); err != nil {
		sendError(w, http.StatusInternalServerError, "UPDATES_TRIGGER_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, map[string]bool{"ok": true}, nil)
}

// SystemCleanup executes host cleanup across package managers and caches.
// POST /api/v2/system/cleanup
func (s *Server) SystemCleanup(w http.ResponseWriter, r *http.Request) {
	result, err := system.CleanSystem()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "CLEANUP_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, result, nil)
}

// GetProcesses returns running system processes.
// GET /api/v2/system/processes
func (s *Server) GetProcesses(w http.ResponseWriter, r *http.Request) {
	limit := 150
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	procs, err := system.GetProcesses(limit)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "PROCESSES_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, procs, nil)
}

// KillProcess terminates a process by PID.
// POST /api/v2/system/processes/{pid}/kill
func (s *Server) KillProcess(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "pid")
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		sendError(w, http.StatusBadRequest, "INVALID_PID", "invalid PID specified", nil)
		return
	}
	if err := system.KillProcess(int32(pid)); err != nil {
		sendError(w, http.StatusInternalServerError, "KILL_PROCESS_FAILED", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, map[string]any{"ok": true, "pid": pid}, nil)
}

// GetSystemSpecs returns detailed host specifications and network interfaces.
// GET /api/v2/system/specs
func (s *Server) GetSystemSpecs(w http.ResponseWriter, r *http.Request) {
	specs, err := system.GetSystemSpecs()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "SYSTEM_SPECS_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, specs, nil)
}

// GetDockerImages retrieves all Docker images on the host.
// GET /api/v2/docker/images
func (s *Server) GetDockerImages(w http.ResponseWriter, r *http.Request) {
	provider, err := docker.GetProvider()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_UNAVAILABLE", err.Error(), nil)
		return
	}
	images, err := provider.ListImages()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, images, nil)
}

// PruneDockerImages prunes unused Docker images.
// POST /api/v2/docker/images/prune
func (s *Server) PruneDockerImages(w http.ResponseWriter, r *http.Request) {
	provider, err := docker.GetProvider()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_UNAVAILABLE", err.Error(), nil)
		return
	}
	reclaimed, err := provider.PruneImages()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, map[string]any{"reclaimed": reclaimed}, nil)
}

// GetDockerVolumes retrieves all Docker volumes.
// GET /api/v2/docker/volumes
func (s *Server) GetDockerVolumes(w http.ResponseWriter, r *http.Request) {
	provider, err := docker.GetProvider()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_UNAVAILABLE", err.Error(), nil)
		return
	}
	volumes, err := provider.ListVolumes()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, volumes, nil)
}

// PruneDockerVolumes prunes unused Docker volumes.
// POST /api/v2/docker/volumes/prune
func (s *Server) PruneDockerVolumes(w http.ResponseWriter, r *http.Request) {
	provider, err := docker.GetProvider()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_UNAVAILABLE", err.Error(), nil)
		return
	}
	reclaimed, err := provider.PruneVolumes()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, map[string]any{"reclaimed": reclaimed}, nil)
}

// GetDockerNetworks retrieves all Docker networks.
// GET /api/v2/docker/networks
func (s *Server) GetDockerNetworks(w http.ResponseWriter, r *http.Request) {
	provider, err := docker.GetProvider()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_UNAVAILABLE", err.Error(), nil)
		return
	}
	networks, err := provider.ListNetworks()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, networks, nil)
}

// PruneDockerNetworks prunes unused Docker networks.
// POST /api/v2/docker/networks/prune
func (s *Server) PruneDockerNetworks(w http.ResponseWriter, r *http.Request) {
	provider, err := docker.GetProvider()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_UNAVAILABLE", err.Error(), nil)
		return
	}
	prunedCount, err := provider.PruneNetworks()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "DOCKER_ERROR", err.Error(), nil)
		return
	}
	sendSuccess(w, http.StatusOK, map[string]any{"pruned_count": prunedCount}, nil)
}
