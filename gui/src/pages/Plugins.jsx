import React, { useState, useEffect } from 'react';
import { api } from '../api';

const PLANNED_PLUGINS = [
  {
    name: 'docker-monitor',
    processName: 'm3tal-docker',
    category: 'Container Management',
    description: 'Docker container fleet monitor, log inspector, container lifecycle controls, and image pruning.',
    capabilities: ['collector', 'actions', 'ui_view', 'websocket_stream'],
    status: 'Planned (External)',
  },
  {
    name: 'stack-orchestrator',
    processName: 'm3tal-stacks',
    category: 'Compose Stacks',
    description: 'Multi-service compose stack orchestration, live health checks, and lifecycle deployment.',
    capabilities: ['orchestrator', 'actions', 'ui_view'],
    status: 'Planned (External)',
  },
  {
    name: 'ai-engine',
    processName: 'm3tal-ai',
    category: 'Local AI & LLM',
    description: 'Local LLM inference manager (Ollama, vLLM), job queue, and model catalog.',
    capabilities: ['inference_runner', 'job_queue', 'ui_view'],
    status: 'Planned (External)',
  },
  {
    name: 'smart-disk',
    processName: 'm3tal-smart',
    category: 'Hardware Diagnostics',
    description: 'Drive health telemetry, NVMe wear life, S.M.A.R.T. self-tests, and temperature history.',
    capabilities: ['collector', 'ui_widget', 'alerts'],
    status: 'Planned (External)',
  },
  {
    name: 'traefik-gateway',
    processName: 'm3tal-traefik',
    category: 'Networking & Proxy',
    description: 'Traefik v3 reverse proxy, automated Let\'s Encrypt SSL/TLS certificates, and dynamic routing.',
    capabilities: ['proxy_generator', 'route_provider'],
    status: 'Planned (External)',
  },
  {
    name: 'gluetun-vpn',
    processName: 'm3tal-vpn',
    category: 'Network Privacy',
    description: 'WireGuard/OpenVPN tunnel connectivity with container egress routing and leak protection.',
    capabilities: ['network_tunnel', 'status_checker'],
    status: 'Planned (External)',
  },
  {
    name: 'samba-manager',
    processName: 'm3tal-samba',
    category: 'File Sharing',
    description: 'Graphical Samba/NFS share manager, guest permissions, user credentials, and active mounts.',
    capabilities: ['actions', 'ui_view', 'system_admin'],
    status: 'Planned (External)',
  },
  {
    name: 'system-bench',
    processName: 'm3tal-bench',
    category: 'Diagnostics',
    description: 'One-click benchmarking for CPU multi-threading, GPU compute, and disk I/O throughput.',
    capabilities: ['benchmark_runner', 'ui_view'],
    status: 'Planned (External)',
  },
];

