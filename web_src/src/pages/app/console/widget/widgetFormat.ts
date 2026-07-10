import { formatRelativeTime } from "@/lib/timezone";

import type { WidgetColumnFormat } from "./types";

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
      return formatDate(value, false);
    case "datetime":
      return formatDate(value, true);
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
  const format = (iso: string) => formatRelativeTime(iso, true).replace(" ago", "");
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Date.parse(value);
    if (Number.isFinite(parsed)) {
      return format(new Date(parsed).toISOString());
    }
  }
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return String(value ?? "");
  const ms = n > 1e12 ? n : n * 1000;
  return format(new Date(ms).toISOString());
}

function formatDate(value: unknown, includeTime: boolean): string {
  if (typeof value !== "string" && typeof value !== "number") return String(value);
  const ms = typeof value === "number" ? value : Date.parse(value);
  if (!Number.isFinite(ms)) return String(value);
  const date = new Date(ms);
  if (includeTime) return date.toLocaleString();
  return date.toLocaleDateString();
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
