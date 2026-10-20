import React, { useState, useEffect } from 'react';
import Dashboard from './pages/Dashboard';
import Stacks from './pages/Stacks';
import Containers from './pages/Containers';
import AI from './pages/AI';
import Plugins from './pages/Plugins';
import { getToken, setToken, getApiBase } from './api';

export default function App() {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [tokenInput, setTokenInput] = useState(getToken());
  const [showSettings, setShowSettings] = useState(false);

  const handleTokenChange = (e) => {
    const val = e.target.value;
    setTokenInput(val);
    setToken(val);
  };

  const renderContent = () => {
    switch (activeTab) {
      case 'dashboard':
        return <Dashboard />;
      case 'stacks':
        return <Stacks />;
      case 'containers':
        return <Containers />;
      case 'ai':
        return <AI />;
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
            <button
              onClick={() => setActiveTab('dashboard')}
              className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-colors flex items-center gap-3 ${
                activeTab === 'dashboard'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
              }`}
            >
              📊 Dashboard
            </button>
            <button
              onClick={() => setActiveTab('stacks')}
              className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-colors flex items-center gap-3 ${
                activeTab === 'stacks'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
              }`}
            >
              📁 Stacks Manager
            </button>
            <button
              onClick={() => setActiveTab('containers')}
              className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-colors flex items-center gap-3 ${
                activeTab === 'containers'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
              }`}
            >
              🐳 Containers
            </button>
            <button
              onClick={() => setActiveTab('ai')}
              className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-colors flex items-center gap-3 ${
                activeTab === 'ai'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
              }`}
            >
              🤖 AI Playground
            </button>
            <button
              onClick={() => setActiveTab('plugins')}
              className={`w-full text-left px-4 py-2.5 rounded-lg text-sm font-semibold transition-colors flex items-center gap-3 ${
                activeTab === 'plugins'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-slate-400 hover:bg-[#21262d] hover:text-white'
              }`}
            >
              🧩 Plugin Registry
            </button>
          </nav>
        </div>

        {/* Settings Footer Panel */}
        <div className="p-4 border-t border-[#30363d]">
          <button
            onClick={() => setShowSettings(!showSettings)}
            className="w-full text-left px-4 py-2 rounded-lg text-xs font-semibold text-slate-500 hover:text-slate-300 transition-colors flex items-center gap-2"
          >
            ⚙️ Connection Settings
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
