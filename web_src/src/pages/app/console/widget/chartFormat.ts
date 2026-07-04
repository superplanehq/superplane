import { formatValue } from "./widgetFormat";
import type { WidgetColumnFormat } from "./types";

export interface ChartValueFormat {
  format?: WidgetColumnFormat;
  prefix?: string;
  suffix?: string;
}

/**
 * Format a numeric chart value with optional prefix/suffix wrapping. Mirrors
 * the number widget's display rules so chart tooltips can show currency, units,
 * or percent values exactly the way KPI cards do.
 */
export function formatSeriesValue(value: unknown, { format, prefix, suffix }: ChartValueFormat): string {
  if (value == null) return "—";
  return `${prefix ?? ""}${formatValue(value, format ?? "number")}${suffix ?? ""}`;
}

/**
 * Render a percentage suffix for donut tooltips (e.g. " (32%)"). Returns an
 * empty string when total is zero or the value can't be expressed as a number.
 */
export function formatPercentOfTotal(value: unknown, total: number): string {
  if (!Number.isFinite(total) || total <= 0) return "";
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return "";
  const pct = (n / total) * 100;
  return ` (${pct.toFixed(pct % 1 === 0 ? 0 : 1)}%)`;
}
