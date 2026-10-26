import { useState } from 'react';
import {
  IconDoctor,
  IconClean,
  IconReconcile,
  IconRefresh,
  IconShield,
  IconServer,
  IconChevronDown,
} from '../Icons';

export default function TopBarWidget({
  metrics,
  health,
  onOpenDoctor,
  onReconcile,
  isReconciling,
  onCleanup,
  isCleaning,
  onRefresh,
  isRefreshing,
  error,
}) {
  const [isExpanded, setIsExpanded] = useState(false);

  const formatUptime = (seconds) => {
    if (!seconds) return '0s';
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (days > 0) return `${days}d ${hours}h ${minutes}m`;
    return `${hours}h ${minutes}m`;
  };

  const systemStatus = typeof health?.status === 'string' && health.status.trim()
    ? health.status.toUpperCase()
    : (metrics ? 'ONLINE' : 'CONNECTING');
  const isHealthy = systemStatus === 'HEALTHY' || systemStatus === 'ONLINE' || systemStatus === 'OK';

  const subsystemEntries = health?.components ? Object.entries(health.components) : [];
  const totalSubsystems = subsystemEntries.length;
  const healthySubsystems = subsystemEntries.filter(
    (entry) => String(entry[1]).toLowerCase() === 'healthy' || String(entry[1]).toLowerCase() === 'ok'
  ).length;

  const healthPercentage = totalSubsystems > 0
    ? Math.round((healthySubsystems / totalSubsystems) * 100)
    : (isHealthy ? 100 : 0);

  return (
    <div className="col-span-1 md:col-span-2 lg:col-span-4 bg-slate-900/40 p-6 rounded-2xl border border-slate-800/60 backdrop-blur-md space-y-4 shadow-sm">
      <div className="flex flex-col lg:flex-row justify-between lg:items-center gap-4">
        <div className="space-y-2">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2">
              System Dashboard
            </h1>
            <p className="text-slate-400 text-sm">
              Modular 4xN telemetry grid and event streaming control plane.
            </p>
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
              onClick={() => setIsExpanded(!isExpanded)}
              className={`flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold transition-all border shadow-sm active:scale-95 ${
                isExpanded
                  ? 'bg-indigo-500/20 text-indigo-300 border-indigo-500/40'
                  : 'bg-slate-800 hover:bg-slate-750 text-slate-300 border-slate-700 hover:text-white'
              }`}
              title={isExpanded ? "Collapse Host Info & Subsystem Health" : "Expand to view Host Info & Subsystem Health details"}
            >
              <span>{isExpanded ? 'Collapse' : 'Expand Info'}</span>
              <IconChevronDown className={`w-3.5 h-3.5 transition-transform duration-200 ${isExpanded ? 'rotate-180 text-indigo-400' : 'text-slate-400'}`} />
            </button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2.5">
          {/* Quick Actions */}
          <button
            onClick={onOpenDoctor}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold bg-teal-500/10 text-teal-300 border border-teal-500/30 hover:bg-teal-500/20 transition-all shadow-sm active:scale-95"
            title="Run comprehensive pre-flight health diagnostics"
          >
            <IconDoctor className="w-4 h-4 text-teal-400" />
            <span>Diagnostics</span>
          </button>

          <button
            onClick={onReconcile}
            disabled={isReconciling}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold bg-indigo-500/10 text-indigo-300 border border-indigo-500/30 hover:bg-indigo-500/20 transition-all shadow-sm active:scale-95 disabled:opacity-50"
            title="Enforce target state across all declared services and configurations"
          >
            <IconReconcile className={`w-4 h-4 text-indigo-400 ${isReconciling ? 'animate-spin' : ''}`} />
            <span>{isReconciling ? 'Reconciling...' : 'Reconcile'}</span>
          </button>

          <button
            onClick={onCleanup}
            disabled={isCleaning}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold bg-slate-800 text-slate-300 border border-slate-700 hover:bg-slate-750 hover:text-white transition-all shadow-sm active:scale-95 disabled:opacity-50"
            title="Clean APT cache, unused Flatpaks, old Snaps, and temporary caches"
          >
            <IconClean className={`w-4 h-4 text-emerald-400 ${isCleaning ? 'animate-spin' : ''}`} />
            <span>{isCleaning ? 'Cleaning...' : 'Clean Up'}</span>
          </button>

          <button
            onClick={onRefresh}
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

      {/* Expandable Details Drawer */}
      {isExpanded && (
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
  );
}
