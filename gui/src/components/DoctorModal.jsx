import React, { useState, useEffect } from 'react';
import { api } from '../api';
import { IconDoctor, IconRefresh } from './Icons';

export default function DoctorModal({ isOpen, onClose }) {
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const runDiagnostics = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.runDoctor();
      if (res.status === 'success' && Array.isArray(res.data)) {
        setResults(res.data);
      } else {
        setError(res.error || 'Failed to complete pre-flight check');
      }
    } catch (err) {
      setError(err.message);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (isOpen) {
      runDiagnostics();
    }
  }, [isOpen]);

  if (!isOpen) return null;

  const passedCount = results.filter((r) => r.status === 'pass').length;
  const failedCount = results.filter((r) => r.status === 'fail').length;
  const warnCount = results.filter((r) => r.status === 'warn').length;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-fade-in">
      <div className="bg-slate-900 border border-slate-700/80 rounded-2xl w-full max-w-2xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
        {/* Header */}
        <div className="p-6 border-b border-slate-800 flex justify-between items-center bg-slate-950/40">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-teal-500/10 text-teal-400 border border-teal-500/20">
              <IconDoctor className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-lg font-bold text-white">System Doctor Diagnostics</h2>
              <p className="text-xs text-slate-400">Pre-flight system, network, and storage verification.</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-slate-800 transition-colors"
          >
            ✕
          </button>
        </div>

        {/* Status Counts */}
        <div className="px-6 py-3 bg-slate-950/20 border-b border-slate-800 flex items-center gap-4 text-xs font-semibold">
          <span className="flex items-center gap-1.5 text-emerald-400">
            <span className="w-2 h-2 rounded-full bg-emerald-400"></span>
            {passedCount} Passed
          </span>
          {warnCount > 0 && (
            <span className="flex items-center gap-1.5 text-amber-400">
              <span className="w-2 h-2 rounded-full bg-amber-400"></span>
              {warnCount} Warnings
            </span>
          )}
          {failedCount > 0 && (
            <span className="flex items-center gap-1.5 text-rose-400">
              <span className="w-2 h-2 rounded-full bg-rose-400 animate-ping"></span>
              {failedCount} Issues Detected
            </span>
          )}
          <div className="ml-auto">
            <button
              onClick={runDiagnostics}
              disabled={loading}
              className="flex items-center gap-1 px-3 py-1 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-md text-xs font-semibold transition-all disabled:opacity-50"
            >
              <IconRefresh className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
              <span>Re-run</span>
            </button>
          </div>
        </div>

        {/* Content list */}
        <div className="flex-1 overflow-y-auto p-6 space-y-2.5">
          {loading && results.length === 0 ? (
            <div className="py-12 text-center text-slate-500 text-sm">
              <div className="w-6 h-6 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin mx-auto mb-3"></div>
              Running system diagnostic tests...
            </div>
          ) : error ? (
            <div className="p-4 rounded-xl bg-rose-500/10 text-rose-400 border border-rose-500/20 text-sm">
              {error}
            </div>
          ) : (
            results.map((r, i) => {
              const isPass = r.status === 'pass';
              const isWarn = r.status === 'warn';
              return (
                <div
                  key={i}
                  className={`p-3.5 rounded-xl border flex items-start justify-between gap-4 text-xs transition-colors ${
                    isPass
                      ? 'bg-slate-950/30 border-slate-800/60'
                      : isWarn
                      ? 'bg-amber-950/20 border-amber-900/40 text-amber-300'
                      : 'bg-rose-950/20 border-rose-900/40 text-rose-300'
                  }`}
                >
                  <div className="space-y-1">
                    <div className="font-semibold text-slate-200 text-sm">{r.name}</div>
                    <div className="text-slate-400 font-mono text-[11px] leading-relaxed">{r.message}</div>
                  </div>
                  <span
                    className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider flex-shrink-0 ${
                      isPass
                        ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                        : isWarn
                        ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                        : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                    }`}
                  >
                    {r.status}
                  </span>
                </div>
              );
            })
          )}
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-slate-800 bg-slate-950/40 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white rounded-lg text-xs font-semibold transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
