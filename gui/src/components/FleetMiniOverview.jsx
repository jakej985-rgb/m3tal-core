import { useState, useEffect, useMemo } from 'react';
import { api } from '../api';
import { IconContainers, IconRefresh } from './Icons';

export default function FleetMiniOverview({
  isEditMode = false,
  onMoveLeft,
  onMoveRight,
  onHide,
  className = '',
}) {
  const [containers, setContainers] = useState([]);
  const [loading, setLoading] = useState(true);

  const fetchFleet = async () => {
    setLoading(true);
    const res = await api.getContainers();
    if (res?.status === 'success' && Array.isArray(res.data)) {
      setContainers(res.data);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchFleet();
    const interval = setInterval(fetchFleet, 5000);
    return () => clearInterval(interval);
  }, []);

  // Analysis of container fleet health, crashes, and duplicate versions
  const analysis = useMemo(() => {
    if (!containers || containers.length === 0) {
      return {
        status: 'empty',
        color: 'slate',
        label: 'NO CONTAINERS',
        sublabel: 'No Docker containers currently managed',
        running: 0,
        stopped: 0,
        crashed: 0,
        unhealthy: 0,
        duplicateConflicts: [],
        criticalItems: [],
        warningItems: [],
        total: 0,
      };
    }

    let running = 0;
    let stopped = 0;
    let crashed = 0;
    let unhealthy = 0;
    const criticalItems = [];
    const warningItems = [];

    // Detect multiple versions of same image base running concurrently
    const serviceImageMap = new Map();

    containers.forEach((c) => {
      const name = (c.names?.[0] || c.ID || 'unknown').replace(/^\//, '');
      const state = (c.state || '').toLowerCase();
      const status = (c.status || '').toLowerCase();

      // Check crash / restart loop / non-zero exit
      const isRestarting = state === 'restarting' || status.includes('restart');
      const isDead = state === 'dead';
      const isNonZeroExit = state === 'exited' && !status.includes('exit (0)') && !status.includes('exited (0)');
      const isUnhealthy = status.includes('unhealthy');

      if (isRestarting || isDead || isNonZeroExit || isUnhealthy) {
        crashed++;
        if (isUnhealthy) unhealthy++;
        criticalItems.push({
          name,
          image: c.image,
          reason: isRestarting
            ? 'Crash / Restart Loop'
            : isUnhealthy
            ? 'Unhealthy Healthcheck'
            : `Exited abnormally (${c.status})`,
        });
      } else if (state === 'running') {
        running++;
      } else {
        stopped++;
        warningItems.push({
          name,
          image: c.image,
          reason: state === 'paused' ? 'Paused' : 'Stopped (Clean exit)',
        });
      }

      // Group images by base repo name
      const imageBase = (c.image || '').split(':')[0];
      if (imageBase && imageBase !== 'unknown') {
        if (!serviceImageMap.has(imageBase)) {
          serviceImageMap.set(imageBase, []);
        }
        serviceImageMap.get(imageBase).push({ name, image: c.image, state });
      }
    });

    const duplicateConflicts = [];
    serviceImageMap.forEach((instances, base) => {
      if (instances.length > 1) {
        const distinctImages = new Set(instances.map((i) => i.image));
        if (distinctImages.size > 1) {
          duplicateConflicts.push({
            base,
            instances,
          });
        }
      }
    });

    // Executive Health Dot & Badge:
    // Red: Crashes, restart loops, unhealthy, or duplicate versions
    // Yellow: Stopped containers (needs action)
    // Green: All containers running & healthy
    let status = 'healthy';
    let color = 'emerald';
    let label = 'ALL UP & HEALTHY';
    let sublabel = '100% Convergence • All services running normally';

    if (criticalItems.length > 0 || duplicateConflicts.length > 0) {
      status = 'critical';
      color = 'rose';
      label = criticalItems.length > 0
        ? `${criticalItems.length} CRITICAL / FAILING`
        : `${duplicateConflicts.length} VERSION CONFLICTS`;
      sublabel = 'Action required: container crashes or mismatched versions';
    } else if (stopped > 0) {
      status = 'warning';
      color = 'amber';
      label = `${stopped} NEED ACTION`;
      sublabel = `${stopped} stopped or paused service(s)`;
    }

    return {
      status,
      color,
      label,
      sublabel,
      running,
      stopped,
      crashed,
      unhealthy,
      duplicateConflicts,
      criticalItems,
      warningItems,
      total: containers.length,
    };
  }, [containers]);

  return (
    <div
      className={`bg-slate-900 border ${
        isEditMode
          ? 'border-amber-500/50 ring-1 ring-amber-500/30'
          : 'border-slate-800'
      } rounded-2xl p-5 flex flex-col justify-between shadow-sm relative overflow-hidden h-full min-h-[260px] lg:h-[260px] ${className}`}
    >
      {/* Edit Mode Customization Toolbar */}
      {isEditMode && (
        <div className="bg-amber-500/10 border-b border-amber-500/20 -mx-5 -mt-5 mb-3 px-3 py-1 flex items-center justify-between text-xs text-amber-300 select-none">
          <span className="font-mono text-[10px] font-bold uppercase tracking-wider">
            2x1 GRID
          </span>
          <div className="flex items-center gap-1">
            {onMoveLeft && (
              <button
                type="button"
                onClick={onMoveLeft}
                className="p-1 rounded hover:bg-amber-500/20 text-amber-200 transition-colors"
                title="Move earlier in grid"
              >
                ◀
              </button>
            )}
            {onMoveRight && (
              <button
                type="button"
                onClick={onMoveRight}
                className="p-1 rounded hover:bg-amber-500/20 text-amber-200 transition-colors"
                title="Move later in grid"
              >
                ▶
              </button>
            )}
            {onHide && (
              <button
                type="button"
                onClick={onHide}
                className="px-1.5 py-0.5 rounded hover:bg-rose-500/20 text-rose-300 transition-colors font-bold text-[11px] ml-1"
                title="Hide this widget"
              >
                ✕ Hide
              </button>
            )}
          </div>
        </div>
      )}

      <div>
        {/* Header */}
        <div className="flex justify-between items-center mb-4 pb-3 border-b border-slate-800/60">
          <div className="flex items-center gap-2.5">
            <div className="p-1.5 rounded-lg bg-slate-800/80 border border-slate-700/60">
              <IconContainers className="w-4 h-4 text-emerald-400" />
            </div>
            <div>
              <h2 className="text-sm font-semibold text-white tracking-tight flex items-center gap-2">
                Container Fleet Overview
              </h2>
              <p className="text-xs text-slate-400">{analysis.sublabel}</p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {/* Executive Status Dot & Badge */}
            <div
              className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-bold border shadow-sm ${
                analysis.status === 'healthy'
                  ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/25'
                  : analysis.status === 'warning'
                  ? 'bg-amber-500/10 text-amber-400 border-amber-500/25'
                  : analysis.status === 'critical'
                  ? 'bg-rose-500/10 text-rose-400 border-rose-500/30'
                  : 'bg-slate-800 text-slate-400 border-slate-700'
              }`}
            >
              <span
                className={`w-2 h-2 rounded-full ${
                  analysis.status === 'healthy'
                    ? 'bg-emerald-400 animate-pulse'
                    : analysis.status === 'warning'
                    ? 'bg-amber-400'
                    : analysis.status === 'critical'
                    ? 'bg-rose-500 animate-ping'
                    : 'bg-slate-600'
                }`}
              />
              <span>{analysis.label}</span>
            </div>

            <button
              onClick={fetchFleet}
              className="p-1 text-slate-500 hover:text-slate-300 transition-colors rounded hover:bg-slate-800"
              title="Refresh containers"
            >
              <IconRefresh className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>

        {/* Metric Counters Grid */}
        <div className="grid grid-cols-4 gap-2 mb-4">
          <div className="bg-slate-950/50 p-2.5 rounded-xl border border-slate-800/60 text-center">
            <span className="text-[10px] uppercase font-bold text-slate-500 tracking-wider block">Total</span>
            <span className="text-lg font-bold text-white font-mono">{analysis.total}</span>
          </div>

          <div className="bg-slate-950/50 p-2.5 rounded-xl border border-slate-800/60 text-center">
            <span className="text-[10px] uppercase font-bold text-emerald-500 tracking-wider block">Running</span>
            <span className="text-lg font-bold text-emerald-400 font-mono">{analysis.running}</span>
          </div>

          <div className="bg-slate-950/50 p-2.5 rounded-xl border border-slate-800/60 text-center">
            <span className="text-[10px] uppercase font-bold text-amber-500 tracking-wider block">Stopped</span>
            <span className="text-lg font-bold text-amber-400 font-mono">{analysis.stopped}</span>
          </div>

          <div className="bg-slate-950/50 p-2.5 rounded-xl border border-slate-800/60 text-center">
            <span className="text-[10px] uppercase font-bold text-rose-500 tracking-wider block">Crashed</span>
            <span className="text-lg font-bold text-rose-400 font-mono">{analysis.crashed}</span>
          </div>
        </div>

        {/* Detailed Alerts or Clean State Banner */}
        {analysis.criticalItems.length > 0 ? (
          <div className="space-y-1.5 mb-2">
            <span className="text-[10px] font-bold text-rose-400 uppercase tracking-wider block">
              ⚠️ Attention Required ({analysis.criticalItems.length})
            </span>
            {analysis.criticalItems.slice(0, 3).map((item, idx) => (
              <div
                key={idx}
                className="flex items-center justify-between p-2 rounded-lg bg-rose-950/20 border border-rose-500/30 text-xs"
              >
                <div className="truncate pr-2">
                  <span className="font-semibold text-rose-300 font-mono">{item.name}</span>
                  <span className="text-[11px] text-slate-400 block truncate">{item.image}</span>
                </div>
                <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-500/20 text-rose-300 border border-rose-500/40 shrink-0">
                  {item.reason}
                </span>
              </div>
            ))}
          </div>
        ) : analysis.duplicateConflicts.length > 0 ? (
          <div className="space-y-1.5 mb-2">
            <span className="text-[10px] font-bold text-amber-400 uppercase tracking-wider block">
              ⚠️ Version Conflict Detected ({analysis.duplicateConflicts.length})
            </span>
            {analysis.duplicateConflicts.slice(0, 2).map((dup, idx) => (
              <div
                key={idx}
                className="p-2 rounded-lg bg-amber-950/20 border border-amber-500/30 text-xs text-amber-200"
              >
                <span className="font-semibold font-mono block">Image: {dup.base}</span>
                <span className="text-[10px] text-slate-400">
                  Multiple distinct image versions currently coexisting ({dup.instances.length} containers).
                </span>
              </div>
            ))}
          </div>
        ) : analysis.total > 0 ? (
          <div className="p-3 rounded-xl bg-emerald-950/20 border border-emerald-500/25 flex items-center gap-2.5 text-xs text-emerald-300">
            <span className="text-base">✓</span>
            <div>
              <span className="font-semibold block">All containers healthy and operational</span>
              <span className="text-[10px] text-slate-400">
                Zero crash loops, healthcheck failures, or version conflicts across {analysis.total} containers.
              </span>
            </div>
          </div>
        ) : (
          <div className="text-center py-4 text-xs text-slate-500 italic">
            No Docker containers currently managed on this host.
          </div>
        )}
      </div>
    </div>
  );
}
