import { ArrowDownRight, ArrowUpRight, Minus, type LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import { factoryCardClassName } from "@/pages/factories/pages/factoryPageLayoutStyles";
import { Timestamp } from "@/components/Timestamp";

import type { OccurrenceTrend, RecurringSuggestionRow } from "./types";
import { QUALITY_DOMAIN_LABELS } from "./types";

const TREND_META: Record<OccurrenceTrend, { label: string; className: string; icon: LucideIcon }> = {
  up: { label: "Rising", className: "text-red-600 dark:text-red-400", icon: ArrowUpRight },
  down: { label: "Falling", className: "text-emerald-600 dark:text-emerald-400", icon: ArrowDownRight },
  flat: { label: "Flat", className: "text-slate-500 dark:text-gray-400", icon: Minus },
};

const GRID_COLUMNS = "grid-cols-[minmax(0,2fr)_minmax(0,1.4fr)_minmax(0,1fr)_88px_96px_minmax(0,1fr)]";

interface RecurringSuggestionsTableProps {
  rows: RecurringSuggestionRow[];
  onOpenPattern: (rowId: string) => void;
}

/**
 * Factory-level aggregation of repeated suggestions: same rule, similar
 * locations, across runs and work orders. Rows link to the pattern card on
 * the Health tab.
 */
export function RecurringSuggestionsTable({ rows, onOpenPattern }: RecurringSuggestionsTableProps) {
  return (
    <section className={cn(factoryCardClassName, "flex flex-col")} aria-label="Recurring suggestions">
      <header className="border-b border-border px-4 py-3">
        <h3 className="workspace-section-title text-foreground">Recurring suggestions</h3>
        <p className="text-[12px] text-muted-foreground">
          Findings that repeat across work orders. Open a pattern to see the standard fix.
        </p>
      </header>

      {rows.length === 0 ? (
        <div className="px-4 py-10 text-center">
          <p className="workspace-body-text text-muted-foreground">No recurring suggestions yet.</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <div
            className={cn(
              "grid min-w-[720px] gap-x-4 border-b border-border px-4 py-2 text-[11px] font-medium uppercase tracking-wide text-muted-foreground",
              GRID_COLUMNS,
            )}
            role="row"
          >
            <span>Pattern</span>
            <span>Rule</span>
            <span>Domain</span>
            <span className="text-right">Count</span>
            <span>Trend</span>
            <span>Last seen</span>
          </div>
          <ul className="divide-y divide-border">
            {rows.map((row) => (
              <li key={row.id}>
                <button
                  type="button"
                  onClick={() => onOpenPattern(row.id)}
                  className={cn(
                    "grid w-full min-w-[720px] items-center gap-x-4 px-4 py-2.5 text-left text-[13px] hover:bg-slate-50 dark:hover:bg-gray-900",
                    GRID_COLUMNS,
                  )}
                >
                  <span className="truncate font-medium text-foreground">{row.patternName}</span>
                  <span className="truncate text-muted-foreground">{row.ruleName}</span>
                  <span className="truncate text-muted-foreground">{QUALITY_DOMAIN_LABELS[row.domain]}</span>
                  <span className="text-right font-medium tabular-nums text-foreground">{row.count}</span>
                  <TrendCell trend={row.trend} />
                  <span className="text-muted-foreground">
                    <Timestamp date={row.lastSeenAt} display="relative" />
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}

function TrendCell({ trend }: { trend: OccurrenceTrend }) {
  const meta = TREND_META[trend];
  const TrendIcon = meta.icon;
  return (
    <span className={cn("inline-flex items-center gap-1 font-medium", meta.className)}>
      <TrendIcon className="size-3.5" aria-hidden />
      {meta.label}
    </span>
  );
}
