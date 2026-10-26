import { useState, useEffect } from 'react';
import { api, subscribeWS } from '../api';
import DoctorModal from '../components/DoctorModal';
import FleetMiniOverview from '../components/FleetMiniOverview';
import NetworkSummary from '../components/NetworkSummary';
import TopBarWidget from '../components/widgets/TopBarWidget';
import StatCardWidget from '../components/widgets/StatCardWidget';
import StorageDetailedWidget from '../components/widgets/StorageDetailedWidget';
import EventLogWidget from '../components/widgets/EventLogWidget';
import PluginWidgetSlot from '../components/widgets/PluginWidgetSlot';

export default function Dashboard() {
  const [metrics, setMetrics] = useState(null);
  const [history, setHistory] = useState([]);
  const [health, setHealth] = useState(null);
  const [events, setEvents] = useState([]);
  const [error, setError] = useState(null);
  const [pluginWidgets, setPluginWidgets] = useState([]);

  // Quick action and modal states
  const [isDoctorOpen, setIsDoctorOpen] = useState(false);
  const [isReconciling, setIsReconciling] = useState(false);
  const [isCleaning, setIsCleaning] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [toast, setToast] = useState(null);

  const showToast = (message, type = 'info') => {
    setToast({ message, type });
    setTimeout(() => {
      setToast(null);
    }, 4500);
  };

  const fetchStats = async () => {
    // 1. Fetch current metrics
    const resMetrics = await api.getMetrics();
    if (resMetrics.status === 'success') {
      setMetrics(resMetrics.data);
      setError(null);
    } else {
      setError(resMetrics.error || 'Failed to fetch metrics');
    }

    // 2. Fetch history
    const resHistory = await api.getSystemMetricsHistory();
    if (resHistory.status === 'success') {
      setHistory(resHistory.data || []);
    }

    // 3. Fetch health status
    const resHealth = await api.getSystemHealth();
    if (resHealth.status === 'success') {
      setHealth(resHealth.data);
    }

    // 4. Fetch dynamic plugin widgets if available
    try {
      const resPlugins = await api.getPlugins();
      if (resPlugins?.status === 'success' && Array.isArray(resPlugins.data)) {
        const widgets = [];
        resPlugins.data.forEach((p) => {
          if (Array.isArray(p.widgets)) {
            p.widgets.forEach((w) => widgets.push({ ...w, plugin_name: p.name }));
          }
        });
        setPluginWidgets(widgets);
      }
    } catch {
      // Non-critical fallback
    }
  };

  useEffect(() => {
    fetchStats();
    const interval = setInterval(fetchStats, 5000);

    // Subscribe to WebSocket events for live notification updates
    const sub = subscribeWS('/api/v2/ws/events', (event) => {
      const timeStr = new Date(event.timestamp * 1000).toLocaleTimeString();
      const newEvent = {
        time: timeStr,
        type: event.type,
        detail: typeof event.payload === 'object' ? JSON.stringify(event.payload) : String(event.payload),
      };
      setEvents((prev) => [newEvent, ...prev].slice(0, 20)); // Keep last 20
    });

    return () => {
      clearInterval(interval);
      sub.close();
    };
  }, []);

  const handleManualRefresh = async () => {
    setIsRefreshing(true);
    await fetchStats();
    setTimeout(() => setIsRefreshing(false), 500);
  };

  const handleReconcile = async () => {
    setIsReconciling(true);
    try {
      const res = await api.reconcileSystem();
      if (res.status === 'success') {
        const actionCount = Array.isArray(res.data?.actions) ? res.data.actions.length : 0;
        showToast(
          actionCount > 0
            ? `System reconciled: ${actionCount} adjustment(s) applied.`
            : 'System state verified: fully converged.',
          'success'
        );
        fetchStats();
      } else {
        showToast(res.error || 'Reconciliation failed', 'error');
      }
    } catch (err) {
      showToast(err.message, 'error');
    }
    setIsReconciling(false);
  };

  const handleSystemCleanup = async () => {
    setIsCleaning(true);
    try {
      const res = await api.systemCleanup();
      if (res.status === 'success') {
        const freed = res.data?.freed_human || '0 B';
        const count = res.data?.items_cleaned?.length || 0;
        showToast(`System cleanup complete (${freed} freed across ${count} tasks).`, 'success');
        fetchStats();
      } else {
        showToast(res.error || 'System cleanup failed', 'error');
      }
    } catch (err) {
      showToast(err.message, 'error');
    }
    setIsCleaning(false);
  };

  const handleUnmountStorage = async (mountpoint) => {
    try {
      const res = await api.unmountStorage(mountpoint);
      if (res.status === 'success') {
        showToast(`Drive at ${mountpoint} safely unmounted. Safe to unplug.`, 'success');
        fetchStats();
      } else {
        showToast(res.error || 'Failed to unmount drive', 'error');
      }
    } catch (err) {
      showToast(err.message, 'error');
    }
  };

  return (
    <div className="space-y-6">
      {/* Floating Action Toast */}
      {toast && (
        <div className={`p-3 rounded-xl border flex items-center justify-between shadow-lg text-xs font-medium animate-fade-in ${
          toast.type === 'success'
            ? 'bg-emerald-950/80 text-emerald-300 border-emerald-500/40'
            : toast.type === 'error'
            ? 'bg-rose-950/80 text-rose-300 border-rose-500/40'
            : 'bg-slate-900/90 text-slate-200 border-slate-700'
        }`}>
          <div className="flex items-center gap-2">
            <span className="text-sm">{toast.type === 'success' ? '✓' : toast.type === 'error' ? '⚠️' : 'ℹ️'}</span>
            <span>{toast.message}</span>
          </div>
          <button
            onClick={() => setToast(null)}
            className="text-slate-400 hover:text-white ml-4 text-xs font-bold"
          >
            ✕
          </button>
        </div>
      )}

      {/* 4-Column Modular Widget Grid Engine */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 items-start">
        {/* 1. Top Bar Widget (4x1) */}
        <TopBarWidget
          metrics={metrics}
          health={health}
          onOpenDoctor={() => setIsDoctorOpen(true)}
          onReconcile={handleReconcile}
          isReconciling={isReconciling}
          onCleanup={handleSystemCleanup}
          isCleaning={isCleaning}
          onRefresh={handleManualRefresh}
          isRefreshing={isRefreshing}
          error={error}
        />

        {/* 2. System Stat Cards Row (1x1 each: CPU, RAM, GPU, Root Disk) */}
        <StatCardWidget type="cpu" metrics={metrics} history={history} />
        <StatCardWidget type="ram" metrics={metrics} history={history} />
        <StatCardWidget type="gpu" metrics={metrics} history={history} />
        <StatCardWidget type="disk" metrics={metrics} history={history} />

        {/* 3. Detailed Storage Widget (4x3) with Partition & Mount Filtering */}
        <StorageDetailedWidget
          diskPartitions={metrics?.disk_partitions || []}
          onCleanup={handleSystemCleanup}
          isCleaning={isCleaning}
          onUnmount={handleUnmountStorage}
        />

        {/* 4. Fleet Overview (2x1) & Network Summary (2x1) */}
        <div className="col-span-1 md:col-span-2 lg:col-span-2">
          <FleetMiniOverview />
        </div>
        <div className="col-span-1 md:col-span-2 lg:col-span-2">
          <NetworkSummary networkInterfaces={metrics?.network_interfaces} />
        </div>

        {/* 5. Dynamic External Plugin Widgets (optional) */}
        {pluginWidgets.map((pw) => (
          <PluginWidgetSlot
            key={pw.id}
            widget={pw}
            onAction={async (pluginName, action) => {
              try {
                const res = await api.pluginAction(pluginName, action);
                if (res.status === 'success') {
                  showToast(`Action executed: ${action}`, 'success');
                } else {
                  showToast(res.error || 'Action failed', 'error');
                }
              } catch (err) {
                showToast(err.message, 'error');
              }
            }}
          />
        ))}

        {/* 6. Live System Event Log (4x2) */}
        <EventLogWidget
          events={events}
          onClear={() => setEvents([])}
        />
      </div>

      {/* Doctor Diagnostics Modal */}
      <DoctorModal isOpen={isDoctorOpen} onClose={() => setIsDoctorOpen(false)} />
    </div>
  );
}
