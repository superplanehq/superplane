/**
 * Timezone utility functions for formatting dates in the user's timezone.
 */

/**
 * Get the user's current timezone
 */
function getUserTimezone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
}

const DATE_TIME_FORMAT: Intl.DateTimeFormatOptions = {
  month: "short",
  day: "numeric",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false, // Use 24-hour format instead of AM/PM
};

/**
 * Intl accepts IANA names ("America/New_York") and offsets in "+HH:MM" form, but
 * not the "GMT+2" label callers use for schedule timezones, so normalize those.
 */
function toIntlTimezone(timezone: string): string {
  const offset = /^(?:GMT|UTC)([+-])(\d{1,2})(?::?(\d{2}))?$/i.exec(timezone.trim());
  if (!offset) {
    return timezone;
  }

  const [, sign, hours, minutes = "00"] = offset;
  return `${sign}${hours.padStart(2, "0")}:${minutes}`;
}

/**
 * Format a timestamp to display in user's local timezone
 * @param timestamp - ISO timestamp string or Date object
 * @param userTimezone - Optional user timezone, defaults to browser timezone
 * @returns Formatted datetime string
 */
export function formatTimestampInUserTimezone(timestamp: string | Date, userTimezone?: string): string {
  const timezone = userTimezone || getUserTimezone();
  const date = typeof timestamp === "string" ? new Date(timestamp) : timestamp;

  let formatted: string;
  try {
    formatted = date.toLocaleDateString("en-US", { ...DATE_TIME_FORMAT, timeZone: toIntlTimezone(timezone) });
  } catch {
    // Unrecognized timezone identifier — fall back to the browser's timezone
    // rather than dropping the timestamp entirely.
    formatted = date.toLocaleDateString("en-US", DATE_TIME_FORMAT);
  }

  return `${formatted} ${timezone}`;
}

/**
 * Format a date string as relative time from now
 * @param dateString - ISO date string or undefined
 * @param abbreviated - Whether to use abbreviated format (e.g., "5m ago" vs "5 minutes ago")
 * @returns Formatted relative time string or 'N/A' if dateString is undefined
 */
export function formatRelativeTime(dateString: string | undefined, abbreviated?: boolean): string {
  if (!dateString) return "N/A";

  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();

  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (abbreviated) {
    if (Math.abs(diffSeconds) < 60) {
      return `${Math.abs(diffSeconds)}s ago`;
    } else if (Math.abs(diffMinutes) < 60) {
      return `${Math.abs(diffMinutes)}m ago`;
    } else if (Math.abs(diffHours) < 24) {
      return `${Math.abs(diffHours)}h ago`;
    } else {
      return `${Math.abs(diffDays)}d ago`;
    }
  } else {
    const rtf = new Intl.RelativeTimeFormat("en", { numeric: "auto" });

    if (Math.abs(diffSeconds) < 60) {
      return rtf.format(-diffSeconds, "second");
    } else if (Math.abs(diffMinutes) < 60) {
      return rtf.format(-diffMinutes, "minute");
    } else if (Math.abs(diffHours) < 24) {
      return rtf.format(-diffHours, "hour");
    } else {
      return rtf.format(-diffDays, "day");
    }
  }
}