export default function Plugins() {
  const [plugins, setPlugins] = useState([]);
  const [activeTab, setActiveTab] = useState('installed'); // 'installed' or 'roadmap'
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(null);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);

  const fetchPlugins = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getPlugins();
      if (res?.status === 'success' && Array.isArray(res.data)) {
        setPlugins(res.data);
      } else if (res?.data && typeof res.data === 'object' && Array.isArray(res.data.plugins)) {
        setPlugins(res.data.plugins);
      } else {
        setPlugins([]);
      }
    } catch (err) {
      setError(err.message || 'Failed to fetch plugins from supervisor');
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchPlugins();
    const interval = setInterval(fetchPlugins, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleStart = async (name) => {
    setActionLoading(name);
    setError(null);
    setSuccess(null);
    try {
      const res = await api.startPlugin(name);
      if (res?.status === 'success') {
        setSuccess(`Plugin "${name}" started successfully.`);
        await fetchPlugins();
      } else {
        setError(res?.error || `Failed to start plugin "${name}"`);
      }
    } catch (err) {
      setError(err.message || `Error starting plugin "${name}"`);
    }
    setActionLoading(null);
  };

  const handleStop = async (name) => {
    setActionLoading(name);
    setError(null);
    setSuccess(null);
    try {
      const res = await api.stopPlugin(name);
      if (res?.status === 'success') {
        setSuccess(`Plugin "${name}" stopped.`);
        await fetchPlugins();
      } else {
        setError(res?.error || `Failed to stop plugin "${name}"`);
      }
    } catch (err) {
      setError(err.message || `Error stopping plugin "${name}"`);
    }
    setActionLoading(null);
  };

  const getStateBadge = (state) => {
    switch (state) {
      case 'running':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-950/60 text-emerald-400 border border-emerald-800">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
            Running
          </span>
        );
      case 'starting':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-cyan-950/60 text-cyan-400 border border-cyan-800">
            Starting...
          </span>
        );
      case 'crashed':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-rose-950/60 text-rose-400 border border-rose-800">
            Crashed
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-slate-800 text-slate-400 border border-slate-700">
            Stopped
          </span>
        );
    }
  };

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-[#30363d] pb-6">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-3">
            <span className="p-2 rounded-lg bg-emerald-950/50 border border-emerald-800/60 text-emerald-400">
              🧩
            </span>
            <span>Plugin Manager</span>
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            Supervise self-contained external plugins communicating over JSON-RPC 2.0.
          </p>
        </div>

        {/* View Tabs */}
        <div className="flex items-center gap-2 bg-[#161b22] border border-[#30363d] p-1 rounded-lg">
          <button
            onClick={() => setActiveTab('installed')}
            className={`px-4 py-1.5 text-xs font-semibold rounded-md transition-colors ${
              activeTab === 'installed'
                ? 'bg-emerald-600 text-white shadow-sm'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            Discovered Plugins ({plugins.length})
          </button>
          <button
            onClick={() => setActiveTab('roadmap')}
            className={`px-4 py-1.5 text-xs font-semibold rounded-md transition-colors ${
              activeTab === 'roadmap'
                ? 'bg-emerald-600 text-white shadow-sm'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            Catalog & Roadmap
          </button>
        </div>
      </div>

      {/* Notifications */}
      {error && (
        <div className="p-4 bg-rose-950/40 border border-rose-800/60 rounded-xl text-rose-400 text-sm flex items-center justify-between">
          <span>⚠️ {error}</span>
          <button onClick={() => setError(null)} className="text-rose-500 hover:text-rose-300">✕</button>
        </div>
      )}
      {success && (
        <div className="p-4 bg-emerald-950/40 border border-emerald-800/60 rounded-xl text-emerald-400 text-sm flex items-center justify-between">
          <span>✅ {success}</span>
          <button onClick={() => setSuccess(null)} className="text-emerald-500 hover:text-emerald-300">✕</button>
        </div>
      )}

      {/* Installed / Discovered Plugins View */}
      {activeTab === 'installed' && (
        <div className="space-y-4">
          <div className="bg-[#161b22] border border-[#30363d] rounded-xl overflow-hidden">
            <div className="p-4 border-b border-[#30363d] flex items-center justify-between">
              <div>
                <h2 className="text-sm font-semibold text-white">Managed Plugin Subprocesses</h2>
                <p className="text-xs text-slate-500 mt-0.5">
                  Search paths: <code className="text-slate-400">~/.config/m3tal/plugins/</code>, <code className="text-slate-400">/var/lib/m3tal/plugins/</code>, <code className="text-slate-400">./plugins/</code>
                </p>
              </div>
              <button
                onClick={fetchPlugins}
                disabled={loading}
                className="px-3 py-1.5 text-xs font-medium rounded-lg bg-[#21262d] hover:bg-[#30363d] text-slate-300 transition-colors"
              >
                {loading ? 'Refreshing...' : '🔄 Refresh'}
              </button>
            </div>

            {plugins.length === 0 ? (
              <div className="p-12 text-center space-y-3">
                <div className="text-3xl">📦</div>
                <h3 className="text-sm font-semibold text-white">No External Plugins Installed</h3>
                <p className="text-xs text-slate-400 max-w-md mx-auto">
                  M3TAL Core is running in pure platform mode with zero background plugin footprint. To install a plugin, drop its package into <code className="text-slate-300">~/.config/m3tal/plugins/</code> or check the Catalog & Roadmap.
                </p>
              </div>
            ) : (
              <div className="divide-y divide-[#30363d]">
                {plugins.map((p) => {
                  const manifest = p.manifest || p.Manifest || {};
                  const meta = manifest.metadata || manifest.Metadata || {};
                  const name = meta.name || p.name || 'unknown';
                  const version = meta.version || '1.0.0';
                  const desc = meta.description || 'No description provided';
                  const state = p.state || p.State || 'stopped';
                  const caps = p.capabilities || p.Capabilities || manifest.spec?.capabilities || [];
                  const procName = meta.name ? (meta.name.startsWith('m3tal-') ? meta.name : `m3tal-${meta.name}`) : 'm3tal-plugin';

                  return (
                    <div key={name} className="p-5 flex flex-col md:flex-row md:items-center justify-between gap-4 hover:bg-[#21262d]/40 transition-colors">
                      <div className="space-y-1.5">
                        <div className="flex items-center gap-3">
                          <span className="font-bold text-white text-base">{name}</span>
                          <span className="text-xs text-slate-400 font-mono">v{version}</span>
                          {getStateBadge(state)}
                          <span className="text-[11px] font-mono text-cyan-400 bg-cyan-950/40 border border-cyan-800/50 px-2 py-0.5 rounded">
                            proc: {procName}
                          </span>
                        </div>
                        <p className="text-xs text-slate-400">{desc}</p>
                        {caps.length > 0 && (
                          <div className="flex items-center gap-1.5 pt-1">
                            {caps.map((c) => (
                              <span key={c} className="text-[10px] font-medium px-2 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700">
                                {c}
                              </span>
                            ))}
                          </div>
                        )}
                        {p.lastError && (
                          <p className="text-xs text-rose-400 font-mono mt-1">Error: {p.lastError}</p>
                        )}
                      </div>

                      <div className="flex items-center gap-2">
                        {state === 'running' ? (
                          <button
                            onClick={() => handleStop(name)}
                            disabled={actionLoading === name}
                            className="px-3 py-1.5 text-xs font-semibold rounded-lg bg-rose-900/60 hover:bg-rose-800 text-rose-200 border border-rose-700 transition-colors"
                          >
                            {actionLoading === name ? 'Stopping...' : '⏹ Stop'}
                          </button>
                        ) : (
                          <button
                            onClick={() => handleStart(name)}
                            disabled={actionLoading === name}
                            className="px-3 py-1.5 text-xs font-semibold rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white shadow-sm transition-colors"
                          >
                            {actionLoading === name ? 'Starting...' : '▶ Start'}
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Catalog & Roadmap View */}
      {activeTab === 'roadmap' && (
        <div className="space-y-4">
          <div className="bg-[#161b22] border border-[#30363d] rounded-xl p-5 space-y-2">
            <h2 className="text-sm font-semibold text-white">M3TAL Plugin Architecture: All Code Lives With The Plugin</h2>
            <p className="text-xs text-slate-400 leading-relaxed">
              In accordance with the M3TAL Plugin Standardization Specification, all third-party plugins are completely self-contained packages. Every plugin runs in its own process named <code className="text-slate-300">m3tal-&lt;plugin-name&gt;</code>, communicates via JSON-RPC 2.0, and mounts its dynamic UI tabs directly into the supervisor without modifying core code.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {PLANNED_PLUGINS.map((item) => (
              <div key={item.name} className="bg-[#161b22] border border-[#30363d] rounded-xl p-5 space-y-3">
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <h3 className="text-sm font-bold text-white">{item.name}</h3>
                    <span className="text-[11px] font-mono text-cyan-400">proc: {item.processName}</span>
                  </div>
                  <span className="text-[10px] font-semibold px-2 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700">
                    {item.category}
                  </span>
                </div>
                <p className="text-xs text-slate-400 leading-relaxed">{item.description}</p>
                <div className="flex flex-wrap items-center gap-1.5 pt-1">
                  {item.capabilities.map((c) => (
                    <span key={c} className="text-[10px] font-medium px-2 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700">
                      {c}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
