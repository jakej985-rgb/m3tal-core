/* app.js — M3TAL Dashboard v2 */

const socket = io();

// ── Clock ────────────────────────────────────────────────────────
function tick() {
    const el = document.getElementById('live-clock');
    if (!el) return;
    const now = new Date();
    const hh = String(now.getHours()).padStart(2, '0');
    const mm = String(now.getMinutes()).padStart(2, '0');
    const ss = String(now.getSeconds()).padStart(2, '0');
    el.textContent = `${hh}:${mm}:${ss}`;
}
setInterval(tick, 1000);
tick();

// ── Resource Chart ───────────────────────────────────────────────
let chart = null;
let currentHours = 1; 
let maxGraphPoints = 60; // Dynamic based on timeframe
const cpuData  = Array(60).fill(null);
const memData  = Array(60).fill(null);
const timeLabels = Array(60).fill('');

function setTimeframe(val) {
    console.log(`Setting timeframe to: ${val}`);
    
    // Update UI active state
    document.querySelectorAll('.time-selectors .panel-badge').forEach(b => {
        b.classList.remove('active');
        if (b.textContent.toLowerCase() === val.toLowerCase()) b.classList.add('active');
    });

    // In a real implementation, this would fetch from /api/metrics/history?hours=...
    // For now, we update the display and refresh the chart
    currentHours = parseInt(val) || 1;
    if (val.includes('d')) currentHours *= 24;
    
    refreshHistory();
}

async function refreshHistory() {
    try {
        const res = await fetch(`/api/metrics/history?hours=${currentHours}`);
        const data = await res.json();
        if (!data || !Array.isArray(data)) return;

        const now = Math.floor(Date.now() / 1000);
        const cutoff = now - (currentHours * 3600);
        
        // Filter history
        let plotData = data.filter(p => p.timestamp >= cutoff);
        
        // Expected points (1 per minute)
        maxGraphPoints = currentHours * 60;
        
        // Pad left with nulls if we don't have enough history
        // This ensures the graph doesn't "shrink" or jump around
        while (plotData.length < maxGraphPoints) {
            plotData.unshift({ cpu: null, mem: null, net_down: null, net_up: null, timestamp: 0 });
        }

        chart.data.labels = plotData.map(p => {
            if (p.timestamp === 0) return "";
            const d = new Date(p.timestamp * 1000);
            if (currentHours > 24) return `${d.getMonth()+1}/${d.getDate()} ${d.getHours()}:00`;
            return `${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`;
        });
        chart.data.datasets[0].data = plotData.map(p => p.cpu);
        chart.data.datasets[1].data = plotData.map(p => p.mem);
        chart.data.datasets[2].data = plotData.map(p => p.net_down || 0);
        chart.data.datasets[3].data = plotData.map(p => p.net_up || 0);
        
        chart.update('none');
    } catch (e) {
        console.error("Failed to refresh history:", e);
    }
}

