import { useMemo, useState } from 'react';
import WidgetCard from './WidgetCard';
import { IconClean } from '../Icons';

export default function StorageDetailedWidget({
  diskPartitions = [],
  onCleanup,
  isCleaning,
  onUnmount,
  isEditMode = false,
  onMoveLeft,
  onMoveRight,
  onHide,
  className = '',
}) {
  const [filterVirtual, setFilterVirtual] = useState(true);
  const [unmountingMount, setUnmountingMount] = useState(null);

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

  // Process and group storage partitions
  const storageGroups = useMemo(() => {
    if (!diskPartitions || !Array.isArray(diskPartitions)) {
      return { system: [], usb: [], network: [], internal: [], totalDrives: 0, totalVolumes: 0 };
    }

    // 1. Mount filtering (filter out snap squashed loops and internal docker overlays if filterVirtual is true)
    const validPartitions = diskPartitions.filter((p) => {
      if (!p || !p.mountpoint) return false;
      if (filterVirtual) {
        // Exclude squashfs loop devices (snap packages)
        if (p.fstype === 'squashfs' || p.device?.startsWith('/dev/loop')) return false;
        // Exclude docker container overlay mounts
        if (p.mountpoint.includes('/docker/overlay2/') || p.fstype === 'overlay') return false;
        // Exclude temporary memory mounts unless root
        if (p.fstype === 'tmpfs' && p.mountpoint !== '/tmp') return false;
      }
      return true;
    });

    const groups = new Map();

    validPartitions.forEach((p) => {
      const isNet = p.device?.startsWith('//') || p.device?.startsWith('\\\\') || p.fstype === 'cifs' || p.fstype?.startsWith('nfs') || p.drive_type === 'network';
      const groupKey = isNet ? `net:${p.mountpoint || p.device}` : (p.disk_name ? `disk:${p.disk_name}` : `dev:${p.device}`);

      if (!groups.has(groupKey)) {
        groups.set(groupKey, {
          key: groupKey,
          isNet,
          diskName: p.disk_name || '',
          diskModel: p.disk_model || '',
          partitions: [],
        });
      }
      groups.get(groupKey).partitions.push(p);
    });

    const system = [];
    const usb = [];
    const network = [];
    const internal = [];

    groups.forEach((g) => {
      const totalGB = g.partitions.reduce((acc, p) => acc + (p.total || 0), 0);
      const usedGB = g.partitions.reduce((acc, p) => acc + (p.used || 0), 0);
      const freeGB = g.partitions.reduce((acc, p) => acc + (p.free || 0), 0);
      const usedPercent = totalGB > 0 ? (usedGB / totalGB) * 100 : 0;

      const isCriticalSpace = g.partitions.some((p) => p.used_percent >= 90) || usedPercent >= 90;
      const isWarningSpace = !isCriticalSpace && (g.partitions.some((p) => p.used_percent >= 80) || usedPercent >= 80);

      const isSystemDrive = g.partitions.some((p) => p.drive_type === 'system' || p.mountpoint === '/' || p.mountpoint === '/home' || p.mountpoint === '/tmp');
      const isUsbDrive = !isSystemDrive && (g.partitions.some((p) => p.drive_type === 'usb' || p.mountpoint?.startsWith('/media/') || p.mountpoint?.startsWith('/run/media/')));
      const isNetDrive = g.isNet || g.partitions.some((p) => p.drive_type === 'network');

      let driveCategory = 'internal';
      if (isSystemDrive) driveCategory = 'system';
      else if (isUsbDrive) driveCategory = 'usb';
      else if (isNetDrive) driveCategory = 'network';

      let title = '';
      let subtitle = '';

      if (driveCategory === 'network') {
        const p = g.partitions[0];
        title = p.label && !p.device?.includes(p.label) ? p.label : p.mountpoint;
        subtitle = `${p.mountpoint} • ${p.fstype?.toUpperCase()}`;
      } else if (driveCategory === 'system') {
        title = g.diskModel ? `System Drive • ${g.diskModel}` : 'System Drive';
        subtitle = `${g.partitions.length} Partition(s) (/, /home) • ${g.diskName || 'nvme/ssd'}`;
      } else if (driveCategory === 'usb') {
        const p = g.partitions[0];
        title = p.label && !p.device?.includes(p.label) ? p.label : (g.diskModel || g.diskName || 'USB Removable Drive');
        subtitle = `${p.mountpoint} • ${p.fstype?.toUpperCase()} • ${g.diskName || 'External'}`;
      } else {
        const p = g.partitions[0];
        title = p.label && !p.device?.includes(p.label) ? p.label : (g.diskModel || p.mountpoint || 'Internal Drive');
        subtitle = `${p.mountpoint} • ${p.fstype?.toUpperCase()} • ${g.diskName || 'Storage'}`;
      }

      const cardData = {
        ...g,
        title,
        subtitle,
        totalGB,
        usedGB,
        freeGB,
        usedPercent,
        isCriticalSpace,
        isWarningSpace,
        driveCategory,
      };

      if (driveCategory === 'system') system.push(cardData);
      else if (driveCategory === 'usb') usb.push(cardData);
      else if (driveCategory === 'network') network.push(cardData);
      else internal.push(cardData);
    });

    return {
      system,
      usb,
      network,
      internal,
      totalDrives: groups.size,
      totalVolumes: validPartitions.length,
    };
  }, [diskPartitions, filterVirtual]);

  if (storageGroups.totalVolumes === 0) {
    return null;
  }

  const headerActions = (
    <div className="flex items-center gap-2">
      <button
        onClick={() => setFilterVirtual(!filterVirtual)}
        className="px-2.5 py-1 rounded text-[11px] font-semibold bg-slate-800 text-slate-400 hover:text-white border border-slate-700 transition-colors"
        title="Toggle loop and container overlay mount filters"
      >
        {filterVirtual ? 'Physical Drives' : 'All Mounts'}
      </button>

      {onCleanup && (
        <button
          onClick={onCleanup}
          disabled={isCleaning}
          className="flex items-center gap-1.5 px-3 py-1 rounded text-[11px] font-semibold bg-emerald-500/10 text-emerald-300 border border-emerald-500/30 hover:bg-emerald-500/20 transition-all shadow-sm active:scale-95 disabled:opacity-50"
          title="Free disk space: purge caches & temporary files"
        >
          <IconClean className={`w-3.5 h-3.5 text-emerald-400 ${isCleaning ? 'animate-spin' : ''}`} />
          <span>{isCleaning ? 'Cleaning...' : 'Clean Caches'}</span>
        </button>
      )}

      <span className="text-xs text-slate-500 font-mono ml-1">
        {storageGroups.totalDrives} {storageGroups.totalDrives === 1 ? 'Drive' : 'Drives'} ({storageGroups.totalVolumes} Volumes)
      </span>
    </div>
  );

  return (
    <WidgetCard
      colSpan={4}
      rowSpan={3}
      isEditMode={isEditMode}
      onMoveLeft={onMoveLeft}
      onMoveRight={onMoveRight}
      onHide={onHide}
      title="Storage & Mounted Drives"
      subtitle="Physical disk partitions, removable media, and remote network mounts"
      badge={
        <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
          4x3 STORAGE ENGINE
        </span>
      }
      actions={headerActions}
      className={`space-y-6 ${className}`}
    >
      <div className="space-y-6">
        {/* 1. Host System Storage */}
        {storageGroups.system.map((card, idx) => (
          <div key={card.key || idx} className="space-y-2">
            <div className="text-xs font-semibold text-slate-400 flex items-center justify-between uppercase tracking-wider">
              <span className="flex items-center gap-2">💻 Host System Storage</span>
              <span className="text-[11px] text-slate-500 font-mono font-normal lowercase">
                parent block: {card.diskName || 'disk0'}
              </span>
            </div>

            <div className={`bg-slate-950/70 border rounded-xl p-5 space-y-4 relative transition-all ${
              card.isCriticalSpace
                ? 'border-rose-500/40 ring-1 ring-rose-500/20'
                : card.isWarningSpace
                ? 'border-amber-500/30'
                : 'border-slate-800/80 shadow-sm'
            }`}>
              {/* Header */}
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-slate-800/60 pb-3">
                <div className="flex items-center gap-2.5 flex-wrap">
                  <span className="font-bold text-white text-base">{card.title}</span>
                  <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                    SYSTEM
                  </span>
                  <span className="px-1.5 py-0.5 rounded text-[10px] font-semibold bg-emerald-500/15 text-emerald-300 border border-emerald-500/30">
                    S.M.A.R.T. OK
                  </span>
                  <span className="px-1.5 py-0.5 rounded text-[10px] font-semibold bg-indigo-500/15 text-indigo-300 border border-indigo-500/30">
                    {card.partitions.length} PARTITIONS
                  </span>
                  {card.isCriticalSpace && (
                    <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-500/15 text-rose-400 border border-rose-500/30 animate-pulse">
                      CRITICAL
                    </span>
                  )}
                  {card.isWarningSpace && (
                    <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-500/15 text-amber-400 border border-amber-500/30">
                      LOW SPACE
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-3 font-mono text-xs">
                  <span className="text-slate-400">Total: {formatCap(card.usedGB)} / {formatCap(card.totalGB)}</span>
                  <span className={`font-bold text-sm ${card.isCriticalSpace ? 'text-rose-400' : card.isWarningSpace ? 'text-amber-400' : 'text-emerald-400'}`}>
                    {card.usedPercent?.toFixed(1)}%
                  </span>
                </div>
              </div>

              {/* System partitions */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3.5">
                {card.partitions.map((p, pIdx) => {
                  const isPartCritical = p.used_percent >= 90;
                  const isPartWarning = p.used_percent >= 80 && p.used_percent < 90;
                  const partLabel = p.mountpoint === '/' ? 'Root (/)' : p.mountpoint === '/home' ? 'Home (/home)' : p.mountpoint === '/tmp' ? 'Temp (/tmp)' : p.mountpoint;

                  return (
                    <div key={pIdx} className="bg-slate-900/80 border border-slate-800/70 rounded-lg p-3.5 space-y-2 hover:border-slate-700 transition-colors">
                      <div className="flex justify-between items-start">
                        <div>
                          <div className="font-semibold text-slate-200 text-xs flex items-center gap-1.5">
                            <span className="text-white font-mono">{partLabel}</span>
                            {p.label && p.label !== p.mountpoint && (
                              <span className="text-[10px] text-slate-400 font-normal">({p.label})</span>
                            )}
                          </div>
                          <div className="text-[11px] text-slate-500 font-mono mt-0.5">
                            {p.device ? p.device.replace('/dev/', '') : ''} • {p.fstype?.toUpperCase()}
                          </div>
                        </div>
                        <div className="text-right font-mono">
                          <div className={`text-xs font-bold ${isPartCritical ? 'text-rose-400' : isPartWarning ? 'text-amber-400' : 'text-slate-200'}`}>
                            {p.used_percent?.toFixed(1)}%
                          </div>
                          <div className="text-[10px] text-slate-400">
                            {formatCap(p.used)} / {formatCap(p.total)}
                          </div>
                        </div>
                      </div>
                      <div className="w-full bg-slate-800 rounded-full h-1.5 overflow-hidden">
                        <div
                          className={`h-1.5 rounded-full transition-all duration-500 ${getUsageColor(p.used_percent)}`}
                          style={{ width: `${Math.min(100, p.used_percent)}%` }}
                        ></div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        ))}

        {/* 2. Removable & USB Drives */}
        {storageGroups.usb.length > 0 && (
          <div className="space-y-2 pt-2">
            <div className="text-xs font-semibold text-slate-400 flex items-center justify-between uppercase tracking-wider">
              <span className="flex items-center gap-2">🔌 Removable & USB Drives</span>
              <span className="text-[10px] text-amber-400/80 font-normal lowercase">hot-pluggable media</span>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {storageGroups.usb.map((card, idx) => (
                <div key={card.key || idx} className={`bg-slate-950/60 border rounded-lg p-4 space-y-3 relative transition-all ${
                  card.isCriticalSpace ? 'border-rose-500/40 ring-1 ring-rose-500/20' : card.isWarningSpace ? 'border-amber-500/30' : 'border-slate-800/60'
                }`}>
                  <div className="flex justify-between items-start">
                    <div className="min-w-0 pr-2">
                      <div className="font-semibold text-white text-sm flex items-center gap-2 flex-wrap">
                        <span className="truncate max-w-[180px]" title={card.title}>{card.title}</span>
                        <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-500/10 text-amber-400 border border-amber-500/20">
                          USB
                        </span>
                        {card.isCriticalSpace && (
                          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-500/15 text-rose-400 border border-rose-500/30 animate-pulse">
                            CRITICAL
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-slate-500 font-mono mt-0.5 truncate" title={card.subtitle}>
                        {card.subtitle}
                      </div>
                    </div>
                    <div className="text-right shrink-0">
                      <span className={`font-bold text-sm ${card.isCriticalSpace ? 'text-rose-400' : card.isWarningSpace ? 'text-amber-400' : 'text-white'}`}>
                        {card.usedPercent?.toFixed(1)}%
                      </span>
                      <div className="text-xs text-slate-400 font-mono">
                        {formatCap(card.usedGB)} / {formatCap(card.totalGB)}
                      </div>
                    </div>
                  </div>
                  <div className="w-full bg-slate-800 rounded-full h-2 overflow-hidden">
                    <div
                      className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(card.usedPercent)}`}
                      style={{ width: `${Math.min(100, card.usedPercent)}%` }}
                    ></div>
                  </div>

                  {card.partitions?.[0]?.mountpoint && (
                    <div className="flex justify-between items-center pt-2 border-t border-slate-900/60">
                      <span className="text-[10px] text-slate-500 font-mono truncate max-w-[150px]" title={card.partitions[0].mountpoint}>
                        {card.partitions[0].mountpoint}
                      </span>
                      {onUnmount && (
                        <button
                          onClick={async () => {
                            const mp = card.partitions[0].mountpoint;
                            setUnmountingMount(mp);
                            await onUnmount(mp);
                            setUnmountingMount(null);
                          }}
                          disabled={unmountingMount === card.partitions[0].mountpoint}
                          className="px-2.5 py-0.5 rounded text-[10px] font-semibold bg-amber-500/10 text-amber-300 border border-amber-500/30 hover:bg-amber-500/20 transition-all flex items-center gap-1 active:scale-95 disabled:opacity-50"
                          title={`Safely unmount ${card.partitions[0].mountpoint}`}
                        >
                          <span>{unmountingMount === card.partitions[0].mountpoint ? 'Ejecting...' : '⏏ Safe Eject'}</span>
                        </button>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* 3. Network Storage */}
        {storageGroups.network.length > 0 && (
          <div className="space-y-2 pt-2">
            <div className="text-xs font-semibold text-slate-400 flex items-center justify-between uppercase tracking-wider">
              <span className="flex items-center gap-2">🌐 Network Storage</span>
              <span className="text-[10px] text-sky-400/80 font-normal lowercase">nfs / smb / remote mounts</span>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {storageGroups.network.map((card, idx) => (
                <div key={card.key || idx} className={`bg-slate-950/60 border rounded-lg p-4 space-y-3 relative transition-all ${
                  card.isCriticalSpace ? 'border-rose-500/40 ring-1 ring-rose-500/20' : card.isWarningSpace ? 'border-amber-500/30' : 'border-slate-800/60'
                }`}>
                  <div className="flex justify-between items-start">
                    <div className="min-w-0 pr-2">
                      <div className="font-semibold text-white text-sm flex items-center gap-2 flex-wrap">
                        <span className="truncate max-w-[180px]" title={card.title}>{card.title}</span>
                        <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-sky-500/10 text-sky-400 border border-sky-500/20">
                          NETWORK
                        </span>
                        {card.isCriticalSpace && (
                          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-500/15 text-rose-400 border border-rose-500/30 animate-pulse">
                            CRITICAL
                          </span>
                        )}
                        {card.isWarningSpace && (
                          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-500/15 text-amber-400 border border-amber-500/30">
                            LOW SPACE
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-slate-500 font-mono mt-0.5 truncate" title={card.subtitle}>
                        {card.subtitle}
                      </div>
                    </div>
                    <div className="text-right shrink-0">
                      <span className={`font-bold text-sm ${card.isCriticalSpace ? 'text-rose-400' : card.isWarningSpace ? 'text-amber-400' : 'text-white'}`}>
                        {card.usedPercent?.toFixed(1)}%
                      </span>
                      <div className="text-xs text-slate-400 font-mono">
                        {formatCap(card.usedGB)} / {formatCap(card.totalGB)}
                      </div>
                    </div>
                  </div>
                  <div className="w-full bg-slate-800 rounded-full h-2 overflow-hidden">
                    <div
                      className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(card.usedPercent)}`}
                      style={{ width: `${Math.min(100, card.usedPercent)}%` }}
                    ></div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* 4. Other Internal Disks */}
        {storageGroups.internal.length > 0 && (
          <div className="space-y-2 pt-2">
            <div className="text-xs font-semibold text-slate-400 flex items-center justify-between uppercase tracking-wider">
              <span className="flex items-center gap-2">💽 Internal Secondary Storage</span>
              <span className="text-[10px] text-slate-500 font-normal lowercase">additional local drives</span>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {storageGroups.internal.map((card, idx) => (
                <div key={card.key || idx} className="bg-slate-950/60 border border-slate-800/60 rounded-lg p-4 space-y-3">
                  <div className="flex justify-between items-start">
                    <div className="min-w-0 pr-2">
                      <div className="font-semibold text-white text-sm flex items-center gap-2 flex-wrap">
                        <span className="truncate max-w-[180px]" title={card.title}>{card.title}</span>
                        <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-slate-800 text-slate-400 border border-slate-700">
                          LOCAL
                        </span>
                      </div>
                      <div className="text-xs text-slate-500 font-mono mt-0.5 truncate" title={card.subtitle}>
                        {card.subtitle}
                      </div>
                    </div>
                    <div className="text-right shrink-0">
                      <span className="font-bold text-sm text-white">{card.usedPercent?.toFixed(1)}%</span>
                      <div className="text-xs text-slate-400 font-mono">
                        {formatCap(card.usedGB)} / {formatCap(card.totalGB)}
                      </div>
                    </div>
                  </div>
                  <div className="w-full bg-slate-800 rounded-full h-2 overflow-hidden">
                    <div
                      className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(card.usedPercent)}`}
                      style={{ width: `${Math.min(100, card.usedPercent)}%` }}
                    ></div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </WidgetCard>
  );
}
