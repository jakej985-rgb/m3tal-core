import React, { useState, useEffect } from 'react';
import Dashboard from './pages/Dashboard';
import SystemMonitor from './pages/SystemMonitor';
import Stacks from './pages/Stacks';
import Containers from './pages/Containers';
import AI from './pages/AI';
import Plugins from './pages/Plugins';
import {
  IconDashboard,
  IconSystemMonitor,
  IconStacks,
  IconContainers,
  IconAI,
  IconPlugins,
  IconSettings,
} from './components/Icons';
import { getToken, setToken, getApiBase, api } from './api';

export default function App() {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [tokenInput, setTokenInput] = useState(getToken());
  const [showSettings, setShowSettings] = useState(false);
  const [pluginsState, setPluginsState] = useState({
    hasStacks: false,
    hasContainers: false,
    hasAI: false,
    loaded: false,
  });

  const checkInstalledPlugins = async () => {
    try {
      const [resPlugins, resCatalog, resContainers] = await Promise.all([
        api.getPlugins(),
        api.getPluginCatalog(),
        api.getContainers(),
      ]);

      let hasStacks = false;
      let hasAI = false;
      let hasContainers = false;

      // 1. Check loaded stacks
      if (resPlugins?.status === 'success' && resPlugins.data?.stacks) {
        if (Array.isArray(resPlugins.data.stacks) && resPlugins.data.stacks.length > 0) {
          hasStacks = true;
          if (resPlugins.data.stacks.some((s) => (s.name || s.id) === 'ai')) {
            hasAI = true;
          }
        }
      }

      // 2. Check catalog installation state
      if (resCatalog?.status === 'success' && Array.isArray(resCatalog.data)) {
        const installed = resCatalog.data.filter((c) => c.installed || c.status === 'enabled');
        if (installed.some((c) => c.kind === 'Stack' || c.name === 'm3tal' || c.name === 'maintenance')) {
          hasStacks = true;
        }
        if (installed.some((c) => c.name === 'ai')) {
          hasAI = true;
        }
      }

      // 3. Check active container fleet
      if (resContainers?.status === 'success' && Array.isArray(resContainers.data) && resContainers.data.length > 0) {
        hasContainers = true;
      }

      setPluginsState({
        hasStacks,
        hasContainers,
        hasAI,
        loaded: true,
      });
    } catch (err) {
      console.warn('Could not refresh plugins state:', err);
    }
  };

  useEffect(() => {
    checkInstalledPlugins();
    const interval = setInterval(checkInstalledPlugins, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleTokenChange = (e) => {
    const val = e.target.value;
    setTokenInput(val);
    setToken(val);
  };

  const renderContent = () => {
    switch (activeTab) {
      case 'dashboard':
        return <Dashboard />;
      case 'system-monitor':
        return <SystemMonitor />;
      case 'stacks':
        return pluginsState.hasStacks ? <Stacks /> : <Dashboard />;
      case 'containers':
        return pluginsState.hasContainers ? <Containers /> : <Dashboard />;
      case 'ai':
        return pluginsState.hasAI ? <AI /> : <Dashboard />;
      case 'plugins':
        return <Plugins />;
      default:
        return <Dashboard />;
    }
  };

  return (
    <div className="flex h-screen bg-[#0d1117] text-[#c9d1d9] overflow-hidden font-sans">
      {/* Sidebar Navigation */}
      <div className="w-64 bg-[#161b22] border-r border-[#30363d] flex flex-col justify-between">
        <div>
          {/* Logo & Header */}
          <div className="p-6 border-b border-[#30363d] flex items-center gap-3">
            <span className="text-xl font-extrabold tracking-wider bg-gradient-to-r from-emerald-400 to-cyan-400 bg-clip-text text-transparent">
              M3TAL CORE
            </span>
            <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700">
              v2.0
            </span>
          </div>

          {/* Navigation Links */}
          <nav className="p-4 space-y-1">
            {/* Always Visible Core Tools */}
            <button
              onClick={() => setActiveTab('dashboard')}
              className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-all flex items-center gap-3 ${
                activeTab === 'dashboard'
                  ? 'bg-emerald-600 text-white shadow-md shadow-emerald-900/30'
                  : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
              }`}
            >
              <IconDashboard className="w-5 h-5 flex-shrink-0" />
              <span>Dashboard</span>
            </button>

            <button
              onClick={() => setActiveTab('system-monitor')}
              className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-all flex items-center gap-3 ${
                activeTab === 'system-monitor'
                  ? 'bg-emerald-600 text-white shadow-md shadow-emerald-900/30'
                  : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
              }`}
            >
              <IconSystemMonitor className="w-5 h-5 flex-shrink-0 text-emerald-400" />
              <span>System Monitor</span>
            </button>

            {/* Plugin-Dependent Items (Hidden on Fresh Install until Installed) */}
            {pluginsState.hasStacks && (
              <button
                onClick={() => setActiveTab('stacks')}
                className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-all flex items-center gap-3 ${
                  activeTab === 'stacks'
                    ? 'bg-emerald-600 text-white shadow-md shadow-emerald-900/30'
                    : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
                }`}
              >
                <IconStacks className="w-5 h-5 flex-shrink-0" />
                <span>Stacks Manager</span>
              </button>
            )}

            {pluginsState.hasContainers && (
              <button
                onClick={() => setActiveTab('containers')}
                className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-all flex items-center gap-3 ${
                  activeTab === 'containers'
                    ? 'bg-emerald-600 text-white shadow-md shadow-emerald-900/30'
                    : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
                }`}
              >
                <IconContainers className="w-5 h-5 flex-shrink-0" />
                <span>Containers</span>
              </button>
            )}

            {pluginsState.hasAI && (
              <button
                onClick={() => setActiveTab('ai')}
                className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-all flex items-center gap-3 ${
                  activeTab === 'ai'
                    ? 'bg-emerald-600 text-white shadow-md shadow-emerald-900/30'
                    : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
                }`}
              >
                <IconAI className="w-5 h-5 flex-shrink-0" />
                <span>AI Playground</span>
              </button>
            )}

            {/* Plugin Registry (Store / Hub) */}
            <button
              onClick={() => setActiveTab('plugins')}
              className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-all flex items-center gap-3 ${
                activeTab === 'plugins'
                  ? 'bg-emerald-600 text-white shadow-md shadow-emerald-900/30'
                  : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
              }`}
            >
              <IconPlugins className="w-5 h-5 flex-shrink-0" />
              <span>Plugin Registry</span>
            </button>
          </nav>
        </div>

        {/* Settings Footer Panel */}
        <div className="p-4 border-t border-[#30363d]">
          <button
            onClick={() => setShowSettings(!showSettings)}
            className="w-full text-left px-4 py-2 rounded-lg text-xs font-semibold text-slate-400 hover:text-white hover:bg-[#21262d] transition-colors flex items-center gap-2"
          >
            <IconSettings className="w-4 h-4 text-slate-500" />
            <span>Connection Settings</span>
          </button>
          {showSettings && (
            <div className="mt-2 p-3 bg-slate-900 border border-[#30363d] rounded-lg space-y-2 text-xs">
              <div>
                <label className="block text-slate-500 mb-1">API Host</label>
                <input
                  type="text"
                  value={getApiBase()}
                  readOnly
                  className="w-full bg-[#0d1117] border border-[#30363d] rounded px-2 py-1 text-slate-400 font-mono focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-slate-500 mb-1">API Token</label>
                <input
                  type="password"
                  value={tokenInput}
                  onChange={handleTokenChange}
                  className="w-full bg-[#0d1117] border border-[#30363d] rounded px-2 py-1 text-white font-mono focus:outline-none focus:border-emerald-500"
                />
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Main Workspace Screen */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden bg-[#0d1117]">
        <main className="flex-1 overflow-y-auto p-6 md:p-8">
          {renderContent()}
        </main>
      </div>
    </div>
  );
}
