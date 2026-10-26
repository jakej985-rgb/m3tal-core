import WidgetCard from './WidgetCard';

/**
 * Slot for dynamically mounting third-party external plugin widgets
 * into the 4-column dashboard grid.
 * 
 * Plugins declare widgets in plugin.yaml:
 * widgets:
 *   - id: "docker-fleet"
 *     title: "Docker Containers"
 *     size: [2, 1] # colSpan: 2, rowSpan: 1
 *     type: "card" | "embed"
 */
export default function PluginWidgetSlot({
  widget,
  onAction,
  isEditMode = false,
  onMoveLeft,
  onMoveRight,
  onHide,
  className = '',
}) {
  if (!widget) return null;

  const colSpan = widget.size?.[0] || widget.colSpan || 1;
  const rowSpan = widget.size?.[1] || widget.rowSpan || 1;

  // If the widget is an embedded iframe (micro-frontend)
  if (widget.type === 'embed' && widget.url) {
    return (
      <WidgetCard
        colSpan={colSpan}
        rowSpan={rowSpan}
        isEditMode={isEditMode}
        onMoveLeft={onMoveLeft}
        onMoveRight={onMoveRight}
        onHide={onHide}
        title={widget.title || 'Plugin Widget'}
        subtitle={widget.plugin_name}
        badge={
          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
            PLUGIN
          </span>
        }
        className={className}
        bodyClassName="p-0 overflow-hidden"
      >
        <iframe
          src={widget.url}
          title={widget.title || 'Plugin Widget'}
          className="w-full h-full min-h-[140px] border-0 rounded-b-xl"
        />
      </WidgetCard>
    );
  }

  // Data-driven standard card widget
  return (
    <WidgetCard
      colSpan={colSpan}
      rowSpan={rowSpan}
      isEditMode={isEditMode}
      onMoveLeft={onMoveLeft}
      onMoveRight={onMoveRight}
      onHide={onHide}
      title={widget.title || 'Plugin Widget'}
      subtitle={widget.subtitle || widget.plugin_name}
      badge={
        <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
          PLUGIN
        </span>
      }
      className={className}
    >
      <div className="space-y-3">
        {widget.value !== undefined && (
          <div className="flex justify-between items-baseline">
            <span className="text-2xl font-bold text-white">{widget.value}</span>
            {widget.status && (
              <span className="text-xs font-semibold text-emerald-400 font-mono">
                {widget.status}
              </span>
            )}
          </div>
        )}

        {widget.description && (
          <p className="text-xs text-slate-400">{widget.description}</p>
        )}

        {Array.isArray(widget.stats) && widget.stats.length > 0 && (
          <div className="grid grid-cols-2 gap-2 pt-2 border-t border-slate-800/60">
            {widget.stats.map((stat, idx) => (
              <div key={idx} className="text-xs">
                <span className="text-slate-500 block text-[11px]">{stat.label}</span>
                <span className="text-slate-200 font-semibold">{stat.value}</span>
              </div>
            ))}
          </div>
        )}

        {widget.action && (
          <div className="pt-2">
            <button
              onClick={() => onAction && onAction(widget.plugin_name, widget.action)}
              className="w-full py-1.5 px-3 rounded text-xs font-semibold bg-indigo-600 hover:bg-indigo-500 text-white transition-all shadow-sm active:scale-95"
            >
              {widget.actionLabel || 'Execute Action'}
            </button>
          </div>
        )}
      </div>
    </WidgetCard>
  );
}
