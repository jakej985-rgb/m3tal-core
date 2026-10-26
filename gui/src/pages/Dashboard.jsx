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

const DEFAULT_LAYOUT = [
  'stat-cpu',
  'stat-ram',
  'stat-gpu',
  'stat-disk',
  'storage-detailed',
  'fleet-containers',
  'network-stats',
  'event-log',
];

const WIDGET_TITLES = {
  'stat-cpu': 'CPU Metric Card',
  'stat-ram': 'RAM Metric Card',
  'stat-gpu': 'GPU Metric Card',
  'stat-disk': 'Root Disk Card',
  'storage-detailed': 'Storage & Drives',
  'fleet-containers': 'Container Fleet',
  'network-stats': 'Network Statistics',
  'event-log': 'Live Event Log',
};

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

  // Edit Dashboard Mode and Layout Persistence
  const STORAGE_KEY = 'm3tal_dashboard_layout_v1';
  const [isEditMode, setIsEditMode] = useState(false);
  const [layoutOrder, setLayoutOrder] = useState(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        const parsed = JSON.parse(saved);
        if (Array.isArray(parsed.order) && parsed.order.length > 0) return parsed.order;
      }
    } catch {
      // Ignore
    }
    return DEFAULT_LAYOUT;
  });

  const [hiddenWidgets, setHiddenWidgets] = useState(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        const parsed = JSON.parse(saved);
        if (Array.isArray(parsed.hidden)) return parsed.hidden;
      }
    } catch {
      // Ignore
    }
    return [];
  });

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({ order: layoutOrder, hidden: hiddenWidgets }));
    } catch {
      // Ignore
    }
  }, [layoutOrder, hiddenWidgets]);

  // Sync external plugin widgets into layout order if not yet added
  useEffect(() => {
    if (pluginWidgets.length > 0) {
      setLayoutOrder((prev) => {
        const next = [...prev];
        pluginWidgets.forEach((pw) => {
          const key = `plugin-${pw.id || pw.title}`;
          if (!next.includes(key) && !hiddenWidgets.includes(key)) {
            next.push(key);
          }
        });
        return next;
      });
    }
  }, [pluginWidgets, hiddenWidgets]);

  const handleMoveWidget = (widgetId, direction) => {
    setLayoutOrder((prev) => {
      const currentIndex = prev.indexOf(widgetId);
      if (currentIndex === -1) return prev;
      const targetIndex = currentIndex + direction;
      if (targetIndex < 0 || targetIndex >= prev.length) return prev;

      const newOrder = [...prev];
      const [movedItem] = newOrder.splice(currentIndex, 1);
      newOrder.splice(targetIndex, 0, movedItem);
      return newOrder;
    });
  };

  const handleHideWidget = (widgetId) => {
    setHiddenWidgets((prev) => (prev.includes(widgetId) ? prev : [...prev, widgetId]));
  };

  const handleRestoreWidget = (widgetId) => {
    setHiddenWidgets((prev) => prev.filter((id) => id !== widgetId));
    setLayoutOrder((prev) => (prev.includes(widgetId) ? prev : [...prev, widgetId]));
  };

  const handleResetLayout = () => {
    setLayoutOrder(DEFAULT_LAYOUT);
    setHiddenWidgets([]);
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch {
      // Ignore
    }
  };

  const getWidgetLabel = (wid) => {
    if (WIDGET_TITLES[wid]) return WIDGET_TITLES[wid];
    if (wid.startsWith('plugin-')) {
      const pw = pluginWidgets.find((w) => `plugin-${w.id || w.title}` === wid);
      if (pw?.title) return `Plugin: ${pw.title}`;
    }
    return wid;
  };

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

  const handlePluginAction = async (pluginName, action) => {
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
        {/* 1. Top Bar Control Plane Widget (4x1) */}
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
          isEditMode={isEditMode}
          onToggleEditMode={() => setIsEditMode((prev) => !prev)}
        />

        {/* Edit Mode Customization Banner */}
        {isEditMode && (
          <div className="col-span-1 md:col-span-2 lg:col-span-4 bg-amber-500/10 border border-amber-500/30 rounded-xl p-3.5 flex flex-wrap items-center justify-between gap-3 text-xs text-amber-200 animate-fade-in shadow-md">
            <div className="flex items-center gap-2">
              <span className="text-base">🛠️</span>
              <div>
                <span className="font-bold text-amber-300">Dashboard Edit Mode:</span>{' '}
                <span>Reorder cards using ◀ and ▶ or ✕ to hide. Positions save to your browser automatically.</span>
              </div>
            </div>
            <div className="flex items-center gap-2 flex-wrap">
              {hiddenWidgets.length > 0 && (
                <div className="flex items-center gap-1.5 flex-wrap">
                  <span className="text-slate-400 text-[11px]">Hidden ({hiddenWidgets.length}):</span>
                  {hiddenWidgets.map((wid) => (
                    <button
                      key={wid}
                      onClick={() => handleRestoreWidget(wid)}
                      className="px-2 py-0.5 rounded bg-amber-500/20 hover:bg-amber-500/30 text-amber-300 border border-amber-500/40 text-[11px] font-semibold transition-colors flex items-center gap-1"
                      title={`Restore ${getWidgetLabel(wid)}`}
                    >
                      <span>+</span>
                      <span>{getWidgetLabel(wid)}</span>
                    </button>
                  ))}
                </div>
              )}
              <button
                onClick={handleResetLayout}
                className="px-2.5 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 border border-slate-700 text-[11px] transition-colors"
                title="Reset dashboard grid layout to factory defaults"
              >
                Reset Default
              </button>
              <button
                onClick={() => setIsEditMode(false)}
                className="px-3 py-1 rounded bg-amber-500 hover:bg-amber-400 text-slate-950 font-bold text-[11px] transition-colors shadow-sm"
              >
                Done Editing
              </button>
            </div>
          </div>
        )}

        {/* Dynamic Re-orderable Widget Grid */}
        {layoutOrder.map((widgetId, index) => {
          if (hiddenWidgets.includes(widgetId)) return null;

          const moveProps = {
            isEditMode,
            onMoveLeft: index > 0 ? () => handleMoveWidget(widgetId, -1) : undefined,
            onMoveRight: index < layoutOrder.length - 1 ? () => handleMoveWidget(widgetId, 1) : undefined,
            onHide: () => handleHideWidget(widgetId),
          };

          switch (widgetId) {
            case 'stat-cpu':
              return <StatCardWidget key={widgetId} type="cpu" metrics={metrics} history={history} {...moveProps} />;
            case 'stat-ram':
              return <StatCardWidget key={widgetId} type="ram" metrics={metrics} history={history} {...moveProps} />;
            case 'stat-gpu':
              return <StatCardWidget key={widgetId} type="gpu" metrics={metrics} history={history} {...moveProps} />;
            case 'stat-disk':
              return <StatCardWidget key={widgetId} type="disk" metrics={metrics} history={history} {...moveProps} />;
            case 'storage-detailed':
              return (
                <StorageDetailedWidget
                  key={widgetId}
                  diskPartitions={metrics?.disk_partitions || []}
                  onCleanup={handleSystemCleanup}
                  isCleaning={isCleaning}
                  onUnmount={handleUnmountStorage}
                  {...moveProps}
                />
              );
            case 'fleet-containers':
              return (
                <div key={widgetId} className="col-span-1 md:col-span-2 lg:col-span-2">
                  <FleetMiniOverview {...moveProps} />
                </div>
              );
            case 'network-stats':
              return (
                <div key={widgetId} className="col-span-1 md:col-span-2 lg:col-span-2">
                  <NetworkSummary networkInterfaces={metrics?.network_interfaces} {...moveProps} />
                </div>
              );
            case 'event-log':
              return (
                <EventLogWidget
                  key={widgetId}
                  events={events}
                  onClear={() => setEvents([])}
                  {...moveProps}
                />
              );
            default:
              if (widgetId.startsWith('plugin-')) {
                const pw = pluginWidgets.find((w) => `plugin-${w.id || w.title}` === widgetId);
                if (pw) {
                  return (
                    <PluginWidgetSlot
                      key={widgetId}
                      widget={pw}
                      onAction={handlePluginAction}
                      {...moveProps}
                    />
                  );
                }
              }
              return null;
          }
        })}
      </div>

      {/* Doctor Diagnostics Modal */}
      <DoctorModal isOpen={isDoctorOpen} onClose={() => setIsDoctorOpen(false)} />
    </div>
  );
}
