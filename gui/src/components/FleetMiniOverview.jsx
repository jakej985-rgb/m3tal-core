import React, { useState, useEffect } from 'react';
import { api } from '../api';
import { IconContainers, IconRefresh } from './Icons';

export default function FleetMiniOverview({ onSelectContainer }) {
  const [containers, setContainers] = useState([]);
  const [loading, setLoading] = useState(true);

  const fetchFleet = async () => {
    setLoading(true);
    const res = await api.getContainers();
    if (res.status === 'success' && Array.isArray(res.data)) {
      setContainers(res.data);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchFleet();
    const interval = setInterval(fetchFleet, 10000);
    return () => clearInterval(interval);
  }, []);

  const runningCount = containers.filter((c) => c.state === 'running').length;
  const stoppedCount = containers.length - runningCount;

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 flex flex-col justify-between">
      <div>
        <div className="flex justify-between items-center mb-4">
          <div className="flex items-center gap-2">
            <IconContainers className="w-5 h-5 text-emerald-400" />
            <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wider">
              Container Fleet
            </h2>
          </div>
          <div className="flex items-center gap-2">
            <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              {runningCount} Running
            </span>
            {stoppedCount > 0 && (
              <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-slate-800 text-slate-400 border border-slate-700">
                {stoppedCount} Stopped
              </span>
            )}
            <button
              onClick={fetchFleet}
              className="p-1 text-slate-500 hover:text-slate-300 transition-colors rounded hover:bg-slate-800"
              title="Refresh containers"
            >
              <IconRefresh className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>

        {containers.length === 0 ? (
          <div className="text-center py-6 text-xs text-slate-500 italic">
            {loading ? 'Checking Docker containers...' : 'No containers currently managed.'}
          </div>
        ) : (
          <div className="space-y-2">
            {containers.slice(0, 4).map((c) => {
              const name = (c.names?.[0] || 'unknown').replace(/^\//, '');
              const isRunning = c.state === 'running';
              return (
                <div
                  key={c.id || name}
                  className="flex items-center justify-between p-2.5 rounded-lg bg-slate-950/40 border border-slate-800/50 text-xs"
                >
                  <div className="flex items-center gap-2.5 min-w-0">
                    <span
                      className={`w-2 h-2 rounded-full flex-shrink-0 ${
                        isRunning ? 'bg-emerald-400 animate-pulse' : 'bg-slate-600'
                      }`}
                    ></span>
                    <div className="truncate">
                      <div className="font-semibold text-slate-200 truncate">{name}</div>
                      <div className="text-[10px] text-slate-500 font-mono truncate">{c.image}</div>
                    </div>
                  </div>
                  <span
                    className={`px-2 py-0.5 rounded text-[10px] font-mono uppercase font-bold ${
                      isRunning
                        ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                        : 'bg-slate-800 text-slate-500'
                    }`}
                  >
                    {c.state}
                  </span>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {containers.length > 4 && (
        <div className="mt-3 text-center">
          <span className="text-[11px] text-slate-500">
            +{containers.length - 4} more services (see Containers tab)
          </span>
        </div>
      )}
    </div>
  );
}
