import WidgetCard from './WidgetCard';

export default function StatCardWidget({
  type = 'cpu', // 'cpu' | 'ram' | 'gpu' | 'disk'
  metrics,
  history = [],
  className = '',
}) {
  if (!metrics) return null;

  const getUsageColor = (val) => {
    if (val >= 85) return 'bg-rose-500';
    if (val >= 60) return 'bg-amber-500';
    return 'bg-teal-400';
  };

  const getUsageGlow = (val) => {
    if (val >= 85) return 'shadow-[0_0_12px_rgba(244,63,94,0.3)]';
    if (val >= 60) return 'shadow-[0_0_12px_rgba(245,158,11,0.3)]';
    return 'shadow-[0_0_12px_rgba(45,212,191,0.3)]';
  };

  // Render Sparkline Component
  const renderSparkline = (metricKey, label) => (
    <div className="mt-5">
      <span className="text-[10px] uppercase font-bold text-slate-500 tracking-wider">
        {label} (5m)
      </span>
      <div className="flex items-end justify-between h-14 gap-[2px] mt-2 bg-slate-950/40 p-2 rounded-lg border border-slate-800/40">
        {history.length === 0 ? (
          <span className="text-slate-600 text-[10px] w-full text-center">Awaiting data...</span>
        ) : (
          history.map((h, i) => {
            const val = h[metricKey] || 0;
            return (
              <div
                key={i}
                className={`flex-1 rounded-t-sm transition-all duration-300 ${getUsageColor(val)}`}
                style={{ height: `${Math.max(4, Math.min(100, val))}%` }}
                title={`${label}: ${val.toFixed(1)}%`}
              ></div>
            );
          })
        )}
      </div>
    </div>
  );

  switch (type) {
    case 'cpu': {
      const cpuVal = metrics.cpu_usage || 0;
      return (
        <WidgetCard
          colSpan={1}
          rowSpan={1}
          className={className}
        >
          <div>
            <div className="flex justify-between items-center mb-1">
              <span className="text-slate-400 font-semibold text-sm">Processor (CPU)</span>
              <span className="text-lg font-bold text-white">{cpuVal.toFixed(1)}%</span>
            </div>
            {metrics.cpu_temp > 0 ? (
              <div className="text-xs text-amber-400/90 mb-3 font-mono">
                🌡️ {metrics.cpu_temp.toFixed(1)}°C ({(metrics.cpu_temp * 1.8 + 32).toFixed(1)}°F)
              </div>
            ) : (
              <div className="text-xs text-slate-500 mb-3 font-mono">Multi-core Compute</div>
            )}
            <div className="w-full bg-slate-800 rounded-full h-2 mt-2">
              <div
                className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(cpuVal)} ${getUsageGlow(cpuVal)}`}
                style={{ width: `${Math.min(100, Math.max(0, cpuVal))}%` }}
              ></div>
            </div>
          </div>
          {renderSparkline('cpu_usage', 'CPU History')}
        </WidgetCard>
      );
    }

    case 'ram': {
      const ramVal = metrics.memory_usage || 0;
      return (
        <WidgetCard
          colSpan={1}
          rowSpan={1}
          className={className}
        >
          <div>
            <div className="flex justify-between items-center mb-1">
              <span className="text-slate-400 font-semibold text-sm">Memory (RAM)</span>
              <span className="text-lg font-bold text-white">{ramVal.toFixed(1)}%</span>
            </div>
            <div className="text-xs text-slate-400 mb-3 font-mono">
              {metrics.memory_used?.toFixed(1) || 0} / {metrics.memory_total?.toFixed(1) || 0} GB
              {metrics.memory_frequency && metrics.memory_frequency !== 'Unknown' && ` (${metrics.memory_frequency})`}
            </div>
            <div className="w-full bg-slate-800 rounded-full h-2 mt-2">
              <div
                className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(ramVal)} ${getUsageGlow(ramVal)}`}
                style={{ width: `${Math.min(100, Math.max(0, ramVal))}%` }}
              ></div>
            </div>
          </div>
          {renderSparkline('memory_usage', 'RAM History')}
        </WidgetCard>
      );
    }

    case 'gpu': {
      const gpuVal = metrics.gpu_usage || 0;
      return (
        <WidgetCard
          colSpan={1}
          rowSpan={1}
          className={className}
        >
          <div>
            <div className="flex justify-between items-center mb-1">
              <span className="text-slate-400 font-semibold text-sm truncate max-w-[170px]" title={metrics.gpu_model}>
                {metrics.gpu_model && metrics.gpu_model !== 'No GPU Detected' ? metrics.gpu_model : 'Graphics (GPU)'}
              </span>
              <span className="text-lg font-bold text-white">{gpuVal.toFixed(1)}%</span>
            </div>
            {metrics.gpu_temp > 0 ? (
              <div className="text-xs text-amber-400/90 mb-3 font-mono">
                🌡️ {metrics.gpu_temp.toFixed(1)}°C ({(metrics.gpu_temp * 1.8 + 32).toFixed(1)}°F)
              </div>
            ) : (
              <div className="text-xs text-slate-500 mb-3 font-mono">Graphics Processor</div>
            )}
            <div className="w-full bg-slate-800 rounded-full h-2 mt-2">
              <div
                className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(gpuVal)} ${getUsageGlow(gpuVal)}`}
                style={{ width: `${Math.min(100, Math.max(0, gpuVal))}%` }}
              ></div>
            </div>
          </div>

          {/* VRAM Gauge */}
          <div className="mt-5 bg-slate-950/40 p-3 rounded-lg border border-slate-800/40">
            <div className="flex justify-between text-xs text-slate-400 mb-1.5 font-mono">
              <span>VRAM</span>
              <span>
                {metrics.gpu_mem_used?.toFixed(0) || 0} / {metrics.gpu_mem_total?.toFixed(0) || 0} MB
              </span>
            </div>
            <div className="w-full bg-slate-800 rounded-full h-1.5">
              <div
                className="h-1.5 rounded-full bg-indigo-400 transition-all duration-500"
                style={{
                  width: `${Math.min(100, metrics.gpu_mem_total > 0 ? ((metrics.gpu_mem_used / metrics.gpu_mem_total) * 100) : 0)}%`,
                }}
              ></div>
            </div>
          </div>
        </WidgetCard>
      );
    }

    case 'disk': {
      const diskVal = metrics.disk_usage || 0;
      return (
        <WidgetCard
          colSpan={1}
          rowSpan={1}
          className={className}
        >
          <div>
            <div className="flex justify-between items-center mb-1">
              <span className="text-slate-400 font-semibold text-sm">System Storage (/)</span>
              <span className="text-lg font-bold text-white">{diskVal.toFixed(1)}%</span>
            </div>
            <div className="text-xs text-slate-400 mb-3 font-mono">
              {metrics.disk_used?.toFixed(1) || 0} / {metrics.disk_total?.toFixed(1) || 0} GB
            </div>
            <div className="w-full bg-slate-800 rounded-full h-2 mt-2">
              <div
                className={`h-2 rounded-full transition-all duration-500 ${getUsageColor(diskVal)} ${getUsageGlow(diskVal)}`}
                style={{ width: `${Math.min(100, Math.max(0, diskVal))}%` }}
              ></div>
            </div>
          </div>
          {renderSparkline('disk_usage', 'Disk History')}
        </WidgetCard>
      );
    }

    default:
      return null;
  }
}
