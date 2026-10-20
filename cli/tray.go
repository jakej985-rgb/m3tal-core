package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/gogpu/systray"
	"github.com/jakej985-rgb/m3tal-core/pkg/client"
	"github.com/jakej985-rgb/m3tal-core/pkg/cmdutil"
	"github.com/spf13/cobra"
)

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
    const r=await fetch('/tray/api/stats');
    const s=await r.json();
    document.getElementById('hostname').textContent=s.hostname||'—';
    document.getElementById('uptime').textContent='up '+fmtUptime(s.uptime||0);

    const cpu=s.cpu_usage||0;
    document.getElementById('cpu-val').textContent=fmt2(cpu)+'%'+(s.cpu_temp>0?' · '+fmt2(s.cpu_temp)+'°C':'');
    setBar('cpu-bar',cpu);

    const mem=s.memory_usage||0;
    document.getElementById('mem-val').textContent=fmt2(s.memory_used||0)+' / '+fmt2(s.memory_total||0)+' GB';
    setBar('mem-bar',mem);

    const disk=s.disk_usage||0;
    document.getElementById('disk-val').textContent=fmt2(s.disk_used||0)+' / '+fmt2(s.disk_total||0)+' GB';
    setBar('disk-bar',disk);

    if(s.gpu_model && s.gpu_model!=='No GPU Detected'){
      document.getElementById('gpu-section').style.display='block';
      document.getElementById('gpu-model').textContent=s.gpu_model;
      document.getElementById('gpu-val').textContent=fmt2(s.gpu_usage||0)+'%'+(s.gpu_temp>0?' · '+fmt2(s.gpu_temp)+'°C':'');
      setBar('gpu-bar',s.gpu_usage||0);
      const vramPct=s.gpu_mem_total>0?((s.gpu_mem_used/s.gpu_mem_total)*100):0;
      document.getElementById('vram-val').textContent=Math.round(s.gpu_mem_used||0)+' / '+Math.round(s.gpu_mem_total||0)+' MB';
      setBar('vram-bar',vramPct);
    }
  }catch(e){console.error('stats',e)}
}

async function loadContainers(){
  try{
    const r=await fetch('/tray/api/containers');
    const list=await r.json();
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

// IconBase64 is the M3TAL logo (32x32 PNG) embedded as base64.
var IconBase64 = "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"

var IconData []byte

func init() {
	var err error
	IconData, err = base64.StdEncoding.DecodeString(IconBase64)
	if err != nil || len(IconData) == 0 {
		// Fallback: 1x1 teal PNG so the icon is never invisible
		IconData, _ = base64.StdEncoding.DecodeString(
			"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==")
	}
}

// hasStatusNotifier checks if org.kde.StatusNotifierWatcher is registered on the session bus
func hasStatusNotifier() bool {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return false
	}
	defer conn.Close()

	var owner string
	err = conn.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, "org.kde.StatusNotifierWatcher").Store(&owner)
	if err != nil {
		return false
	}
	return owner != ""
}

var trayCmd = &cobra.Command{
	Use:   "tray",
	Short: "Run the M3TAL system tray monitor",
	Long:  "Starts a system tray icon that launches a browser-based stats monitor popup showing real-time CPU, GPU, storage, and container metrics.",
	Run: cmdutil.WithClient(func(c *client.Client, cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		runTray(c, port)
	}),
}

func init() {
	trayCmd.Flags().StringP("port", "p", "18088", "Port to run the tray web server on")
}

func runTray(c *client.Client, _ string) {
	log.Printf("[tray] runTray invoked - targeting API daemon at %s\n", c.BaseURL)

	// Run systray (pure Go, zero CGO, blocks on Run)
	mode := os.Getenv("M3TAL_TRAY")
	if mode == "off" {
		log.Println("[tray] M3TAL_TRAY is set to off; system tray disabled")
		return
	}
	if mode != "force" && !hasStatusNotifier() {
		log.Println("[tray] StatusNotifierWatcher not available; system tray disabled")
		return
	}

	targetURL := fmt.Sprintf("%s/tray", c.BaseURL)
	log.Printf("[tray] Tray UI target URL: %s\n", targetURL)

	tray := systray.New()
	tray.SetIcon(IconData)
	tray.SetTooltip("M3TAL Core System Tray")

	menu := systray.NewMenu()
	menu.Add("Open Monitor", func() {
		openBrowser(targetURL)
	})
	menu.Add("Open Dashboard", func() {
		openBrowser("http://localhost:8082")
	})
	menu.AddSeparator()
	menu.Add("Quit", func() {
		tray.Remove()
		os.Exit(0)
	})

	tray.SetMenu(menu)
	tray.OnClick(func() {
		// Left-click: open a small popup window (like calendar) at fixed size
		openPopup(targetURL)
	})
	tray.Show()
	tray.Run()
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// openPopup opens a small standalone popup window (360x520) like a calendar card.
// Tries chromium-based --app popup first, then falls back to xdg-open.
func openPopup(url string) {
	if runtime.GOOS != "linux" {
		openBrowser(url)
		return
	}
	// Try chrome/chromium --app mode for a borderless popup window
	for _, browser := range []string{"google-chrome", "chromium", "chromium-browser", "brave-browser"} {
		if _, err := exec.LookPath(browser); err == nil {
			cmd := exec.Command(browser,
				"--app="+url,
				"--window-size=360,560",
				"--window-position=9999,9999", // position near bottom-right; DE may reposition
				"--no-first-run",
				"--no-default-browser-check",
			)
			if err := cmd.Start(); err == nil {
				go stripDecorations("M3TAL Tray")
				return
			}
		}
	}
	// Fallback: open in default browser
	openBrowser(url)
}

// stripDecorations strips window decorations and sets hints via xprop to make the popup window borderless.
func stripDecorations(title string) {
	for i := 0; i < 20; i++ {
		// 1. Remove decorations using xprop
		cmd1 := exec.Command("xprop", "-name", title, "-f", "_MOTIF_WM_HINTS", "32c", "-set", "_MOTIF_WM_HINTS", "2,0,0,0,0")
		_ = cmd1.Run()

		// 2. Set window type to UTILITY to treat it like a drawer applet popup (e.g. Calendar)
		cmd2 := exec.Command("xprop", "-name", title, "-f", "_NET_WM_WINDOW_TYPE", "32a", "-set", "_NET_WM_WINDOW_TYPE", "_NET_WM_WINDOW_TYPE_UTILITY")
		_ = cmd2.Run()

		// 3. Keep on top and skip standard taskbar listing
		cmd3 := exec.Command("wmctrl", "-r", title, "-b", "add,above,skip_taskbar")
		_ = cmd3.Run()

		time.Sleep(100 * time.Millisecond)
	}
}
