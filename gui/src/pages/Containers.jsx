import React, { useState, useEffect, useRef } from 'react';
import { api, subscribeWS } from '../api';

export default function Containers() {
  const [containers, setContainers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [activeLogContainer, setActiveLogContainer] = useState(null);
  const [logs, setLogs] = useState([]);
  
  const logTerminalEndRef = useRef(null);
  const wsSubscriptionRef = useRef(null);

  const fetchContainers = async () => {
    setLoading(true);
    const res = await api.getContainers();
    if (res.status === 'success') {
      setContainers(res.data);
      setError(null);
    } else {
      setError(res.error || 'Failed to fetch containers');
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchContainers();
    
    // Auto-refresh when container events occur
    const sub = subscribeWS('/api/v2/ws/events', (event) => {
      if (event.type?.startsWith('container.')) {
        fetchContainers();
      }
    });

    return () => sub.close();
  }, []);

  // Listen to log streaming
  useEffect(() => {
    if (activeLogContainer) {
      setLogs([]);
      
      // Close previous log socket if active
      if (wsSubscriptionRef.current) {
        wsSubscriptionRef.current.close();
      }

      wsSubscriptionRef.current = subscribeWS(
        `/api/v2/ws/logs/${activeLogContainer}`,
        (logMsg) => {
          // If JSON message
          if (logMsg.raw) {
            setLogs((prev) => [...prev, logMsg.data]);
          } else {
            setLogs((prev) => [...prev, JSON.stringify(logMsg)]);
          }
        }
      );
    } else {
      if (wsSubscriptionRef.current) {
        wsSubscriptionRef.current.close();
        wsSubscriptionRef.current = null;
      }
    }

    return () => {
      if (wsSubscriptionRef.current) {
        wsSubscriptionRef.current.close();
      }
    };
  }, [activeLogContainer]);

  // Auto-scroll logs
  useEffect(() => {
    if (logTerminalEndRef.current) {
      logTerminalEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs]);

  const handleAction = async (name, action) => {
    const res = await api.controlContainer(name, action);
    if (res.status !== 'success') {
      alert(`Action failed: ${res.error}`);
    }
    fetchContainers();
  };

  const getPortsString = (ports) => {
    if (!ports || ports.length === 0) return 'none';
    return ports
      .filter((p) => p.public_port)
      .map((p) => `${p.public_port}->${p.private_port}`)
      .join(', ');
  };

  const getStatusColor = (state) => {
    const s = state.toLowerCase();
    if (s === 'running') return 'text-emerald-400';
    if (s === 'exited' || s === 'dead') return 'text-rose-400';
    return 'text-amber-400';
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white">Container Instances</h1>
          <p className="text-slate-400 text-sm">Monitor services, control containers, and inspect active logs.</p>
        </div>
        <button
          onClick={fetchContainers}
          className="px-4 py-2 text-sm font-semibold rounded-lg bg-slate-800 hover:bg-slate-700 text-white border border-slate-700/60 transition-colors"
        >
          Refresh List
        </button>
      </div>

      {error && (
        <div className="p-4 rounded-xl bg-rose-500/10 text-rose-400 border border-rose-500/20 text-sm">
          {error}
        </div>
      )}

      {loading && containers.length === 0 ? (
        <div className="flex justify-center items-center h-48 text-slate-500">
          Loading Docker services...
        </div>
      ) : (
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
          {/* Containers Table */}
          <div className="xl:col-span-2 bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-sm">
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-slate-800/80 bg-slate-950/20 text-xs font-semibold text-slate-400 uppercase tracking-wider">
                    <th className="p-4">Name</th>
                    <th className="p-4">State</th>
                    <th className="p-4">Image</th>
                    <th className="p-4">Ports</th>
                    <th className="p-4 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/40 text-sm">
                  {containers.map((c) => {
                    const cname = c.names[0].lstrip ? c.names[0].lstrip('/') : c.names[0].replace(/^\//, '');
                    return (
                      <tr key={c.id} className="hover:bg-slate-950/10 transition-colors">
                        <td className="p-4 font-semibold text-white">{cname}</td>
                        <td className="p-4 font-mono">
                          <span className={`inline-flex items-center gap-1.5 ${getStatusColor(c.state)}`}>
                            <span className={`w-1.5 h-1.5 rounded-full ${c.state === 'running' ? 'bg-emerald-400 animate-pulse' : 'bg-rose-400'}`}></span>
                            {c.state.toUpperCase()}
                          </span>
                        </td>
                        <td className="p-4 text-xs font-mono text-slate-400 max-w-[150px] truncate" title={c.image}>
                          {c.image}
                        </td>
                        <td className="p-4 text-xs font-mono text-slate-400">{getPortsString(c.ports)}</td>
                        <td className="p-4 text-right space-x-2">
                          <button
                            onClick={() => setActiveLogContainer(cname)}
                            className="px-2.5 py-1 text-xs font-semibold rounded bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700/60"
                          >
                            Logs
                          </button>
                          {c.state === 'running' ? (
                            <button
                              onClick={() => handleAction(cname, 'stop')}
                              className="px-2.5 py-1 text-xs font-semibold rounded bg-rose-950/30 hover:bg-rose-950/60 text-rose-400 border border-rose-900/40"
                            >
                              Stop
                            </button>
                          ) : (
                            <button
                              onClick={() => handleAction(cname, 'start')}
                              className="px-2.5 py-1 text-xs font-semibold rounded bg-emerald-950/30 hover:bg-emerald-950/60 text-emerald-400 border border-emerald-900/40"
                            >
                              Start
                            </button>
                          )}
                          <button
                            onClick={() => handleAction(cname, 'restart')}
                            className="px-2.5 py-1 text-xs font-semibold rounded bg-slate-850 hover:bg-slate-800 text-slate-400 border border-slate-800"
                          >
                            Restart
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                  {containers.length === 0 && (
                    <tr>
                      <td colSpan="5" className="text-center p-8 text-slate-500">
                        No containers currently managed.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Logs Console Pane */}
          <div className="xl:col-span-1 bg-slate-900 border border-slate-800 rounded-xl p-6 flex flex-col h-[500px]">
            <div className="flex justify-between items-center mb-4">
              <div>
                <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider">Log Streaming</h2>
                <span className="text-xs text-slate-500 font-mono">
                  {activeLogContainer ? `tail -f ${activeLogContainer}` : 'Select a container to stream logs'}
                </span>
              </div>
              {activeLogContainer && (
                <button
                  onClick={() => setActiveLogContainer(null)}
                  className="text-xs text-rose-400 hover:underline"
                >
                  Close Stream
                </button>
              )}
            </div>

            <div className="flex-1 bg-slate-950 rounded-lg p-4 font-mono text-[11px] overflow-y-auto space-y-1 text-emerald-400 border border-slate-950 relative">
              {activeLogContainer ? (
                <>
                  {logs.map((log, idx) => (
                    <div key={idx} className="whitespace-pre-wrap leading-relaxed border-l-2 border-emerald-500/20 pl-2">
                      {log}
                    </div>
                  ))}
                  <div ref={logTerminalEndRef}></div>
                </>
              ) : (
                <div className="flex h-full items-center justify-center text-slate-700 italic">
                  Select a container to open WebSocket log console...
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
