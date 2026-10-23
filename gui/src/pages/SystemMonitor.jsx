import React, { useState, useEffect } from 'react';
import { api } from '../api';
import { IconSystemMonitor, IconRefresh } from '../components/Icons';

export default function SystemMonitor() {
  const [activeTab, setActiveTab] = useState('processes'); // 'processes', 'performance', 'specs'
  const [processes, setProcesses] = useState([]);
  const [specs, setSpecs] = useState(null);
  const [metrics, setMetrics] = useState(null);
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [sortField, setSortField] = useState('cpu_percent');
  const [sortAsc, setSortAsc] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [killModalProc, setKillModalProc] = useState(null);
  const [toast, setToast] = useState(null);

  const showToast = (message, type = 'info') => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 4000);
  };

  const fetchData = async () => {
    try {
      // 1. Fetch metrics
      const resMetrics = await api.getMetrics();
      if (resMetrics.status === 'success') {
        setMetrics(resMetrics.data);
      }

      // 2. Fetch history for sparklines
      const resHistory = await api.getSystemMetricsHistory();
      if (resHistory.status === 'success') {
        setHistory(resHistory.data || []);
      }

      // 3. Fetch processes if on processes or performance tab
      if (activeTab === 'processes' || activeTab === 'performance') {
        const resProcs = await api.getProcesses(150);
        if (resProcs.status === 'success' && Array.isArray(resProcs.data)) {
          setProcesses(resProcs.data);
        }
      }

      // 4. Fetch system specs
      if (activeTab === 'specs' || activeTab === 'performance') {
        const resSpecs = await api.getSystemSpecs();
        if (resSpecs.status === 'success') {
          setSpecs(resSpecs.data);
        }
      }
    } catch (err) {
      console.error('SystemMonitor fetch error:', err);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData();
    if (!autoRefresh) return;
    const interval = setInterval(fetchData, 3500);
    return () => clearInterval(interval);
  }, [activeTab, autoRefresh]);

  const handleKillProcess = async (proc) => {
    try {
      const res = await api.killProcess(proc.pid);
      if (res.status === 'success') {
        showToast(`Process "${proc.name}" (PID ${proc.pid}) terminated.`, 'success');
        setProcesses((prev) => prev.filter((p) => p.pid !== proc.pid));
      } else {
        showToast(res.error || `Failed to kill PID ${proc.pid}`, 'error');
      }
    } catch (err) {
      showToast(err.message, 'error');
    }
    setKillModalProc(null);
  };

  const handleSort = (field) => {
    if (sortField === field) {
      setSortAsc(!sortAsc);
    } else {
      setSortField(field);
      setSortAsc(false);
    }
  };

  const filteredProcesses = processes
    .filter((p) => {
      if (!searchQuery) return true;
      const q = searchQuery.toLowerCase();
      return (
        p.name?.toLowerCase().includes(q) ||
        String(p.pid).includes(q) ||
        p.username?.toLowerCase().includes(q)
      );
    })
    .sort((a, b) => {
      let valA = a[sortField];
      let valB = b[sortField];
      if (typeof valA === 'string') valA = valA.toLowerCase();
      if (typeof valB === 'string') valB = valB.toLowerCase();
      if (valA < valB) return sortAsc ? -1 : 1;
      if (valA > valB) return sortAsc ? 1 : -1;
      return 0;
    });

  const formatBytes = (bytes) => {
    if (!bytes && bytes !== 0) return '0 B';
    if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`;
    return `${bytes} B`;
  };

  const formatCap = (gb) => {
    if (!gb && gb !== 0) return '0 GB';
    if (gb >= 1000) return `${(gb / 1024).toFixed(1)} TB`;
    return `${gb.toFixed(1)} GB`;
  };

  const getUsageColor = (val) => {
    if (val >= 85) return 'bg-rose-500';
    if (val >= 60) return 'bg-amber-500';
    return 'bg-teal-400';
  };

  return (
    <div className="space-y-6 max-w-7xl mx-auto pb-12">
      {/* Top Header Row with Navigation Tabs & Live Telemetry Ribbon */}
      <div className="flex flex-col lg:flex-row justify-between lg:items-center gap-4 bg-slate-900/40 p-6 rounded-2xl border border-slate-800/60 backdrop-blur-md">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2.5">
            <IconSystemMonitor className="w-6 h-6 text-emerald-400" />
            <span>System Monitor</span>
          </h1>
          <p className="text-slate-400 text-sm">Advanced task manager, per-process telemetry, and hardware profiler.</p>
        </div>

        {/* Navigation & Controls */}
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center bg-slate-900 border border-slate-800 p-1 rounded-xl">
            <button
              onClick={() => setActiveTab('processes')}
              className={`px-4 py-1.5 text-xs font-semibold rounded-lg transition-all ${
                activeTab === 'processes'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              Processes ({processes.length})
            </button>
            <button
              onClick={() => setActiveTab('performance')}
              className={`px-4 py-1.5 text-xs font-semibold rounded-lg transition-all ${
                activeTab === 'performance'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              Performance
            </button>
            <button
              onClick={() => setActiveTab('specs')}
              className={`px-4 py-1.5 text-xs font-semibold rounded-lg transition-all ${
                activeTab === 'specs'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              System Specs
            </button>
          </div>

          <button
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold border transition-all ${
              autoRefresh
                ? 'bg-emerald-500/10 text-emerald-300 border-emerald-500/30'
                : 'bg-slate-800 text-slate-400 border-slate-700'
            }`}
            title="Toggle live 3-second auto refresh"
          >
            {autoRefresh ? 'Live: ON' : 'Paused'}
          </button>

          <button
            onClick={fetchData}
            className="p-1.5 rounded-lg bg-slate-800 text-slate-300 border border-slate-700 hover:text-white hover:bg-slate-750 transition-all active:scale-95"
            title="Refresh statistics now"
          >
            <IconRefresh className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Toast Notification */}
      {toast && (
        <div
          className={`p-3 rounded-xl border flex items-center justify-between text-xs font-medium animate-fade-in ${
            toast.type === 'success'
              ? 'bg-emerald-950/80 text-emerald-300 border-emerald-500/40'
              : toast.type === 'error'
              ? 'bg-rose-950/80 text-rose-300 border-rose-500/40'
              : 'bg-slate-900/90 text-slate-200 border-slate-700'
          }`}
        >
          <span>{toast.message}</span>
          <button onClick={() => setToast(null)} className="text-slate-400 hover:text-white ml-4">
            ✕
          </button>
        </div>
      )}

      {/* Telemetry Summary Cards Ribbon */}
      {metrics && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-4">
            <div className="text-slate-400 text-xs font-semibold mb-1">Processor (CPU)</div>
            <div className="text-xl font-bold text-white flex items-baseline gap-2">
              <span>{metrics.cpu_usage?.toFixed(1)}%</span>
              {metrics.cpu_temp > 0 && (
                <span className="text-xs text-amber-400 font-mono">🌡️ {metrics.cpu_temp.toFixed(0)}°C</span>
              )}
            </div>
            <div className="w-full bg-slate-800 rounded-full h-1.5 mt-2">
              <div
                className={`h-1.5 rounded-full ${getUsageColor(metrics.cpu_usage)}`}
                style={{ width: `${Math.min(100, metrics.cpu_usage)}%` }}
              ></div>
            </div>
          </div>

          <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-4">
            <div className="text-slate-400 text-xs font-semibold mb-1">Memory (RAM)</div>
            <div className="text-xl font-bold text-white flex items-baseline gap-2">
              <span>{metrics.memory_usage?.toFixed(1)}%</span>
              <span className="text-xs text-slate-400 font-mono">
                {metrics.memory_used?.toFixed(1)} / {metrics.memory_total?.toFixed(1)} GB
              </span>
            </div>
            <div className="w-full bg-slate-800 rounded-full h-1.5 mt-2">
              <div
                className={`h-1.5 rounded-full ${getUsageColor(metrics.memory_usage)}`}
                style={{ width: `${Math.min(100, metrics.memory_usage)}%` }}
              ></div>
            </div>
          </div>

          <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-4">
            <div className="text-slate-400 text-xs font-semibold mb-1">Active GPU</div>
            <div className="text-xl font-bold text-white flex items-baseline gap-2">
              <span>{metrics.gpu_usage?.toFixed(1) || 0}%</span>
              {metrics.gpu_mem_total > 0 && (
                <span className="text-xs text-indigo-400 font-mono">
                  {Math.round(metrics.gpu_mem_used)} MB
                </span>
              )}
            </div>
            <div className="w-full bg-slate-800 rounded-full h-1.5 mt-2">
              <div
                className={`h-1.5 rounded-full ${getUsageColor(metrics.gpu_usage || 0)}`}
                style={{ width: `${Math.min(100, metrics.gpu_usage || 0)}%` }}
              ></div>
            </div>
          </div>

          <div className="bg-slate-900/60 border border-slate-800/80 rounded-xl p-4">
            <div className="text-slate-400 text-xs font-semibold mb-1">Primary Disk (/)</div>
            <div className="text-xl font-bold text-white flex items-baseline gap-2">
              <span>{metrics.disk_usage?.toFixed(1)}%</span>
              <span className="text-xs text-slate-400 font-mono">
                {metrics.disk_used?.toFixed(0)} / {metrics.disk_total?.toFixed(0)} GB
              </span>
            </div>
            <div className="w-full bg-slate-800 rounded-full h-1.5 mt-2">
              <div
                className={`h-1.5 rounded-full ${getUsageColor(metrics.disk_usage)}`}
                style={{ width: `${Math.min(100, metrics.disk_usage)}%` }}
              ></div>
            </div>
          </div>
        </div>
      )}

      {/* TAB 1: PROCESSES TABLE */}
      {activeTab === 'processes' && (
        <div className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden shadow-xl">
          {/* Table Toolbar */}
          <div className="p-4 bg-slate-950/40 border-b border-slate-800 flex flex-col sm:flex-row justify-between items-center gap-3">
            <div className="relative w-full sm:w-80">
              <input
                type="text"
                placeholder="Search processes by name, PID, user..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full bg-slate-900 border border-slate-700/80 rounded-lg px-3.5 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500"
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery('')}
                  className="absolute right-2.5 top-2 text-slate-500 hover:text-white text-xs"
                >
                  ✕
                </button>
              )}
            </div>

            <div className="flex items-center gap-4 text-xs text-slate-400 font-mono">
              <span>Showing {filteredProcesses.length} of {processes.length} Processes</span>
            </div>
          </div>

          {/* Process List */}
          <div className="overflow-x-auto max-h-[620px]">
            <table className="w-full text-left text-xs border-collapse">
              <thead className="bg-slate-950/60 sticky top-0 border-b border-slate-800 text-[11px] font-semibold text-slate-400 uppercase tracking-wider select-none z-10">
                <tr>
                  <th
                    className="p-3.5 cursor-pointer hover:text-white"
                    onClick={() => handleSort('name')}
                  >
                    Process Name {sortField === 'name' && (sortAsc ? '▲' : '▼')}
                  </th>
                  <th
                    className="p-3.5 cursor-pointer hover:text-white"
                    onClick={() => handleSort('pid')}
                  >
                    PID {sortField === 'pid' && (sortAsc ? '▲' : '▼')}
                  </th>
                  <th
                    className="p-3.5 cursor-pointer hover:text-white"
                    onClick={() => handleSort('username')}
                  >
                    User {sortField === 'username' && (sortAsc ? '▲' : '▼')}
                  </th>
                  <th
                    className="p-3.5 cursor-pointer hover:text-white"
                    onClick={() => handleSort('cpu_percent')}
                  >
                    CPU % {sortField === 'cpu_percent' && (sortAsc ? '▲' : '▼')}
                  </th>
                  <th
                    className="p-3.5 cursor-pointer hover:text-white"
                    onClick={() => handleSort('memory_rss')}
                  >
                    Memory (RSS) {sortField === 'memory_rss' && (sortAsc ? '▲' : '▼')}
                  </th>
                  <th className="p-3.5">Status</th>
                  <th className="p-3.5 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60 font-mono">
                {filteredProcesses.map((p) => {
                  const isHighCpu = p.cpu_percent >= 20;
                  const isRoot = p.username === 'root';

                  return (
                    <tr
                      key={p.pid}
                      className="hover:bg-slate-800/40 transition-colors group"
                    >
                      <td className="p-3.5 font-sans">
                        <div className="font-semibold text-white text-xs flex items-center gap-2">
                          <span>{p.name}</span>
                        </div>
                        {p.cmdline && p.cmdline !== p.name && (
                          <div className="text-[10px] text-slate-500 font-mono truncate max-w-sm mt-0.5" title={p.cmdline}>
                            {p.cmdline}
                          </div>
                        )}
                      </td>
                      <td className="p-3.5 text-slate-400">{p.pid}</td>
                      <td className="p-3.5">
                        <span
                          className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                            isRoot
                              ? 'bg-amber-500/10 text-amber-300 border border-amber-500/20'
                              : 'bg-cyan-500/10 text-cyan-300 border border-cyan-500/20'
                          }`}
                        >
                          {p.username || 'unknown'}
                        </span>
                      </td>
                      <td className="p-3.5">
                        <div className="flex items-center gap-2">
                          <span className={`font-semibold ${isHighCpu ? 'text-amber-400' : 'text-slate-300'}`}>
                            {p.cpu_percent.toFixed(1)}%
                          </span>
                          <div className="w-16 bg-slate-800 rounded-full h-1.5 hidden sm:block">
                            <div
                              className={`h-1.5 rounded-full ${getUsageColor(p.cpu_percent)}`}
                              style={{ width: `${Math.min(100, p.cpu_percent)}%` }}
                            ></div>
                          </div>
                        </div>
                      </td>
                      <td className="p-3.5 text-slate-300">
                        <div>{formatBytes(p.memory_rss)}</div>
                        <div className="text-[10px] text-slate-500">{p.memory_percent.toFixed(1)}%</div>
                      </td>
                      <td className="p-3.5">
                        <span className="flex items-center gap-1.5 text-slate-300 text-[11px] capitalize">
                          <span
                            className={`w-2 h-2 rounded-full ${
                              p.status === 'running' ? 'bg-emerald-400 animate-pulse' : 'bg-slate-500'
                            }`}
                          ></span>
                          {p.status}
                        </span>
                      </td>
                      <td className="p-3.5 text-right">
                        <button
                          onClick={() => setKillModalProc(p)}
                          className="px-2.5 py-1 bg-slate-800 hover:bg-rose-500/20 hover:text-rose-300 text-slate-400 border border-slate-700/80 hover:border-rose-500/30 rounded text-[10px] font-bold uppercase tracking-wider transition-all"
                        >
                          End Task
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* TAB 2: PERFORMANCE (Windows Task Manager View) */}
      {activeTab === 'performance' && (
        <div className="space-y-6">
          {/* CPU & Memory In-Depth View */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* CPU Performance Card */}
            <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-4">
              <div className="flex justify-between items-start">
                <div>
                  <h2 className="text-base font-bold text-white">Central Processor (CPU)</h2>
                  <p className="text-xs text-slate-400 font-mono mt-0.5">{specs?.cpu_model || 'Processor'}</p>
                </div>
                <div className="text-right">
                  <div className="text-2xl font-extrabold text-white">{metrics?.cpu_usage?.toFixed(1)}%</div>
                  <div className="text-xs text-slate-400 font-mono">Utilization</div>
                </div>
              </div>

              {/* Sparkline */}
              <div className="h-28 bg-slate-950/60 rounded-xl p-3 border border-slate-800/80 flex items-end justify-between gap-1">
                {history.map((h, i) => (
                  <div
                    key={i}
                    className={`flex-1 rounded-t-sm transition-all duration-300 ${getUsageColor(h.cpu_usage)}`}
                    style={{ height: `${Math.max(4, h.cpu_usage)}%` }}
                    title={`CPU: ${h.cpu_usage?.toFixed(1)}%`}
                  ></div>
                ))}
              </div>

              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 pt-2 text-xs font-mono border-t border-slate-800/60">
                <div>
                  <div className="text-slate-500">Cores / Threads</div>
                  <div className="text-white font-bold">{specs?.cpu_cores || 1} / {specs?.cpu_threads || 1}</div>
                </div>
                <div>
                  <div className="text-slate-500">Clock Speed</div>
                  <div className="text-white font-bold">{specs?.cpu_mhz ? `${(specs.cpu_mhz / 1000).toFixed(2)} GHz` : '—'}</div>
                </div>
                <div>
                  <div className="text-slate-500">Temperature</div>
                  <div className="text-amber-400 font-bold">{metrics?.cpu_temp ? `${metrics.cpu_temp.toFixed(1)}°C` : '—'}</div>
                </div>
                <div>
                  <div className="text-slate-500">Architecture</div>
                  <div className="text-white font-bold">{specs?.kernel_arch || 'x86_64'}</div>
                </div>
              </div>
            </div>

            {/* Memory Performance Card */}
            <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-4">
              <div className="flex justify-between items-start">
                <div>
                  <h2 className="text-base font-bold text-white">System Memory (RAM)</h2>
                  <p className="text-xs text-slate-400 font-mono mt-0.5">
                    {metrics?.memory_frequency && metrics.memory_frequency !== 'Unknown' ? metrics.memory_frequency : 'DDR Memory'}
                  </p>
                </div>
                <div className="text-right">
                  <div className="text-2xl font-extrabold text-white">{metrics?.memory_usage?.toFixed(1)}%</div>
                  <div className="text-xs text-slate-400 font-mono">
                    {metrics?.memory_used?.toFixed(1)} / {metrics?.memory_total?.toFixed(1)} GB
                  </div>
                </div>
              </div>

              {/* Sparkline */}
              <div className="h-28 bg-slate-950/60 rounded-xl p-3 border border-slate-800/80 flex items-end justify-between gap-1">
                {history.map((h, i) => (
                  <div
                    key={i}
                    className={`flex-1 rounded-t-sm transition-all duration-300 ${getUsageColor(h.memory_usage)}`}
                    style={{ height: `${Math.max(4, h.memory_usage)}%` }}
                    title={`RAM: ${h.memory_usage?.toFixed(1)}%`}
                  ></div>
                ))}
              </div>

              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 pt-2 text-xs font-mono border-t border-slate-800/60">
                <div>
                  <div className="text-slate-500">In Use</div>
                  <div className="text-white font-bold">{metrics?.memory_used?.toFixed(1)} GB</div>
                </div>
                <div>
                  <div className="text-slate-500">Available</div>
                  <div className="text-white font-bold">
                    {metrics ? (metrics.memory_total - metrics.memory_used).toFixed(1) : 0} GB
                  </div>
                </div>
                <div>
                  <div className="text-slate-500">Swap Total</div>
                  <div className="text-white font-bold">{specs?.swap_total?.toFixed(1) || 0} GB</div>
                </div>
                <div>
                  <div className="text-slate-500">Swap Used</div>
                  <div className="text-white font-bold">{specs?.swap_used?.toFixed(1) || 0} GB</div>
                </div>
              </div>
            </div>
          </div>

          {/* Network Adapters Telemetry */}
          {specs && specs.network_interfaces && specs.network_interfaces.length > 0 && (
            <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-4">
              <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider">
                Network Adapters & Live Throughput
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {specs.network_interfaces.map((iface, idx) => (
                  <div
                    key={idx}
                    className="bg-slate-950/60 border border-slate-800/60 rounded-xl p-4 space-y-2.5"
                  >
                    <div className="flex justify-between items-center">
                      <span className="font-bold text-white text-sm">{iface.name}</span>
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                          iface.is_up
                            ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                            : 'bg-slate-800 text-slate-500 border border-slate-700'
                        }`}
                      >
                        {iface.is_up ? 'ACTIVE' : 'DOWN'}
                      </span>
                    </div>

                    <div className="text-xs text-slate-400 font-mono space-y-1">
                      {iface.ip_addresses && iface.ip_addresses.length > 0 ? (
                        iface.ip_addresses.slice(0, 2).map((ip, ipIdx) => (
                          <div key={ipIdx} className="truncate text-slate-300">
                            {ip}
                          </div>
                        ))
                      ) : (
                        <div className="text-slate-600 italic">No assigned IP</div>
                      )}
                    </div>

                    <div className="border-t border-slate-800/60 pt-2 flex justify-between text-[11px] font-mono text-slate-400">
                      <div>
                        <span className="text-slate-500">Sent: </span>
                        <span className="text-white font-semibold">{formatBytes(iface.bytes_sent)}</span>
                      </div>
                      <div>
                        <span className="text-slate-500">Recv: </span>
                        <span className="text-white font-semibold">{formatBytes(iface.bytes_recv)}</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* TAB 3: SYSTEM SPECS (Neofetch / Specs manifesto) */}
      {activeTab === 'specs' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* OS & Kernel Box */}
          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-4">
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider flex items-center gap-2">
              <span>🐧</span> Operating System & Kernel
            </h2>
            <div className="border-t border-slate-800/60 pt-3 space-y-3 font-mono text-xs">
              <div className="flex justify-between py-1 border-b border-slate-800/40">
                <span className="text-slate-500">Hostname</span>
                <span className="text-white font-medium">{specs?.hostname || metrics?.hostname || 'localhost'}</span>
              </div>
              <div className="flex justify-between py-1 border-b border-slate-800/40">
                <span className="text-slate-500">Platform Distribution</span>
                <span className="text-emerald-400 font-medium capitalize">
                  {specs?.platform || 'Linux'} {specs?.platform_version || ''}
                </span>
              </div>
              <div className="flex justify-between py-1 border-b border-slate-800/40">
                <span className="text-slate-500">Kernel Version</span>
                <span className="text-white font-medium">{specs?.kernel_version || 'Linux Kernel'}</span>
              </div>
              <div className="flex justify-between py-1 border-b border-slate-800/40">
                <span className="text-slate-500">Kernel Architecture</span>
                <span className="text-white font-medium">{specs?.kernel_arch || 'x86_64'}</span>
              </div>
              <div className="flex justify-between py-1">
                <span className="text-slate-500">Virtualization / Hypervisor</span>
                <span className="text-cyan-400 font-medium">{specs?.virtualization || 'Bare Metal'}</span>
              </div>
            </div>
          </div>

          {/* Hardware & Processor Box */}
          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-4">
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider flex items-center gap-2">
              <span>⚡</span> Hardware & Processors
            </h2>
            <div className="border-t border-slate-800/60 pt-3 space-y-3 font-mono text-xs">
              <div className="flex justify-between py-1 border-b border-slate-800/40">
                <span className="text-slate-500">Processor Model</span>
                <span className="text-white font-medium truncate max-w-[280px]" title={specs?.cpu_model}>
                  {specs?.cpu_model || 'Standard CPU'}
                </span>
              </div>
              <div className="flex justify-between py-1 border-b border-slate-800/40">
                <span className="text-slate-500">Physical Cores</span>
                <span className="text-white font-medium">{specs?.cpu_cores || 1} Cores</span>
              </div>
              <div className="flex justify-between py-1 border-b border-slate-800/40">
                <span className="text-slate-500">Logical Threads</span>
                <span className="text-white font-medium">{specs?.cpu_threads || 1} Threads</span>
              </div>
              <div className="flex justify-between py-1 border-b border-slate-800/40">
                <span className="text-slate-500">Installed Physical RAM</span>
                <span className="text-white font-medium">{specs?.memory_total?.toFixed(1) || 0} GB</span>
              </div>
              <div className="flex justify-between py-1">
                <span className="text-slate-500">Graphics Processor (GPU)</span>
                <span className="text-indigo-400 font-medium">{specs?.gpu_model || 'No Dedicated GPU'}</span>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Process Termination Confirmation Modal */}
      {killModalProc && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm animate-fade-in">
          <div className="bg-slate-900 border border-slate-700 rounded-2xl p-6 w-full max-w-md space-y-4 shadow-2xl">
            <div className="flex items-center gap-3 text-rose-400">
              <span className="text-2xl">⚠️</span>
              <h3 className="text-lg font-bold text-white">End Process?</h3>
            </div>
            <p className="text-xs text-slate-300 leading-relaxed">
              Are you sure you want to terminate <strong className="text-white font-mono">{killModalProc.name}</strong> (PID {killModalProc.pid})? Unsaved data within this process may be lost.
            </p>
            <div className="flex justify-end gap-3 pt-2">
              <button
                onClick={() => setKillModalProc(null)}
                className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg text-xs font-semibold transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => handleKillProcess(killModalProc)}
                className="px-4 py-2 bg-rose-600 hover:bg-rose-500 text-white rounded-lg text-xs font-semibold transition-colors shadow-sm"
              >
                End Process
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
