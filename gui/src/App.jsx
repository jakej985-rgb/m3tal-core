import React, { useState, useEffect } from 'react';
import Dashboard from './pages/Dashboard';
import SystemMonitor from './pages/SystemMonitor';
import Plugins from './pages/Plugins';
import {
  IconDashboard,
  IconSystemMonitor,
  IconPlugins,
  IconSettings,
} from './components/Icons';
import { getToken, setToken, getApiBase, api } from './api';

export default function App() {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [tokenInput, setTokenInput] = useState(getToken());
  const [showSettings, setShowSettings] = useState(false);
  const [pluginTabs, setPluginTabs] = useState([]);

  // Fetch dynamic tabs registered by running external plugins
  const fetchPluginTabs = async () => {
    try {
      const res = await api.getPluginTabs();
      if (res?.status === 'success' && Array.isArray(res.data)) {
        setPluginTabs(res.data);
      } else {
        setPluginTabs([]);
      }
    } catch {
      // Non-critical, fallback to empty dynamic tabs
      setPluginTabs([]);
    }
  };

  useEffect(() => {
    fetchPluginTabs();
    const interval = setInterval(fetchPluginTabs, 15000);
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
      case 'plugins':
        return <Plugins />;
      default:
        // Check if activeTab matches a dynamic plugin tab
        const dynamicTab = pluginTabs.find((t) => t.id === activeTab);
        if (dynamicTab) {
          return (
            <div className="space-y-4">
              <h1 className="text-2xl font-bold text-white flex items-center gap-2">
                <span>{dynamicTab.title}</span>
                <span className="text-xs font-semibold px-2 py-0.5 rounded bg-emerald-950/60 text-emerald-400 border border-emerald-800">
                  Plugin View
                </span>
              </h1>
              <div className="bg-[#161b22] border border-[#30363d] rounded-xl p-6 text-slate-300">
                <p className="text-sm text-slate-400">
                  Dynamic view provided by plugin: <span className="text-white font-mono">{dynamicTab.plugin || dynamicTab.id}</span>
                </p>
              </div>
            </div>
          );
        }
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

            {/* Dynamic Plugin Tabs */}
            {pluginTabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-all flex items-center gap-3 ${
                  activeTab === tab.id
                    ? 'bg-emerald-600 text-white shadow-md shadow-emerald-900/30'
                    : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
                }`}
              >
                <IconPlugins className="w-5 h-5 flex-shrink-0 text-cyan-400" />
                <span>{tab.title}</span>
              </button>
            ))}

            {/* Plugin Manager */}
            <button
              onClick={() => setActiveTab('plugins')}
              className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-all flex items-center gap-3 ${
                activeTab === 'plugins'
                  ? 'bg-emerald-600 text-white shadow-md shadow-emerald-900/30'
                  : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
              }`}
            >
              <IconPlugins className="w-5 h-5 flex-shrink-0" />
              <span>Plugin Manager</span>
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
