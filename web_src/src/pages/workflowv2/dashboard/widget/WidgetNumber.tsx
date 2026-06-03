import { useMemo } from "react";
import { Hash, Loader2 } from "lucide-react";

import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import { WidgetEmptyState } from "../WidgetEmptyState";

import { aggregateNumber, aggregateNumberPerSource, applyFilters, combinePartials } from "./widgetData";
import { formatValue } from "./widgetFormat";
import { getValueAtPath } from "./fieldPath";
import type { WidgetNumberRender } from "./types";
import type { MemoryNumberSource, WidgetNumberCombine } from "../panelTypes";

/**
 * Layout variant for the rendered number block.
 * - `panel` (default): fills the panel body with vertical centering and outer padding.
 *   Used by single-value and composite number panels.
 * - `inline`: drops the `h-full` and outer padding so multiple instances can sit
 *   side-by-side inside a flex-wrap row in a multi-number panel.
 */
export type WidgetNumberVariant = "panel" | "inline";

interface WidgetNumberProps {
  render: WidgetNumberRender;
  rows: unknown[];
  isLoading: boolean;
  totalCount?: number;
  /** Composite memory mode: aggregate each source independently and combine the partials. */
  composite?: {
    entries: CanvasMemoryEntry[];
    sources: MemoryNumberSource[];
    combine: WidgetNumberCombine;
  };
  variant?: WidgetNumberVariant;
}

export function WidgetNumber({ render, rows, isLoading, totalCount, composite, variant = "panel" }: WidgetNumberProps) {
  const filtered = useMemo(() => applyFilters(rows, render.filters), [rows, render.filters]);
  const value = useMemo(() => {
    if (composite) {
      const partials = composite.sources.map((source) =>
        aggregateNumberPerSource(composite.entries, source, render.filters),
      );
      return combinePartials(partials, composite.combine);
    }
    if (render.aggregation === "count" && !render.filters?.length && totalCount !== undefined) {
      return totalCount;
    }
    if (!render.aggregation) return null;
    return aggregateNumber(filtered, render.aggregation, render.field);
  }, [composite, filtered, render.aggregation, render.field, render.filters, totalCount]);
  const sparkline = useMemo(() => {
    if (!render.sparklineField || composite) return null;
    return filtered
      .map((row) => {
        const raw = getValueAtPath(row, render.sparklineField!);
        const n = typeof raw === "number" ? raw : Number(raw);
        return Number.isFinite(n) ? n : null;
      })
      .filter((n): n is number => n != null);
  }, [composite, filtered, render.sparklineField]);

  if (isLoading) {
    if (variant === "inline") {
      return (
        <div className="flex items-center justify-center py-1">
          <Loader2 className="size-4 animate-spin text-slate-400" />
        </div>
      );
    }
    return (
      <div className="flex h-full items-center justify-center p-4">
        <Loader2 className="size-4 animate-spin text-slate-400" />
      </div>
    );
  }

  if (value == null && variant === "panel") {
    return <WidgetEmptyState icon={Hash} message="No data to display." testId="widget-number-empty" />;
  }

  return <NumberDisplay render={render} value={value} sparkline={sparkline} variant={variant} />;
}

interface NumberDisplayProps {
  render: WidgetNumberRender;
  value: number | null;
  sparkline: number[] | null;
  variant: WidgetNumberVariant;
}

function NumberDisplay({ render, value, sparkline, variant }: NumberDisplayProps) {
  const display =
    value == null
      ? "—"
      : `${render.prefix ?? ""}${formatValue(value, render.format ?? "number")}${render.suffix ?? ""}`;
  const className =
    variant === "inline"
      ? "flex flex-col items-start justify-center gap-1 text-left"
      : "flex h-full flex-col items-start justify-center gap-1 p-4";
  return (
    <div className={className} data-testid="widget-number">
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
  // Compute the line points and close the path back along the baseline so
  // the SVG renders as a filled area underneath the line.
  const linePoints = values.map((v, i) => {
    const x = i * stepX;
    const y = height - ((v - min) / range) * height;
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });
  const lineCoords = linePoints.join(" ");
  const firstX = (0).toFixed(1);
  const lastX = ((values.length - 1) * stepX).toFixed(1);
  const baselineY = height.toFixed(1);
  const areaPath = `M${linePoints[0]} L${linePoints.slice(1).join(" L")} L${lastX},${baselineY} L${firstX},${baselineY} Z`;
  return (
    <svg width={width} height={height} className="text-sky-500" viewBox={`0 0 ${width} ${height}`} aria-hidden>
      <path d={areaPath} fill="currentColor" fillOpacity={0.2} stroke="none" />
      <polyline points={lineCoords} fill="none" stroke="currentColor" strokeWidth={1.5} strokeLinejoin="round" />
    </svg>
  );
}
