import { useState, useEffect, useMemo } from 'react';
import Dashboard from './pages/Dashboard';
import SystemMonitor from './pages/SystemMonitor';
import Plugins from './pages/Plugins';
import Settings from './pages/Settings';
import {
  IconDashboard,
  IconSystemMonitor,
  IconContainers,
  IconPlugins,
  IconSettings,
  IconLayers,
} from './components/Icons';
import { getToken, getApiBase, api } from './api';

export default function App() {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [activeSubTabs, setActiveSubTabs] = useState({});
  const [tokenInput] = useState(getToken());
  const [pluginTabs, setPluginTabs] = useState([]);
  const [coreVersion, setCoreVersion] = useState('v0.18.0');

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

  const fetchCoreVersion = async () => {
    try {
      const res = await api.getVersion();
      if (res?.status === 'success' && res.data) {
        setCoreVersion(res.data.display || `v${res.data.version}`);
      }
    } catch {
      // Fallback stays v0.18.0
    }
  };

  useEffect(() => {
    fetchCoreVersion();
    fetchPluginTabs();
    const interval = setInterval(fetchPluginTabs, 15000);
    return () => clearInterval(interval);
  }, []);

  // Group plugin tabs into categories with ordered sub-tabs
  const pluginCategories = useMemo(() => {
    const groups = {};
    pluginTabs.forEach((tab) => {
      let catKey = tab.category;
      if (!catKey) {
        if (
          tab.plugin === 'docker-monitor' ||
          tab.id === 'docker_containers' ||
          tab.plugin === 'compose' ||
          tab.id === 'compose_editor'
        ) {
          catKey = 'containers';
        } else {
          catKey = tab.plugin || tab.id;
        }
      }
      if (!groups[catKey]) {
        groups[catKey] = {
          key: catKey,
          title: tab.category_title || (catKey === 'containers' ? 'Containers' : tab.title),
          order: tab.order || 50,
          tabs: []
        };
      }
      groups[catKey].tabs.push(tab);
      groups[catKey].tabs.sort((a, b) => (a.order || 50) - (b.order || 50));
    });
    return Object.values(groups).sort((a, b) => a.order - b.order);
  }, [pluginTabs]);

  const renderContent = () => {
    switch (activeTab) {
      case 'dashboard':
        return <Dashboard />;
      case 'system-monitor':
        return <SystemMonitor />;
      case 'plugins':
        return <Plugins />;
      case 'settings':
        return <Settings />;
      default: {
        // 1. Check if activeTab matches a plugin category (e.g. 'containers')
        const currentCat = pluginCategories.find((c) => c.key === activeTab || c.tabs.some((t) => t.id === activeTab));
        if (currentCat) {
          const subTabs = currentCat.tabs;
          const activeSubTabId =
            activeSubTabs[currentCat.key] ||
            (subTabs.some((t) => t.id === activeTab) ? activeTab : subTabs[0]?.id);
          const currentSubTab = subTabs.find((t) => t.id === activeSubTabId) || subTabs[0];

          if (currentSubTab) {
            const pluginName = currentSubTab.plugin || (currentSubTab.id === 'docker_containers' ? 'docker-monitor' : currentSubTab.id);
            const pluginUrl = currentSubTab.url || `/plugins/${pluginName}/`;
            return (
              <div className="w-full h-full flex flex-col -m-6 md:-m-8" style={{ height: 'calc(100vh - 0px)' }}>
                {/* Dynamic Top Sub-Tab Switcher (rendered when category has multiple plugins) */}
                {subTabs.length > 1 && (
                  <div className="bg-[#161b22] border-b border-[#30363d] px-6 py-2.5 flex items-center gap-2 flex-shrink-0">
                    {subTabs.map((sub) => {
                      const isSubActive = sub.id === currentSubTab.id;
                      let displayTitle = sub.title;
                      if (currentCat.key === 'containers') {
                        if (
                          sub.id === 'docker_containers' ||
                          sub.plugin === 'docker-monitor' ||
                          sub.title === 'Containers'
                        ) {
                          displayTitle = 'Fleet';
                        } else if (
                          sub.id === 'compose_editor' ||
                          sub.plugin === 'compose' ||
                          sub.title === 'Compose Editor'
                        ) {
                          displayTitle = 'Compose';
                        }
                      }
                      return (
                        <button
                          key={sub.id}
                          onClick={() => setActiveSubTabs((prev) => ({ ...prev, [currentCat.key]: sub.id }))}
                          className={`px-4 py-1.5 rounded-md text-xs font-semibold transition-all flex items-center gap-2 ${
                            isSubActive
                              ? 'bg-emerald-600 text-white shadow shadow-emerald-900/30'
                              : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
                          }`}
                        >
                          {currentCat.key === 'containers' && (
                            displayTitle === 'Fleet' ? (
                              <IconContainers className="w-3.5 h-3.5 text-cyan-300" />
                            ) : (
                              <IconLayers className="w-3.5 h-3.5 text-cyan-300" />
                            )
                          )}
                          <span>{displayTitle}</span>
                        </button>
                      );
                    })}
                  </div>
                )}
                <iframe
                  key={currentSubTab.id}
                  src={`${pluginUrl}?token=${tokenInput}`}
                  title={currentSubTab.title}
                  className="w-full h-full flex-1 border-0 bg-[#0d1117]"
                />
              </div>
            );
          }
        }
        return <Dashboard />;
      }
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
            <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-slate-800 text-emerald-400 border border-slate-700 font-mono">
              {coreVersion}
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

            {/* Dynamic Plugin Category Navigation */}
            {pluginCategories.map((cat) => {
              const isCatActive = activeTab === cat.key || cat.tabs.some((t) => t.id === activeTab);
              return (
                <button
                  key={cat.key}
                  onClick={() => setActiveTab(cat.key)}
                  className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-all flex items-center gap-3 ${
                    isCatActive
                      ? 'bg-emerald-600 text-white shadow-md shadow-emerald-900/30'
                      : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
                  }`}
                >
                  {cat.key === 'containers' ? (
                    <IconContainers className="w-5 h-5 flex-shrink-0 text-cyan-400" />
                  ) : (
                    <IconPlugins className="w-5 h-5 flex-shrink-0 text-cyan-400" />
                  )}
                  <span>{cat.title}</span>
                  {cat.tabs.length > 1 && (
                    <span className="ml-auto text-[10px] font-bold bg-[#0d1117] text-slate-400 px-1.5 py-0.5 rounded border border-[#30363d]">
                      {cat.tabs.length}
                    </span>
                  )}
                </button>
              );
            })}

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

            {/* Settings */}
            <button
              onClick={() => setActiveTab('settings')}
              className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-all flex items-center gap-3 ${
                activeTab === 'settings'
                  ? 'bg-emerald-600 text-white shadow-md shadow-emerald-900/30'
                  : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
              }`}
            >
              <IconSettings className="w-5 h-5 flex-shrink-0" />
              <span>Settings</span>
            </button>
          </nav>
        </div>

        {/* Connection Status & Settings Footer Panel */}
        <div className="p-4 border-t border-[#30363d] flex items-center justify-between">
          <div className="flex items-center gap-2.5 min-w-0">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-pulse flex-shrink-0" />
            <div className="truncate">
              <div className="text-[11px] font-semibold text-slate-300 leading-tight">API Connected</div>
              <div className="text-[10px] text-slate-500 font-mono truncate">{getApiBase()}</div>
            </div>
          </div>
          <button
            onClick={() => setActiveTab('settings')}
            className={`p-1.5 rounded-lg transition-colors flex-shrink-0 ${
              activeTab === 'settings'
                ? 'bg-emerald-600 text-white'
                : 'text-slate-400 hover:text-white hover:bg-[#21262d]'
            }`}
            title="Open Settings"
          >
            <IconSettings className="w-4 h-4" />
          </button>
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
