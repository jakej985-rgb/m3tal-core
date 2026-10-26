import { useState, useEffect, useCallback } from 'react';
import { api } from '../api';
import {
  IconPlugins,
  IconDownload,
  IconDocument,
  IconLock,
  IconTrash,
  IconPlay,
  IconStop,
  IconCheck,
  IconCopy,
  IconRefresh,
} from '../components/Icons';

function parseSemVer(v) {
  if (!v) return [0, 0, 0];
  const clean = String(v).replace(/^v/, '').split('-')[0].split('+')[0];
  const parts = clean.split('.').map((p) => parseInt(p, 10) || 0);
  while (parts.length < 3) parts.push(0);
  return parts.slice(0, 3);
}

function compareSemVer(v1, v2) {
  const [maj1, min1, pat1] = parseSemVer(v1);
  const [maj2, min2, pat2] = parseSemVer(v2);

  if (maj1 > maj2) return { cmp: 1, type: 'dev' };
  if (maj1 < maj2) return { cmp: -1, type: 'major' };

  if (min1 > min2) return { cmp: 1, type: 'dev' };
  if (min1 < min2) return { cmp: -1, type: 'minor' };

  if (pat1 > pat2) return { cmp: 1, type: 'dev' };
  if (pat1 < pat2) return { cmp: -1, type: 'patch' };

  return { cmp: 0, type: 'equal' };
}

const FALLBACK_PLANNED_PLUGINS = [
  {
    name: 'docker-monitor',
    title: 'Containers',
    process_name: 'm3tal-docker',
    package_name: 'm3tal-plugin-docker-monitor',
    category: 'Container Management',
    description: 'Docker container fleet monitor, log inspector, container lifecycle controls, and image pruning.',
    capabilities: ['collector', 'actions', 'ui_view', 'ui_widget', 'settings', 'websocket_stream'],
    version: '',
    status: 'planned',
    has_deb: false,
    apt_command: 'sudo apt install -y m3tal-plugin-docker-monitor',
  },
  {
    name: 'compose',
    title: 'Compose Editor',
    process_name: 'm3tal-compose',
    package_name: 'm3tal-plugin-compose',
    category: 'Container Management',
    description: 'Docker Compose visual editor, syntax validator, and multi-container stack lifecycle manager.',
    capabilities: ['collector', 'actions', 'ui_view', 'ui_widget', 'settings'],
    version: '',
    status: 'planned',
    has_deb: false,
    apt_command: 'sudo apt install -y m3tal-plugin-compose',
  },
  {
    name: 'mesh',
    title: 'Cluster & Mesh Federation',
    process_name: 'm3tal-mesh',
    package_name: 'm3tal-plugin-mesh-cluster',
    category: 'Networking & Fleet',
    description: 'Local network discovery, multi-node clustering, and telemetry federation across M3TAL workstations, servers, and NAS nodes.',
    capabilities: ['collector', 'actions', 'ui_view'],
    version: '',
    status: 'planned',
    has_deb: false,
    apt_command: 'sudo apt install -y m3tal-plugin-mesh-cluster',
    modes: [
      {
        id: 'cluster',
        label: 'Cluster',
        subLabel: 'Full Peer',
        description: 'Full peer (Send + Receive). Broadcasts local machine metrics AND renders the multi-node fleet dashboard. Recommended for primary workstations.',
        package_name: 'm3tal-plugin-mesh-cluster',
        apt_command: 'sudo apt install -y m3tal-plugin-mesh-cluster',
        process_name: 'm3tal-mesh',
      },
      {
        id: 'mesh',
        label: 'Mesh',
        subLabel: 'Viewer',
        description: 'Reader & dashboard UI. Scans LAN to view and monitor other connected M3TAL nodes from your desktop.',
        package_name: 'm3tal-plugin-mesh',
        apt_command: 'sudo apt install -y m3tal-plugin-mesh',
        process_name: 'm3tal-mesh',
      },
      {
        id: 'node',
        label: 'Node',
        subLabel: 'Sender',
        description: 'Headless sender daemon. Broadcasts CPU, GPU, and disk metrics without running a GUI. Ideal for servers & NAS.',
        package_name: 'm3tal-plugin-mesh-node',
        apt_command: 'sudo apt install -y m3tal-plugin-mesh-node',
        process_name: 'm3tal-mesh-node',
      },
    ],
  },
  {
    name: 'stack-orchestrator',
    title: 'Compose Stacks (Template DB)',
    process_name: 'm3tal-stacks',
    package_name: 'm3tal-plugin-stacks',
    category: 'Compose Stacks',
    description: 'Curated database of verified Docker Compose templates, argument presets, and 1-click community stacks.',
    capabilities: ['orchestrator', 'actions', 'ui_view'],
    version: '',
    status: 'planned',
    apt_command: 'sudo apt install -y m3tal-plugin-stacks',
  },
  {
    name: 'ai-engine',
    title: 'Local AI & LLM',
    process_name: 'm3tal-ai',
    package_name: 'm3tal-plugin-ai',
    category: 'Local AI & LLM',
    description: 'Local LLM inference manager (Ollama, vLLM), GPU VRAM stats, asynchronous job queue, and model catalog.',
    capabilities: ['inference_runner', 'job_queue', 'ui_view'],
    version: '',
    status: 'planned',
    apt_command: 'sudo apt install -y m3tal-plugin-ai',
  },
  {
    name: 'smart-disk',
    title: 'Smart Disk Telemetry',
    process_name: 'm3tal-smart',
    package_name: 'm3tal-plugin-smart',
    category: 'Hardware Diagnostics',
    description: 'Drive health telemetry, NVMe wear life, S.M.A.R.T. self-tests, and temperature history.',
    capabilities: ['collector', 'ui_widget', 'alerts'],
    version: '',
    status: 'planned',
    apt_command: 'sudo apt install -y m3tal-plugin-smart',
  },
  {
    name: 'traefik-gateway',
    title: 'Traefik Gateway',
    process_name: 'm3tal-traefik',
    package_name: 'm3tal-plugin-traefik',
    category: 'Networking & Proxy',
    description: 'Traefik v3 reverse proxy, automated Let\'s Encrypt SSL/TLS certificates, and dynamic routing.',
    capabilities: ['proxy_generator', 'route_provider'],
    version: '',
    status: 'planned',
    apt_command: 'sudo apt install -y m3tal-plugin-traefik',
  },
  {
    name: 'gluetun-vpn',
    title: 'VPN Tunnel',
    process_name: 'm3tal-vpn',
    package_name: 'm3tal-plugin-vpn',
    category: 'Network Privacy',
    description: 'WireGuard/OpenVPN tunnel connectivity with container egress routing and leak protection.',
    capabilities: ['network_tunnel', 'status_checker'],
    version: '',
    status: 'planned',
    apt_command: 'sudo apt install -y m3tal-plugin-vpn',
  },
  {
    name: 'samba-manager',
    title: 'Samba Manager',
    process_name: 'm3tal-samba',
    package_name: 'm3tal-plugin-samba',
    category: 'File Sharing',
    description: 'Graphical Samba/NFS share manager, guest permissions, user credentials, and active mounts.',
    capabilities: ['actions', 'ui_view', 'system_admin'],
    version: '',
    status: 'planned',
    apt_command: 'sudo apt install -y m3tal-plugin-samba',
  },
  {
    name: 'system-bench',
    title: 'System Benchmarks',
    process_name: 'm3tal-bench',
    package_name: 'm3tal-plugin-bench',
    category: 'Diagnostics',
    description: 'One-click benchmarking for CPU multi-threading, GPU compute, and disk I/O throughput.',
    capabilities: ['benchmark_runner', 'ui_view'],
    version: '',
    status: 'planned',
    apt_command: 'sudo apt install -y m3tal-plugin-bench',
  },
];

