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

  // Map rowSpan
  const rowSpanClass = {
    1: 'row-span-1',
    2: 'row-span-2',
    3: 'row-span-3',
  }[rowSpan] || 'row-span-1';

  const hasHeader = Boolean(title || subtitle || Icon || badge || actions);

  return (
    <div
      className={`bg-slate-900 border border-slate-800/80 rounded-2xl p-5 shadow-sm backdrop-blur-md relative overflow-hidden flex flex-col justify-between transition-all hover:border-slate-700/80 ${colSpanClass} ${rowSpanClass} ${className}`}
    >
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

      <div className={`flex-1 flex flex-col justify-between ${bodyClassName}`}>
        {children}
      </div>
    </div>
  );
}
