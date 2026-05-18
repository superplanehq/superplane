import { useMemo } from "react";
import { Loader2 } from "lucide-react";

import { applyFilters, aggregateNumber } from "./widgetData";
import { formatValue } from "./widgetFormat";
import { getValueAtPath } from "./fieldPath";
import type { WidgetNumberRender } from "./types";

interface WidgetNumberProps {
  render: WidgetNumberRender;
  rows: unknown[];
  isLoading: boolean;
  totalCount?: number;
}

export function WidgetNumber({ render, rows, isLoading, totalCount }: WidgetNumberProps) {
  const filtered = useMemo(() => applyFilters(rows, render.filters), [rows, render.filters]);
  const value = useMemo(() => {
    if (render.aggregation === "count" && !render.filters?.length && totalCount !== undefined) {
      return totalCount;
    }
    return aggregateNumber(filtered, render.aggregation, render.field);
  }, [filtered, render.aggregation, render.field, render.filters, totalCount]);
  const sparkline = useMemo(() => {
    if (!render.sparklineField) return null;
    return filtered
      .map((row) => {
        const raw = getValueAtPath(row, render.sparklineField!);
        const n = typeof raw === "number" ? raw : Number(raw);
        return Number.isFinite(n) ? n : null;
      })
      .filter((n): n is number => n != null);
  }, [filtered, render.sparklineField]);

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <Loader2 className="size-4 animate-spin text-slate-400" />
      </div>
    );
  }

  const display = value == null ? "—" : formatValue(value, render.format ?? "number");
  return (
    <div className="flex h-full flex-col items-start justify-center gap-1 p-4" data-testid="widget-number">
      {render.label ? (
        <span className="text-xs font-medium uppercase tracking-wide text-slate-500">{render.label}</span>
      ) : null}
      <span className="text-2xl font-semibold text-slate-900">{display}</span>
      {sparkline && sparkline.length > 1 ? <Sparkline values={sparkline} /> : null}
    </div>
  );
}

function Sparkline({ values }: { values: number[] }) {
  const width = 120;
  const height = 28;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const stepX = values.length > 1 ? width / (values.length - 1) : 0;
  const points = values
    .map((v, i) => {
      const x = i * stepX;
      const y = height - ((v - min) / range) * height;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");
  return (
    <svg width={width} height={height} className="text-sky-500" viewBox={`0 0 ${width} ${height}`} aria-hidden>
      <polyline points={points} fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinejoin="round" />
    </svg>
  );
}