export default function Plugins() {
  const [plugins, setPlugins] = useState([]);
  const [catalog, setCatalog] = useState(FALLBACK_PLANNED_PLUGINS);
  const [activeTab, setActiveTab] = useState('installed'); // 'installed' or 'roadmap'
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(null);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const [copiedName, setCopiedName] = useState(null);
  const [selectedModes, setSelectedModes] = useState({});

  const getActiveMode = (item) => {
    if (!item?.modes || item.modes.length === 0) return null;
    const selectedId = selectedModes[item.name];
    if (selectedId) {
      return item.modes.find((m) => m.id === selectedId) || item.modes[0];
    }
    // Default to 'cluster' if present, otherwise first mode
    return item.modes.find((m) => m.id === 'cluster') || item.modes[0];
  };

  // Modals state
  const [selectedChangelog, setSelectedChangelog] = useState(null);
  const [installTarget, setInstallTarget] = useState(null);
  const [uninstallTarget, setUninstallTarget] = useState(null);
  const [sudoPassword, setSudoPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [installMethod, setInstallMethod] = useState('sudo'); // 'sudo' or 'user'
  const [installLoading, setInstallLoading] = useState(false);
  const [installError, setInstallError] = useState(null);

  const fetchPlugins = useCallback(async () => {
    setLoading(true);
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
  }, []);

  const fetchCatalog = useCallback(async () => {
    try {
      const res = await api.getPluginCatalog();
      if (res?.status === 'success' && Array.isArray(res.data) && res.data.length > 0) {
        setCatalog(res.data);
      }
    } catch {
      // Use fallback
    }
  }, []);

  const reloadAll = useCallback(async () => {
    await Promise.all([fetchPlugins(), fetchCatalog()]);
  }, [fetchPlugins, fetchCatalog]);

  const handleRefreshApt = async () => {
    setActionLoading('apt_refresh');
    setError(null);
    setSuccess(null);
    try {
      await api.refreshRepositories();
      await reloadAll();
      setSuccess('APT repository index refreshed successfully. Candidate versions updated from keyring.');
    } catch (err) {
      setError(err.message || 'Failed to refresh APT repositories');
    } finally {
      setActionLoading(null);
    }
  };

  useEffect(() => {
    reloadAll();
    const interval = setInterval(reloadAll, 12000);
    return () => clearInterval(interval);
  }, [reloadAll]);

  const handleStart = async (name) => {
    setActionLoading(name);
    setError(null);
    setSuccess(null);
    try {
      const res = await api.startPlugin(name);
      if (res?.status === 'success') {
        setSuccess(`Plugin "${name}" started successfully.`);
        await reloadAll();
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
        await reloadAll();
      } else {
        setError(res?.error || `Failed to stop plugin "${name}"`);
      }
    } catch (err) {
      setError(err.message || `Error stopping plugin "${name}"`);
    }
    setActionLoading(null);
  };

  const openInstallModal = (item) => {
    setInstallTarget(item);
    setSudoPassword('');
    setInstallError(null);
    setInstallMethod('sudo');
  };

  const closeInstallModal = () => {
    setInstallTarget(null);
    setSudoPassword('');
    setInstallError(null);
    setInstallLoading(false);
  };

  const handleConfirmInstall = async () => {
    if (!installTarget) return;
    setInstallLoading(true);
    setInstallError(null);

    const activeMode = getActiveMode(installTarget);
    const targetPkg = activeMode?.package_name || installTarget.package_name || installTarget.name;

    try {
      const res = await api.installPlugin(installTarget.name, {
        method: installMethod,
        password: sudoPassword,
        package: targetPkg,
        mode: activeMode?.id,
      });

      if (res?.status === 'success') {
        const actionVerb = installTarget.isUpdate ? 'updated' : 'installed';
        const versionSuffix = installTarget.version ? ` to v${installTarget.version}` : '';
        setSuccess(`Plugin "${installTarget.title || installTarget.name}" ${actionVerb} successfully${versionSuffix} via ${installMethod}!`);
        closeInstallModal();
        await reloadAll();
      } else {
        setInstallError(res?.error || 'Installation failed.');
      }
    } catch (err) {
      setInstallError(err.message || 'Installation request failed.');
    }
    setInstallLoading(false);
  };

  const handleConfirmUninstall = async () => {
    if (!uninstallTarget) return;
    setActionLoading(uninstallTarget.name);
    setError(null);
    try {
      const res = await api.uninstallPlugin(uninstallTarget.name, {
        method: 'sudo',
        password: sudoPassword,
      });
      if (res?.status === 'success') {
        setSuccess(`Plugin "${uninstallTarget.name}" uninstalled.`);
        setUninstallTarget(null);
        await reloadAll();
      } else {
        setError(res?.error || `Failed to uninstall plugin "${uninstallTarget.name}"`);
      }
    } catch (err) {
      setError(err.message || `Error uninstalling plugin "${uninstallTarget.name}"`);
    }
    setActionLoading(null);
  };

  const copyToClipboard = (text, name) => {
    navigator.clipboard.writeText(text);
    setCopiedName(name);
    setTimeout(() => setCopiedName(null), 2500);
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
    <div className="space-y-6 max-w-7xl mx-auto pb-12">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-[#30363d] pb-6">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-3">
            <span className="p-2 rounded-lg bg-emerald-950/50 border border-emerald-800/60 text-emerald-400">
              <IconPlugins className="w-6 h-6" />
            </span>
            <span>Plugin Manager</span>
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            Supervise external plugins, inspect release changelogs & roadmap specifications, and install via APT or Sudo.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={handleRefreshApt}
            disabled={actionLoading === 'apt_refresh'}
            title="Poll APT keyring for latest plugin candidate versions"
            className="px-3 py-1.5 text-xs font-semibold rounded-lg border border-[#30363d] bg-[#161b22] hover:bg-[#21262d] text-emerald-400 hover:text-emerald-300 transition-colors flex items-center gap-1.5 shadow-sm"
          >
            <IconRefresh className={`w-3.5 h-3.5 ${actionLoading === 'apt_refresh' ? 'animate-spin' : ''}`} />
            <span>{actionLoading === 'apt_refresh' ? 'Checking APT...' : 'Check for Updates'}</span>
          </button>

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
                  Search paths: <code className="text-slate-400">~/.config/m3tal/plugins/</code>, <code className="text-slate-400">/usr/lib/m3tal/plugins/</code>, <code className="text-slate-400">./plugins/</code>
                </p>
              </div>
              <button
                onClick={reloadAll}
                disabled={loading}
                className="px-3 py-1.5 text-xs font-medium rounded-lg bg-[#21262d] hover:bg-[#30363d] text-slate-300 transition-colors flex items-center gap-1.5"
              >
                <IconRefresh className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
                <span>{loading ? 'Refreshing...' : 'Refresh'}</span>
              </button>
            </div>

            {plugins.length === 0 ? (
              <div className="p-12 text-center space-y-4">
                <div className="text-4xl text-slate-500">📦</div>
                <h3 className="text-base font-semibold text-white">No External Plugins Installed</h3>
                <p className="text-xs text-slate-400 max-w-md mx-auto leading-relaxed">
                  M3TAL Core is running in pure platform mode with zero background plugin footprint. To install a plugin with 1 click, explore the <strong>Catalog & Roadmap</strong> tab!
                </p>
                <button
                  onClick={() => setActiveTab('roadmap')}
                  className="px-4 py-2 text-xs font-bold rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white shadow-md transition-all inline-flex items-center gap-2"
                >
                  <span>Browse Catalog & Roadmap</span>
                  <span>→</span>
                </button>
              </div>
            ) : (
              <div className="divide-y divide-[#30363d]">
                {plugins.map((p) => {
                  const manifest = p.manifest || p.Manifest || {};
                  const meta = manifest.metadata || manifest.Metadata || {};
                  const name = meta.name || p.name || 'unknown';
                  const version = meta.version || p.version || '1.0.0';
                  const desc = meta.description || p.description || 'No description provided';
                  const state = p.state || p.State || 'stopped';
                  const caps = p.capabilities || p.Capabilities || manifest.spec?.capabilities || [];
                  const procName = meta.name ? (meta.name.startsWith('m3tal-') ? meta.name : `m3tal-${meta.name}`) : 'm3tal-plugin';

                  // Match catalog entry for changelog and updates
                  const catalogItem = catalog.find((c) => {
                    const cName = c.name || '';
                    const cleanC = cName.replace(/^m3tal-/, '');
                    const cleanP = name.replace(/^m3tal-/, '');
                    return cName === name || cleanC === cleanP;
                  });

                  const semverComp = (catalogItem && catalogItem.version) ? compareSemVer(version, catalogItem.version) : null;
                  const hasUpdate = semverComp && semverComp.cmp === -1;
                  const isDevAhead = semverComp && semverComp.cmp === 1;
                  const isUpToDate = semverComp && semverComp.cmp === 0;

                  return (
                    <div key={name} className="p-5 flex flex-col md:flex-row md:items-center justify-between gap-4 hover:bg-[#21262d]/40 transition-colors">
                      <div className="space-y-1.5">
                        <div className="flex flex-wrap items-center gap-3">
                          <span className="font-bold text-white text-base">{meta.title || name}</span>
                          <span className="text-xs text-slate-400 font-mono">v{version}</span>
                          {getStateBadge(state)}
                          {hasUpdate && (
                            <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-bold bg-amber-950/80 text-amber-300 border border-amber-700/80 animate-pulse">
                              <span className="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                              Update available: v{catalogItem.version} ({semverComp.type})
                            </span>
                          )}
                          {isUpToDate && (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-950/40 text-emerald-400 border border-emerald-800/60">
                              ✓ Up to date
                            </span>
                          )}
                          {isDevAhead && (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-blue-950/40 text-blue-400 border border-blue-800/60">
                              Dev build (ahead of catalog)
                            </span>
                          )}
                          <span className="text-[11px] font-mono text-cyan-400 bg-cyan-950/40 border border-cyan-800/50 px-2 py-0.5 rounded">
                            proc: {procName}
                          </span>
                        </div>
                        <p className="text-xs text-slate-400">{desc}</p>
                        {caps.length > 0 && (
                          <div className="flex flex-wrap items-center gap-1.5 pt-1">
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

                      <div className="flex items-center gap-2 flex-wrap">
                        {hasUpdate && (
                          <button
                            onClick={() => openInstallModal({ ...catalogItem, isUpdate: true, installedVersion: version })}
                            className="px-3 py-1.5 text-xs font-bold rounded-lg bg-amber-500 hover:bg-amber-400 text-slate-950 shadow-md transition-all flex items-center gap-1.5"
                          >
                            <span>⚡ Update to v{catalogItem.version}</span>
                          </button>
                        )}

                        {catalogItem && (
                          <button
                            onClick={() => setSelectedChangelog(catalogItem)}
                            className="px-3 py-1.5 text-xs font-medium rounded-lg bg-[#21262d] hover:bg-[#30363d] text-slate-300 border border-[#30363d] transition-colors flex items-center gap-1.5"
                          >
                            <IconDocument className="w-3.5 h-3.5 text-cyan-400" />
                            <span>Changelog</span>
                          </button>
                        )}

                        {state === 'running' ? (
                          <button
                            onClick={() => handleStop(name)}
                            disabled={actionLoading === name}
                            className="px-3 py-1.5 text-xs font-semibold rounded-lg bg-rose-900/60 hover:bg-rose-800 text-rose-200 border border-rose-700 transition-colors flex items-center gap-1.5"
                          >
                            <IconStop className="w-3.5 h-3.5" />
                            <span>{actionLoading === name ? 'Stopping...' : 'Stop'}</span>
                          </button>
                        ) : (
                          <button
                            onClick={() => handleStart(name)}
                            disabled={actionLoading === name}
                            className="px-3 py-1.5 text-xs font-semibold rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white shadow-sm transition-colors flex items-center gap-1.5"
                          >
                            <IconPlay className="w-3.5 h-3.5" />
                            <span>{actionLoading === name ? 'Starting...' : 'Start'}</span>
                          </button>
                        )}

                        <button
                          onClick={() => setUninstallTarget({ name })}
                          disabled={actionLoading === name}
                          className="p-1.5 text-xs font-medium rounded-lg bg-slate-800 hover:bg-rose-950/60 hover:text-rose-300 text-slate-400 border border-slate-700 transition-colors"
                          title="Uninstall Plugin"
                        >
                          <IconTrash className="w-4 h-4" />
                        </button>
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
        <div className="space-y-6">
          <div className="bg-[#161b22] border border-[#30363d] rounded-xl p-5 space-y-2">
            <h2 className="text-sm font-semibold text-white">M3TAL Plugin Architecture: All Code Lives With The Plugin</h2>
            <p className="text-xs text-slate-400 leading-relaxed">
              In accordance with the M3TAL Plugin Standardization Specification, all external plugins are completely self-contained packages. Every plugin runs in its own canonical process named <code className="text-slate-300">m3tal-&lt;plugin-name&gt;</code>, communicates via JSON-RPC 2.0, and mounts dynamic UI tabs directly into the supervisor without modifying core code.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {catalog.map((item) => {
              const activeMode = getActiveMode(item);
              const cleanItemName = (item.name || '').replace(/^m3tal-/, '');
              const cleanProcName = (item.process_name || item.processName || '').replace(/^m3tal-/, '');
              const effectiveProcName = activeMode?.process_name || item.process_name || item.processName;
              const effectiveAptCommand = activeMode?.apt_command || item.apt_command;

              const matchingPlugin = plugins.find((p) => {
                const pName = p.manifest?.metadata?.name || p.name || '';
                const cleanP = pName.replace(/^m3tal-/, '');
                return (
                  pName === item.name ||
                  cleanP === cleanItemName ||
                  cleanP === cleanProcName ||
                  (activeMode && (pName === activeMode.id || cleanP === activeMode.id))
                );
              });

              const isInstalled = item.status === 'installed' || Boolean(matchingPlugin);
              const installedVersion = matchingPlugin?.manifest?.metadata?.version || matchingPlugin?.version || item.installed_version;

              const semver = (isInstalled && installedVersion && item.version) ? compareSemVer(installedVersion, item.version) : null;
              const hasUpdate = semver && semver.cmp === -1;
              const isDevAhead = semver && semver.cmp === 1;

              const hasDeb = Boolean(item.has_deb || item.deb_filename || item.name === 'docker-monitor' || item.name === 'compose');
              const downloadUrl = api.getPluginDownloadUrl(item.name);

              return (
                <div key={item.name} className="bg-[#161b22] border border-[#30363d] rounded-xl p-5 flex flex-col justify-between space-y-4 hover:border-slate-600 transition-colors">
                  <div className="space-y-3">
                    {/* Header */}
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <div className="flex items-center gap-2 flex-wrap">
                          <h3 className="text-base font-bold text-white">{item.title || item.name}</h3>
                          {isInstalled && hasUpdate ? (
                            <span className="flex items-center gap-1.5 text-xs font-mono font-semibold px-2 py-0.5 rounded bg-amber-950/70 text-amber-300 border border-amber-700/80">
                              <span>v{installedVersion}</span>
                              <span className="text-slate-400">→</span>
                              <span className="text-emerald-400 font-bold">v{item.version}</span>
                            </span>
                          ) : isInstalled && isDevAhead ? (
                            <span className="text-xs font-mono font-semibold px-2 py-0.5 rounded bg-blue-950/60 text-blue-300 border border-blue-700">
                              v{installedVersion} (dev)
                            </span>
                          ) : (
                            <span className="text-xs font-mono font-semibold px-2 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700">
                              v{isInstalled ? (installedVersion || item.version) : (item.version || '0.1.0')}
                            </span>
                          )}
                        </div>
                        <span className="text-[11px] font-mono text-cyan-400">proc: {effectiveProcName}</span>
                      </div>

                      <div className="flex flex-col items-end gap-1.5">
                        <span className="text-[10px] font-semibold px-2 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700">
                          {item.category}
                        </span>
                        {isInstalled ? (
                          hasUpdate ? (
                            <span className="inline-flex items-center gap-1.5 text-[11px] font-bold text-amber-300 bg-amber-950/70 border border-amber-600 px-2.5 py-0.5 rounded-full animate-pulse">
                              <span className="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                              Update Available ({semver.type.toUpperCase()})
                            </span>
                          ) : isDevAhead ? (
                            <span className="inline-flex items-center gap-1.5 text-[11px] font-bold text-blue-300 bg-blue-950/60 border border-blue-700 px-2.5 py-0.5 rounded-full">
                              <span className="w-1.5 h-1.5 rounded-full bg-blue-400"></span>
                              Dev Build (Ahead)
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1.5 text-[11px] font-bold text-emerald-400 bg-emerald-950/60 border border-emerald-800 px-2.5 py-0.5 rounded-full">
                              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                              v{installedVersion || item.version} • Up to Date
                            </span>
                          )
                        ) : hasDeb ? (
                          <span className="text-[11px] font-bold text-cyan-300 bg-cyan-950/60 border border-cyan-800 px-2.5 py-0.5 rounded-full">
                            Available in APT / DEB
                          </span>
                        ) : (
                          <span className="text-[11px] font-medium text-slate-400 bg-slate-800/80 border border-slate-700 px-2.5 py-0.5 rounded-full">
                            Planned Roadmap
                          </span>
                        )}
                      </div>
                    </div>

                    {/* Description */}
                    <p className="text-xs text-slate-400 leading-relaxed">{item.description}</p>

                    {/* Update Capability Diff Notice */}
                    {isInstalled && hasUpdate && (item.capabilities_diff?.length > 0) && (
                      <div className="flex items-center gap-2 p-2 rounded-lg bg-amber-950/30 border border-amber-800/40 text-xs text-amber-200">
                        <span className="font-semibold text-amber-300">New in v{item.version}:</span>
                        <div className="flex flex-wrap gap-1">
                          {item.capabilities_diff.map((c) => (
                            <span key={c} className="px-1.5 py-0.5 rounded bg-amber-900/60 border border-amber-700/60 font-mono text-[10px] text-amber-100">
                              {c}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Mode Selector for Multi-Target Plugins (e.g. Mesh / Node / Cluster) */}
                    {item.modes && item.modes.length > 0 && (
                      <div className="pt-2 space-y-2">
                        <div className="text-[11px] font-semibold text-slate-400">Target Role / Installation Mode:</div>
                        <div className="grid grid-cols-3 gap-1.5 p-1 bg-[#0d1117] border border-[#30363d] rounded-lg">
                          {item.modes.map((m) => {
                            const isSelected = activeMode?.id === m.id;
                            return (
                              <button
                                key={m.id}
                                onClick={() => setSelectedModes((prev) => ({ ...prev, [item.name]: m.id }))}
                                className={`px-2 py-2 text-xs font-bold rounded-lg transition-all text-center flex flex-col items-center justify-center ${
                                  isSelected
                                    ? 'bg-emerald-600 text-white shadow-md shadow-emerald-950/60 ring-1 ring-emerald-400'
                                    : 'text-slate-400 hover:text-white hover:bg-[#21262d]'
                                }`}
                                title={m.description}
                              >
                                <span className="text-xs">{m.label}</span>
                                {m.subLabel && <span className="text-[10px] font-normal opacity-75">({m.subLabel})</span>}
                              </button>
                            );
                          })}
                        </div>
                        {activeMode && (
                          <div className="p-2.5 rounded-lg bg-[#0d1117]/80 border border-[#21262d] text-[11px] text-slate-300 leading-relaxed">
                            <span className="font-semibold text-cyan-400">{activeMode.label}: </span>
                            <span>{activeMode.description}</span>
                          </div>
                        )}
                      </div>
                    )}

                    {/* Capabilities */}
                    {item.capabilities && (
                      <div className="flex flex-wrap items-center gap-1.5 pt-1">
                        {item.capabilities.map((c) => (
                          <span key={c} className="text-[10px] font-medium px-2 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700">
                            {c}
                          </span>
                        ))}
                      </div>
                    )}

                    {/* APT Command Snippet */}
                    {effectiveAptCommand && (
                      <div className="pt-2">
                        <div className="flex items-center justify-between p-2 rounded-lg bg-[#0d1117] border border-[#30363d] text-xs font-mono text-slate-400">
                          <span className="truncate">{effectiveAptCommand}</span>
                          <button
                            onClick={() => copyToClipboard(effectiveAptCommand, `${item.name}-${activeMode?.id || 'default'}`)}
                            className="ml-2 px-2.5 py-1 text-[10px] font-semibold rounded bg-[#21262d] hover:bg-[#30363d] text-slate-300 transition-colors flex items-center gap-1"
                          >
                            {copiedName === `${item.name}-${activeMode?.id || 'default'}` ? (
                              <>
                                <IconCheck className="w-3 h-3 text-emerald-400" />
                                <span className="text-emerald-400">Copied</span>
                              </>
                            ) : (
                              <>
                                <IconCopy className="w-3 h-3 text-slate-400" />
                                <span>Copy</span>
                              </>
                            )}
                          </button>
                        </div>
                      </div>
                    )}
                  </div>

                  {/* Actions Bar */}
                  <div className="pt-3 border-t border-[#30363d] flex flex-wrap items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      {/* Changelog or Roadmap Specs Button */}
                      <button
                        onClick={() => setSelectedChangelog(item)}
                        className="px-3 py-1.5 text-xs font-medium rounded-lg bg-[#21262d] hover:bg-[#30363d] text-slate-300 border border-[#30363d] transition-colors flex items-center gap-1.5"
                      >
                        <IconDocument className="w-3.5 h-3.5 text-cyan-400" />
                        <span>{hasDeb ? 'Changelog' : 'Roadmap Specs'}</span>
                      </button>

                      {/* Download Button */}
                      {hasDeb && (
                        <a
                          href={activeMode ? api.getPluginDownloadUrl(activeMode.id) : downloadUrl}
                          download={activeMode?.deb_filename || item.deb_filename || `m3tal-plugin-${activeMode?.id || item.name}.deb`}
                          className="px-3 py-1.5 text-xs font-medium rounded-lg bg-[#21262d] hover:bg-[#30363d] text-cyan-300 border border-cyan-900/60 transition-colors flex items-center gap-1.5"
                          title="Download official Debian (.deb) package"
                        >
                          <IconDownload className="w-3.5 h-3.5 text-cyan-400" />
                          <span>Download .deb</span>
                        </a>
                      )}
                    </div>

                    {/* Install / Uninstall Controls */}
                    <div>
                      {isInstalled ? (
                        <div className="flex items-center gap-2">
                          {hasUpdate && (
                            <button
                              onClick={() => openInstallModal({ ...item, isUpdate: true, installedVersion })}
                              className="px-3.5 py-1.5 text-xs font-bold rounded-lg bg-amber-500 hover:bg-amber-400 text-slate-950 shadow-md transition-all flex items-center gap-1.5"
                            >
                              <span>⚡ Update to v{item.version}</span>
                            </button>
                          )}
                          <button
                            onClick={() => setActiveTab('installed')}
                            className="px-3 py-1.5 text-xs font-semibold rounded-lg bg-emerald-950/80 text-emerald-300 border border-emerald-800 hover:bg-emerald-900 transition-colors"
                          >
                            Manage in Installed →
                          </button>
                        </div>
                      ) : hasDeb ? (
                        <button
                          onClick={() => openInstallModal({
                            ...item,
                            name: activeMode?.id || item.name,
                            title: activeMode ? `${item.title || item.name} (${activeMode.label})` : (item.title || item.name),
                            deb_filename: activeMode?.deb_filename || item.deb_filename,
                          })}
                          className="px-4 py-1.5 text-xs font-bold rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white shadow-md transition-all flex items-center gap-1.5"
                        >
                          <IconDownload className="w-3.5 h-3.5 text-white" />
                          <span>Install {activeMode?.label || 'Plugin'}</span>
                        </button>
                      ) : (
                        <span className="text-xs text-slate-500 italic">In Development</span>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Sudo Install Confirmation Modal */}
      {installTarget && (
        <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-[#161b22] border border-[#30363d] rounded-2xl max-w-lg w-full p-6 space-y-5 shadow-2xl animate-in fade-in zoom-in-95 duration-150">
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-3">
                <span className="p-2.5 rounded-xl bg-emerald-950/60 border border-emerald-800 text-emerald-400">
                  <IconLock className="w-6 h-6 text-emerald-400" />
                </span>
                <div>
                  <h3 className="text-lg font-bold text-white">
                    {installTarget.isUpdate ? 'Confirm Plugin Update' : 'Confirm Plugin Installation'}
                  </h3>
                  <p className="text-xs text-slate-400 mt-0.5">
                    {installTarget.isUpdate ? (
                      <>
                        Updating <span className="text-white font-semibold">{installTarget.title || installTarget.name}</span> from{' '}
                        <span className="font-mono text-slate-300 font-semibold">v{installTarget.installedVersion || '0.1.0'}</span> to{' '}
                        <span className="font-mono text-emerald-400 font-bold">v{installTarget.version}</span>
                      </>
                    ) : (
                      <>
                        Installing <span className="text-white font-semibold">{installTarget.title || installTarget.name}</span> (v{installTarget.version || '0.1.0'})
                      </>
                    )}
                  </p>
                </div>
              </div>
              <button
                onClick={closeInstallModal}
                disabled={installLoading}
                className="text-slate-400 hover:text-white text-lg p-1"
              >
                ✕
              </button>
            </div>

            {installError && (
              <div className="p-3 bg-rose-950/60 border border-rose-800 rounded-lg text-rose-300 text-xs">
                ⚠️ {installError}
              </div>
            )}

            <div className="space-y-4 text-xs text-slate-300">
              <div className="p-3 bg-[#0d1117] rounded-xl border border-[#30363d] space-y-2">
                <div className="flex justify-between items-center text-slate-400">
                  <span>Package Archive:</span>
                  <span className="font-mono text-cyan-400">{installTarget.deb_filename || `m3tal-plugin-${installTarget.name}.deb`}</span>
                </div>
                <div className="flex justify-between items-center text-slate-400">
                  <span>Target Destination:</span>
                  <span className="font-mono text-slate-300">
                    {installMethod === 'sudo' ? `/usr/lib/m3tal/plugins/${installTarget.name}/` : `~/.config/m3tal/plugins/${installTarget.name}/`}
                  </span>
                </div>
              </div>

              {/* Install Method Selection */}
              <div className="space-y-2">
                <label className="font-semibold text-white">Choose Installation Target:</label>
                <div className="grid grid-cols-1 gap-2">
                  <label
                    className={`p-3 rounded-xl border cursor-pointer flex items-start gap-3 transition-colors ${
                      installMethod === 'sudo'
                        ? 'bg-emerald-950/30 border-emerald-700 text-white'
                        : 'bg-[#0d1117] border-[#30363d] text-slate-400 hover:border-slate-600'
                    }`}
                  >
                    <input
                      type="radio"
                      name="installMethod"
                      value="sudo"
                      checked={installMethod === 'sudo'}
                      onChange={() => setInstallMethod('sudo')}
                      className="mt-0.5 text-emerald-500 focus:ring-emerald-500"
                    />
                    <div>
                      <div className="font-bold text-white">System Debian Package via Sudo (Recommended)</div>
                      <div className="text-[11px] text-slate-400 mt-0.5">
                        Installs to system directory <code className="text-slate-300">/usr/lib/m3tal/plugins/</code> via <code className="text-slate-300">dpkg</code>.
                      </div>
                    </div>
                  </label>

                  <label
                    className={`p-3 rounded-xl border cursor-pointer flex items-start gap-3 transition-colors ${
                      installMethod === 'user'
                        ? 'bg-emerald-950/30 border-emerald-700 text-white'
                        : 'bg-[#0d1117] border-[#30363d] text-slate-400 hover:border-slate-600'
                    }`}
                  >
                    <input
                      type="radio"
                      name="installMethod"
                      value="user"
                      checked={installMethod === 'user'}
                      onChange={() => setInstallMethod('user')}
                      className="mt-0.5 text-emerald-500 focus:ring-emerald-500"
                    />
                    <div>
                      <div className="font-bold text-white">User-Space Profile (No Sudo)</div>
                      <div className="text-[11px] text-slate-400 mt-0.5">
                        Extracts payload directly into <code className="text-slate-300">~/.config/m3tal/plugins/</code>.
                      </div>
                    </div>
                  </label>
                </div>
              </div>

              {/* Sudo Password Input */}
              {installMethod === 'sudo' && (
                <div className="space-y-1.5 pt-1">
                  <label className="font-semibold text-white flex items-center justify-between">
                    <span>Sudo Password Confirmation:</span>
                    <span className="text-[10px] text-slate-400 font-normal">Passed securely to sudo -S</span>
                  </label>
                  <div className="relative">
                    <input
                      type={showPassword ? 'text' : 'password'}
                      value={sudoPassword}
                      onChange={(e) => setSudoPassword(e.target.value)}
                      placeholder="Enter your Linux sudo password"
                      className="w-full px-3.5 py-2.5 rounded-xl bg-[#0d1117] border border-[#30363d] focus:border-emerald-500 focus:outline-none text-white text-xs font-mono"
                      autoFocus
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') handleConfirmInstall();
                      }}
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-3 top-2.5 text-slate-400 hover:text-white text-xs"
                    >
                      {showPassword ? 'Hide' : 'Show'}
                    </button>
                  </div>
                </div>
              )}
            </div>

            {/* Modal Actions */}
            <div className="flex items-center justify-end gap-3 pt-2 border-t border-[#30363d]">
              <button
                onClick={closeInstallModal}
                disabled={installLoading}
                className="px-4 py-2 text-xs font-semibold rounded-lg bg-[#21262d] hover:bg-[#30363d] text-slate-300 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmInstall}
                disabled={installLoading || (installMethod === 'sudo' && !sudoPassword)}
                className={`px-5 py-2 text-xs font-bold rounded-lg text-white shadow-md transition-all flex items-center gap-2 ${
                  installLoading || (installMethod === 'sudo' && !sudoPassword)
                    ? 'bg-slate-700 cursor-not-allowed opacity-60'
                    : installTarget.isUpdate
                      ? 'bg-amber-500 hover:bg-amber-400 text-slate-950 font-black'
                      : 'bg-emerald-600 hover:bg-emerald-500'
                }`}
              >
                {installLoading ? (
                  <>
                    <span className="w-3.5 h-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
                    <span>{installTarget.isUpdate ? 'Updating Package...' : 'Installing Package...'}</span>
                  </>
                ) : (
                  <>
                    <span>{installTarget.isUpdate ? 'Confirm & Update' : 'Confirm & Install'}</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Changelog & Roadmap Specs Modal */}
      {selectedChangelog && (
        <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-[#161b22] border border-[#30363d] rounded-2xl max-w-2xl w-full max-h-[85vh] flex flex-col shadow-2xl animate-in fade-in zoom-in-95 duration-150 overflow-hidden">
            {/* Modal Header */}
            <div className="p-5 border-b border-[#30363d] flex items-center justify-between">
              <div className="flex items-center gap-3">
                <span className="p-2 rounded-lg bg-cyan-950/60 border border-cyan-800 text-cyan-400">
                  <IconDocument className="w-5 h-5 text-cyan-400" />
                </span>
                <div>
                  <h3 className="text-base font-bold text-white">
                    {selectedChangelog.title || selectedChangelog.name}
                  </h3>
                  <div className="flex items-center gap-2 mt-0.5">
                    <span className="text-xs font-mono text-emerald-400">v{selectedChangelog.version || '0.1.0'}</span>
                    <span className="text-slate-500">•</span>
                    <span className="text-xs text-slate-400 font-mono">Package: {selectedChangelog.package_name || selectedChangelog.packageName || `m3tal-plugin-${selectedChangelog.name}`}</span>
                  </div>
                </div>
              </div>
              <button
                onClick={() => setSelectedChangelog(null)}
                className="text-slate-400 hover:text-white p-1 text-lg"
              >
                ✕
              </button>
            </div>

            {/* Modal Content */}
            <div className="p-6 overflow-y-auto space-y-4 text-slate-300 text-xs leading-relaxed font-sans">
              {selectedChangelog.changelog ? (
                <div className="prose prose-invert max-w-none space-y-3">
                  <pre className="p-4 bg-[#0d1117] border border-[#30363d] rounded-xl whitespace-pre-wrap font-sans text-xs text-slate-300 leading-relaxed">
                    {selectedChangelog.changelog}
                  </pre>
                </div>
              ) : (
                <div className="space-y-4">
                  <div className="p-4 bg-[#0d1117] border border-[#30363d] rounded-xl space-y-2">
                    <h4 className="text-sm font-bold text-white">{selectedChangelog.title || selectedChangelog.name} Specification</h4>
                    <p className="text-slate-400 leading-relaxed">{selectedChangelog.description}</p>
                  </div>
                  <div className="space-y-2">
                    <h5 className="font-semibold text-white">Target Capabilities:</h5>
                    <div className="flex flex-wrap gap-1.5 pt-1">
                      {selectedChangelog.capabilities?.map((c) => (
                        <span key={c} className="font-mono text-cyan-300 bg-cyan-950/40 border border-cyan-800/60 px-2 py-0.5 rounded text-[11px]">{c}</span>
                      ))}
                    </div>
                  </div>
                  {selectedChangelog.apt_command && (
                    <div className="p-3 bg-[#0d1117] border border-[#30363d] rounded-xl space-y-1">
                      <div className="text-slate-400 text-[11px]">APT Package Command (upon release):</div>
                      <code className="text-emerald-400 font-mono">{selectedChangelog.apt_command}</code>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Modal Footer */}
            <div className="p-4 border-t border-[#30363d] bg-[#161b22] flex items-center justify-between">
              {selectedChangelog.has_deb && (
                <a
                  href={api.getPluginDownloadUrl(selectedChangelog.name)}
                  download={selectedChangelog.deb_filename || `m3tal-plugin-${selectedChangelog.name}.deb`}
                  className="px-3 py-1.5 text-xs font-medium rounded-lg bg-[#21262d] hover:bg-[#30363d] text-cyan-300 border border-cyan-900/60 transition-colors inline-flex items-center gap-1.5"
                >
                  <IconDownload className="w-3.5 h-3.5 text-cyan-400" />
                  <span>Download .deb Archive</span>
                </a>
              )}
              <button
                onClick={() => setSelectedChangelog(null)}
                className="ml-auto px-4 py-1.5 text-xs font-semibold rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white transition-colors"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Uninstall Confirmation Modal */}
      {uninstallTarget && (
        <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-[#161b22] border border-[#30363d] rounded-2xl max-w-md w-full p-6 space-y-4 shadow-2xl">
            <h3 className="text-base font-bold text-white">Confirm Plugin Removal</h3>
            <p className="text-xs text-slate-400">
              Are you sure you want to uninstall <strong className="text-white">{uninstallTarget.name}</strong>? Its subprocess will be terminated and package removed.
            </p>
            <div className="flex items-center justify-end gap-3 pt-2">
              <button
                onClick={() => setUninstallTarget(null)}
                className="px-4 py-1.5 text-xs font-semibold rounded-lg bg-[#21262d] text-slate-300 hover:bg-[#30363d]"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmUninstall}
                className="px-4 py-1.5 text-xs font-bold rounded-lg bg-rose-600 hover:bg-rose-500 text-white"
              >
                Uninstall
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
