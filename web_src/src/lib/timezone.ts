/**
 * Timezone utility functions for formatting dates in the user's timezone.
 */

/**
 * Get the user's current timezone
 */
function getUserTimezone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
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
  const options: Intl.DateTimeFormatOptions = {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false, // Use 24-hour format instead of AM/PM
  };

  return date.toLocaleDateString("en-US", options) + ` ${timezone}`;
}

/**
 * Format an optional ISO timestamp for execution/event detail rows; returns "-" when missing or invalid.
 */
export function formatOptionalIsoTimestamp(value?: string): string {
  if (!value) {
    return "-";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }

  return formatTimestampInUserTimezone(date);
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