function initChart() {
    const canvas = document.getElementById('resource-chart');
    if (!canvas) return;

    chart = new Chart(canvas, {
        type: 'line',
        data: {
            labels: timeLabels,
            datasets: [
                {
                    label: 'CPU',
                    data: cpuData,
                    borderColor: '#22c55e',
                    backgroundColor: 'rgba(34,197,94,0.08)',
                    borderWidth: 1.5,
                    tension: 0.4,
                    fill: true,
                    pointRadius: 0,
                    yAxisID: 'y'
                },
                {
                    label: 'MEM',
                    data: memData,
                    borderColor: '#a855f7',
                    backgroundColor: 'rgba(168,85,247,0.08)',
                    borderWidth: 1.5,
                    tension: 0.4,
                    fill: true,
                    pointRadius: 0,
                    yAxisID: 'y'
                },
                {
                    label: 'DOWN',
                    data: [],
                    borderColor: '#3b82f6',
                    backgroundColor: 'rgba(59,130,246,0.08)',
                    borderWidth: 1.5,
                    tension: 0.4,
                    fill: true,
                    pointRadius: 0,
                    yAxisID: 'y1'
                },
                {
                    label: 'UP',
                    data: [],
                    borderColor: '#f59e0b',
                    backgroundColor: 'rgba(245,158,11,0.08)',
                    borderWidth: 1.5,
                    tension: 0.4,
                    fill: true,
                    pointRadius: 0,
                    yAxisID: 'y1'
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: { duration: 400 },
            interaction: { mode: 'index', intersect: false },
            scales: {
                x: {
                    display: false,
                    grid: { display: false }
                },
                y: {
                    min: 0, max: 100,
                    grid: { color: 'rgba(255,255,255,0.04)', drawBorder: false },
                    ticks: {
                        color: '#4b5e75',
                        font: { family: "'JetBrains Mono', monospace", size: 10 },
                        callback: v => `${v}%`,
                        maxTicksLimit: 5,
                    },
                    border: { display: false }
                },
                y1: {
                    position: 'right',
                    beginAtZero: true,
                    grid: { display: false },
                    ticks: {
                        color: '#4b5e75',
                        font: { family: "'JetBrains Mono', monospace", size: 10 },
                        callback: function(v) {
                            if (v >= 1) return v.toFixed(1) + ' MB/s';
                            if (v > 0) return (v * 1024).toFixed(0) + ' KB/s';
                            return '0 MB/s';
                        },
                        maxTicksLimit: 5,
                    },
                    border: { display: false }
                }
            },
            plugins: {
                legend: { display: false },
                tooltip: {
                    backgroundColor: 'rgba(13,17,23,0.9)',
                    borderColor: 'rgba(0,212,170,0.2)',
                    borderWidth: 1,
                    titleColor: '#94a3b8',
                    bodyColor: '#e2e8f0',
                    bodyFont: { family: "'JetBrains Mono', monospace", size: 11 },
                }
            }
        }
    });
}

function pushChartPoint(cpu, mem, netDownStr, netUpStr, data = {}) {
    if (!chart) return;
    
    // Only push live updates to the chart if we are in the 1H (Real-time) view
    // Otherwise, it will cause the historical data to shift and eventually vanish
    if (currentHours > 1) return;
    
    const now = new Date();
    const timeStr = `${String(now.getHours()).padStart(2,'0')}:${String(now.getMinutes()).padStart(2,'0')}`;
    
    // Parse network strings (fallback if raw not available)
    const parseSpeed = (v, s) => {
        if (typeof v === 'number') return v;
        if (!s || typeof s !== 'string') return 0;
        const parts = s.split(' ');
        const val = parseFloat(parts[0]);
        if (isNaN(val)) return 0;
        const unit = (parts[1] || '').toUpperCase();
        if (unit.includes('K')) return val / 1024;
        if (unit.includes('G')) return val * 1024;
        if (unit.includes('B') && !unit.includes('M')) return val / (1024 * 1024);
        return val;
    };

    chart.data.labels.push(timeStr);
    chart.data.datasets[0].data.push(cpu);
    chart.data.datasets[1].data.push(mem);
    chart.data.datasets[2].data.push(parseSpeed(data.down_raw, netDownStr));
    chart.data.datasets[3].data.push(parseSpeed(data.up_raw, netUpStr));
    
    if (chart.data.labels.length > maxGraphPoints) {
        chart.data.labels.shift();
        chart.data.datasets.forEach(ds => ds.data.shift());
    }
    
    chart.update('none');
}

// Global state for single-line metric synchronization
const liveStats = {
    cpu: 0,
    mem: 0,
    cpuTemp: '--',
    gpuLoad: 0,
    gpuTemp: '--',
    gpuMem: '--',
    gpuActive: false,
    netDown: '0.0',
    netUp: '0.0',
    netLoad: '0'
};

function updateCpuFull() {
    setText('stat-cpu-usage-val', `${liveStats.cpu.toFixed(1)}%`);
    setText('stat-cpu-temp-val-split', `${liveStats.cpuTemp}°C`);
    const total = liveStats.memTotal || 16.0;
    setText('stat-cpu-mem-label', `${total.toFixed(1)} GB RAM`);
    setText('stat-cpu-mem-val-split', `${liveStats.mem.toFixed(1)} GB`);
}

function updateGpuFull() {
    if (liveStats.gpuActive) {
        setText('stat-gpu-usage-val', `${liveStats.gpuLoad}%`);
        setText('stat-gpu-temp-val-split', `${liveStats.gpuTemp}°C`);
        const total = liveStats.gpuMemTotal || 1024;
        const totalStr = total >= 1024 ? `${(total/1024).toFixed(0)}GB` : `${total}MB`;
        setText('stat-gpu-mem-label', `${totalStr} VRAM`);
        setText('stat-gpu-mem-val-split', `${liveStats.gpuMem}MB`);
    } else {
        setText('stat-gpu-usage-val', 'OFF');
        setText('stat-gpu-temp-val-split', '--°C');
        setText('stat-gpu-mem-label', 'VRAM');
        setText('stat-gpu-mem-val-split', 'SB');
    }
}

function updateNetworkFull() {
    setText('stat-net-down-val', liveStats.netDown);
    setText('stat-net-up-val', liveStats.netUp);
    setText('stat-net-load-val', `${liveStats.netLoad}%`);
}

// ── Socket – real-time metrics ────────────────────────────────────
socket.on('metrics_update', (data) => {
    const sys = data.system || {};
    liveStats.cpu = sys.cpu || 0;
    liveStats.mem = sys.mem_gb || 0;
    liveStats.memTotal = sys.mem_total || 0;

    // Update the single-line display
    updateCpuFull();

    // Update Network display
    const net = data.network || {};
    liveStats.netDown = net.down != null ? net.down : '0.0';
    liveStats.netUp   = net.up   != null ? net.up   : '0.0';
    liveStats.netLoad = net.load != null ? Math.round(net.load) : '0';
    updateNetworkFull();

    // Legacy stat cards (if still in HTML)
    setText('stat-cpu', `${liveStats.cpu.toFixed(1)}%`);
    setText('stat-mem', `${liveStats.mem.toFixed(1)} GB`);

    // Push to chart
    pushChartPoint(liveStats.cpu, sys.mem || 0, liveStats.netDown, liveStats.netUp, net);
});

// ── Mobile Drawer ──────────────────────────────────────────────────
function toggleDrawer() {
    const drawer = document.getElementById('mobile-drawer');
    if (drawer) {
        drawer.classList.toggle('open');
    }
}

// Close drawer when clicking outside
document.addEventListener('click', (e) => {
    const drawer = document.getElementById('mobile-drawer');
    const toggle = document.getElementById('menu-toggle');
    if (drawer && drawer.classList.contains('open')) {
        if (!drawer.contains(e.target) && !toggle.contains(e.target)) {
            drawer.classList.remove('open');
        }
    }
});

// ── Helpers ───────────────────────────────────────────────────────
function setText(id, val) {
    const el = document.getElementById(id);
    if (el) el.textContent = val;
}

function getStatusClass(status) {
    const s = (status || '').toLowerCase();
    if (s === 'running' || s === 'online') return 'running';
    if (s === 'restarting') return 'restarting';
    if (s === 'offline' || s === 'exited') return 'offline';
    if (s === 'missing') return 'missing';
    return 'unknown';
}

function getCpuClass(cpu) {
    if (cpu >= 80) return 'cpu-crit';
    if (cpu >= 50) return 'cpu-high';
    return '';
}

function renderBar(percent) {
    const total = 10;
    const filled = Math.round((Math.min(percent, 100) / 100) * total);
    const bar = "█".repeat(filled) + "░".repeat(total - filled);
    
    let color = "#22c55e"; // Green
    if (percent >= 80) color = "#ef4444"; // Red
    else if (percent >= 50) color = "#f59e0b"; // Yellow
    
    return `<span style="color: ${color}; font-family: 'JetBrains Mono', monospace; letter-spacing: -1px;">${bar}</span>`;
}

// ── UI Interactions ───────────────────────────────────────────────
function togglePanel(header) {
    const panel = header.parentElement;
    panel.classList.toggle('collapsed');
}

function toggleRow(rowId) {
    const detailsRow = document.getElementById(rowId);
    if (!detailsRow) return;
    
    const isVisible = detailsRow.style.display !== 'none';
    
    // Hide all other details rows first (optional, for accordion effect)
    // document.querySelectorAll('.details-row').forEach(r => r.style.display = 'none');
    
    detailsRow.style.display = isVisible ? 'none' : 'table-row';
}

// ── Health score ──────────────────────────────────────────────────
async function refreshHealth() {
    try {
        const res  = await fetch('/api/health/report');
        const data = await res.json();
        const score = data.score || 0;
        const verdict = data.verdict || 'Healthy';

        // Main Score
        const scoreEl = document.getElementById('health-score');
        if (scoreEl) scoreEl.textContent = score;
        
        // Mini Card Score (Standardized ID)
        const healthVal = document.getElementById('stat-ai-health-val');
        if (healthVal) {
            healthVal.textContent = `${score}%`;
        }
        
        const ring = document.getElementById('health-ring');
        if (ring) {
            const offset = 220 - (220 * score / 100);
            ring.style.strokeDashoffset = offset;
        }

        // Mini Card Score
        setText('stat-ai-health-val', `${score}%`);
        
        const ringMini = document.getElementById('gsi-ring-mini');
        if (ringMini) {
            const offset = 220 - (220 * score / 100);
            ringMini.style.strokeDashoffset = offset;
        }

        const verdictEl = document.getElementById('system-verdict');
        if (verdictEl) {
            const agents = data.agent_health || {};
            const total = Object.keys(agents).length;
            const online = Object.values(agents).filter(a => {
                const s = (a.status || '').toLowerCase();
                return s === 'healthy' || s === 'ok' || s === 'online' || s === 'up' || s === 'running';
            }).length;
            
            verdictEl.textContent = `${online} / ${total}`;
            
            let statusClass = 'offline';
            if (online === total && total > 0 && score >= 90) statusClass = 'running';
            else if (online > 0 && score >= 60) statusClass = 'restarting';
            
            verdictEl.className = `badge ${statusClass}`;
        }
    } catch (_) {}
}

// ── Hardware Metrics ──────────────────────────────────────────────
async function refreshHardware() {
    try {
        const [tRes, sRes, gRes] = await Promise.all([
            fetch('/api/metrics/temperature'),
            fetch('/api/metrics/storage'),
            fetch('/api/metrics/gpu')
        ]);
        const tData = await tRes.json();
        const sData = await sRes.json();
        const gData = await gRes.json();

        // Update GPU Card (Single Line)
        liveStats.gpuActive = gData.active;
        if (gData.active) {
            liveStats.gpuLoad = gData.load !== undefined ? gData.load : 0;
            liveStats.gpuTemp = gData.temp != null ? Math.round(gData.temp) : '--';
            liveStats.gpuMem  = gData.mem_used != null ? gData.mem_used : '--';
            liveStats.gpuMemTotal = gData.mem_total != null ? gData.mem_total : 1024;
        }
        updateGpuFull();

        const gpuIcon = document.getElementById('stat-gpu-card')?.querySelector('.stat-icon');
        if (gpuIcon) {
            if (gData.load >= 80 || gData.temp >= 80) {
                gpuIcon.style.background = 'rgba(239, 68, 68, 0.15)'; gpuIcon.style.color = '#ef4444';
            } else if (gData.load >= 50 || gData.temp >= 70) {
                gpuIcon.style.background = 'rgba(245, 158, 11, 0.15)'; gpuIcon.style.color = '#f59e0b';
            } else {
                gpuIcon.style.background = 'rgba(249, 115, 22, 0.15)'; gpuIcon.style.color = '#f97316';
            }
        }

        // Update Temperature Card & CPU Split
        liveStats.cpuTemp = tData.cpu_temp != null ? Math.round(tData.cpu_temp) : '--';
        liveStats.gpuTemp = tData.gpu_temp != null ? Math.round(tData.gpu_temp) : '--';
        updateCpuFull();

        const cpuTempEl = document.getElementById('stat-cpu-temp-val');
        const gpuTempEl = document.getElementById('stat-gpu-temp-val');
        
        if (cpuTempEl && gpuTempEl) {
            cpuTempEl.textContent = `${liveStats.cpuTemp}°C`;
            gpuTempEl.textContent = `${liveStats.gpuTemp}°C`;
            
            const maxTemp = Math.max(tData.cpu_temp || 0, tData.gpu_temp || 0);
            const tempIcon = cpuTempEl.closest('.stat-card').querySelector('.stat-icon');
            if (maxTemp >= 85) {
                tempIcon.style.background = 'rgba(239, 68, 68, 0.15)'; tempIcon.style.color = '#ef4444';
            } else if (maxTemp >= 75) {
                tempIcon.style.background = 'rgba(245, 158, 11, 0.15)'; tempIcon.style.color = '#f59e0b';
            } else {
                tempIcon.style.background = 'rgba(34, 197, 94, 0.15)'; tempIcon.style.color = '#22c55e';
            }
        }

        // Update Storage Card
        const storageGrid = document.getElementById('stat-storage-grid');
        if (storageGrid && sData.disks) {
            let maxUsage = 0;
            let gridHtml = '';
            const driveKeys = Object.keys(sData.disks).sort(); 
            
            if (driveKeys.length === 0) {
                gridHtml = '<div class="stat-sub">No drives detected</div>';
            } else {
                let rows = { names: '', space: '', temp: '' };
                
                driveKeys.forEach(key => {
                    const disk = sData.disks[key];
                    if (disk.percent > maxUsage) maxUsage = disk.percent;
                    
                    const free = disk.free != null ? `${disk.free}G` : '--';
                    const temp = disk.temp != null ? `${Math.round(disk.temp)}°C` : '--';
                    
                    rows.names += `<div>${key}</div>`;
                    rows.space += `<div>${free}</div>`;
                    rows.temp  += `<div>${temp}</div>`;
                });

                gridHtml = `
                    <div class="storage-table">
                        <div class="row names">${rows.names}</div>
                        <div class="row space">${rows.space}</div>
                        <div class="row temp">${rows.temp}</div>
                    </div>
                `;
            }
            
            storageGrid.innerHTML = gridHtml;
            
            const storageIcon = document.getElementById('stat-storage-card').querySelector('.stat-icon');
            if (maxUsage >= 95) {
                storageIcon.style.background = 'rgba(239, 68, 68, 0.15)'; storageIcon.style.color = '#ef4444';
            } else if (maxUsage >= 85) {
                storageIcon.style.background = 'rgba(245, 158, 11, 0.15)'; storageIcon.style.color = '#f59e0b';
            } else {
                storageIcon.style.background = 'rgba(14, 165, 233, 0.15)'; storageIcon.style.color = '#0ea5e9';
            }
        }
    } catch (_) {}
}

// ── Network Links (Traefik Discovery) ──────────────────────────────
async function refreshLinks() {
    try {
        const res = await fetch('/api/network/routes');
        const routes = await res.json();
        const grid = document.getElementById('dynamic-links-grid');
        if (!grid) return;

        if (routes.length === 0) {
            grid.innerHTML = '<div class="stat-sub">No web routes discovered</div>';
            return;
        }

        grid.innerHTML = routes.map(r => `
            <a href="${r.url}" class="big-btn" target="_blank">
                <span style="font-size: 1.2rem;">🌐</span>
                ${r.name}
            </a>
        `).join('');
    } catch (_) {}
}

// ── Container table ───────────────────────────────────────────────
async function refreshFleet() {
    try {
        const [hRes, rRes, tRes, sRes] = await Promise.all([
            fetch('/api/health'),
            fetch('/api/health/report'),
            fetch('/api/metrics/temperature'),
            fetch('/api/metrics/storage')
        ]);
        const hData = await hRes.json();
        const rData = await rRes.json();
        const tData = await tRes.json();
        const sData = await sRes.json();

        // Go backend writes containers as a flat array in health.json
        const containers = hData.containers || [];
        const online = containers.filter(c => (c.state || '').toLowerCase() === 'running').length;
        const total = containers.length;
        setText('stat-fleet-count', `${online} / ${total} UP`);

        // Uptime from health report
        const uptimeSubEl = document.getElementById('stat-fleet-uptime');
        if (uptimeSubEl && rData.uptime) uptimeSubEl.textContent = rData.uptime;

        // Table body
        const tbody = document.getElementById('fleet-tbody');
        if (!tbody) return;

        if (containers.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading-text">Waiting for agent data…</td></tr>';
            return;
        }

        // Sort: running first
        const order = { running: 0, created: 1, restarting: 2, paused: 3, exited: 4, dead: 5 };
        containers.sort((a, b) => (order[(a.state||'').toLowerCase()] ?? 6) - (order[(b.state||'').toLowerCase()] ?? 6));

        // Tracking open rows
        if (!window.openRows) window.openRows = new Set();

        let html = '';
        containers.forEach(c => {
            const name   = c.name || 'unknown';
            const state  = c.state || 'unknown';
            const status = c.status || state;
            const sc     = getStatusClass(state);
            const cpu    = c.cpu != null ? c.cpu.toFixed(1) + '%' : '—';
            const mem    = c.mem != null ? c.mem.toFixed(1) + '%' : '—';
            const cpuClass = c.cpu != null ? getCpuClass(c.cpu) : '';

            // Sub-metrics for details
            const cpuTemp = tData.cpu_temp != null ? Math.round(tData.cpu_temp) : '--';
            const gpuTemp = tData.gpu_temp != null ? Math.round(tData.gpu_temp) : '--';

            const rowId = `details-${name.replace(/[^a-z0-9]/gi, '-')}`;
            
            // Stats parsing
            const memLimitGB = c.mem_limit ? (c.mem_limit / (1024*1024*1024)).toFixed(1) + ' GB' : '—';
            const memUsedMB  = c.mem_usage ? (c.mem_usage / (1024*1024)).toFixed(0) + ' MB' : '—';
            const isRowOpen  = window.openRows.has(rowId);

            // Network Rates
            if (!window.prevNet) window.prevNet = {};
            const prev = window.prevNet[name] || { rx: c.net_rx, tx: c.net_tx, t: Date.now() };
            const dt = (Date.now() - prev.t) / 1000;
            const rxRate = dt > 0 ? (c.net_rx - prev.rx) / dt : 0;
            const txRate = dt > 0 ? (c.net_tx - prev.tx) / dt : 0;
            window.prevNet[name] = { rx: c.net_rx, tx: c.net_tx, t: Date.now() };

            const formatNet = (b) => b < 1024*1024 ? (b/1024).toFixed(1) + ' KB/s' : (b/(1024*1024)).toFixed(1) + ' MB/s';
            const formatTotal = (b) => b < 1024*1024*1024 ? (b/(1024*1024)).toFixed(0) + ' MB' : (b/(1024*1024*1024)).toFixed(1) + ' GB';

            const getUsageClass = (val) => val > 80 ? 'high' : val > 40 ? 'mid' : 'low';

            html += `
                <tr class="container-row" onclick="toggleRow('${rowId}')">
                    <td><span class="container-name">${name}</span></td>
                    <td><span class="badge ${sc}">${state.toUpperCase()}</span></td>
                    <td class="metric-cell ${cpuClass}">${cpu}</td>
                    <td class="metric-cell">${mem}</td>
                    <td class="metric-cell">${status}</td>
                    <td>
                        <div class="actions-cell">
                            <button class="action-btn logs" title="Logs" onclick="event.stopPropagation(); doAction('logs','${name}')">≡</button>
                        </div>
                    </td>
                </tr>
                <tr id="${rowId}" class="details-row" style="display: ${isRowOpen ? 'table-row' : 'none'};">
                    <td colspan="6">
                        <div class="details-box">
                            <!-- 1: STATUS & HEALTH -->
                            <div class="details-metrics">
                                <div style="opacity:0.5; font-size:0.7rem; margin-bottom:0.4rem;">STATUS & HEALTH</div>
                                <div><strong>STATE:</strong> <span style="color:var(--text-1)">${state}</span></div>
                                <div><strong>MANAGED:</strong> ${c.managed ? '<span style="color:var(--green)">YES</span>' : 'NO'}</div>
                                <div style="font-size:0.7rem; opacity:0.8; margin-top:0.4rem;">${status}</div>
                            </div>

                            <!-- 2: CPU ENGINE -->
                            <div class="details-metrics">
                                <div style="opacity:0.5; font-size:0.7rem; margin-bottom:0.4rem;">CPU ENGINE</div>
                                <div style="display:flex; justify-content:space-between;">
                                    <strong>USAGE:</strong> <span style="color:var(--text-1)">${cpu}</span>
                                </div>
                                <div class="usage-track" style="height:4px; margin-top:0.5rem;">
                                    <div class="usage-fill ${getUsageClass(c.cpu)}" style="width:${cpu}"></div>
                                </div>
                            </div>

                            <!-- 3: MEMORY ALLOC -->
                            <div class="details-metrics">
                                <div style="opacity:0.5; font-size:0.7rem; margin-bottom:0.4rem;">MEMORY ALLOC</div>
                                <div style="display:flex; justify-content:space-between;">
                                    <strong>USE:</strong> <span style="color:var(--teal)">${memUsedMB}</span>
                                    <span style="opacity:0.4">/ ${memLimitGB}</span>
                                </div>
                                <div class="usage-track" style="height:4px; margin-top:0.5rem;">
                                    <div class="usage-fill ${getUsageClass(c.mem)}" style="width:${mem}"></div>
                                </div>
                            </div>

                            <!-- 4: NETWORK I/O -->
                            <div class="details-metrics">
                                <div style="opacity:0.5; font-size:0.7rem; margin-bottom:0.4rem;">NETWORK I/O</div>
                                <div style="display:grid; grid-template-columns: 1fr 1fr; gap:0.5rem;">
                                    <div>
                                        <div style="font-size:0.65rem; opacity:0.5;">DOWN</div>
                                        <div style="color:var(--blue)">${formatNet(rxRate)}</div>
                                        <div style="font-size:0.65rem; opacity:0.3;">${formatTotal(c.net_rx)}</div>
                                    </div>
                                    <div>
                                        <div style="font-size:0.65rem; opacity:0.5;">UP</div>
                                        <div style="color:var(--purple)">${formatNet(txRate)}</div>
                                        <div style="font-size:0.65rem; opacity:0.3;">${formatTotal(c.net_tx)}</div>
                                    </div>
                                </div>
                            </div>

                            <!-- 5: ACTIONS (Now part of the grid for responsiveness) -->
                            <div class="details-actions">
                                <button class="big-btn scan action-btn" onclick="event.stopPropagation(); window.location.href='/logs?container=${name}'">≡ Full Logs</button>
                                <button class="big-btn heal action-btn" onclick="event.stopPropagation(); doAction('restart','${name}')">↺ Restart</button>
                                <button class="big-btn reboot action-btn" onclick="event.stopPropagation(); doAction('stop','${name}')">■ Stop</button>
                            </div>
                        </div>
                    </td>
                </tr>
            `;
        });
        tbody.innerHTML = html;
    } catch (_) {}
}

function toggleRow(id) {
    const el = document.getElementById(id);
    if (!el) return;
    const isVisible = el.style.display !== 'none';
    el.style.display = isVisible ? 'none' : 'table-row';
    
    if (!window.openRows) window.openRows = new Set();
    if (isVisible) {
        window.openRows.delete(id);
    } else {
        window.openRows.add(id);
    }
}

// ── Activity feed ─────────────────────────────────────────────────
async function refreshActivity() {
    try {
        const [aRes, hRes] = await Promise.all([
            fetch('/api/anomalies'),
            fetch('/api/health/report')
        ]);
        const aData = await aRes.json();
        const hData = await hRes.json();

        const feed = document.getElementById('activity-feed');
        if (!feed) return;

        const issues = [
            ...(aData.issues || []).map(i => ({
                title: i.target || 'Container',
                sub:   i.reason || i.message || '',
                type:  (i.type === 'critical') ? 'warn' : 'warn',
                time:  formatTime()
            })),
            ...(hData.issues || []).map(msg => ({
                title: 'System',
                sub:   msg,
                type:  'warn',
                time:  formatTime()
            }))
        ];

        const now = formatTime();

        const pinned = [{
            title: issues.length === 0 ? 'All systems operational' : `${issues.length} issue(s) detected`,
            sub:   issues.length === 0 ? 'No issues detected'       : 'Review anomalies below',
            type:  issues.length === 0 ? 'ok' : 'warn',
            time:  now
        }];

        const all = [...pinned, ...issues].slice(0, 8);

        const iconMap = { ok: '✓', warn: '⚠', info: 'ℹ' };

        feed.innerHTML = all.map(item => `
            <div class="activity-item">
                <div class="activity-icon ${item.type}">${iconMap[item.type] || 'ℹ'}</div>
                <div class="activity-text">
                    <div class="activity-title">${item.title}</div>
                    ${item.sub ? `<div class="activity-sub">${item.sub}</div>` : ''}
                </div>
                <div class="activity-time">${item.time}</div>
            </div>
        `).join('');
    } catch (_) {}
}


function formatTime() {
    const n = new Date();
    return `${String(n.getHours()).padStart(2,'0')}:${String(n.getMinutes()).padStart(2,'0')}`;
}

// ── Actions (wired to /api/action) ───────────────────────────
async function doAction(action, container) {
    console.log(`Action: ${action} on ${container}`);
    const btn = event.currentTarget;
    const origHtml = btn.innerHTML;
    btn.innerHTML = '⏳';
    btn.disabled = true;

    try {
        const res = await fetch('/api/action', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({ action, container })
        });
        const data = await res.json();
        
        if (data.ok) {
            if (action === 'logs') {
                window.location.href = `/logs?container=${container}`;
            } else {
                btn.style.background = 'var(--green-dim)';
                btn.style.color = 'var(--green)';
                setTimeout(() => {
                    btn.style.background = '';
                    btn.style.color = '';
                }, 2000);
            }
        } else {
            alert(`Error: ${data.error || 'Failed'}`);
            btn.style.background = 'var(--red-dim)';
            btn.style.color = 'var(--red)';
        }
    } catch (e) {
        alert(`Request failed: ${e.message}`);
    } finally {
        setTimeout(() => {
            btn.innerHTML = origHtml;
            btn.disabled = false;
        }, action === 'logs' ? 0 : 2000);
    }
}

async function doGlobalAction(action) {
    console.log(`Global action: ${action}`);
    
    // Add confirmation for reboot
    if (action === 'reboot' && !confirm('Are you sure you want to reboot the entire host system?')) {
        return;
    }

    const btn = event.currentTarget;
    const origHtml = btn.innerHTML;
    btn.innerHTML = '⏳ Processing...';
    btn.disabled = true;

    try {
        const res = await fetch('/api/action', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({ action })
        });
        const data = await res.json();
        
        if (data.ok) {
            if (action === 'status') {
                 alert(`Status:\nScore: ${data.score}%\nVerdict: ${data.verdict}\nSystem: ${data.system}`);
            } else {
                 btn.innerHTML = `✅ ${data.message || 'Success'}`;
            }
        } else {
            alert(`Error: ${data.error || 'Failed'}`);
            btn.innerHTML = '❌ Error';
        }
    } catch (e) {
        alert(`Request failed: ${e.message}`);
        btn.innerHTML = '❌ Failed';
    } finally {
        setTimeout(() => {
            btn.innerHTML = origHtml;
            btn.disabled = false;
        }, 3000);
    }
}

