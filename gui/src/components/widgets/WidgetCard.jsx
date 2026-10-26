/**
 * Reusable WidgetCard container enforcing the 4-column dashboard grid sizing.
 * 
 * Sizing conventions:
 * - colSpan: 1 (1 column wide, 25% on desktop)
 *            2 (2 columns wide, 50% on desktop)
 *            3 (3 columns wide, 75% on desktop)
 *            4 (4 columns wide, 100% full width)
 * - rowSpan: 1, 2, 3 (height units in grid rows)
 */
export default function WidgetCard({
  colSpan = 1,
  rowSpan = 1,
  title,
  subtitle,
  icon: Icon,
  badge,
  actions,
  isEditMode = false,
  onMoveLeft,
  onMoveRight,
  onHide,
  className = '',
  bodyClassName = '',
  children,
}) {
  // Map colSpan to responsive Tailwind grid column spans
  const colSpanClass = {
    1: 'col-span-1',
    2: 'col-span-1 md:col-span-2',
    3: 'col-span-1 md:col-span-2 lg:col-span-3',
    4: 'col-span-1 md:col-span-2 lg:col-span-4',
  }[colSpan] || 'col-span-1';

  // Map rowSpan to explicit modular grid unit heights:
  // Base 1x1 block unit H = 260px, Gap G = 24px (gap-6)
  // rowSpan 1: H = 260px (1 block down)
  // rowSpan 2: 2 * 260 + 24 = 544px (2 blocks down)
  // rowSpan 3: 3 * 260 + 48 = 828px (3 blocks down)
  const rowSpanClass = {
    1: 'row-span-1 min-h-[260px] lg:h-[260px]',
    2: 'row-span-2 min-h-[544px] lg:h-[544px]',
    3: 'row-span-3 min-h-[828px] lg:h-[828px]',
  }[rowSpan] || 'row-span-1 min-h-[260px] lg:h-[260px]';

  const hasHeader = Boolean(title || subtitle || Icon || badge || actions);

  return (
    <div
      className={`bg-slate-900 border ${
        isEditMode
          ? 'border-amber-500/50 ring-1 ring-amber-500/30'
          : 'border-slate-800/80 hover:border-slate-700/80'
      } rounded-2xl p-5 shadow-sm backdrop-blur-md relative overflow-hidden flex flex-col justify-between transition-all ${colSpanClass} ${rowSpanClass} ${className}`}
    >
      {/* Edit Mode Customization Toolbar */}
      {isEditMode && (
        <div className="bg-amber-500/10 border-b border-amber-500/20 -mx-5 -mt-5 mb-3 px-3 py-1 flex items-center justify-between text-xs text-amber-300 select-none">
          <span className="font-mono text-[10px] font-bold uppercase tracking-wider">
            {colSpan}x{rowSpan} GRID
          </span>
          <div className="flex items-center gap-1">
            {onMoveLeft && (
              <button
                type="button"
                onClick={onMoveLeft}
                className="p-1 rounded hover:bg-amber-500/20 text-amber-200 transition-colors"
                title="Move earlier in grid"
              >
                ◀
              </button>
            )}
            {onMoveRight && (
              <button
                type="button"
                onClick={onMoveRight}
                className="p-1 rounded hover:bg-amber-500/20 text-amber-200 transition-colors"
                title="Move later in grid"
              >
                ▶
              </button>
            )}
            {onHide && (
              <button
                type="button"
                onClick={onHide}
                className="px-1.5 py-0.5 rounded hover:bg-rose-500/20 text-rose-300 transition-colors font-bold text-[11px] ml-1"
                title="Hide this widget"
              >
                ✕ Hide
              </button>
            )}
          </div>
        </div>
      )}
      {hasHeader && (
        <div className="flex items-center justify-between gap-3 mb-4 pb-3 border-b border-slate-800/60 shrink-0">
          <div className="flex items-center gap-2.5 min-w-0">
            {Icon && (
              <div className="p-1.5 rounded-lg bg-slate-800/80 border border-slate-700/60 text-slate-300 shrink-0">
                <Icon className="w-4 h-4 text-emerald-400" />
              </div>
            )}
            <div className="min-w-0">
              {title && (
                <div className="text-sm font-semibold text-white tracking-tight flex items-center gap-2 truncate">
                  <span className="truncate">{title}</span>
                  {badge && <span className="shrink-0">{badge}</span>}
                </div>
              )}
              {subtitle && (
                <div className="text-xs text-slate-400 truncate mt-0.5">{subtitle}</div>
              )}
            </div>
          </div>
          {actions && <div className="flex items-center gap-2 shrink-0">{actions}</div>}
        </div>
      )}

      <div className={`flex-1 min-h-0 flex flex-col justify-between ${bodyClassName}`}>
        {children}
      </div>
    </div>
  );
}
