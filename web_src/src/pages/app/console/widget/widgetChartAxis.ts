import { coerceWidgetTimestamp, formatValue } from "./widgetFormat";
import type { WidgetChartSeries, WidgetColumnFormat } from "./types";

type SeriesWithFormat = Pick<WidgetChartSeries, "format">;

/**
 * Format an X-axis tick. Timestamp buckets render as a short date only
 * (e.g. `Jul 6`); precise time is shown in the bar tooltip instead.
 */
/** Indices of data rows whose X-axis date label should render (first of each day). */
export function buildXAxisTickShowIndices(
  data: Array<Record<string, unknown>>,
  xFormat: WidgetColumnFormat | undefined,
): Set<number> | null {
  if (!data.some((row) => isTimestampAxisBucket(row.x, xFormat))) return null;
  const show = new Set<number>();
  let lastLabel = "";
  data.forEach((row, index) => {
    const label = formatXAxisTick(row.x, xFormat);
    if (label && label !== lastLabel) {
      show.add(index);
      lastLabel = label;
    }
  });
  return show;
}

export function formatXAxisTick(value: unknown, format: WidgetColumnFormat | undefined): string {
  if (value == null || value === "") return "";
  if (isTimestampAxisBucket(value, format)) {
    const compact = formatDateTimeAxisTick(value, "date");
    if (compact) return compact;
  }
  if (!format) return String(value);
  return formatValue(value, format);
}

/** Tooltip category label — includes time for timestamp buckets. */
export function formatXTooltipLabel(value: unknown, format: WidgetColumnFormat | undefined): string {
  if (value == null || value === "") return "";
  if (format === "date" && isTimestampAxisBucket(value, format)) {
    const dateOnly = formatDateTimeAxisTick(value, "date");
    if (dateOnly) return dateOnly;
  }
  if (coerceWidgetTimestamp(value) != null) {
    return formatDateTimeAxisTick(value, "datetime") ?? String(value);
  }
  return formatXAxisTick(value, format);
}

function isTimestampAxisBucket(value: unknown, format: WidgetColumnFormat | undefined): boolean {
  if (coerceWidgetTimestamp(value) == null) return false;
  return !format || format === "datetime" || format === "date";
}

// Compact axis-only formats (kept locale-fixed so tick strings match
// `Jul 6` / `May 26 4:10 PM` regardless of the viewer's locale — the hover
// details block still reflects the user's local timezone).
function formatDateTimeAxisTick(value: unknown, format: "datetime" | "date"): string | null {
  const date = coerceWidgetTimestamp(value);
  if (!date) return null;
  if (format === "date") {
    return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
  }
  return date.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
  });
}

/**
 * Format a Y-axis tick. Honors the configured `yFormat` (currency, duration,
 * percent, ...); otherwise falls back to a locale-aware numeric default with
 * thousands separators above 1k.
 *
 * Duration ticks use a compact, space-free shape so Recharts does not wrap
 * labels like `18m 32s` onto multiple SVG lines.
 */
export function formatYTick(value: number, format: WidgetColumnFormat | undefined): string {
  if (!Number.isFinite(value)) return String(value);
  if (format === "duration") return formatDurationAxisTick(value);
  if (format) return formatValue(value, format);
  if (Math.abs(value) >= 1000) return value.toLocaleString();
  return String(value);
}

function formatDurationAxisTick(ms: number): string {
  const abs = Math.abs(ms);
  const sign = ms < 0 ? "-" : "";
  if (abs < 1000) return `${sign}${Math.round(abs)}ms`;
  const seconds = abs / 1000;
  if (seconds < 60) {
    const rounded = seconds < 10 ? seconds.toFixed(1) : String(Math.round(seconds));
    return `${sign}${rounded}s`;
  }
  const totalMinutes = Math.floor(seconds / 60);
  const remSec = Math.floor(seconds % 60);
  if (totalMinutes < 60) {
    return remSec > 0 ? `${sign}${totalMinutes}m${remSec}s` : `${sign}${totalMinutes}m`;
  }
  const totalHours = Math.floor(totalMinutes / 60);
  const remMin = totalMinutes % 60;
  if (totalHours < 24) {
    return remMin > 0 ? `${sign}${totalHours}h${remMin}m` : `${sign}${totalHours}h`;
  }
  const days = Math.floor(totalHours / 24);
  const remHours = totalHours % 24;
  return remHours > 0 ? `${sign}${days}d${remHours}h` : `${sign}${days}d`;
}

const CARTESIAN_Y_AXIS_FORMATS = new Set<WidgetColumnFormat>([
  "number",
  "percent",
  "duration",
  "date",
  "datetime",
  "relative",
]);

/** Reuse a single series' value format on the Y axis when authors omit `yFormat`. */
export function resolveCartesianYFormat(
  yFormat: WidgetColumnFormat | undefined,
  series: SeriesWithFormat[],
): WidgetColumnFormat | undefined {
  if (yFormat) return yFormat;
  if (series.length !== 1) return undefined;
  const format = series[0]?.format;
  return format && CARTESIAN_Y_AXIS_FORMATS.has(format) ? format : undefined;
}

/** Size the Y-axis gutter from the longest formatted tick we expect to render. */
export function estimateYAxisWidth(
  data: Array<Record<string, unknown>>,
  seriesKeys: string[],
  yFormat: WidgetColumnFormat | undefined,
  hasYLabel: boolean,
): number {
  let maxValue = 0;
  for (const row of data) {
    for (const key of seriesKeys) {
      const n = Number(row[key]);
      if (Number.isFinite(n)) maxValue = Math.max(maxValue, Math.abs(n));
    }
  }
  const candidates = [0, maxValue, maxValue / 2, maxValue / 4];
  let maxChars = 0;
  for (const value of candidates) {
    if (!Number.isFinite(value)) continue;
    maxChars = Math.max(maxChars, formatYTick(value, yFormat).length);
  }
  const tickWidth = Math.ceil(maxChars * 6.5) + 12;
  return Math.max(hasYLabel ? 56 : 40, tickWidth);
}
