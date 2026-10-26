import { useState, useEffect, useMemo } from 'react';
import { api } from '../api';
import { IconNetwork, IconRefresh } from './Icons';

export default function NetworkSummary({
  networkInterfaces: propInterfaces,
  isEditMode = false,
  onMoveLeft,
  onMoveRight,
  onHide,
  className = '',
}) {
  const [interfaces, setInterfaces] = useState(propInterfaces || []);
  const [loading, setLoading] = useState(!propInterfaces || propInterfaces.length === 0);

  const fetchNetwork = async () => {
    setLoading(true);
    try {
      const res = await api.getNetworkStats();
      if (res?.status === 'success' && Array.isArray(res.data)) {
        setInterfaces(res.data);
      }
    } catch {
      // Non-critical fallback
    }
    setLoading(false);
  };

  useEffect(() => {
    if (propInterfaces && propInterfaces.length > 0) {
      setInterfaces(propInterfaces);
    } else {
      fetchNetwork();
    }
  }, [propInterfaces]);

  useEffect(() => {
    const interval = setInterval(fetchNetwork, 5000);
    return () => clearInterval(interval);
  }, []);

  const formatBytes = (bytes) => {
    if (!bytes && bytes !== 0) return '0 B';
    if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${bytes} B`;
  };

  // Compute aggregated stats
  const { totalSent, totalRecv, activeCount, primaryIP } = useMemo(() => {
    let sent = 0;
    let recv = 0;
    let active = 0;
    let mainIP = '127.0.0.1';

    interfaces.forEach((iface) => {
      sent += iface.bytes_sent || 0;
      recv += iface.bytes_recv || 0;
      if (iface.is_up) {
        active++;
        if (Array.isArray(iface.ip_addresses)) {
          // Find first non-loopback IPv4
          const ipv4 = iface.ip_addresses.find(
            (addr) => !addr.startsWith('127.') && !addr.startsWith('fe80') && !addr.includes(':')
          );
          if (ipv4 && mainIP === '127.0.0.1') {
            mainIP = ipv4.split('/')[0];
          }
        }
      }
    });

    return {
      totalSent: sent,
      totalRecv: recv,
      activeCount: active,
      primaryIP: mainIP,
    };
  }, [interfaces]);

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
        <div className="flex justify-between items-center mb-3 pb-2.5 border-b border-slate-800/60">
          <div className="flex items-center gap-2.5">
            <div className="p-1.5 rounded-lg bg-slate-800/80 border border-slate-700/60">
              <IconNetwork className="w-4 h-4 text-sky-400" />
            </div>
            <div>
              <h2 className="text-sm font-semibold text-white tracking-tight flex items-center gap-2">
                Network Statistics
              </h2>
              <p className="text-xs text-slate-400 font-mono">
                Host IP: <span className="text-slate-200">{primaryIP}</span>
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <span
              className={`px-2.5 py-0.5 rounded-full text-[10px] font-bold border shadow-sm ${
                activeCount > 0
                  ? 'bg-sky-500/10 text-sky-400 border-sky-500/25'
                  : 'bg-slate-800 text-slate-400 border-slate-700'
              }`}
            >
              {activeCount} {activeCount === 1 ? 'LINK ACTIVE' : 'LINKS ACTIVE'}
            </span>

            <button
              onClick={fetchNetwork}
              className="p-1 text-slate-500 hover:text-slate-300 transition-colors rounded hover:bg-slate-800"
              title="Refresh network telemetry"
            >
              <IconRefresh className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>

        {/* Aggregated Bandwidth Counters */}
        <div className="grid grid-cols-2 gap-2.5 mb-3">
          <div className="bg-slate-950/50 p-2.5 rounded-xl border border-slate-800/60 flex items-center justify-between">
            <div>
              <span className="text-[10px] uppercase font-bold text-slate-500 tracking-wider block">
                ⬇ Total Received (RX)
              </span>
              <span className="text-sm font-bold text-white font-mono">{formatBytes(totalRecv)}</span>
            </div>
            <span className="text-xs text-sky-400 font-mono">IN</span>
          </div>

          <div className="bg-slate-950/50 p-2.5 rounded-xl border border-slate-800/60 flex items-center justify-between">
            <div>
              <span className="text-[10px] uppercase font-bold text-slate-500 tracking-wider block">
                ⬆ Total Transmitted (TX)
              </span>
              <span className="text-sm font-bold text-white font-mono">{formatBytes(totalSent)}</span>
            </div>
            <span className="text-xs text-emerald-400 font-mono">OUT</span>
          </div>
        </div>

        {/* Interface List */}
        {interfaces.length === 0 ? (
          <div className="text-center py-4 text-xs text-slate-500 italic">
            {loading ? 'Discovering network interfaces...' : 'No active network interfaces detected.'}
          </div>
        ) : (
          <div className="space-y-1.5 max-h-[85px] overflow-y-auto pr-1">
            {interfaces.slice(0, 3).map((iface) => {
              // Extract primary IPv4
              const ipv4 = (iface.ip_addresses || []).find(
                (addr) => !addr.startsWith('127.') && !addr.startsWith('fe80') && !addr.includes(':')
              ) || iface.ip_addresses?.[0] || 'Unassigned';

              return (
                <div
                  key={iface.name}
                  className="flex items-center justify-between p-2 rounded-lg bg-slate-950/40 border border-slate-800/50 text-xs"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <span
                      className={`w-2 h-2 rounded-full flex-shrink-0 ${
                        iface.is_up ? 'bg-sky-400 animate-pulse' : 'bg-slate-600'
                      }`}
                    />
                    <div className="truncate">
                      <div className="font-semibold text-slate-200 font-mono flex items-center gap-1.5">
                        <span className="text-xs">{iface.name}</span>
                        {iface.is_up && (
                          <span className="px-1 py-0.1 rounded text-[8px] font-bold bg-sky-500/15 text-sky-300 border border-sky-500/30">
                            UP
                          </span>
                        )}
                      </div>
                      <div className="text-[10px] text-slate-400 font-mono truncate">{ipv4}</div>
                    </div>
                  </div>

                  <div className="text-right font-mono text-[10px] shrink-0 text-slate-400 leading-tight">
                    <div>
                      <span className="text-slate-500">RX:</span> {formatBytes(iface.bytes_recv)}
                    </div>
                    <div>
                      <span className="text-slate-500">TX:</span> {formatBytes(iface.bytes_sent)}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {interfaces.length > 3 && (
        <div className="mt-1 text-center">
          <span className="text-[10px] text-slate-500 font-mono">
            +{interfaces.length - 3} more interface(s)
          </span>
        </div>
      )}
    </div>
  );
}
