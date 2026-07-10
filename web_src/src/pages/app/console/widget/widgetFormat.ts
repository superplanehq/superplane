import { formatTimeAgo } from "@/lib/date";
import { formatAbsolute, formatDate as formatDateOnly } from "@/lib/datetime";

import type { WidgetColumnFormat } from "./types";

/**
 * Coerce a widget cell value into a `Date`. Accepts ISO strings, `Date`
 * instances, and epoch seconds/milliseconds (using the existing `> 1e12`
 * heuristic to disambiguate). Returns `null` for anything that can't be
 * parsed, so callers can fall back to the raw string.
 */
export function coerceWidgetTimestamp(value: unknown): Date | null {
  if (value == null) return null;
  if (value instanceof Date) {
    return Number.isNaN(value.getTime()) ? null : value;
  }
  if (typeof value === "string") {
    if (value.trim() === "") return null;
    const parsed = Date.parse(value);
    if (Number.isFinite(parsed)) return new Date(parsed);
    return null;
  }
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return null;
  const ms = n > 1e12 ? n : n * 1000;
  return new Date(ms);
}

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
      return formatDate(value);
    case "datetime":
      return formatDatetime(value);
    case "relative":
      return formatRelative(value);
    case "duration":
      return formatDuration(value);
    case "status":
    case "badge":
      return String(value).toLowerCase();
    case "code":
    case "text":
    case "link":
    case "avatar":
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

function formatRelative(value: unknown): string {
  const date = coerceWidgetTimestamp(value);
  if (!date) return String(value ?? "");
  // Compact, live-updating relative text with an "ago" suffix so the label
  // matches the runs sidebar; the hover details always expose the verbose
  // "5 minutes ago" phrasing.
  return formatTimeAgo(date, true);
}

function formatDate(value: unknown): string {
  const date = coerceWidgetTimestamp(value);
  if (!date) return String(value);
  return formatDateOnly(date);
}

function formatDatetime(value: unknown): string {
  const date = coerceWidgetTimestamp(value);
  if (!date) return String(value);
  return formatAbsolute(date);
}

/**
 * Format a numeric duration. The input is **always interpreted as
 * milliseconds** so the heuristic that used to silently switch units based
 * on magnitude can't mis-classify small ms values (e.g. 4527 ms used to be
 * read as seconds and printed as `1h 15m`).
 *
 * Use CEL to convert other units before passing them in, e.g.
 * `{{ seconds * 1000 }}` or `{{ minutes * 60000 }}`. Negative values are
 * formatted with a leading `-`.
 *
 * Output rules:
 * - `< 1000 ms`        → `547ms`
 * - `< 60 s`           → `4.5s` (one decimal under 10s, integer otherwise)
 * - `< 60 min`         → `1m 23s`
 * - `< 24 h`           → `2h 5m`
 * - `>= 24 h`          → `3d 4h`
 */
function formatDuration(value: unknown): string {
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return String(value);
  if (n === 0) return "0ms";
  const sign = n < 0 ? "-" : "";
  const ms = Math.abs(n);
  return `${sign}${formatPositiveDurationMs(ms)}`;
}

function formatPositiveDurationMs(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`;
  const seconds = ms / 1000;
  if (seconds < 60) return `${seconds.toFixed(seconds < 10 ? 1 : 0)}s`;
  const totalMinutes = Math.floor(seconds / 60);
  const remSec = Math.floor(seconds % 60);
  if (totalMinutes < 60) return `${totalMinutes}m ${remSec}s`;
  const totalHours = Math.floor(totalMinutes / 60);
  const remMin = totalMinutes % 60;
  if (totalHours < 24) return `${totalHours}h ${remMin}m`;
  const days = Math.floor(totalHours / 24);
  const remHours = totalHours % 24;
  return `${days}d ${remHours}h`;
}
