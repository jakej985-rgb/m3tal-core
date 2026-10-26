import WidgetCard from './WidgetCard';

export default function EventLogWidget({
  events = [],
  onClear,
  className = '',
}) {
  const headerActions = (
    <div className="flex items-center gap-3">
      {onClear && events.length > 0 && (
        <button
          onClick={onClear}
          className="text-[11px] font-semibold text-slate-400 hover:text-white px-2 py-0.5 rounded bg-slate-800 border border-slate-700 hover:bg-slate-750 transition-colors"
        >
          Clear
        </button>
      )}
      <span className="text-xs text-slate-500 font-mono">
        {events.length} {events.length === 1 ? 'event' : 'events'} recorded
      </span>
    </div>
  );

  return (
    <WidgetCard
      colSpan={4}
      rowSpan={2}
      title="Live System Event Log"
      subtitle="Streaming real-time daemon, service state transitions, and audit records"
      badge={
        <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-500/10 text-amber-400 border border-amber-500/20">
          STREAMING (WS)
        </span>
      }
      actions={headerActions}
      className={`min-h-[340px] flex flex-col ${className}`}
      bodyClassName="h-[280px]"
    >
      <div className="h-full overflow-y-auto space-y-2 pr-2 text-xs font-mono">
        {events.length === 0 ? (
          <div className="flex h-full items-center justify-center text-slate-600 italic">
            Waiting for system events (service states, system reconciliations, hardware alerts)...
          </div>
        ) : (
          events.map((ev, i) => (
            <div
              key={i}
              className="flex gap-4 p-2.5 bg-slate-950/40 rounded border border-slate-800/40 hover:bg-slate-800/20 transition-colors"
            >
              <span className="text-slate-500 whitespace-nowrap">{ev.time}</span>
              <span className="text-amber-400 font-semibold whitespace-nowrap">{ev.type}</span>
              <span className="text-slate-300 break-all">{ev.detail}</span>
            </div>
          ))
        )}
      </div>
    </WidgetCard>
  );
}
