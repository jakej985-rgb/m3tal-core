import { useState, useEffect, useMemo } from 'react';
import { api, subscribeWS } from '../api';
import DoctorModal from '../components/DoctorModal';
import FleetMiniOverview from '../components/FleetMiniOverview';
import NetworkSummary from '../components/NetworkSummary';
import { IconDoctor, IconClean, IconReconcile, IconRefresh, IconShield, IconServer, IconChevronDown } from '../components/Icons';

export default function Dashboard() {
  const [metrics, setMetrics] = useState(null);
  const [history, setHistory] = useState([]);
  const [health, setHealth] = useState(null);
  const [events, setEvents] = useState([]);
  const [error, setError] = useState(null);

  // Quick action and header expansion states
  const [isDoctorOpen, setIsDoctorOpen] = useState(false);
  const [isReconciling, setIsReconciling] = useState(false);
  const [isCleaning, setIsCleaning] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isSystemInfoExpanded, setIsSystemInfoExpanded] = useState(false);
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

  const handleSystemCleanup = async () => {
    setIsCleaning(true);
    try {
      const res = await api.systemCleanup();
      if (res.status === 'success') {
        const freed = res.data?.freed_human || '0 B';
        const count = res.data?.items_cleaned?.length || 0;
        showToast(`System cleanup complete (${freed} freed across ${count} tasks).`, 'success');
        fetchStats();
      } else {
        showToast(res.error || 'System cleanup failed', 'error');
      }
    } catch (err) {
      showToast(err.message, 'error');
    }
    setIsCleaning(false);
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

  // Categorize storage into System (top row), USB/Removable, Network, and Internal drives
  const storageGroups = useMemo(() => {
    if (!metrics?.disk_partitions) return { system: [], usb: [], network: [], internal: [], totalDrives: 0, totalVolumes: 0 };

    const groups = new Map();

    metrics.disk_partitions.forEach((p) => {
      const isNet = p.device?.startsWith('//') || p.device?.startsWith('\\\\') || p.fstype === 'cifs' || p.fstype?.startsWith('nfs') || p.drive_type === 'network';
      const groupKey = isNet ? `net:${p.mountpoint || p.device}` : (p.disk_name ? `disk:${p.disk_name}` : `dev:${p.device}`);

      if (!groups.has(groupKey)) {
        groups.set(groupKey, {
          key: groupKey,
          isNet,
          diskName: p.disk_name || '',
          diskModel: p.disk_model || '',
          partitions: [],
        });
      }
      groups.get(groupKey).partitions.push(p);
    });

    const system = [];
    const usb = [];
    const network = [];
    const internal = [];

    groups.forEach((g) => {
      const totalGB = g.partitions.reduce((acc, p) => acc + (p.total || 0), 0);
      const usedGB = g.partitions.reduce((acc, p) => acc + (p.used || 0), 0);
      const freeGB = g.partitions.reduce((acc, p) => acc + (p.free || 0), 0);
      const usedPercent = totalGB > 0 ? (usedGB / totalGB) * 100 : 0;

      const isCriticalSpace = g.partitions.some((p) => p.used_percent >= 90) || usedPercent >= 90;
      const isWarningSpace = !isCriticalSpace && (g.partitions.some((p) => p.used_percent >= 80) || usedPercent >= 80);

      const isSystemDrive = g.partitions.some((p) => p.drive_type === 'system' || p.mountpoint === '/' || p.mountpoint === '/home' || p.mountpoint === '/tmp');
      const isUsbDrive = !isSystemDrive && (g.partitions.some((p) => p.drive_type === 'usb' || p.mountpoint?.startsWith('/media/') || p.mountpoint?.startsWith('/run/media/')));
      const isNetDrive = g.isNet || g.partitions.some((p) => p.drive_type === 'network');

      let driveCategory = 'internal';
      if (isSystemDrive) driveCategory = 'system';
      else if (isUsbDrive) driveCategory = 'usb';
      else if (isNetDrive) driveCategory = 'network';

      let title = '';
      let subtitle = '';

      if (driveCategory === 'network') {
        const p = g.partitions[0];
        title = p.label && !p.device?.includes(p.label) ? p.label : p.mountpoint;
        subtitle = `${p.mountpoint} • ${p.fstype?.toUpperCase()}`;
      } else if (driveCategory === 'system') {
        title = g.diskModel ? `System Drive • ${g.diskModel} (${g.diskName})` : `System Drive (${g.diskName})`;
        subtitle = `${g.partitions.length} Linux System Partitions (/, /home, /tmp) • ${g.diskName}`;
      } else if (driveCategory === 'usb') {
        const p = g.partitions[0];
        title = p.label && !p.device?.includes(p.label) ? p.label : (g.diskModel || g.diskName || 'USB Drive');
        subtitle = `${p.mountpoint} • ${p.fstype?.toUpperCase()} • ${g.diskName}`;
      } else {
        const p = g.partitions[0];
        title = p.label && !p.device?.includes(p.label) ? p.label : (g.diskModel || p.mountpoint);
        subtitle = `${p.mountpoint} • ${p.fstype?.toUpperCase()} • ${g.diskName}`;
      }

      const cardData = {
        ...g,
        title,
        subtitle,
        totalGB,
        usedGB,
        freeGB,
        usedPercent,
        isCriticalSpace,
        isWarningSpace,
        driveCategory,
      };

      if (driveCategory === 'system') system.push(cardData);
      else if (driveCategory === 'usb') usb.push(cardData);
      else if (driveCategory === 'network') network.push(cardData);
      else internal.push(cardData);
    });

    return {
      system,
      usb,
      network,
      internal,
      totalDrives: groups.size,
      totalVolumes: metrics.disk_partitions.length,
    };
  }, [metrics?.disk_partitions]);

  // Resilient status badge computation
  const systemStatus = typeof health?.status === 'string' && health.status.trim()
    ? health.status.toUpperCase()
    : (metrics ? 'ONLINE' : 'CONNECTING');
  const isHealthy = systemStatus === 'HEALTHY' || systemStatus === 'ONLINE' || systemStatus === 'OK';

  // Subsystem health aggregation
  const subsystemEntries = useMemo(() => {
    if (!health?.components) return [];
    return Object.entries(health.components);
  }, [health?.components]);

  const totalSubsystems = subsystemEntries.length;
  const healthySubsystems = subsystemEntries.filter(
    ([_, status]) => String(status).toLowerCase() === 'healthy' || String(status).toLowerCase() === 'ok'
  ).length;

  const healthPercentage = totalSubsystems > 0 
    ? Math.round((healthySubsystems / totalSubsystems) * 100) 
    : (isHealthy ? 100 : 0);

  return (
    <div className="space-y-6">
      {/* Top Header Card with expandable Host Info & Subsystem Health */}
      <div className="bg-slate-900/40 p-6 rounded-2xl border border-slate-800/60 backdrop-blur-md space-y-4">
        <div className="flex flex-col lg:flex-row justify-between lg:items-center gap-4">
          <div className="space-y-2">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2">
                System Dashboard
              </h1>
              <p className="text-slate-400 text-sm">Real-time resource utilization and event streaming.</p>
            </div>

            {/* Collapsed Status Summary: Host Uptime & Subsystem Health */}
            <div className="flex flex-wrap items-center gap-2 pt-0.5">
              {/* Host & Uptime */}
              <div className="flex items-center gap-1.5 px-3 py-1 rounded-lg bg-slate-800/80 border border-slate-700/80 text-xs text-slate-300 shadow-sm">
                <IconServer className="w-3.5 h-3.5 text-indigo-400" />
                <span className="font-semibold text-white">{metrics?.hostname || 'Host'}</span>
                <span className="text-slate-500">•</span>
                <span className="text-emerald-400 font-mono">Up {formatUptime(metrics?.uptime)}</span>
              </div>

              {/* Subsystem Health with Icon & Percentage */}
              <div className={`flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold border shadow-sm ${
                healthPercentage >= 90
                  ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                  : healthPercentage >= 60
                  ? 'bg-amber-500/10 text-amber-400 border-amber-500/30'
                  : 'bg-rose-500/10 text-rose-400 border-rose-500/30'
              }`}>
                <span className={`w-1.5 h-1.5 rounded-full ${
                  healthPercentage >= 90 ? 'bg-emerald-400 animate-pulse' : 'bg-amber-400 animate-ping'
                }`} />
                <IconShield className="w-3.5 h-3.5" />
                <span>{healthPercentage}% Health</span>
                <span className="opacity-75 font-normal">({healthySubsystems}/{totalSubsystems || 1} OK)</span>
              </div>

              {/* Expand / Collapse Toggle Button */}
              <button
                onClick={() => setIsSystemInfoExpanded(!isSystemInfoExpanded)}
                className={`flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold transition-all border shadow-sm active:scale-95 ${
                  isSystemInfoExpanded
                    ? 'bg-indigo-500/20 text-indigo-300 border-indigo-500/40'
                    : 'bg-slate-800 hover:bg-slate-750 text-slate-300 border-slate-700 hover:text-white'
                }`}
                title={isSystemInfoExpanded ? "Collapse Host Info & Subsystem Health" : "Expand to view all Host Info & Subsystem Health details"}
              >
                <span>{isSystemInfoExpanded ? 'Collapse' : 'Expand Info'}</span>
                <IconChevronDown className={`w-3.5 h-3.5 transition-transform duration-200 ${isSystemInfoExpanded ? 'rotate-180 text-indigo-400' : 'text-slate-400'}`} />
              </button>
            </div>
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
              onClick={handleSystemCleanup}
              disabled={isCleaning}
              className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold bg-slate-800 text-slate-300 border border-slate-700 hover:bg-slate-750 hover:text-white transition-all shadow-sm active:scale-95 disabled:opacity-50"
              title="Clean APT cache, unused Flatpaks, old Snaps, and temporary caches"
            >
              <IconClean className={`w-4 h-4 text-emerald-400 ${isCleaning ? 'animate-spin' : ''}`} />
              <span>{isCleaning ? 'Cleaning...' : 'Clean Up'}</span>
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

        {/* Expandable Details Drawer: Full Host Info & Subsystem Health Checklist */}
        {isSystemInfoExpanded && (
          <div className="border-t border-slate-800/80 pt-5 mt-4 grid grid-cols-1 lg:grid-cols-2 gap-6 animate-fade-in">
            {/* Host System Information */}
            <div className="bg-slate-950/40 rounded-xl p-4 border border-slate-800/80 space-y-3">
              <div className="flex items-center justify-between pb-2 border-b border-slate-800/60">
                <h3 className="text-xs font-semibold text-slate-300 uppercase tracking-wider flex items-center gap-2">
                  <IconServer className="w-4 h-4 text-indigo-400" />
                  Host System Details
                </h3>
                <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-300 border border-indigo-500/20 font-bold">
                  LOCAL NODE
                </span>
              </div>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 text-xs">
                <div>
                  <span className="text-slate-500 block text-[11px]">Hostname</span>
                  <span className="text-white font-medium">{metrics?.hostname || 'localhost'}</span>
                </div>
                <div>
                  <span className="text-slate-500 block text-[11px]">OS Platform</span>
                  <span className="text-white font-medium">Linux</span>
                </div>
                <div>
                  <span className="text-slate-500 block text-[11px]">Total Uptime</span>
                  <span className="text-emerald-400 font-medium font-mono">{formatUptime(metrics?.uptime)}</span>
                </div>
                <div>
                  <span className="text-slate-500 block text-[11px]">API Control Plane</span>
                  <span className="text-emerald-400 font-medium flex items-center gap-1.5">
                    <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                    Online (v2.0)
                  </span>
                </div>
                <div>
                  <span className="text-slate-500 block text-[11px]">Processor Usage</span>
                  <span className="text-slate-200 font-medium font-mono">{metrics?.cpu_usage?.toFixed(1) || 0}%</span>
                </div>
                <div>
                  <span className="text-slate-500 block text-[11px]">Memory Usage</span>
                  <span className="text-slate-200 font-medium font-mono">{metrics?.memory_percent?.toFixed(1) || 0}%</span>
                </div>
              </div>
            </div>

            {/* Subsystem Health Checklist */}
            <div className="bg-slate-950/40 rounded-xl p-4 border border-slate-800/80 space-y-3">
              <div className="flex items-center justify-between pb-2 border-b border-slate-800/60">
                <h3 className="text-xs font-semibold text-slate-300 uppercase tracking-wider flex items-center gap-2">
                  <IconShield className="w-4 h-4 text-emerald-400" />
                  Subsystem Health
                </h3>
                <span className={`text-[10px] font-mono px-2 py-0.5 rounded font-bold border ${
                  healthPercentage >= 90
                    ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                    : 'bg-amber-500/10 text-amber-400 border-amber-500/30'
                }`}>
                  {healthPercentage}% HEALTHY ({healthySubsystems}/{totalSubsystems || 1})
                </span>
              </div>

              <div className="space-y-2 max-h-48 overflow-y-auto pr-1">
                {subsystemEntries.length > 0 ? (
                  subsystemEntries.map(([name, status]) => {
                    const isSubHealthy = String(status).toLowerCase() === 'healthy' || String(status).toLowerCase() === 'ok';
                    return (
                      <div key={name} className="flex justify-between items-center text-xs py-1.5 border-b border-slate-900/60 last:border-0">
                        <span className="text-slate-300 capitalize font-medium">{name}</span>
                        <div className="flex items-center gap-2">
                          {!isSubHealthy && health?.details?.[name] && (
                            <span className="text-[10px] text-rose-400 max-w-[180px] truncate" title={health.details[name]}>
                              {health.details[name]}
                            </span>
                          )}
                          <span className={`px-2.5 py-0.5 rounded text-[10px] font-bold ${
                            isSubHealthy ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                          }`}>
                            {String(status).toUpperCase()}
                          </span>
                        </div>
                      </div>
                    );
                  })
                ) : (
                  <div className="text-xs text-slate-500 italic py-2">
                    All core system subsystems operational.
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
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

      {/* Storage & Mounted Drives Panel */}
      {storageGroups && storageGroups.totalVolumes > 0 && (
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 space-y-6">
          <div className="flex justify-between items-center">
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider flex items-center gap-2">
              <span>💿</span> Storage & Mounted Drives
            </h2>
            <span className="text-xs text-slate-500 font-mono">
              {storageGroups.totalDrives} {storageGroups.totalDrives === 1 ? 'Drive' : 'Drives'} ({storageGroups.totalVolumes} Volumes)
            </span>
          </div>

          {/* 1. TOP ROW: SYSTEM STORAGE (LOCKED ACROSS TOP ROW) */}
          {storageGroups.system.map((card, idx) => (
            <div key={card.key || idx} className="space-y-2">
              <div className="text-xs font-semibold text-slate-400 flex items-center gap-2 uppercase tracking-wider">
                <span>💻</span> Host System Storage
              </div>
              <div className={`bg-slate-950/70 border rounded-xl p-5 space-y-4 relative transition-all ${
                card.isCriticalSpace
                  ? 'border-rose-500/40 ring-1 ring-rose-500/20'
                  : card.isWarningSpace
                  ? 'border-amber-500/30'
                  : 'border-slate-800/80 shadow-sm'
              }`}>
                {/* Header */}
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-slate-800/60 pb-3">
                  <div className="flex items-center gap-2.5 flex-wrap">
                    <span className="font-bold text-white text-base">{card.title}</span>
                    <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                      SYSTEM
                    </span>
                    <span className="px-1.5 py-0.5 rounded text-[10px] font-semibold bg-indigo-500/15 text-indigo-300 border border-indigo-500/30">
                      {card.partitions.length} PARTITIONS
                    </span>
                    {card.isCriticalSpace && (
                      <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-500/15 text-rose-400 border border-rose-500/30 animate-pulse">
                        CRITICAL
                      </span>
                    )}
                    {card.isWarningSpace && (
                      <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-500/15 text-amber-400 border border-amber-500/30">
                        LOW SPACE
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-3 font-mono text-xs">
                    <span className="text-slate-400">Total: {formatCap(card.usedGB)} / {formatCap(card.totalGB)}</span>
                    <span className={`font-bold text-sm ${card.isCriticalSpace ? 'text-rose-400' : card.isWarningSpace ? 'text-amber-400' : 'text-emerald-400'}`}>
                      {card.usedPercent?.toFixed(1)}%
                    </span>
                  </div>
                </div>

                {/* System partitions displayed side-by-side in responsive columns */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-3.5">
                  {card.partitions.map((p, pIdx) => {
                    const isPartCritical = p.used_percent >= 90;
                    const isPartWarning = p.used_percent >= 80 && p.used_percent < 90;
                    const partLabel = p.mountpoint === '/' ? 'Root (/)' : p.mountpoint === '/home' ? 'Home (/home)' : p.mountpoint === '/tmp' ? 'Temp (/tmp)' : p.mountpoint;

                    return (
                      <div key={pIdx} className="bg-slate-900/80 border border-slate-800/70 rounded-lg p-3.5 space-y-2 hover:border-slate-700 transition-colors">
                        <div className="flex justify-between items-start">
                          <div>
                            <div className="font-semibold text-slate-200 text-xs flex items-center gap-1.5">
                              <span className="text-white font-mono">{partLabel}</span>
                              {p.label && p.label !== p.mountpoint && (
                                <span className="text-[10px] text-slate-400 font-normal">({p.label})</span>
                              )}
                            </div>
                            <div className="text-[11px] text-slate-500 font-mono mt-0.5">
                              {p.device ? p.device.replace('/dev/', '') : ''} • {p.fstype?.toUpperCase()}
                            </div>
                          </div>
                          <div className="text-right font-mono">
                            <div className={`text-xs font-bold ${isPartCritical ? 'text-rose-400' : isPartWarning ? 'text-amber-400' : 'text-slate-200'}`}>
                              {p.used_percent?.toFixed(1)}%
                            </div>
                            <div className="text-[10px] text-slate-400">
                              {formatCap(p.used)} / {formatCap(p.total)}
                            </div>
                          </div>
                        </div>
                        <div className="w-full bg-slate-800 rounded-full h-1.5 overflow-hidden">
                          <div
                            className={`h-1.5 rounded-full transition-all duration-500 ${getUsageColor(p.used_percent)}`}
                            style={{ width: `${Math.min(100, p.used_percent)}%` }}
                          ></div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          ))}

          {/* 2. REMOVABLE & USB STORAGE (DEDICATED SECTION) */}
          {storageGroups.usb.length > 0 && (
            <div className="space-y-2 pt-2">
              <div className="text-xs font-semibold text-slate-400 flex items-center gap-2 uppercase tracking-wider">
                <span>🔌</span> Removable & USB Drives
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {storageGroups.usb.map((card, idx) => (
                  <div key={card.key || idx} className={`bg-slate-950/60 border rounded-lg p-4 space-y-3 relative transition-all ${
                    card.isCriticalSpace ? 'border-rose-500/40 ring-1 ring-rose-500/20' : card.isWarningSpace ? 'border-amber-500/30' : 'border-slate-800/60'
                  }`}>
                    <div className="flex justify-between items-start">
                      <div className="min-w-0 pr-2">
                        <div className="font-semibold text-white text-sm flex items-center gap-2 flex-wrap">
                          <span className="truncate max-w-[180px]" title={card.title}>{card.title}</span>
                          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-500/10 text-amber-400 border border-amber-500/20">
                            USB
                          </span>
                          {card.isCriticalSpace && (
                            <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-500/15 text-rose-400 border border-rose-500/30 animate-pulse">
                              CRITICAL
                            </span>
                          )}
                        </div>
                        <div className="text-xs text-slate-500 font-mono mt-0.5 truncate" title={card.subtitle}>
                          {card.subtitle}
                        </div>
                      </div>
                      <div className="text-right shrink-0">
                        <span className={`font-bold text-sm ${card.isCriticalSpace ? 'text-rose-400' : card.isWarningSpace ? 'text-amber-400' : 'text-white'}`}>
                          {card.usedPercent?.toFixed(1)}%
                        </span>
                        <div className="text-xs text-slate-400 font-mono">
                          {formatCap(card.usedGB)} / {formatCap(card.totalGB)}
                        </div>
                      </div>
                    </div>
                    <div className="w-full bg-slate-800 rounded-full h-2 overflow-hidden">
                      <div
                        className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(card.usedPercent)}`}
                        style={{ width: `${Math.min(100, card.usedPercent)}%` }}
                      ></div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* 3. NETWORK STORAGE (LOCKED TO DEDICATED ROW) */}
          {storageGroups.network.length > 0 && (
            <div className="space-y-2 pt-2">
              <div className="text-xs font-semibold text-slate-400 flex items-center gap-2 uppercase tracking-wider">
                <span>🌐</span> Network Storage
              </div>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {storageGroups.network.map((card, idx) => (
                  <div key={card.key || idx} className={`bg-slate-950/60 border rounded-lg p-4 space-y-3 relative transition-all ${
                    card.isCriticalSpace ? 'border-rose-500/40 ring-1 ring-rose-500/20 shadow-[0_0_15px_rgba(244,63,94,0.1)]' : card.isWarningSpace ? 'border-amber-500/30' : 'border-slate-800/60'
                  }`}>
                    <div className="flex justify-between items-start">
                      <div className="min-w-0 pr-2">
                        <div className="font-semibold text-white text-sm flex items-center gap-2 flex-wrap">
                          <span className="truncate max-w-[180px]" title={card.title}>{card.title}</span>
                          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-sky-500/10 text-sky-400 border border-sky-500/20">
                            NETWORK
                          </span>
                          {card.isCriticalSpace && (
                            <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-500/15 text-rose-400 border border-rose-500/30 animate-pulse">
                              CRITICAL
                            </span>
                          )}
                          {card.isWarningSpace && (
                            <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-500/15 text-amber-400 border border-amber-500/30">
                              LOW SPACE
                            </span>
                          )}
                        </div>
                        <div className="text-xs text-slate-500 font-mono mt-0.5 truncate" title={card.subtitle}>
                          {card.subtitle}
                        </div>
                      </div>
                      <div className="text-right shrink-0">
                        <span className={`font-bold text-sm ${card.isCriticalSpace ? 'text-rose-400' : card.isWarningSpace ? 'text-amber-400' : 'text-white'}`}>
                          {card.usedPercent?.toFixed(1)}%
                        </span>
                        <div className="text-xs text-slate-400 font-mono">
                          {formatCap(card.usedGB)} / {formatCap(card.totalGB)}
                        </div>
                      </div>
                    </div>
                    <div className="w-full bg-slate-800 rounded-full h-2 overflow-hidden">
                      <div
                        className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(card.usedPercent)}`}
                        style={{ width: `${Math.min(100, card.usedPercent)}%` }}
                      ></div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* 4. OTHER INTERNAL STORAGE (if any) */}
          {storageGroups.internal.length > 0 && (
            <div className="space-y-2 pt-2">
              <div className="text-xs font-semibold text-slate-400 flex items-center gap-2 uppercase tracking-wider">
                <span>💽</span> Internal Storage
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {storageGroups.internal.map((card, idx) => (
                  <div key={card.key || idx} className="bg-slate-950/60 border border-slate-800/60 rounded-lg p-4 space-y-3">
                    <div className="flex justify-between items-start">
                      <div className="min-w-0 pr-2">
                        <div className="font-semibold text-white text-sm flex items-center gap-2 flex-wrap">
                          <span className="truncate max-w-[180px]" title={card.title}>{card.title}</span>
                          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-slate-800 text-slate-400 border border-slate-700">
                            LOCAL
                          </span>
                        </div>
                        <div className="text-xs text-slate-500 font-mono mt-0.5 truncate" title={card.subtitle}>
                          {card.subtitle}
                        </div>
                      </div>
                      <div className="text-right shrink-0">
                        <span className="font-bold text-sm text-white">{card.usedPercent?.toFixed(1)}%</span>
                        <div className="text-xs text-slate-400 font-mono">
                          {formatCap(card.usedGB)} / {formatCap(card.totalGB)}
                        </div>
                      </div>
                    </div>
                    <div className="w-full bg-slate-800 rounded-full h-2 overflow-hidden">
                      <div
                        className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(card.usedPercent)}`}
                        style={{ width: `${Math.min(100, card.usedPercent)}%` }}
                      ></div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Fleet & Network Operational Widgets */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <FleetMiniOverview />
        <NetworkSummary />
      </div>

      {/* Full-Width Live System Event Log */}
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 flex flex-col h-[380px]">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider flex items-center gap-2">
            Live System Event Log
          </h2>
          <span className="text-xs text-slate-500 font-mono">
            {events.length} {events.length === 1 ? 'event' : 'events'} recorded
          </span>
        </div>
        <div className="flex-1 overflow-y-auto space-y-2 pr-2 text-xs font-mono">
          {events.length === 0 ? (
            <div className="flex h-full items-center justify-center text-slate-600 italic">
              Waiting for system events (service states, system reconciliations, hardware alerts)...
            </div>
          ) : (
            events.map((ev, i) => (
              <div key={i} className="flex gap-4 p-2 bg-slate-950/40 rounded border border-slate-800/40 hover:bg-slate-800/20 transition-colors">
                <span className="text-slate-500 whitespace-nowrap">{ev.time}</span>
                <span className="text-amber-400 font-semibold whitespace-nowrap">{ev.type}</span>
                <span className="text-slate-300 break-all">{ev.detail}</span>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Doctor Diagnostics Modal */}
      <DoctorModal isOpen={isDoctorOpen} onClose={() => setIsDoctorOpen(false)} />
    </div>
  );
}