async function rescanStacks(e) {
    const btn = (e && e.currentTarget) ? e.currentTarget : (window.event ? window.event.currentTarget : null);
    const origHtml = btn ? btn.innerHTML : '🔍 Rescan Stacks';
    if (btn) {
        btn.innerHTML = '⏳ Scanning...';
        btn.disabled = true;
    }

    try {
        const res = await fetch('/api/stacks/scan', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'}
        });
        const data = await res.json();
        if (data.ok) {
            alert('Scan completed successfully! Discovered ' + (data.stacks ? data.stacks.length : 0) + ' stacks.');
            refreshFleet();
        } else {
            alert('Scan failed: ' + (data.error || 'Unknown error'));
        }
    } catch (e) {
        alert('Scan request failed: ' + e.message);
    } finally {
        if (btn) {
            btn.innerHTML = origHtml;
            btn.disabled = false;
        }
    }
}

// ── Boot ──────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
    // Telegram Web App Initialization
    if (window.Telegram && window.Telegram.WebApp) {
        window.Telegram.WebApp.ready();
        window.Telegram.WebApp.expand();
    }

    initChart();
    refreshHealth();
    refreshHardware();
    refreshFleet();
    refreshActivity();
    refreshLinks();

    setInterval(refreshHealth,   5000);
    setInterval(refreshHardware, 10000);
    setInterval(refreshFleet,    8000);
    setInterval(refreshActivity, 12000);
    setInterval(refreshLinks,    60000); // Check for new routes every minute
});

async function refreshLinks() {
    const container = document.getElementById('dynamic-links-grid');
    if (!container) return;

    try {
        const res = await fetch('/api/network/routes');
        const links = await res.json();

        if (!links || links.length === 0) {
            container.innerHTML = '<div class="loading-text">No web routes discovered</div>';
            return;
        }

        container.innerHTML = links.map(link => `
            <a href="${link.url}" target="_blank" class="network-link">
                <div class="link-info">
                    <span class="link-name">${link.name}</span>
                </div>
                <div class="link-icon-container">
                    <img src="${link.icon}" alt="${link.name}" class="link-icon" onerror="this.style.opacity='0.5';">
                </div>
                <div class="link-status ${link.status === 'enabled' ? 'up' : 'down'}"></div>
            </a>
        `).join('');
    } catch (e) {
        console.error("Failed to refresh links:", e);
    }
}

function setText(id, text) {
    const el = document.getElementById(id);
    if (el) el.textContent = text;
}
