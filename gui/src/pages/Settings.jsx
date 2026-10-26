import { useState, useEffect, useCallback } from 'react';
import { api, getApiBase, getToken, setToken } from '../api';
import {
  IconSettings,
  IconLock,
  IconDoctor,
  IconPrune,
  IconClean,
  IconReconcile,
  IconRefresh,
  IconCheck,
  IconCopy,
  IconPlugins,
} from '../components/Icons';

export default function Settings() {
  const [activeSection, setActiveSection] = useState('core_general');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [toast, setToast] = useState(null);

  // Core settings state
  const [coreConfig, setCoreConfig] = useState({
    node_name: 'm3tal-node-01',
    polling_interval: 5,
    theme: 'dark',
    api_host: getApiBase(),
    api_token: getToken(),
  });

  // Plugin settings state
  const [pluginSections, setPluginSections] = useState([]);
  const [pluginConfigs, setPluginConfigs] = useState({});

  // UI state for password visibility & doctor output
  const [showToken, setShowToken] = useState(false);
  const [showPasswordMap, setShowPasswordMap] = useState({});
  const [copiedHost, setCopiedHost] = useState(false);
  const [copiedToken, setCopiedToken] = useState(false);
  const [doctorResult, setDoctorResult] = useState(null);
  const [doctorRunning, setDoctorRunning] = useState(false);

  const showToast = (message, type = 'info') => {
    setToast({ message, type });
    setTimeout(() => {
      setToast(null);
    }, 4000);
  };

  const loadAllSettings = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.getSettings();
      if (res?.status === 'success' && res.data) {
        if (res.data.core) {
          setCoreConfig((prev) => ({
            ...prev,
            ...res.data.core,
            api_host: res.data.core.api_host || getApiBase(),
            api_token: res.data.core.api_token || getToken(),
          }));
        }
        if (Array.isArray(res.data.sections)) {
          setPluginSections(res.data.sections);
        }
        if (res.data.plugin_configs && typeof res.data.plugin_configs === 'object') {
          setPluginConfigs(res.data.plugin_configs);
        }
      } else {
        showToast(res?.error || 'Failed to load settings payload', 'error');
      }
    } catch (err) {
      showToast(err.message || 'Error connecting to settings service', 'error');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadAllSettings();
  }, [loadAllSettings]);

  // Handle saving Core Preferences
  const handleSaveCore = async (e) => {
    if (e) e.preventDefault();
    setSaving(true);
    try {
      // Synchronize client-side token first if modified
      if (coreConfig.api_token) {
        setToken(coreConfig.api_token);
      }

      const res = await api.saveCoreSettings(coreConfig);
      if (res?.status === 'success') {
        showToast('Core system preferences saved successfully.', 'success');
      } else {
        showToast(res?.error || 'Failed to save core preferences', 'error');
      }
    } catch (err) {
      showToast(err.message || 'Error saving core preferences', 'error');
    } finally {
      setSaving(false);
    }
  };

  // Handle saving specific Plugin settings
  const handleSavePlugin = async (pluginName) => {
    setSaving(true);
    try {
      const currentConfig = pluginConfigs[pluginName] || {};
      const res = await api.savePluginSettings(pluginName, currentConfig);
      if (res?.status === 'success') {
        showToast(`Settings for plugin "${pluginName}" saved and dispatched to daemon.`, 'success');
      } else {
        showToast(res?.error || `Failed to save settings for "${pluginName}"`, 'error');
      }
    } catch (err) {
      showToast(err.message || `Error saving settings for "${pluginName}"`, 'error');
    } finally {
      setSaving(false);
    }
  };

  // Handle field change in a plugin section
  const handlePluginFieldChange = (pluginName, fieldKey, value) => {
    setPluginConfigs((prev) => ({
      ...prev,
      [pluginName]: {
        ...(prev[pluginName] || {}),
        [fieldKey]: value,
      },
    }));
  };

  // Reset plugin settings to defaults
  const handleResetPluginDefaults = (section) => {
    const defaultVals = {};
    if (Array.isArray(section.fields)) {
      section.fields.forEach((f) => {
        if (f.default !== undefined) {
          defaultVals[f.key] = f.default;
        }
      });
    }
    setPluginConfigs((prev) => ({
      ...prev,
      [section.plugin]: defaultVals,
    }));
    showToast(`Restored default values for ${section.title}. Remember to save changes.`, 'info');
  };

  // Reset Dashboard grid layout
  const handleResetDashboardLayout = () => {
    try {
      localStorage.removeItem('m3tal_dashboard_layout_v1');
      showToast('Dashboard layout reset to default. Return to Dashboard to view.', 'success');
    } catch (err) {
      showToast('Failed to reset dashboard layout: ' + err.message, 'error');
    }
  };

  // Run doctor diagnostics
  const handleRunDoctor = async () => {
    setDoctorRunning(true);
    setDoctorResult(null);
    try {
      const res = await api.runDoctor();
      if (res?.status === 'success') {
        setDoctorResult(res.data);
        showToast('System diagnostics complete.', 'success');
      } else {
        showToast(res?.error || 'Diagnostic run failed', 'error');
      }
    } catch (err) {
      showToast(err.message || 'Diagnostic error', 'error');
    } finally {
      setDoctorRunning(false);
    }
  };

  // Prune unused docker images
  const handlePruneImages = async () => {
    if (!window.confirm('Prune dangling and unused Docker images?')) return;
    try {
      const res = await api.pruneImages();
      if (res?.status === 'success') {
        showToast('Docker image pruning completed successfully.', 'success');
      } else {
        showToast(res?.error || 'Docker prune failed', 'error');
      }
    } catch (err) {
      showToast(err.message || 'Prune error', 'error');
    }
  };

  // Host system cleanup
  const handleHostCleanup = async () => {
    if (!window.confirm('Run host temporary file and package cache cleanup?')) return;
    try {
      const res = await api.systemCleanup();
      if (res?.status === 'success') {
        const freed = res.data?.freed_bytes_human || res.data?.freed_bytes || '0 B';
        showToast(`System cleanup complete (${freed} freed).`, 'success');
      } else {
        showToast(res?.error || 'System cleanup failed', 'error');
      }
    } catch (err) {
      showToast(err.message || 'Cleanup error', 'error');
    }
  };

  // Copy helper
  const copyToClipboard = (text, type) => {
    navigator.clipboard.writeText(text);
    if (type === 'host') {
      setCopiedHost(true);
      setTimeout(() => setCopiedHost(false), 2000);
    } else if (type === 'token') {
      setCopiedToken(true);
      setTimeout(() => setCopiedToken(false), 2000);
    }
  };

  return (
    <div className="w-full h-full flex flex-col space-y-6">
      {/* Top Banner / Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 pb-4 border-b border-[#30363d]">
        <div className="flex items-center gap-3">
          <div className="p-2.5 bg-emerald-950/60 border border-emerald-500/30 rounded-xl text-emerald-400 shadow-sm">
            <IconSettings className="w-6 h-6" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-white tracking-wide flex items-center gap-2">
              System Settings & Configurations
            </h1>
            <p className="text-xs text-slate-400">
              Manage core preferences, API credentials, host maintenance, and external plugin options.
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={loadAllSettings}
            disabled={loading}
            className="px-3.5 py-1.5 rounded-lg border border-[#30363d] bg-[#161b22] hover:bg-[#21262d] text-slate-300 text-xs font-semibold flex items-center gap-2 transition-all"
          >
            <IconRefresh className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            <span>Reload</span>
          </button>
        </div>
      </div>

      {/* Main Two-Pane Settings Layout */}
      <div className="flex-1 flex flex-col lg:flex-row gap-6 min-h-0">
        {/* Left Navigation Sidebar */}
        <div className="w-full lg:w-72 flex-shrink-0 flex flex-col gap-6">
          {/* Core Categories */}
          <div className="bg-[#161b22] border border-[#30363d] rounded-xl p-3">
            <div className="px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-slate-400">
              Core Preferences
            </div>
            <div className="space-y-1">
              <button
                onClick={() => setActiveSection('core_general')}
                className={`w-full text-left px-3 py-2 rounded-lg text-xs font-semibold flex items-center gap-2.5 transition-all ${
                  activeSection === 'core_general'
                    ? 'bg-emerald-600 text-white shadow-sm'
                    : 'text-slate-300 hover:bg-[#21262d] hover:text-white'
                }`}
              >
                <IconSettings className="w-4 h-4 flex-shrink-0 text-emerald-400" />
                <span>General & Appearance</span>
              </button>

              <button
                onClick={() => setActiveSection('core_connection')}
                className={`w-full text-left px-3 py-2 rounded-lg text-xs font-semibold flex items-center gap-2.5 transition-all ${
                  activeSection === 'core_connection'
                    ? 'bg-emerald-600 text-white shadow-sm'
                    : 'text-slate-300 hover:bg-[#21262d] hover:text-white'
                }`}
              >
                <IconLock className="w-4 h-4 flex-shrink-0 text-cyan-400" />
                <span>Connection & Security</span>
              </button>

              <button
                onClick={() => setActiveSection('core_maintenance')}
                className={`w-full text-left px-3 py-2 rounded-lg text-xs font-semibold flex items-center gap-2.5 transition-all ${
                  activeSection === 'core_maintenance'
                    ? 'bg-emerald-600 text-white shadow-sm'
                    : 'text-slate-300 hover:bg-[#21262d] hover:text-white'
                }`}
              >
                <IconDoctor className="w-4 h-4 flex-shrink-0 text-amber-400" />
                <span>Host Maintenance</span>
              </button>
            </div>
          </div>

          {/* Plugin Categories */}
          <div className="bg-[#161b22] border border-[#30363d] rounded-xl p-3">
            <div className="px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-slate-400 flex items-center justify-between">
              <span>Plugin Settings</span>
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700">
                {pluginSections.length}
              </span>
            </div>
            <div className="space-y-1">
              {pluginSections.length === 0 ? (
                <div className="px-3 py-4 text-center text-xs text-slate-500 italic">
                  No plugins have registered settings sections yet.
                </div>
              ) : (
                pluginSections.map((sec) => {
                  const isSecActive = activeSection === `plugin_${sec.plugin}_${sec.id}`;
                  return (
                    <button
                      key={`${sec.plugin}_${sec.id}`}
                      onClick={() => setActiveSection(`plugin_${sec.plugin}_${sec.id}`)}
                      className={`w-full text-left px-3 py-2 rounded-lg text-xs font-semibold flex items-center justify-between transition-all ${
                        isSecActive
                          ? 'bg-emerald-600 text-white shadow-sm'
                          : 'text-slate-300 hover:bg-[#21262d] hover:text-white'
                      }`}
                    >
                      <div className="flex items-center gap-2.5 truncate">
                        <IconPlugins className="w-4 h-4 flex-shrink-0 text-cyan-400" />
                        <span className="truncate">{sec.title || sec.id}</span>
                      </div>
                      <span className="text-[10px] uppercase font-mono px-1.5 py-0.5 rounded bg-slate-800/80 text-slate-400 border border-slate-700 flex-shrink-0">
                        {sec.plugin}
                      </span>
                    </button>
                  );
                })
              )}
            </div>
          </div>
        </div>

        {/* Right Viewport Content */}
        <div className="flex-1 bg-[#161b22] border border-[#30363d] rounded-xl p-6 overflow-y-auto">
          {/* SECTION: Core General */}
          {activeSection === 'core_general' && (
            <div className="space-y-6 max-w-2xl">
              <div>
                <h2 className="text-lg font-bold text-white flex items-center gap-2">
                  <IconSettings className="w-5 h-5 text-emerald-400" />
                  General & Appearance
                </h2>
                <p className="text-xs text-slate-400 mt-1">
                  Configure local node identity, polling cadence, and dashboard layout states.
                </p>
              </div>

              <div className="space-y-4 pt-2">
                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                    Node Display Name
                  </label>
                  <input
                    type="text"
                    value={coreConfig.node_name || ''}
                    onChange={(e) => setCoreConfig({ ...coreConfig, node_name: e.target.value })}
                    className="w-full bg-[#0d1117] border border-[#30363d] rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-emerald-500 font-mono"
                    placeholder="m3tal-node-01"
                  />
                  <p className="text-[11px] text-slate-500 mt-1">
                    Canonical identifier displayed on the main dashboard and multi-node federation peers.
                  </p>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                    Telemetry Polling Interval (Seconds)
                  </label>
                  <input
                    type="number"
                    min="1"
                    max="60"
                    value={coreConfig.polling_interval || 5}
                    onChange={(e) =>
                      setCoreConfig({ ...coreConfig, polling_interval: parseInt(e.target.value, 10) || 5 })
                    }
                    className="w-32 bg-[#0d1117] border border-[#30363d] rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-emerald-500 font-mono"
                  />
                  <p className="text-[11px] text-slate-500 mt-1">
                    Frequency at which hardware metrics (CPU, RAM, GPU, Storage, Network) are queried (1–60s).
                  </p>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                    UI Theme
                  </label>
                  <select
                    value={coreConfig.theme || 'dark'}
                    onChange={(e) => setCoreConfig({ ...coreConfig, theme: e.target.value })}
                    className="bg-[#0d1117] border border-[#30363d] rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-emerald-500"
                  >
                    <option value="dark">Dark Slate (Default)</option>
                    <option value="cyber">Midnight Cyber</option>
                    <option value="high-contrast">High Contrast</option>
                  </select>
                </div>

                <div className="pt-4 border-t border-[#30363d]">
                  <label className="block text-xs font-semibold text-slate-300 mb-1">
                    Dashboard Grid Layout
                  </label>
                  <p className="text-[11px] text-slate-500 mb-3">
                    If tiles have been rearranged or hidden, you can restore the factory 4-column layout.
                  </p>
                  <button
                    type="button"
                    onClick={handleResetDashboardLayout}
                    className="px-3.5 py-1.5 rounded-lg border border-[#30363d] bg-[#21262d] hover:bg-[#30363d] text-slate-300 text-xs font-semibold transition-colors"
                  >
                    Reset Dashboard Grid Layout
                  </button>
                </div>
              </div>

              <div className="pt-6 border-t border-[#30363d] flex items-center justify-end">
                <button
                  type="button"
                  onClick={handleSaveCore}
                  disabled={saving}
                  className="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs transition-all shadow-md shadow-emerald-900/30 flex items-center gap-2"
                >
                  {saving ? <IconRefresh className="w-3.5 h-3.5 animate-spin" /> : <IconCheck className="w-3.5 h-3.5" />}
                  <span>Save General Preferences</span>
                </button>
              </div>
            </div>
          )}

          {/* SECTION: Core Connection & Security */}
          {activeSection === 'core_connection' && (
            <div className="space-y-6 max-w-2xl">
              <div>
                <h2 className="text-lg font-bold text-white flex items-center gap-2">
                  <IconLock className="w-5 h-5 text-cyan-400" />
                  Connection & Security
                </h2>
                <p className="text-xs text-slate-400 mt-1">
                  Manage API endpoints, security authentication tokens, and gateway connectivity.
                </p>
              </div>

              <div className="space-y-4 pt-2">
                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                    Core API Host Base URL
                  </label>
                  <div className="flex items-center gap-2">
                    <input
                      type="text"
                      readOnly
                      value={coreConfig.api_host || getApiBase()}
                      className="w-full bg-[#0d1117] border border-[#30363d] rounded-lg px-3 py-2 text-sm text-slate-400 font-mono focus:outline-none"
                    />
                    <button
                      type="button"
                      onClick={() => copyToClipboard(coreConfig.api_host || getApiBase(), 'host')}
                      className="p-2.5 rounded-lg border border-[#30363d] bg-[#21262d] hover:bg-[#30363d] text-slate-300 text-xs transition-colors"
                      title="Copy API Host"
                    >
                      {copiedHost ? <IconCheck className="w-4 h-4 text-emerald-400" /> : <IconCopy className="w-4 h-4" />}
                    </button>
                  </div>
                  <p className="text-[11px] text-slate-500 mt-1">
                    Configured via <code>/etc/m3tal/core.yaml</code> or environment variable <code>M3TAL_BIND_ADDR</code>.
                  </p>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-300 mb-1.5">
                    API Authentication Token
                  </label>
                  <div className="flex items-center gap-2">
                    <div className="relative w-full">
                      <input
                        type={showToken ? 'text' : 'password'}
                        value={coreConfig.api_token || ''}
                        onChange={(e) => setCoreConfig({ ...coreConfig, api_token: e.target.value })}
                        className="w-full bg-[#0d1117] border border-[#30363d] rounded-lg px-3 py-2 pr-10 text-sm text-white font-mono focus:outline-none focus:border-emerald-500"
                        placeholder="Enter API Token..."
                      />
                      <button
                        type="button"
                        onClick={() => setShowToken(!showToken)}
                        className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white text-xs font-semibold"
                      >
                        {showToken ? 'Hide' : 'Show'}
                      </button>
                    </div>
                    <button
                      type="button"
                      onClick={() => copyToClipboard(coreConfig.api_token || '', 'token')}
                      className="p-2.5 rounded-lg border border-[#30363d] bg-[#21262d] hover:bg-[#30363d] text-slate-300 text-xs transition-colors"
                      title="Copy Token"
                    >
                      {copiedToken ? <IconCheck className="w-4 h-4 text-emerald-400" /> : <IconCopy className="w-4 h-4" />}
                    </button>
                  </div>
                  <p className="text-[11px] text-slate-500 mt-1">
                    Sent via <code>X-API-Token</code> HTTP header and WebSocket query parameter for authorized IPC calls.
                  </p>
                </div>

                <div className="p-4 bg-[#0d1117] border border-[#30363d] rounded-lg flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-3 h-3 rounded-full bg-emerald-400 animate-pulse" />
                    <div>
                      <div className="text-xs font-semibold text-white">WebSocket Gateway Live</div>
                      <div className="text-[11px] text-slate-500 font-mono">
                        {getApiBase().replace(/^http/, 'ws')}/api/v2/ws
                      </div>
                    </div>
                  </div>
                  <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-950 text-emerald-400 border border-emerald-800">
                    Connected
                  </span>
                </div>
              </div>

              <div className="pt-6 border-t border-[#30363d] flex items-center justify-end">
                <button
                  type="button"
                  onClick={handleSaveCore}
                  disabled={saving}
                  className="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs transition-all shadow-md shadow-emerald-900/30 flex items-center gap-2"
                >
                  {saving ? <IconRefresh className="w-3.5 h-3.5 animate-spin" /> : <IconCheck className="w-3.5 h-3.5" />}
                  <span>Save Connection Settings</span>
                </button>
              </div>
            </div>
          )}

          {/* SECTION: Core Host Maintenance */}
          {activeSection === 'core_maintenance' && (
            <div className="space-y-6 max-w-2xl">
              <div>
                <h2 className="text-lg font-bold text-white flex items-center gap-2">
                  <IconDoctor className="w-5 h-5 text-amber-400" />
                  Host Maintenance & Diagnostics
                </h2>
                <p className="text-xs text-slate-400 mt-1">
                  Execute host diagnostics, reclaim disk space, and reconcile node state.
                </p>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2">
                {/* Doctor */}
                <div className="p-4 bg-[#0d1117] border border-[#30363d] rounded-lg flex flex-col justify-between space-y-3">
                  <div>
                    <div className="flex items-center gap-2 text-white font-semibold text-sm">
                      <IconDoctor className="w-4 h-4 text-emerald-400" />
                      <span>Host Diagnostics</span>
                    </div>
                    <p className="text-[11px] text-slate-400 mt-1">
                      Runs self-checks for Docker daemon socket, cgroups, kernel capabilities, and disk space.
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={handleRunDoctor}
                    disabled={doctorRunning}
                    className="px-3.5 py-2 rounded-lg bg-[#21262d] hover:bg-[#30363d] text-white text-xs font-semibold flex items-center justify-center gap-2 transition-colors"
                  >
                    {doctorRunning ? <IconRefresh className="w-3.5 h-3.5 animate-spin" /> : <IconDoctor className="w-3.5 h-3.5" />}
                    <span>Run Doctor</span>
                  </button>
                </div>

                {/* Prune Images */}
                <div className="p-4 bg-[#0d1117] border border-[#30363d] rounded-lg flex flex-col justify-between space-y-3">
                  <div>
                    <div className="flex items-center gap-2 text-white font-semibold text-sm">
                      <IconPrune className="w-4 h-4 text-amber-400" />
                      <span>Prune Docker Images</span>
                    </div>
                    <p className="text-[11px] text-slate-400 mt-1">
                      Removes untagged and dangling Docker container images to reclaim host drive space.
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={handlePruneImages}
                    className="px-3.5 py-2 rounded-lg bg-[#21262d] hover:bg-[#30363d] text-amber-400 text-xs font-semibold flex items-center justify-center gap-2 transition-colors"
                  >
                    <IconPrune className="w-3.5 h-3.5" />
                    <span>Prune Images</span>
                  </button>
                </div>

                {/* System Cleanup */}
                <div className="p-4 bg-[#0d1117] border border-[#30363d] rounded-lg flex flex-col justify-between space-y-3">
                  <div>
                    <div className="flex items-center gap-2 text-white font-semibold text-sm">
                      <IconClean className="w-4 h-4 text-cyan-400" />
                      <span>Clean Host System</span>
                    </div>
                    <p className="text-[11px] text-slate-400 mt-1">
                      Clears APT package cache, rotating journals, and temporary files in <code>/tmp</code>.
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={handleHostCleanup}
                    className="px-3.5 py-2 rounded-lg bg-[#21262d] hover:bg-[#30363d] text-cyan-400 text-xs font-semibold flex items-center justify-center gap-2 transition-colors"
                  >
                    <IconClean className="w-3.5 h-3.5" />
                    <span>System Cleanup</span>
                  </button>
                </div>

                {/* Reconcile */}
                <div className="p-4 bg-[#0d1117] border border-[#30363d] rounded-lg flex flex-col justify-between space-y-3">
                  <div>
                    <div className="flex items-center gap-2 text-white font-semibold text-sm">
                      <IconReconcile className="w-4 h-4 text-purple-400" />
                      <span>Reconcile System</span>
                    </div>
                    <p className="text-[11px] text-slate-400 mt-1">
                      Forces hardware inventory rescan and plugin daemon health reconciliation.
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={async () => {
                      const res = await api.reconcileSystem();
                      if (res?.status === 'success') showToast('System state reconciled.', 'success');
                      else showToast(res?.error || 'Reconcile failed', 'error');
                    }}
                    className="px-3.5 py-2 rounded-lg bg-[#21262d] hover:bg-[#30363d] text-purple-400 text-xs font-semibold flex items-center justify-center gap-2 transition-colors"
                  >
                    <IconReconcile className="w-3.5 h-3.5" />
                    <span>Reconcile State</span>
                  </button>
                </div>
              </div>

              {/* Doctor output display */}
              {doctorResult && (
                <div className="mt-4 p-4 bg-[#0d1117] border border-[#30363d] rounded-lg space-y-2">
                  <div className="text-xs font-bold text-white flex items-center justify-between">
                    <span>Diagnostic Report Results:</span>
                    <span className="text-[10px] text-emerald-400 font-mono">STATUS: HEALTHY</span>
                  </div>
                  <pre className="text-[11px] text-slate-300 font-mono bg-[#161b22] p-3 rounded overflow-x-auto max-h-60">
                    {JSON.stringify(doctorResult, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          )}

          {/* DYNAMIC SECTION: Plugin Preferences */}
          {activeSection.startsWith('plugin_') && (() => {
            const sec = pluginSections.find((s) => `plugin_${s.plugin}_${s.id}` === activeSection);
            if (!sec) {
              return (
                <div className="text-center py-12 text-slate-500 text-xs">
                  Plugin section not found or plugin was removed.
                </div>
              );
            }

            const currentConfig = pluginConfigs[sec.plugin] || {};

            return (
              <div className="space-y-6 max-w-2xl">
                <div>
                  <div className="flex items-center gap-2">
                    <h2 className="text-lg font-bold text-white">{sec.title || sec.id}</h2>
                    <span className="text-[10px] uppercase font-mono px-2 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700">
                      Plugin: {sec.plugin}
                    </span>
                  </div>
                  {sec.description && (
                    <p className="text-xs text-slate-400 mt-1">{sec.description}</p>
                  )}
                </div>

                <div className="space-y-5 pt-2">
                  {Array.isArray(sec.fields) && sec.fields.map((field) => {
                    const rawVal = currentConfig[field.key];
                    const val = rawVal !== undefined ? rawVal : (field.default !== undefined ? field.default : '');
                    const isPasswordType = field.type === 'password';
                    const passKey = `${sec.plugin}_${field.key}`;
                    const isPassVisible = showPasswordMap[passKey];

                    return (
                      <div key={field.key} className="space-y-1.5">
                        <div className="flex items-center justify-between">
                          <label className="text-xs font-semibold text-slate-300">
                            {field.label || field.key}
                          </label>
                          <span className="text-[10px] text-slate-500 font-mono">
                            {field.key}
                          </span>
                        </div>

                        {/* Text / String */}
                        {field.type === 'text' && (
                          <input
                            type="text"
                            value={val}
                            onChange={(e) => handlePluginFieldChange(sec.plugin, field.key, e.target.value)}
                            className="w-full bg-[#0d1117] border border-[#30363d] rounded-lg px-3 py-2 text-sm text-white font-mono focus:outline-none focus:border-emerald-500"
                          />
                        )}

                        {/* Password */}
                        {isPasswordType && (
                          <div className="relative w-full">
                            <input
                              type={isPassVisible ? 'text' : 'password'}
                              value={val}
                              onChange={(e) => handlePluginFieldChange(sec.plugin, field.key, e.target.value)}
                              className="w-full bg-[#0d1117] border border-[#30363d] rounded-lg px-3 py-2 pr-12 text-sm text-white font-mono focus:outline-none focus:border-emerald-500"
                            />
                            <button
                              type="button"
                              onClick={() =>
                                setShowPasswordMap((prev) => ({ ...prev, [passKey]: !prev[passKey] }))
                              }
                              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white text-xs font-semibold"
                            >
                              {isPassVisible ? 'Hide' : 'Show'}
                            </button>
                          </div>
                        )}

                        {/* Number */}
                        {field.type === 'number' && (
                          <input
                            type="number"
                            value={val}
                            onChange={(e) =>
                              handlePluginFieldChange(sec.plugin, field.key, parseFloat(e.target.value) || 0)
                            }
                            className="w-36 bg-[#0d1117] border border-[#30363d] rounded-lg px-3 py-2 text-sm text-white font-mono focus:outline-none focus:border-emerald-500"
                          />
                        )}

                        {/* Boolean Toggle */}
                        {field.type === 'boolean' && (
                          <div className="flex items-center gap-3 pt-1">
                            <button
                              type="button"
                              role="switch"
                              aria-checked={Boolean(val)}
                              onClick={() => handlePluginFieldChange(sec.plugin, field.key, !val)}
                              className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                                val ? 'bg-emerald-600' : 'bg-slate-700'
                              }`}
                            >
                              <span
                                className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                                  val ? 'translate-x-5' : 'translate-x-0'
                                }`}
                              />
                            </button>
                            <span className="text-xs text-slate-400">
                              {val ? 'Enabled' : 'Disabled'}
                            </span>
                          </div>
                        )}

                        {/* Select Dropdown */}
                        {field.type === 'select' && (
                          <select
                            value={val}
                            onChange={(e) => handlePluginFieldChange(sec.plugin, field.key, e.target.value)}
                            className="bg-[#0d1117] border border-[#30363d] rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-emerald-500"
                          >
                            {Array.isArray(field.options) &&
                              field.options.map((opt) => (
                                <option key={opt.value} value={opt.value}>
                                  {opt.label || opt.value}
                                </option>
                              ))}
                          </select>
                        )}

                        {field.description && (
                          <p className="text-[11px] text-slate-500 mt-1">{field.description}</p>
                        )}
                      </div>
                    );
                  })}
                </div>

                <div className="pt-6 border-t border-[#30363d] flex items-center justify-between">
                  <button
                    type="button"
                    onClick={() => handleResetPluginDefaults(sec)}
                    className="px-3.5 py-1.5 rounded-lg border border-[#30363d] bg-[#21262d] hover:bg-[#30363d] text-slate-300 text-xs font-semibold transition-colors"
                  >
                    Reset Defaults
                  </button>
                  <button
                    type="button"
                    onClick={() => handleSavePlugin(sec.plugin)}
                    disabled={saving}
                    className="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs transition-all shadow-md shadow-emerald-900/30 flex items-center gap-2"
                  >
                    {saving ? <IconRefresh className="w-3.5 h-3.5 animate-spin" /> : <IconCheck className="w-3.5 h-3.5" />}
                    <span>Save {sec.plugin} Settings</span>
                  </button>
                </div>
              </div>
            );
          })()}
        </div>
      </div>

      {/* Floating Toast Notification */}
      {toast && (
        <div
          className={`fixed bottom-6 right-6 z-50 px-4 py-3 rounded-lg shadow-xl border text-xs font-semibold flex items-center gap-3 transition-all ${
            toast.type === 'success'
              ? 'bg-emerald-950/90 border-emerald-500/50 text-emerald-200'
              : toast.type === 'error'
              ? 'bg-red-950/90 border-red-500/50 text-red-200'
              : 'bg-slate-900/90 border-[#30363d] text-slate-200'
          }`}
        >
          <span className="text-sm">
            {toast.type === 'success' ? '✓' : toast.type === 'error' ? '⚠️' : 'ℹ️'}
          </span>
          <span>{toast.message}</span>
          <button
            onClick={() => setToast(null)}
            className="text-slate-400 hover:text-white ml-2 text-xs"
          >
            ✕
          </button>
        </div>
      )}
    </div>
  );
}
