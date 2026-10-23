import { useState, useEffect } from 'react';
import { api, subscribeWS } from '../api';
import DoctorModal from '../components/DoctorModal';
import FleetMiniOverview from '../components/FleetMiniOverview';
import NetworkSummary from '../components/NetworkSummary';
import { IconDoctor, IconPrune, IconReconcile, IconRefresh } from '../components/Icons';

export default function Dashboard() {
  const [metrics, setMetrics] = useState(null);
  const [history, setHistory] = useState([]);
  const [health, setHealth] = useState(null);
  const [events, setEvents] = useState([]);
  const [error, setError] = useState(null);

  // Quick action states
  const [isDoctorOpen, setIsDoctorOpen] = useState(false);
  const [isReconciling, setIsReconciling] = useState(false);
  const [isPruning, setIsPruning] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [toast, setToast] = useState(null);

  const showToast = (message, type = 'info') => {
    setToast({ message, type });
    setTimeout(() => {
      setToast(null);
    }, 4500);
  };

  const fetchStats = async () => {
    // 1. Fetch current metrics
    const resMetrics = await api.getMetrics();
    if (resMetrics.status === 'success') {
      setMetrics(resMetrics.data);
      setError(null);
    } else {
      setError(resMetrics.error || 'Failed to fetch metrics');
    }

    // 2. Fetch history
    const resHistory = await api.getSystemMetricsHistory();
    if (resHistory.status === 'success') {
      setHistory(resHistory.data || []);
    }

    // 3. Fetch health status
    const resHealth = await api.getSystemHealth();
    if (resHealth.status === 'success') {
      setHealth(resHealth.data);
    }
  };

  useEffect(() => {
    fetchStats();
    const interval = setInterval(fetchStats, 5000);

    // 4. Subscribe to WebSocket events for live notification updates
    const sub = subscribeWS('/api/v2/ws/events', (event) => {
      const timeStr = new Date(event.timestamp * 1000).toLocaleTimeString();
      const newEvent = {
        time: timeStr,
        type: event.type,
        detail: typeof event.payload === 'object' ? JSON.stringify(event.payload) : String(event.payload),
      };
      setEvents((prev) => [newEvent, ...prev].slice(0, 20)); // Keep last 20
    });

    return () => {
      clearInterval(interval);
      sub.close();
    };
  }, []);

  const handleManualRefresh = async () => {
    setIsRefreshing(true);
    await fetchStats();
    setTimeout(() => setIsRefreshing(false), 500);
  };

  const handleReconcile = async () => {
    setIsReconciling(true);
    try {
      const res = await api.reconcileSystem();
      if (res.status === 'success') {
        const actionCount = Array.isArray(res.data?.actions) ? res.data.actions.length : 0;
        showToast(
          actionCount > 0 
            ? `System reconciled: ${actionCount} adjustment(s) applied.` 
            : 'System state verified: fully converged.',
          'success'
        );
        fetchStats();
      } else {
        showToast(res.error || 'Reconciliation failed', 'error');
      }
    } catch (err) {
      showToast(err.message, 'error');
    }
    setIsReconciling(false);
  };

  const handlePrune = async () => {
    setIsPruning(true);
    try {
      const res = await api.pruneImages();
      if (res.status === 'success') {
        const reclaimed = res.data?.reclaimed;
        let reclaimedStr = '';
        if (typeof reclaimed === 'number') {
          if (reclaimed > 1024 * 1024 * 1024) {
            reclaimedStr = ` (${(reclaimed / (1024 * 1024 * 1024)).toFixed(2)} GB freed)`;
          } else if (reclaimed > 1024 * 1024) {
            reclaimedStr = ` (${(reclaimed / (1024 * 1024)).toFixed(1)} MB freed)`;
          } else if (reclaimed > 0) {
            reclaimedStr = ` (${(reclaimed / 1024).toFixed(0)} KB freed)`;
          } else {
            reclaimedStr = ' (0 B freed)';
          }
        }
        showToast(`Image cleanup complete${reclaimedStr}.`, 'success');
        fetchStats();
      } else {
        showToast(res.error || 'Pruning failed', 'error');
      }
    } catch (err) {
      showToast(err.message, 'error');
    }
    setIsPruning(false);
  };

  const formatUptime = (seconds) => {
    if (!seconds) return '0s';
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (days > 0) return `${days}d ${hours}h ${minutes}m`;
    return `${hours}h ${minutes}m`;
  };

  const formatCap = (gb) => {
    if (!gb && gb !== 0) return '0 GB';
    if (gb >= 1000) return `${(gb / 1024).toFixed(1)} TB`;
    return `${gb.toFixed(1)} GB`;
  };

  const getUsageColor = (val) => {
    if (val >= 85) return 'bg-rose-500';
    if (val >= 60) return 'bg-amber-500';
    return 'bg-teal-400';
  };

  const getUsageGlow = (val) => {
    if (val >= 85) return 'shadow-[0_0_12px_rgba(244,63,94,0.3)]';
    if (val >= 60) return 'shadow-[0_0_12px_rgba(245,158,11,0.3)]';
    return 'shadow-[0_0_12px_rgba(45,212,191,0.3)]';
  };

  // Resilient status badge computation
  const systemStatus = typeof health?.status === 'string' && health.status.trim()
    ? health.status.toUpperCase()
    : (metrics ? 'ONLINE' : 'CONNECTING');
  const isHealthy = systemStatus === 'HEALTHY' || systemStatus === 'ONLINE' || systemStatus === 'OK';

  return (
    <div className="space-y-6">
      {/* Top Header Row with health badges & Quick Actions */}
      <div className="flex flex-col lg:flex-row justify-between lg:items-center gap-4 bg-slate-900/40 p-6 rounded-2xl border border-slate-800/60 backdrop-blur-md">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2">
            System Dashboard
          </h1>
          <p className="text-slate-400 text-sm">Real-time resource utilization and event streaming.</p>
        </div>

        <div className="flex flex-wrap items-center gap-2.5">
          {/* Quick Actions */}
          <button
            onClick={() => setIsDoctorOpen(true)}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold bg-teal-500/10 text-teal-300 border border-teal-500/30 hover:bg-teal-500/20 transition-all shadow-sm active:scale-95"
            title="Run comprehensive pre-flight health diagnostics"
          >
            <IconDoctor className="w-4 h-4 text-teal-400" />
            <span>Diagnostics</span>
          </button>

          <button
            onClick={handleReconcile}
            disabled={isReconciling}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold bg-indigo-500/10 text-indigo-300 border border-indigo-500/30 hover:bg-indigo-500/20 transition-all shadow-sm active:scale-95 disabled:opacity-50"
            title="Enforce target state across all declared services and configurations"
          >
            <IconReconcile className={`w-4 h-4 text-indigo-400 ${isReconciling ? 'animate-spin' : ''}`} />
            <span>{isReconciling ? 'Reconciling...' : 'Reconcile'}</span>
          </button>

          <button
            onClick={handlePrune}
            disabled={isPruning}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold bg-slate-800 text-slate-300 border border-slate-700 hover:bg-slate-750 transition-all shadow-sm active:scale-95 disabled:opacity-50"
            title="Prune dangling or unused container images"
          >
            <IconPrune className={`w-4 h-4 text-slate-400 ${isPruning ? 'animate-spin' : ''}`} />
            <span>{isPruning ? 'Pruning...' : 'Prune Images'}</span>
          </button>

          <button
            onClick={handleManualRefresh}
            disabled={isRefreshing}
            className="p-1.5 rounded-lg bg-slate-800/80 text-slate-400 border border-slate-700/80 hover:text-white hover:bg-slate-700 transition-all active:scale-95 disabled:opacity-50"
            title="Refresh statistics now"
          >
            <IconRefresh className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} />
          </button>

          {/* System Status Pill */}
          <div className={`ml-2 px-3.5 py-1.5 rounded-full text-xs font-semibold flex items-center gap-2 border shadow-sm ${
            isHealthy 
              ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20' 
              : 'bg-rose-500/10 text-rose-400 border-rose-500/20'
          }`}>
            <span className={`w-2 h-2 rounded-full ${
              isHealthy ? 'bg-emerald-400 animate-pulse' : 'bg-rose-400 animate-ping'
            }`}></span>
            SYSTEM: {systemStatus}
          </div>
          
          {error && (
            <div className="px-3 py-1 text-xs rounded-full bg-rose-500/10 text-rose-400 border border-rose-500/20">
              {error}
            </div>
          )}
        </div>
      </div>

      {/* Floating Action Toast */}
      {toast && (
        <div className={`p-3 rounded-xl border flex items-center justify-between shadow-lg text-xs font-medium animate-fade-in ${
          toast.type === 'success'
            ? 'bg-emerald-950/80 text-emerald-300 border-emerald-500/40'
            : toast.type === 'error'
            ? 'bg-rose-950/80 text-rose-300 border-rose-500/40'
            : 'bg-slate-900/90 text-slate-200 border-slate-700'
        }`}>
          <div className="flex items-center gap-2">
            <span className="text-sm">{toast.type === 'success' ? '✓' : toast.type === 'error' ? '⚠️' : 'ℹ️'}</span>
            <span>{toast.message}</span>
          </div>
          <button
            onClick={() => setToast(null)}
            className="text-slate-400 hover:text-white ml-4 text-xs font-bold"
          >
            ✕
          </button>
        </div>
      )}

      {metrics && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {/* CPU Card */}
          <div className="bg-slate-900 border border-slate-800/80 rounded-xl p-6 relative overflow-hidden flex flex-col justify-between">
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-slate-400 font-semibold text-sm">Processor (CPU)</span>
                <span className="text-lg font-bold text-white">{metrics.cpu_usage?.toFixed(1)}%</span>
              </div>
              {metrics.cpu_temp > 0 && (
                <div className="text-xs text-amber-400/90 mb-3 font-mono">
                  🌡️ {metrics.cpu_temp.toFixed(1)}°C ({(metrics.cpu_temp * 1.8 + 32).toFixed(1)}°F)
                </div>
              )}
              <div className="w-full bg-slate-800 rounded-full h-2 mt-2">
                <div
                  className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(metrics.cpu_usage)} ${getUsageGlow(metrics.cpu_usage)}`}
                  style={{ width: `${Math.min(100, metrics.cpu_usage)}%` }}
                ></div>
              </div>
            </div>

            {/* Sparkline History */}
            <div className="mt-6">
              <span className="text-[10px] uppercase font-bold text-slate-500 tracking-wider">CPU History (5m)</span>
              <div className="flex items-end justify-between h-14 gap-[2px] mt-2 bg-slate-950/40 p-2 rounded-lg border border-slate-800/40">
                {history.length === 0 ? (
                  <span className="text-slate-600 text-[10px] w-full text-center">Awaiting data...</span>
                ) : (
                  history.map((h, i) => (
                    <div
                      key={i}
                      className={`flex-1 rounded-t-sm transition-all duration-300 ${getUsageColor(h.cpu_usage)}`}
                      style={{ height: `${Math.max(4, h.cpu_usage)}%` }}
                      title={`CPU: ${h.cpu_usage?.toFixed(1)}%`}
                    ></div>
                  ))
                )}
              </div>
            </div>
          </div>

          {/* Memory Card */}
          <div className="bg-slate-900 border border-slate-800/80 rounded-xl p-6 relative overflow-hidden flex flex-col justify-between">
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-slate-400 font-semibold text-sm">Memory (RAM)</span>
                <span className="text-lg font-bold text-white">{metrics.memory_usage?.toFixed(1)}%</span>
              </div>
              <div className="text-xs text-slate-400 mb-3 font-mono">
                {metrics.memory_used?.toFixed(1)} / {metrics.memory_total?.toFixed(1)} GB
                {metrics.memory_frequency && metrics.memory_frequency !== 'Unknown' && ` (${metrics.memory_frequency})`}
              </div>
              <div className="w-full bg-slate-800 rounded-full h-2 mt-2">
                <div
                  className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(metrics.memory_usage)} ${getUsageGlow(metrics.memory_usage)}`}
                  style={{ width: `${Math.min(100, metrics.memory_usage)}%` }}
                ></div>
              </div>
            </div>

            {/* Sparkline History */}
            <div className="mt-6">
              <span className="text-[10px] uppercase font-bold text-slate-500 tracking-wider">RAM History (5m)</span>
              <div className="flex items-end justify-between h-14 gap-[2px] mt-2 bg-slate-950/40 p-2 rounded-lg border border-slate-800/40">
                {history.length === 0 ? (
                  <span className="text-slate-600 text-[10px] w-full text-center">Awaiting data...</span>
                ) : (
                  history.map((h, i) => (
                    <div
                      key={i}
                      className={`flex-1 rounded-t-sm transition-all duration-300 ${getUsageColor(h.memory_usage)}`}
                      style={{ height: `${Math.max(4, h.memory_usage)}%` }}
                      title={`RAM: ${h.memory_usage?.toFixed(1)}%`}
                    ></div>
                  ))
                )}
              </div>
            </div>
          </div>

          {/* GPU Card */}
          <div className="bg-slate-900 border border-slate-800/80 rounded-xl p-6 relative overflow-hidden flex flex-col justify-between">
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-slate-400 font-semibold text-sm truncate max-w-[170px]" title={metrics.gpu_model}>
                  {metrics.gpu_model && metrics.gpu_model !== 'No GPU Detected' ? metrics.gpu_model : 'GPU'}
                </span>
                <span className="text-lg font-bold text-white">{metrics.gpu_usage?.toFixed(1) || 0}%</span>
              </div>
              {metrics.gpu_temp > 0 ? (
                <div className="text-xs text-amber-400/90 mb-3 font-mono">
                  🌡️ {metrics.gpu_temp.toFixed(1)}°C ({(metrics.gpu_temp * 1.8 + 32).toFixed(1)}°F)
                </div>
              ) : (
                <div className="text-xs text-slate-500 mb-3 font-mono">Graphics Processor</div>
              )}
              
              <div className="w-full bg-slate-800 rounded-full h-2 mt-2">
                <div
                  className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(metrics.gpu_usage || 0)}`}
                  style={{ width: `${Math.min(100, metrics.gpu_usage || 0)}%` }}
                ></div>
              </div>
            </div>

            {/* VRAM Details */}
            <div className="mt-6 bg-slate-950/40 p-3 rounded-lg border border-slate-800/40">
              <div className="flex justify-between text-xs text-slate-400 mb-1.5 font-mono">
                <span>VRAM</span>
                <span>
                  {metrics.gpu_mem_used?.toFixed(0) || 0} / {metrics.gpu_mem_total?.toFixed(0) || 0} MB
                </span>
              </div>
              <div className="w-full bg-slate-800 rounded-full h-1.5">
                <div
                  className="h-1.5 rounded-full bg-indigo-400 transition-all duration-500"
                  style={{
                    width: `${Math.min(100, metrics.gpu_mem_total > 0 ? ((metrics.gpu_mem_used / metrics.gpu_mem_total) * 100) : 0)}%`,
                  }}
                ></div>
              </div>
            </div>
          </div>

          {/* Primary Disk Card */}
          <div className="bg-slate-900 border border-slate-800/80 rounded-xl p-6 relative overflow-hidden flex flex-col justify-between">
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-slate-400 font-semibold text-sm">System Storage (/)</span>
                <span className="text-lg font-bold text-white">{metrics.disk_usage?.toFixed(1)}%</span>
              </div>
              <div className="text-xs text-slate-400 mb-3 font-mono">
                {metrics.disk_used?.toFixed(1)} / {metrics.disk_total?.toFixed(1)} GB
              </div>
              <div className="w-full bg-slate-800 rounded-full h-2 mt-2">
                <div
                  className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(metrics.disk_usage)} ${getUsageGlow(metrics.disk_usage)}`}
                  style={{ width: `${Math.min(100, metrics.disk_usage)}%` }}
                ></div>
              </div>
            </div>

            {/* Sparkline History */}
            <div className="mt-6">
              <span className="text-[10px] uppercase font-bold text-slate-500 tracking-wider">Disk History (5m)</span>
              <div className="flex items-end justify-between h-14 gap-[2px] mt-2 bg-slate-950/40 p-2 rounded-lg border border-slate-800/40">
                {history.length === 0 ? (
                  <span className="text-slate-600 text-[10px] w-full text-center">Awaiting data...</span>
                ) : (
                  history.map((h, i) => (
                    <div
                      key={i}
                      className={`flex-1 rounded-t-sm transition-all duration-300 ${getUsageColor(h.disk_usage)}`}
                      style={{ height: `${Math.max(4, h.disk_usage)}%` }}
                      title={`Disk: ${h.disk_usage?.toFixed(1)}%`}
                    ></div>
                  ))
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Storage & Mounted Drives Panel (disk1, disk2, network shares, pools) */}
      {metrics && metrics.disk_partitions && metrics.disk_partitions.length > 0 && (
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider flex items-center gap-2">
              <span>💿</span> Storage & Mounted Drives
            </h2>
            <span className="text-xs text-slate-500 font-mono">{metrics.disk_partitions.length} Volumes Mounted</span>
          </div>
          <div className="border-t border-slate-800/60 pt-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {metrics.disk_partitions.map((p, idx) => {
              const isNet = p.device?.startsWith('//') || p.device?.startsWith('\\\\') || p.fstype === 'cifs' || p.fstype?.startsWith('nfs');
              const label = p.label && !p.device?.includes(p.label) ? p.label : p.mountpoint;
              const isCriticalSpace = p.used_percent >= 90;
              const isWarningSpace = p.used_percent >= 80 && p.used_percent < 90;

              return (
                <div 
                  key={idx} 
                  className={`bg-slate-950/60 border rounded-lg p-4 space-y-2 relative transition-all ${
                    isCriticalSpace 
                      ? 'border-rose-500/40 ring-1 ring-rose-500/20 shadow-[0_0_15px_rgba(244,63,94,0.1)]' 
                      : isWarningSpace 
                      ? 'border-amber-500/30' 
                      : 'border-slate-800/60'
                  }`}
                >
                  <div className="flex justify-between items-start">
                    <div>
                      <div className="font-semibold text-white text-sm flex items-center gap-2 flex-wrap">
                        <span>{label}</span>
                        {isNet ? (
                          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-sky-500/10 text-sky-400 border border-sky-500/20">
                            NETWORK
                          </span>
                        ) : (
                          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-slate-800 text-slate-400 border border-slate-700">
                            LOCAL
                          </span>
                        )}
                        {isCriticalSpace && (
                          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-500/15 text-rose-400 border border-rose-500/30 animate-pulse">
                            CRITICAL
                          </span>
                        )}
                        {isWarningSpace && (
                          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-500/15 text-amber-400 border border-amber-500/30">
                            LOW SPACE
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-slate-500 font-mono mt-0.5">
                        {p.mountpoint} • {p.fstype?.toUpperCase()}
                      </div>
                    </div>
                    <div className="text-right">
                      <span className={`font-bold text-sm ${
                        isCriticalSpace ? 'text-rose-400' : isWarningSpace ? 'text-amber-400' : 'text-white'
                      }`}>
                        {p.used_percent?.toFixed(1)}%
                      </span>
                      <div className="text-xs text-slate-400 font-mono">
                        {formatCap(p.used)} / {formatCap(p.total)}
                      </div>
                    </div>
                  </div>
                  <div className="w-full bg-slate-800 rounded-full h-2">
                    <div
                      className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(p.used_percent)}`}
                      style={{ width: `${Math.min(100, p.used_percent)}%` }}
                    ></div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Fleet & Network Operational Widgets */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <FleetMiniOverview />
        <NetworkSummary />
      </div>

      {metrics && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Host Info */}
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 lg:col-span-1 space-y-4">
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider">Host Information</h2>
            <div className="border-t border-slate-800/60 pt-3">
              <div className="flex justify-between text-sm py-1.5">
                <span className="text-slate-500">Hostname</span>
                <span className="text-white font-medium">{metrics.hostname}</span>
              </div>
              <div className="flex justify-between text-sm py-1.5">
                <span className="text-slate-500">OS Platform</span>
                <span className="text-white font-medium">Linux</span>
              </div>
              <div className="flex justify-between text-sm py-1.5">
                <span className="text-slate-500">System Uptime</span>
                <span className="text-white font-medium">{formatUptime(metrics.uptime)}</span>
              </div>
              <div className="flex justify-between text-sm py-1.5">
                <span className="text-slate-500">API Status</span>
                <span className="text-emerald-400 font-medium flex items-center gap-1.5">
                  <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
                  Online
                </span>
              </div>
            </div>

            {/* Health Checklist */}
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider pt-4">Subsystem Health</h2>
            <div className="border-t border-slate-800/60 pt-3 space-y-2.5">
              {health && health.components ? (
                Object.entries(health.components).map(([name, status]) => (
                  <div key={name} className="flex justify-between items-center text-sm">
                    <span className="text-slate-500 capitalize">{name}</span>
                    <div className="flex items-center gap-2">
                      {status === 'unhealthy' && health.details && health.details[name] && (
                        <span className="text-[10px] text-rose-400 max-w-[150px] truncate" title={health.details[name]}>
                          {health.details[name]}
                        </span>
                      )}
                      <span className={`px-2.5 py-0.5 rounded text-[10px] font-bold ${
                        status === 'healthy' ? 'bg-emerald-500/10 text-emerald-400' : 'bg-rose-500/10 text-rose-400'
                      }`}>
                        {String(status).toUpperCase()}
                      </span>
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-xs text-slate-500 animate-pulse">Loading components status...</div>
              )}
            </div>
          </div>

          {/* Event Stream */}
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 lg:col-span-2 flex flex-col h-[380px]">
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-4">Live System Event Log</h2>
            <div className="flex-1 overflow-y-auto space-y-2 pr-2 text-xs font-mono">
              {events.length === 0 ? (
                <div className="flex h-full items-center justify-center text-slate-600 italic">
                  Waiting for system events (container states, stack updates, AI jobs)...
                </div>
              ) : (
                events.map((ev, i) => (
                  <div key={i} className="flex gap-4 p-2 bg-slate-950/40 rounded border border-slate-800/40">
                    <span className="text-slate-500">{ev.time}</span>
                    <span className="text-amber-400 font-semibold">{ev.type}</span>
                    <span className="text-slate-300 break-all">{ev.detail}</span>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      )}

      {/* Doctor Diagnostics Modal */}
      <DoctorModal isOpen={isDoctorOpen} onClose={() => setIsDoctorOpen(false)} />
    </div>
  );
}
