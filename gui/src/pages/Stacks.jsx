import React, { useState, useEffect } from 'react';
import { api, subscribeWS } from '../api';

export default function Stacks() {
  const [stacks, setStacks] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [actioning, setActioning] = useState(null);

  const fetchStacks = async () => {
    setLoading(true);
    const res = await api.getStacks();
    if (res.status === 'success') {
      setStacks(res.data);
      setError(null);
    } else {
      setError(res.error || 'Failed to fetch stacks');
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchStacks();
    
    // Subscribe to stack events to auto-refresh status instantly!
    const sub = subscribeWS('/api/v2/ws/events', (event) => {
      if (event.type === 'stack.updated' || event.type?.startsWith('container.')) {
        fetchStacks();
      }
    });

    return () => sub.close();
  }, []);

  const handleUp = async (name) => {
    setActioning({ name, action: 'up' });
    const res = await api.deployStack(name);
    if (res.status !== 'success') {
      alert(`Error deploying stack: ${res.error}`);
    }
    setActioning(null);
    fetchStacks();
  };

  const handleDown = async (name) => {
    setActioning({ name, action: 'down' });
    const res = await api.stopStack(name);
    if (res.status !== 'success') {
      alert(`Error stopping stack: ${res.error}`);
    }
    setActioning(null);
    fetchStacks();
  };

  const getStatusBadge = (status) => {
    const s = status.toLowerCase();
    if (s === 'running' || s === 'success') {
      return (
        <span className="px-2.5 py-0.5 text-xs font-semibold rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/25">
          RUNNING
        </span>
      );
    }
    if (s === 'stopped') {
      return (
        <span className="px-2.5 py-0.5 text-xs font-semibold rounded-full bg-slate-500/10 text-slate-400 border border-slate-500/25">
          STOPPED
        </span>
      );
    }
    if (s === 'failed') {
      return (
        <span className="px-2.5 py-0.5 text-xs font-semibold rounded-full bg-rose-500/10 text-rose-400 border border-rose-500/25">
          FAILED
        </span>
      );
    }
    return (
      <span className="px-2.5 py-0.5 text-xs font-semibold rounded-full bg-amber-500/10 text-amber-400 border border-amber-500/25">
        DISCOVERED
      </span>
    );
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white">Stack Configurations</h1>
          <p className="text-slate-400 text-sm">Deploy and stop Docker Compose stacks.</p>
        </div>
        <button
          onClick={fetchStacks}
          className="px-4 py-2 text-sm font-semibold rounded-lg bg-slate-800 hover:bg-slate-700 text-white border border-slate-700/60 transition-colors"
        >
          Refresh Stacks
        </button>
      </div>

      {error && (
        <div className="p-4 rounded-xl bg-rose-500/10 text-rose-400 border border-rose-500/20 text-sm">
          {error}
        </div>
      )}

      {loading && stacks.length === 0 ? (
        <div className="flex justify-center items-center h-48 text-slate-500">
          Loading stacks list...
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-6">
          {stacks.map((stack) => (
            <div key={stack.name} className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-sm">
              <div className="p-6 border-b border-slate-800/60 flex flex-col sm:flex-row justify-between sm:items-center gap-4">
                <div className="space-y-1">
                  <div className="flex items-center gap-3">
                    <span className="text-lg font-bold text-white">{stack.name}</span>
                    {getStatusBadge(stack.status)}
                  </div>
                  <div className="text-xs text-slate-500 font-mono break-all">{stack.compose_path}</div>
                </div>
                
                <div className="flex items-center gap-3">
                  {actioning?.name === stack.name ? (
                    <span className="text-sm text-amber-400 flex items-center gap-2">
                      <span className="w-4 h-4 border-2 border-amber-400 border-t-transparent rounded-full animate-spin"></span>
                      Executing {actioning.action}...
                    </span>
                  ) : (
                    <>
                      <button
                        onClick={() => handleUp(stack.name)}
                        className="px-4 py-1.5 text-sm font-semibold rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white shadow-sm transition-colors"
                      >
                        Deploy (Up)
                      </button>
                      <button
                        onClick={() => handleDown(stack.name)}
                        className="px-4 py-1.5 text-sm font-semibold rounded-lg bg-slate-800 hover:bg-slate-700 text-white border border-slate-700 transition-colors"
                      >
                        Stop (Down)
                      </button>
                    </>
                  )}
                </div>
              </div>

              {stack.services && stack.services.length > 0 && (
                <div className="px-6 py-4 bg-slate-950/20">
                  <div className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Declared Services</div>
                  <div className="flex flex-wrap gap-2">
                    {stack.services.map((srv) => (
                      <span key={srv} className="px-2.5 py-1 text-xs font-medium font-mono rounded bg-slate-850 text-slate-300 border border-slate-800/40">
                        {srv}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ))}
          {stacks.length === 0 && (
            <div className="text-center p-8 bg-slate-900 border border-slate-800 rounded-xl text-slate-500">
              No Docker Compose files discovered in the configured stack folder.
            </div>
          )}
        </div>
      )}
    </div>
  );
}
