import { useState, useEffect } from 'react';
import { api } from '../api';
import { IconDoctor, IconRefresh } from './Icons';

export default function DoctorModal({ isOpen, onClose }) {
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [activeFilter, setActiveFilter] = useState('all'); // 'all', 'fail', 'warn', 'pass'

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
      setActiveFilter('all');
    }
  }, [isOpen]);

  if (!isOpen) return null;

  const getStatusType = (status) => {
    const s = String(status || '').trim().toLowerCase();
    if (s === 'pass' || s === 'ok' || s === 'passed' || s === 'healthy') return 'pass';
    if (s === 'warn' || s === 'warning') return 'warn';
    return 'fail';
  };

  const passedCount = results.filter((r) => getStatusType(r.status) === 'pass').length;
  const warnCount = results.filter((r) => getStatusType(r.status) === 'warn').length;
  const failedCount = results.filter((r) => getStatusType(r.status) === 'fail').length;

  const filteredResults = results.filter((r) => {
    if (activeFilter === 'all') return true;
    return getStatusType(r.status) === activeFilter;
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md animate-fade-in">
      <div className="bg-[#0f172a] border border-slate-700/80 rounded-2xl w-full max-w-2xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
        {/* Header */}
        <div className="p-6 border-b border-slate-800 flex justify-between items-center bg-slate-950/60">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-xl bg-teal-500/10 text-teal-400 border border-teal-500/20 shadow-sm">
              <IconDoctor className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-lg font-bold text-white tracking-tight">System Doctor Diagnostics</h2>
              <p className="text-xs text-slate-400">Pre-flight system, network, configuration, and storage health checks.</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white p-1.5 rounded-lg hover:bg-slate-800 transition-colors"
            title="Close dialog"
          >
            ✕
          </button>
        </div>

        {/* Status Counters & Filter Ribbon */}
        <div className="px-6 py-3 bg-slate-950/40 border-b border-slate-800 flex flex-wrap items-center justify-between gap-3 text-xs">
          <div className="flex items-center gap-2">
            <button
              onClick={() => setActiveFilter('all')}
              className={`px-2.5 py-1 rounded-md font-semibold transition-all ${
                activeFilter === 'all'
                  ? 'bg-slate-700 text-white shadow-sm'
                  : 'text-slate-400 hover:text-white hover:bg-slate-800'
              }`}
            >
              All ({results.length})
            </button>

            <button
              onClick={() => setActiveFilter('pass')}
              className={`px-2.5 py-1 rounded-md font-semibold flex items-center gap-1.5 transition-all ${
                activeFilter === 'pass'
                  ? 'bg-emerald-500/25 text-emerald-300 border border-emerald-500/40 shadow-sm'
                  : 'text-emerald-400/80 hover:text-emerald-300 hover:bg-emerald-500/10'
              }`}
            >
              <span className="w-2 h-2 rounded-full bg-emerald-400"></span>
              {passedCount} Passed
            </button>

            {warnCount > 0 && (
              <button
                onClick={() => setActiveFilter('warn')}
                className={`px-2.5 py-1 rounded-md font-semibold flex items-center gap-1.5 transition-all ${
                  activeFilter === 'warn'
                    ? 'bg-amber-500/25 text-amber-200 border border-amber-500/40 shadow-sm'
                    : 'text-amber-400/80 hover:text-amber-300 hover:bg-amber-500/10'
                }`}
              >
                <span className="w-2 h-2 rounded-full bg-amber-400"></span>
                {warnCount} Warnings
              </button>
            )}

            {failedCount > 0 && (
              <button
                onClick={() => setActiveFilter('fail')}
                className={`px-2.5 py-1 rounded-md font-semibold flex items-center gap-1.5 transition-all ${
                  activeFilter === 'fail'
                    ? 'bg-rose-500/25 text-rose-200 border border-rose-500/40 shadow-sm'
                    : 'text-rose-400/90 hover:text-rose-300 hover:bg-rose-500/10'
                }`}
              >
                <span className="w-2 h-2 rounded-full bg-rose-400 animate-ping"></span>
                {failedCount} Critical Issues
              </button>
            )}
          </div>

          <button
            onClick={runDiagnostics}
            disabled={loading}
            className="flex items-center gap-1.5 px-3 py-1 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded-md font-semibold transition-all disabled:opacity-50"
          >
            <IconRefresh className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            <span>Re-run</span>
          </button>
        </div>

        {/* Content list */}
        <div className="flex-1 overflow-y-auto p-6 space-y-3">
          {loading && results.length === 0 ? (
            <div className="py-16 text-center text-slate-400 text-sm">
              <div className="w-7 h-7 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin mx-auto mb-3"></div>
              Running system diagnostic tests...
            </div>
          ) : error ? (
            <div className="p-4 rounded-xl bg-rose-950/40 text-rose-300 border border-rose-500/30 text-sm">
              <div className="font-semibold mb-1">Diagnostic Run Failed</div>
              <div className="font-mono text-xs text-rose-400">{error}</div>
            </div>
          ) : filteredResults.length === 0 ? (
            <div className="py-12 text-center text-slate-500 text-sm italic">
              No diagnostic checks match filter &quot;{activeFilter}&quot;.
            </div>
          ) : (
            filteredResults.map((r, i) => {
              const statusType = getStatusType(r.status);
              const isPass = statusType === 'pass';
              const isWarn = statusType === 'warn';

              return (
                <div
                  key={i}
                  className={`p-4 rounded-xl border flex items-start justify-between gap-4 text-xs transition-all ${
                    isPass
                      ? 'bg-emerald-950/20 border-emerald-500/30 hover:border-emerald-500/50'
                      : isWarn
                      ? 'bg-amber-950/25 border-amber-500/40 ring-1 ring-amber-500/20 hover:border-amber-500/60'
                      : 'bg-rose-950/30 border-rose-500/50 ring-1 ring-rose-500/30 shadow-[0_0_15px_rgba(244,63,94,0.15)] hover:border-rose-500/70'
                  }`}
                >
                  <div className="flex items-start gap-3">
                    <span
                      className={`w-2 h-2 rounded-full mt-1.5 flex-shrink-0 ${
                        isPass
                          ? 'bg-emerald-400'
                          : isWarn
                          ? 'bg-amber-400'
                          : 'bg-rose-400 animate-pulse'
                      }`}
                    ></span>

                    <div className="space-y-1">
                      <div
                        className={`text-sm font-semibold tracking-wide ${
                          isPass
                            ? 'text-white'
                            : isWarn
                            ? 'text-amber-100'
                            : 'text-rose-100'
                        }`}
                      >
                        {r.name}
                      </div>
                      <div
                        className={`font-mono text-[11px] leading-relaxed break-words ${
                          isPass
                            ? 'text-slate-300'
                            : isWarn
                            ? 'text-amber-200/90'
                            : 'text-rose-200/90'
                        }`}
                      >
                        {r.message}
                      </div>
                    </div>
                  </div>

                  <span
                    className={`px-2.5 py-1 rounded text-[10px] font-bold uppercase tracking-wider flex-shrink-0 flex items-center gap-1 ${
                      isPass
                        ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40'
                        : isWarn
                        ? 'bg-amber-500/20 text-amber-300 border border-amber-500/50'
                        : 'bg-rose-500/25 text-rose-200 border border-rose-500/50 animate-pulse'
                    }`}
                  >
                    <span>{isPass ? '✓' : isWarn ? '⚠️' : '✕'}</span>
                    <span>{isPass ? 'PASS' : isWarn ? 'WARN' : 'FAIL'}</span>
                  </span>
                </div>
              );
            })
          )}
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-slate-800 bg-slate-950/60 flex items-center justify-between text-xs text-slate-400">
          <span>Click status filters above to isolate warnings or critical failures.</span>
          <button
            onClick={onClose}
            className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white rounded-lg font-semibold transition-colors shadow-sm"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
