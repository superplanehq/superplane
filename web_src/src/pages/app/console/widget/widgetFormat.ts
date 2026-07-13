import { formatTimeAgo } from "@/lib/date";
import { formatAbsolute, formatDate as formatDateOnly } from "@/lib/datetime";

import type { WidgetColumnFormat } from "./types";

/** Pure digit / decimal strings (epoch candidates); excludes ISO and other date text. */
const PURE_NUMERIC_RE = /^-?\d+(\.\d+)?$/;

/**
 * Coerce a widget cell value into a `Date`. Accepts ISO strings, `Date`
 * instances, and plausible epoch seconds/milliseconds — including numeric
 * strings like `"1717390000"` that JSON/CEL often emit (using `>= 1e11` to
 * disambiguate ms vs seconds). Returns `null` for anything that can't be
 * parsed, so callers can fall back to the raw string.
 *
 * Short digit strings / small numbers (`"404"`, `12`) are rejected so status
 * codes and categories never become early-1970 or year-404 dates.
 *
 * Shared by widget formatters and CEL builtins (`formatDate`, `epochMs`) so
 * timestamp parsing stays consistent across the console package.
 */
export function coerceWidgetTimestamp(value: unknown): Date | null {
  if (value == null) return null;
  if (value instanceof Date) {
    return Number.isFinite(value.getTime()) ? value : null;
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (trimmed === "") return null;
    // Skip Date.parse for pure digits — it treats values like "404" as years.
    if (PURE_NUMERIC_RE.test(trimmed)) {
      return dateFromEpochNumber(Number(trimmed));
    }
    const parsed = Date.parse(trimmed);
    if (Number.isFinite(parsed)) return new Date(parsed);
    return null;
  }
  return dateFromEpochNumber(typeof value === "number" ? value : Number(value));
}

/** Milliseconds from ~1973 onward; below this, values are treated as seconds. */
const EPOCH_MS_MAGNITUDE = 1e11;

function dateFromEpochNumber(n: number): Date | null {
  if (!isPlausibleEpochNumber(n)) return null;
  // Use magnitude so negative pre-1970 ms values (e.g. -1.5e12) are not
  // mistaken for seconds and multiplied by 1000.
  const ms = Math.abs(n) >= EPOCH_MS_MAGNITUDE ? n : n * 1000;
  const date = new Date(ms);
  return Number.isFinite(date.getTime()) ? date : null;
}

/**
 * Epoch seconds (~1e9–1e10) or milliseconds (~1e11–1e13). The bands meet at
 * `1e11` so 1973–2001 ms epochs are accepted; status codes / hours stay out.
 */
function isPlausibleEpochNumber(n: number): boolean {
  if (!Number.isFinite(n)) return false;
  const abs = Math.abs(n);
  return abs >= 1e9 && abs < 1e14;
}

/**
 * Resolved progress values for a table cell.
 *
 * - `percent` is the raw ratio in percent points and may be < 0 or > 100 when
 *   the current value under- or overshoots the target. UI code uses this for
 *   the tooltip / percentage label so users still see the real overshoot.
 * - `barPercent` is clamped to `[0, 100]` and drives the bar fill width.
 * - `current` and `target` are the coerced numeric values (never null when
 *   `computeProgress` returns a non-null result).
 */
export interface WidgetProgress {
  current: number;
  target: number;
  percent: number;
  barPercent: number;
}

/**
 * Coerce a raw current/target pair into progress values for a bar cell.
 *
 * Returns `null` when either value can't be coerced to a finite number or the
 * target is <= 0 — the caller renders the empty-state placeholder in that
 * case. Both inputs are treated as absolute numbers; unlike `percent`
 * formatting, `0.5` is not silently promoted to 50%.
 */
export function computeProgress(current: unknown, target: unknown): WidgetProgress | null {
  const currentNum = toFiniteNumber(current);
  const targetNum = toFiniteNumber(target);
  if (currentNum === null || targetNum === null) return null;
  if (targetNum <= 0) return null;
  const percent = (currentNum / targetNum) * 100;
  const barPercent = Math.max(0, Math.min(100, percent));
  return { current: currentNum, target: targetNum, percent, barPercent };
}

/** Format a raw percentage number using the same rounding rules as `formatPercent`. */
export function formatPercentageDisplay(percent: number): string {
  return `${percent.toFixed(percent % 1 === 0 ? 0 : 1)}%`;
}

function toFiniteNumber(value: unknown): number | null {
  if (typeof value === "number") return Number.isFinite(value) ? value : null;
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (trimmed === "") return null;
    const n = Number(trimmed);
    return Number.isFinite(n) ? n : null;
  }
  return null;
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
      // `progress` falls through here — its ProgressCell renders bespoke UI
      // instead of the formatted string, so there is nothing extra to do.
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
  // Compact relative text without an "ago" suffix (e.g. "5m", "in 2h") so
  // dense table cells stay short; the hover details always expose the verbose
  // "5 minutes ago" / "in 3 hours" phrasing.
  return formatTimeAgo(date, false);
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
