import React, { useState, useEffect } from 'react';
import { api } from '../api';

export default function Plugins() {
  const [plugins, setPlugins] = useState(null);
  const [catalog, setCatalog] = useState(null);
  const [activeTab, setActiveTab] = useState('installed'); // 'installed' or 'catalog'
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(null); // 'kind/name'
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const [modalWarnings, setModalWarnings] = useState(null);

  const fetchPlugins = async () => {
    setLoading(true);
    const res = await api.getPlugins();
    if (res.status === 'success') {
      setPlugins(res.data);
    } else {
      setError(res.error || 'Failed to fetch plugins');
    }
    setLoading(false);
  };

  const fetchCatalog = async () => {
    setLoading(true);
    const res = await api.getPluginCatalog();
    if (res.status === 'success') {
      setCatalog(res.data);
    } else {
      setError(res.error || 'Failed to fetch catalog');
    }
    setLoading(false);
  };

  const loadData = async () => {
    setError(null);
    if (activeTab === 'installed') {
      await fetchPlugins();
    } else {
      await fetchCatalog();
    }
  };

  useEffect(() => {
    loadData();
  }, [activeTab]);

  const handleEnableToggle = async (name, kind, currentlyEnabled) => {
    // If name is null, we are just triggering a reload
    if (!name) return;

    setActionLoading(`${kind}/${name}`);
    setError(null);
    setSuccess(null);
    setModalWarnings(null);

    const action = currentlyEnabled ? api.disablePlugin : api.enablePlugin;
    const res = await action(name, kind);

    if (res.status === 'success') {
      setSuccess(`Plugin "${name}" ${currentlyEnabled ? 'disabled' : 'enabled'} successfully!`);
      setTimeout(() => setSuccess(null), 5000);
      await loadData();
    } else {
      if (res.error === 'unsatisfied dependencies' && res.data && res.data.warnings) {
        setModalWarnings({
          pluginName: name,
          warnings: res.data.warnings
        });
      } else {
        setError(res.error || `Failed to perform action on "${name}"`);
      }
    }
    setActionLoading(null);
  };

  const handleInstall = async (name, kind) => {
    setActionLoading(`${kind}/${name}`);
    setError(null);
    setSuccess(null);
    
    const res = await api.installPlugin(name, kind);
    if (res.status === 'success') {
      setSuccess(`Plugin "${name}" installed successfully!`);
      setTimeout(() => setSuccess(null), 5000);
      await loadData();
    } else {
      setError(res.error || `Failed to install plugin "${name}"`);
    }
    setActionLoading(null);
  };

  const handleUninstall = async (name, kind) => {
    if (!window.confirm(`Are you sure you want to uninstall plugin "${name}"?`)) {
      return;
    }
    setActionLoading(`${kind}/${name}`);
    setError(null);
    setSuccess(null);
    
    const res = await api.uninstallPlugin(name, kind);
    if (res.status === 'success') {
      setSuccess(`Plugin "${name}" uninstalled successfully!`);
      setTimeout(() => setSuccess(null), 5000);
      await loadData();
    } else {
      setError(res.error || `Failed to uninstall plugin "${name}"`);
    }
    setActionLoading(null);
  };

  const renderWarnings = (warnings) => {
    if (!warnings || warnings.length === 0) return null;
    return (
      <div className="mt-2 space-y-1">
        {warnings.map((w, idx) => (
          <div key={idx} className="flex items-center gap-1.5 text-xs text-rose-455 font-medium bg-rose-500/10 border border-rose-500/20 px-2.5 py-1 rounded-md">
            <svg className="w-3.5 h-3.5 flex-shrink-0 text-rose-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <span className="text-rose-350">{w}</span>
          </div>
        ))}
      </div>
    );
  };

  return (
    <div className="space-y-6 max-w-7xl mx-auto pb-12">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2">
            <span className="text-emerald-400">🧩</span> Plugin Hub
          </h1>
          <p className="text-slate-400 text-sm">Extend M3TAL functionalities with catalog and dependency-aware plugins.</p>
        </div>
        
        {/* Navigation Tabs */}
        <div className="flex items-center bg-slate-900 border border-slate-800 p-1 rounded-xl">
          <button
            onClick={() => setActiveTab('installed')}
            className={`px-4 py-2 text-sm font-semibold rounded-lg transition-all ${
              activeTab === 'installed'
                ? 'bg-slate-800 text-emerald-400 shadow-sm'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Installed Plugins
          </button>
          <button
            onClick={() => setActiveTab('catalog')}
            className={`px-4 py-2 text-sm font-semibold rounded-lg transition-all ${
              activeTab === 'catalog'
                ? 'bg-slate-800 text-emerald-400 shadow-sm'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Discover Catalog
          </button>
        </div>
      </div>

      {/* Notifications */}
      {error && (
        <div className="p-4 rounded-xl bg-rose-500/10 text-rose-455 border border-rose-500/20 text-sm flex items-center justify-between shadow-sm">
          <div className="flex items-center gap-2">
            <svg className="w-5 h-5 flex-shrink-0 text-rose-450" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span className="text-rose-350">{error}</span>
          </div>
          <button onClick={() => setError(null)} className="text-rose-400/80 hover:text-rose-300">
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      )}

      {success && (
        <div className="p-4 rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-sm flex items-center gap-2 shadow-sm">
          <svg className="w-5 h-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>{success}</span>
        </div>
      )}

      {/* Unsatisfied Dependencies Warning Modal */}
      {modalWarnings && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-lg w-full overflow-hidden shadow-2xl animate-in fade-in zoom-in-95 duration-200">
            <div className="p-6 border-b border-slate-800/80 flex items-start gap-4">
              <div className="p-3 bg-rose-500/10 text-rose-400 border border-rose-500/20 rounded-xl flex-shrink-0">
                <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
              </div>
              <div className="space-y-1">
                <h3 className="text-lg font-bold text-white">Unsatisfied Dependencies</h3>
                <p className="text-slate-400 text-sm">
                  Plugin <span className="text-emerald-400 font-semibold">"{modalWarnings.pluginName}"</span> cannot be enabled yet because the following requirements are missing or disabled:
                </p>
              </div>
            </div>
            <div className="p-6 bg-slate-950/30 space-y-3">
              {modalWarnings.warnings.map((w, idx) => (
                <div key={idx} className="p-3 rounded-xl bg-slate-900/60 border border-slate-800/50 flex items-center gap-2 text-sm text-slate-350">
                  <span className="w-1.5 h-1.5 bg-rose-400 rounded-full flex-shrink-0"></span>
                  <span>{w}</span>
                </div>
              ))}
            </div>
            <div className="p-6 border-t border-slate-800/80 flex justify-end gap-3">
              <button
                onClick={() => setModalWarnings(null)}
                className="px-4 py-2 rounded-xl text-sm font-semibold bg-slate-800 hover:bg-slate-700 text-white transition-colors"
              >
                Got It
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Main Content */}
      {loading && !plugins && !catalog ? (
        <div className="flex flex-col justify-center items-center h-64 text-slate-500 gap-2">
          <div className="w-8 h-8 border-4 border-emerald-400 border-t-transparent rounded-full animate-spin"></div>
          <span className="text-sm">Fetching plugin data...</span>
        </div>
      ) : activeTab === 'installed' && plugins ? (
        <div className="space-y-8">
          <div className="text-sm font-semibold text-slate-400 bg-slate-900 border border-slate-800 rounded-xl px-5 py-4 flex items-center justify-between">
            <span>Status: <span className="text-emerald-400 font-bold">{plugins.summary || '0 plugins loaded'}</span></span>
            <button
              onClick={loadData}
              className="text-xs text-emerald-400 hover:text-emerald-300 font-bold flex items-center gap-1 transition-colors"
            >
              🔄 Refresh List
            </button>
          </div>

          {/* Installed Sections mapping */}
          {['stacks', 'routes', 'middleware'].map((kind) => {
            const list = plugins[kind] || [];
            const headerTitle = kind === 'stacks' ? '📁 Stack Plugins' : kind === 'routes' ? '🌐 Route Plugins' : '🔒 Middleware Plugins';
            
            return (
              <div key={kind} className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden shadow-sm">
                <div className="px-6 py-4 bg-slate-950/20 border-b border-slate-800/80 flex justify-between items-center">
                  <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider">{headerTitle}</h2>
                  <span className="text-xs bg-slate-800 text-slate-400 px-2 py-0.5 rounded-full font-mono">{list.length} loaded</span>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full text-left border-collapse">
                    <thead>
                      <tr className="border-b border-slate-800/80 bg-slate-950/10 text-xs font-semibold text-slate-400 uppercase tracking-wider">
                        <th className="p-4 pl-6">Name / Details</th>
                        <th className="p-4">Author & Version</th>
                        <th className="p-4">Specification</th>
                        <th className="p-4 text-center">Status</th>
                        <th className="p-4 pr-6 text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-800/40 text-sm">
                      {list.map((item, idx) => {
                        const name = item.metadata?.name || item.name || item.service;
                        const version = item.metadata?.version || '1.0.0';
                        const author = item.metadata?.author || 'System';
                        const enabled = item.enabled;
                        const itemKind = kind === 'stacks' ? 'Stack' : kind === 'routes' ? 'Route' : 'Middleware';
                        const isLoading = actionLoading === `${itemKind}/${name}`;

                        return (
                          <tr key={idx} className="hover:bg-slate-950/10 transition-colors">
                            <td className="p-4 pl-6">
                              <div className="font-semibold text-white">{name}</div>
                              <div className="text-xs text-slate-400 mt-0.5 line-clamp-1">
                                {item.metadata?.description || 'No description provided.'}
                              </div>
                              {renderWarnings(item.warnings)}
                            </td>
                            <td className="p-4">
                              <div className="text-slate-300 text-xs font-medium">{author}</div>
                              <div className="text-[10px] font-mono text-slate-500 mt-0.5">v{version}</div>
                            </td>
                            <td className="p-4">
                              {kind === 'stacks' && (
                                <span className="text-xs font-mono text-slate-400 bg-slate-950/40 px-2 py-1 rounded border border-slate-800">
                                  {item.composePath}
                                </span>
                              )}
                              {kind === 'routes' && (
                                <div className="space-y-1">
                                  <div className="text-emerald-400 font-mono text-xs">{item.domain}:{item.port}</div>
                                  <div className="text-[10px] text-slate-500">Entrypoints: {item.entrypoints || 'web'}</div>
                                </div>
                              )}
                              {kind === 'middleware' && (
                                <div className="space-y-0.5">
                                  <div className="text-xs font-mono text-slate-300">{item.type}</div>
                                  <div className="text-[10px] text-slate-500 truncate max-w-[150px]">{JSON.stringify(item.config)}</div>
                                </div>
                              )}
                            </td>
                            <td className="p-4 text-center">
                              <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold ${
                                enabled
                                  ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                                  : 'bg-slate-800 text-slate-450 border border-slate-700/60'
                              }`}>
                                {enabled ? 'Active' : 'Disabled'}
                              </span>
                            </td>
                            <td className="p-4 pr-6 text-right">
                              <div className="flex items-center justify-end gap-2">
                                <button
                                  onClick={() => handleEnableToggle(name, itemKind, enabled)}
                                  disabled={isLoading}
                                  className={`px-3 py-1.5 rounded-lg text-xs font-bold transition-all flex items-center gap-1 ${
                                    enabled
                                      ? 'bg-slate-800 hover:bg-slate-700 text-rose-400 hover:text-rose-300 border border-slate-700/50'
                                      : 'bg-emerald-500 hover:bg-emerald-400 text-slate-955 shadow-sm shadow-emerald-500/10'
                                  } disabled:opacity-40`}
                                >
                                  {isLoading ? (
                                    <div className="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin"></div>
                                  ) : enabled ? (
                                    'Disable'
                                  ) : (
                                    'Enable'
                                  )}
                                </button>
                                
                                {item.sourcePath && (item.sourcePath.includes('/etc/m3tal') || item.sourcePath.includes('deploy/plugins') || item.sourcePath.includes('.disabled')) && (
                                  <button
                                    onClick={() => handleUninstall(name, itemKind)}
                                    disabled={isLoading}
                                    className="p-1.5 rounded-lg text-slate-400 hover:text-rose-455 hover:bg-slate-800/50 transition-all border border-transparent hover:border-slate-805"
                                    title="Uninstall Plugin"
                                  >
                                    <svg className="w-4 h-4 text-slate-400 hover:text-rose-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                    </svg>
                                  </button>
                                )}
                              </div>
                            </td>
                          </tr>
                        );
                      })}
                      {list.length === 0 && (
                        <tr>
                          <td colSpan="5" className="text-center p-8 text-slate-500 italic">No plugins of this kind loaded.</td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            );
          })}
        </div>
      ) : activeTab === 'catalog' && catalog ? (
        <div className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden shadow-sm">
          <div className="px-6 py-4 bg-slate-950/20 border-b border-slate-800/80 flex justify-between items-center">
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider">Discover Available Plugins</h2>
            <span className="text-xs bg-slate-800 text-slate-400 px-2 py-0.5 rounded-full font-mono">{catalog.length} catalog items</span>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-slate-800/80 bg-slate-950/10 text-xs font-semibold text-slate-400 uppercase tracking-wider">
                  <th className="p-4 pl-6">Plugin Name</th>
                  <th className="p-4">Kind / Info</th>
                  <th className="p-4">Capabilities & Requirements</th>
                  <th className="p-4 text-center">Status</th>
                  <th className="p-4 pr-6 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/40 text-sm">
                {catalog.map((item, idx) => {
                  const isLoading = actionLoading === `${item.kind}/${item.name}`;
                  const status = item.status; // 'enabled', 'disabled', 'not_installed'

                  return (
                    <tr key={idx} className="hover:bg-slate-950/10 transition-colors">
                      <td className="p-4 pl-6 max-w-sm">
                        <div className="font-semibold text-white flex items-center gap-1.5">
                          {item.name}
                          <span className="text-[10px] text-slate-500 font-mono">v{item.version}</span>
                        </div>
                        <div className="text-xs text-slate-400 mt-1">
                          {item.description}
                        </div>
                        <div className="text-[10px] text-slate-500 mt-1.5">By {item.author || 'M3TAL Team'}</div>
                      </td>
                      <td className="p-4">
                        <span className="inline-flex items-center px-2 py-0.5 rounded-md text-[10px] font-bold font-mono bg-slate-950/50 text-slate-300 border border-slate-800 uppercase">
                          {item.kind}
                        </span>
                        {item.category && (
                          <div className="text-[10px] text-slate-500 mt-1 uppercase font-semibold">
                            {item.category} {item.subcategory ? `> ${item.subcategory}` : ''}
                          </div>
                        )}
                      </td>
                      <td className="p-4">
                        <div className="space-y-1.5 max-w-xs">
                          {item.provides && item.provides.length > 0 && (
                            <div className="flex flex-wrap gap-1 items-center">
                              <span className="text-[10px] text-slate-500 font-bold uppercase mr-1">Provides:</span>
                              {item.provides.map((p) => (
                                <span key={p} className="px-1.5 py-0.2 text-[9px] font-mono rounded bg-emerald-950/30 text-emerald-400 border border-emerald-900/40">
                                  {p}
                                </span>
                              ))}
                            </div>
                          )}
                          {item.requires && item.requires.length > 0 && (
                            <div className="flex flex-wrap gap-1 items-center">
                              <span className="text-[10px] text-slate-500 font-bold uppercase mr-1">Requires:</span>
                              {item.requires.map((p) => (
                                <span key={p} className="px-1.5 py-0.2 text-[9px] font-mono rounded bg-amber-950/30 text-amber-400 border border-amber-900/40">
                                  {p}
                                </span>
                              ))}
                            </div>
                          )}
                          {item.dependencies && item.dependencies.length > 0 && (
                            <div className="flex flex-wrap gap-1 items-center">
                              <span className="text-[10px] text-slate-500 font-bold uppercase mr-1">Depends:</span>
                              {item.dependencies.map((d) => (
                                <span key={d.name} className={`px-1.5 py-0.2 text-[9px] font-mono rounded ${
                                  d.required
                                    ? 'bg-rose-955/30 text-rose-455 border border-rose-900/40'
                                    : 'bg-slate-800 text-slate-400 border border-slate-700/60'
                                }`}>
                                  {d.name} {d.required ? '*' : ''}
                                </span>
                              ))}
                            </div>
                          )}
                        </div>
                      </td>
                      <td className="p-4 text-center">
                        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold ${
                          status === 'enabled'
                            ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                            : status === 'disabled'
                            ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                            : 'bg-slate-800 text-slate-450 border border-slate-700/60'
                        }`}>
                          {status === 'enabled' ? 'Active' : status === 'disabled' ? 'Installed (Disabled)' : 'Not Installed'}
                        </span>
                      </td>
                      <td className="p-4 pr-6 text-right">
                        {status === 'not_installed' ? (
                          <button
                            onClick={() => handleInstall(item.name, item.kind)}
                            disabled={isLoading}
                            className="px-3.5 py-1.5 bg-emerald-500 hover:bg-emerald-400 text-slate-955 text-xs font-bold rounded-lg transition-all disabled:opacity-40 shadow-sm"
                          >
                            {isLoading ? (
                              <div className="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin"></div>
                            ) : (
                              'Install'
                            )}
                          </button>
                        ) : (
                          <div className="flex items-center justify-end gap-2">
                            <button
                              onClick={() => handleEnableToggle(item.name, item.kind, status === 'enabled')}
                              disabled={isLoading}
                              className={`px-3 py-1.5 rounded-lg text-xs font-bold transition-all ${
                                status === 'enabled'
                                  ? 'bg-slate-800 hover:bg-slate-700 text-rose-400 hover:text-rose-300 border border-slate-705'
                                  : 'bg-emerald-500 hover:bg-emerald-400 text-slate-955 shadow-sm shadow-emerald-500/10'
                              } disabled:opacity-40`}
                            >
                              {isLoading ? (
                                <div className="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin"></div>
                              ) : status === 'enabled' ? (
                                'Disable'
                              ) : (
                                'Enable'
                              )}
                            </button>
                            <button
                              onClick={() => handleUninstall(item.name, item.kind)}
                              disabled={isLoading}
                              className="p-1.5 rounded-lg text-slate-400 hover:text-rose-455 hover:bg-slate-800/50 transition-all border border-transparent hover:border-slate-800"
                              title="Uninstall"
                            >
                              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                              </svg>
                            </button>
                          </div>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <div className="text-center p-12 bg-slate-900 border border-slate-800 rounded-2xl text-slate-500">
          No catalog plugins available.
        </div>
      )}
    </div>
  );
}
