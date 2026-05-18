import type { WidgetColumnFormat } from "./types";

/**
 * Format a raw value for display, consistent with the format hints declared in
 * the widget YAML. Falls back to the value's string representation when the
 * format doesn't fit (e.g. asking for `date` on a non-string value).
 */
export function formatValue(value: unknown, format: WidgetColumnFormat | undefined): string {
  if (value == null) return "";
  switch (format) {
    case "number":
      return formatNumber(value);
    case "percent":
      return formatPercent(value);
    case "date":
      return formatDate(value, false);
    case "datetime":
      return formatDate(value, true);
    case "duration":
      return formatDuration(value);
    case "status":
      return String(value).toLowerCase();
    case "code":
    case "text":
    case "link":
    case undefined:
      return String(value);
    default:
      return String(value);
  }
}

function formatNumber(value: unknown): string {
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return String(value);
  return n.toLocaleString();
}

function formatPercent(value: unknown): string {
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return String(value);
  // Values between 0 and 1 are treated as fractions; otherwise displayed as-is.
  const scaled = n > 0 && n <= 1 ? n * 100 : n;
  return `${scaled.toFixed(scaled % 1 === 0 ? 0 : 1)}%`;
}

function formatDate(value: unknown, includeTime: boolean): string {
  if (typeof value !== "string" && typeof value !== "number") return String(value);
  const ms = typeof value === "number" ? value : Date.parse(value);
  if (!Number.isFinite(ms)) return String(value);
  const date = new Date(ms);
  if (includeTime) return date.toLocaleString();
  return date.toLocaleDateString();
}

function formatDuration(value: unknown): string {
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return String(value);
  // Heuristic: treat very large numbers (>10000) as milliseconds, others as seconds.
  const seconds = n > 10000 ? n / 1000 : n;
  if (seconds < 60) return `${seconds.toFixed(seconds < 10 ? 1 : 0)}s`;
  const minutes = Math.floor(seconds / 60);
  const rem = Math.floor(seconds % 60);
  if (minutes < 60) return `${minutes}m ${rem}s`;
  const hours = Math.floor(minutes / 60);
  const remMin = minutes % 60;
  return `${hours}h ${remMin}m`;
}
