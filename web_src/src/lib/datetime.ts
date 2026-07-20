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

const DATE_ONLY_OPTIONS: Intl.DateTimeFormatOptions = {
  day: "2-digit",
  month: "short",
  year: "numeric",
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

/**
 * Locale-aware date-only timestamp in the user's local timezone, e.g.
 * `"02 Jun 2026"`. Use for surfaces that only want a calendar day.
 */
export function formatDate(value: TimestampInput, locale?: string): string {
  const date = toDate(value);
  if (!date) return "";
  return new Intl.DateTimeFormat(locale, DATE_ONLY_OPTIONS).format(date);
}

const SECOND = 1000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;
const WEEK = 7 * DAY;
const MONTH = 30 * DAY;
const YEAR = 365 * DAY;
const DEFAULT_LOCALE_KEY = "__default__";
const relativeTimeFormatters = new Map<string, Intl.RelativeTimeFormat>();

function relativeFormatter(locale?: string): Intl.RelativeTimeFormat {
  const key = locale ?? DEFAULT_LOCALE_KEY;
  const cached = relativeTimeFormatters.get(key);
  if (cached) return cached;

  const formatter = new Intl.RelativeTimeFormat(locale, { numeric: "always" });
  relativeTimeFormatters.set(key, formatter);
  return formatter;
}

function formatRelativeUnit(
  diff: number,
  unitSize: number,
  unit: Intl.RelativeTimeFormatUnit,
  formatter: Intl.RelativeTimeFormat,
): string {
  return formatter.format(Math.round(diff / unitSize), unit);
}

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
  const rtf = relativeFormatter(locale);

  if (Math.round(abs / SECOND) < 60) return formatRelativeUnit(diff, SECOND, "second", rtf);
  if (Math.round(abs / MINUTE) < 60) return formatRelativeUnit(diff, MINUTE, "minute", rtf);
  if (Math.round(abs / HOUR) < 24) return formatRelativeUnit(diff, HOUR, "hour", rtf);
  if (Math.round(abs / DAY) < 7) return formatRelativeUnit(diff, DAY, "day", rtf);
  if (Math.round(abs / WEEK) < 4) return formatRelativeUnit(diff, WEEK, "week", rtf);
  if (Math.round(abs / MONTH) < 12) return formatRelativeUnit(diff, MONTH, "month", rtf);
  return formatRelativeUnit(diff, YEAR, "year", rtf);
}

/**
 * Full-precision ISO 8601 timestamp (UTC), e.g. `"2026-06-02T10:01:10.561Z"`.
 */
export function formatISO(value: TimestampInput): string {
  const date = toDate(value);
  if (!date) return "";
  return date.toISOString();
}
