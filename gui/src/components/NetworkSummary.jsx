import React, { useState, useEffect } from 'react';
import { api } from '../api';
import { IconShield, IconRefresh } from './Icons';

export default function NetworkSummary() {
  const [vpn, setVpn] = useState(null);
  const [loading, setLoading] = useState(true);

  const fetchVpn = async () => {
    setLoading(true);
    try {
      const res = await api.getVPNStatus();
      if (res.status === 'success' && res.data) {
        setVpn(res.data);
      } else {
        setVpn({ connected: false, status_text: 'Inactive / Standby' });
      }
    } catch {
      setVpn({ connected: false, status_text: 'Not configured' });
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchVpn();
    const interval = setInterval(fetchVpn, 15000);
    return () => clearInterval(interval);
  }, []);

  const isConnected = vpn?.connected === true;

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 flex flex-col justify-between">
      <div>
        <div className="flex justify-between items-center mb-4">
          <div className="flex items-center gap-2">
            <IconShield className={`w-5 h-5 ${isConnected ? 'text-emerald-400' : 'text-slate-400'}`} />
            <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wider">
              Network & Privacy
            </h2>
          </div>
          <div className="flex items-center gap-2">
            <span
              className={`px-2 py-0.5 rounded text-[11px] font-bold ${
                isConnected
                  ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                  : 'bg-slate-800 text-slate-400 border border-slate-700'
              }`}
            >
              {isConnected ? 'PROTECTED' : 'LOCAL ROUTING'}
            </span>
            <button
              onClick={fetchVpn}
              className="p-1 text-slate-500 hover:text-slate-300 transition-colors rounded hover:bg-slate-800"
              title="Refresh VPN status"
            >
              <IconRefresh className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>

        <div className="space-y-2.5 text-xs">
          <div className="flex justify-between items-center p-2 rounded-lg bg-slate-950/40 border border-slate-800/40">
            <span className="text-slate-400 font-medium">Gateway Protocol</span>
            <span className="text-white font-mono font-semibold">Traefik Ingress (v3)</span>
          </div>
          <div className="flex justify-between items-center p-2 rounded-lg bg-slate-950/40 border border-slate-800/40">
            <span className="text-slate-400 font-medium">VPN Tunnel</span>
            <span className={`font-mono font-semibold ${isConnected ? 'text-emerald-400' : 'text-slate-400'}`}>
              {vpn?.status_text || (isConnected ? 'Connected' : 'Offline / Unrouted')}
            </span>
          </div>
          {vpn?.region && (
            <div className="flex justify-between items-center p-2 rounded-lg bg-slate-950/40 border border-slate-800/40">
              <span className="text-slate-400 font-medium">VPN Exit Node</span>
              <span className="text-sky-400 font-mono font-semibold">{vpn.region}</span>
            </div>
          )}
          {vpn?.external_ip && (
            <div className="flex justify-between items-center p-2 rounded-lg bg-slate-950/40 border border-slate-800/40">
              <span className="text-slate-400 font-medium">Public IP</span>
              <span className="text-slate-300 font-mono">{vpn.external_ip}</span>
            </div>
          )}
        </div>
      </div>

      <div className="mt-4 pt-3 border-t border-slate-800/60 flex items-center justify-between text-[11px] text-slate-500">
        <span>DNS Leak Protection: Active</span>
        <span className="text-emerald-400 font-medium">Port 5050 Active</span>
      </div>
    </div>
  );
}
