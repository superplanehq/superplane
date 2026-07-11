import { useMemo, type ReactNode } from "react";
import { Hash, Loader2 } from "lucide-react";

import { Timestamp, type TimestampDisplay } from "@/components/Timestamp";
import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import { WidgetEmptyState } from "../WidgetEmptyState";
import { CONSOLE_WIDGET_LABEL_CLASSES } from "../consoleTableStyles";

import { aggregateNumber, aggregateNumberPerSource, applyFilters, combinePartials } from "./widgetData";
import { coerceWidgetTimestamp, formatValue } from "./widgetFormat";
import { getValueAtPath } from "./fieldPath";
import type { WidgetColumnFormat, WidgetNumberRender } from "./types";
import type { MemoryNumberSource, WidgetNumberCombine } from "../panelTypes";

const TIMESTAMP_NUMBER_FORMATS: Record<"date" | "datetime" | "relative", TimestampDisplay> = {
  date: "date",
  datetime: "datetime",
  relative: "relative",
};

function isTimestampNumberFormat(format: WidgetColumnFormat | undefined): format is "date" | "datetime" | "relative" {
  return format === "date" || format === "datetime" || format === "relative";
}

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
          <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-500" />
        </div>
      );
    }
    return (
      <div className="flex h-full items-center justify-center p-4">
        <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-500" />
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

const VALUE_CLASS = "text-4xl font-medium text-slate-900 dark:text-gray-100";

function renderValueNode(value: number, format: WidgetColumnFormat | undefined, formatted: string) {
  if (isTimestampNumberFormat(format)) {
    const date = coerceWidgetTimestamp(value);
    if (date) {
      return (
        <Timestamp
          date={date}
          display={TIMESTAMP_NUMBER_FORMATS[format]}
          relativeStyle="abbreviated"
          includeAgo={false}
          className={VALUE_CLASS}
        />
      );
    }
  }
  return <span className={VALUE_CLASS}>{formatted}</span>;
}

function ValueBlock({
  valueNode,
  prefix,
  suffix,
  suffixClassName,
}: {
  valueNode: ReactNode;
  prefix: string | undefined;
  suffix: string | undefined;
  suffixClassName: string;
}) {
  if (!prefix && !suffix) return <>{valueNode}</>;

  // Suffix needs a small gap from the value. Prefix stays flush (currency-style
  // "R$" / "$") — only introduce flex gap when a suffix is present.
  if (suffix) {
    return (
      <div className="flex items-baseline gap-0.5">
        {prefix ? <span className={VALUE_CLASS}>{prefix}</span> : null}
        {valueNode}
        <span className={suffixClassName}>{suffix}</span>
      </div>
    );
  }

  return (
    <div className="flex items-baseline">
      <span className={VALUE_CLASS}>{prefix}</span>
      {valueNode}
    </div>
  );
}

function NumberDisplay({ render, value, sparkline, variant }: NumberDisplayProps) {
  const format = render.format;
  const formatted = value == null ? null : formatValue(value, format ?? "number");
  const className =
    variant === "inline"
      ? "flex flex-col items-start justify-center gap-1 text-left"
      : "flex h-full flex-col items-start justify-center gap-1 p-4";
  const suffixClassName =
    variant === "inline"
      ? "text-base font-semibold text-slate-900 dark:text-gray-100"
      : "text-xl font-semibold text-slate-900 dark:text-gray-100";
  const hasSparkline = sparkline != null && sparkline.length > 1;
  const valueBlock =
    formatted == null || value == null ? (
      <span className={VALUE_CLASS}>—</span>
    ) : (
      <ValueBlock
        valueNode={renderValueNode(value, format, formatted)}
        prefix={render.prefix}
        suffix={render.suffix}
        suffixClassName={suffixClassName}
      />
    );
  return (
    <div className={className} data-testid="widget-number">
      {render.label ? (
        <span className={CONSOLE_WIDGET_LABEL_CLASSES} data-testid="widget-number-label">
          {render.label}
        </span>
      ) : null}
      {hasSparkline ? (
        <div className={variant === "inline" ? "flex flex-col gap-2" : "flex flex-col gap-3"}>
          {valueBlock}
          <Sparkline values={sparkline} />
        </div>
      ) : (
        valueBlock
      )}
    </div>
  );
}

function Sparkline({ values }: { values: number[] }) {
  const width = 120;
  const height = 28;
  const strokeWidth = 1.5;
  // Inset the plot so round joins / stroke width aren't clipped at the SVG edges.
  const padY = Math.ceil(strokeWidth / 2) + 1;
  const plotTop = padY;
  const plotBottom = height - padY;
  const plotHeight = plotBottom - plotTop;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const stepX = values.length > 1 ? width / (values.length - 1) : 0;
  // Compute the line points and close the path back along the baseline so
  // the SVG renders as a filled area underneath the line.
  const linePoints = values.map((v, i) => {
    const x = i * stepX;
    const y = plotTop + plotHeight - ((v - min) / range) * plotHeight;
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });
  const lineCoords = linePoints.join(" ");
  const firstX = (0).toFixed(1);
  const lastX = ((values.length - 1) * stepX).toFixed(1);
  const baselineY = plotBottom.toFixed(1);
  const areaPath = `M${linePoints[0]} L${linePoints.slice(1).join(" L")} L${lastX},${baselineY} L${firstX},${baselineY} Z`;
  return (
    <svg
      width={width}
      height={height}
      className="block text-sky-500 dark:text-indigo-400"
      viewBox={`0 0 ${width} ${height}`}
      aria-hidden
    >
      <path d={areaPath} fill="currentColor" fillOpacity={0.2} stroke="none" />
      <polyline
        points={lineCoords}
        fill="none"
        stroke="currentColor"
        strokeWidth={strokeWidth}
        strokeLinejoin="round"
        strokeLinecap="round"
      />
    </svg>
  );
}
