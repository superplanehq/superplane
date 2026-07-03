/**
 * Shared date/time formatting helpers used by the `Timestamp` component and
 * anywhere timestamps are shown to users. Formatting is locale-aware via the
 * platform `Intl` APIs; pass a fixed `locale` only for deterministic tests.
 */

export type TimestampInput = Date | string | number;

/**
 * Coerce the accepted timestamp inputs into a `Date`.
 * Returns `null` when the value is missing or not a valid date.
 */
export function toDate(value: TimestampInput | null | undefined): Date | null {
  if (value === null || value === undefined || value === "") return null;
  const date = value instanceof Date ? value : new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

const ABSOLUTE_OPTIONS: Intl.DateTimeFormatOptions = {
  day: "2-digit",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  hour12: false,
};

/**
 * Locale-aware absolute timestamp in the user's local timezone, including a
 * short timezone name, e.g. `"02 Jun 2026, 12:01:10 CEST"`.
 */
export function formatAbsolute(value: TimestampInput, locale?: string): string {
  const date = toDate(value);
  if (!date) return "";
  return new Intl.DateTimeFormat(locale, { ...ABSOLUTE_OPTIONS, timeZoneName: "short" }).format(date);
}

/**
 * Locale-aware absolute timestamp rendered in UTC (no timezone suffix, since
 * callers label it as UTC), e.g. `"02 Jun 2026, 10:01:10"`.
 */
export function formatUTC(value: TimestampInput, locale?: string): string {
  const date = toDate(value);
  if (!date) return "";
  return new Intl.DateTimeFormat(locale, { ...ABSOLUTE_OPTIONS, timeZone: "UTC" }).format(date);
}

const SECOND = 1000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;
const WEEK = 7 * DAY;
const MONTH = 30 * DAY;
const YEAR = 365 * DAY;

/**
 * Locale-aware relative time from now, e.g. `"1 day ago"` or `"in 3 hours"`.
 * Uses `numeric: "always"` so results read `"1 day ago"` rather than
 * `"yesterday"`, matching the reference design.
 */
export function formatRelative(value: TimestampInput, locale?: string, now: number = Date.now()): string {
  const date = toDate(value);
  if (!date) return "";

  const diff = date.getTime() - now;
  const abs = Math.abs(diff);
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: "always" });

  if (abs < MINUTE) return rtf.format(Math.round(diff / SECOND), "second");
  if (abs < HOUR) return rtf.format(Math.round(diff / MINUTE), "minute");
  if (abs < DAY) return rtf.format(Math.round(diff / HOUR), "hour");
  if (abs < WEEK) return rtf.format(Math.round(diff / DAY), "day");
  if (abs < MONTH) return rtf.format(Math.round(diff / WEEK), "week");
  if (abs < YEAR) return rtf.format(Math.round(diff / MONTH), "month");
  return rtf.format(Math.round(diff / YEAR), "year");
}

/**
 * Full-precision ISO 8601 timestamp (UTC), e.g. `"2026-06-02T10:01:10.561Z"`.
 */
export function formatISO(value: TimestampInput): string {
  const date = toDate(value);
  if (!date) return "";
  return date.toISOString();
}
