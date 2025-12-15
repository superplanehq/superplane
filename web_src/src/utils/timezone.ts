/**
 * Timezone utility functions for handling time conversion between user's local timezone and UTC
 */

/**
 * Get the user's current timezone
 */
export function getUserTimezone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
}

/**
 * Convert a time string from user's local timezone to UTC
 * @param timeString - Time string in HH:MM format (e.g., "14:30")
 * @param userTimezone - Optional user timezone, defaults to browser timezone
 * @returns Time string in HH:MM format in UTC
 */
export function convertLocalTimeToUTC(timeString: string, userTimezone?: string): string {
  const timezone = userTimezone || getUserTimezone();

  // Create a date object with today's date and the given time in the user's timezone
  const today = new Date();
  const [hours, minutes] = timeString.split(":").map(Number);

  // Create a date in the user's timezone
  const localDate = new Date(today.getFullYear(), today.getMonth(), today.getDate(), hours, minutes);

  // Get the timezone offset for the user's timezone
  const tempDate = new Date(localDate.toLocaleString("en-US", { timeZone: timezone }));
  const offset = localDate.getTime() - tempDate.getTime();

  // Adjust the date to UTC
  const utcDate = new Date(localDate.getTime() - offset);

  const utcHours = utcDate.getUTCHours().toString().padStart(2, "0");
  const utcMinutes = utcDate.getUTCMinutes().toString().padStart(2, "0");

  return `${utcHours}:${utcMinutes}`;
}

/**
 * Convert a time string from UTC to user's local timezone
 * @param utcTimeString - Time string in HH:MM format in UTC (e.g., "14:30")
 * @param userTimezone - Optional user timezone, defaults to browser timezone
 * @returns Time string in HH:MM format in user's local timezone
 */
export function convertUTCToLocalTime(utcTimeString: string, userTimezone?: string): string {
  const timezone = userTimezone || getUserTimezone();

  // Create a date object with today's date and the given UTC time
  const today = new Date();
  const [hours, minutes] = utcTimeString.split(":").map(Number);

  // Create a UTC date
  const utcDate = new Date(Date.UTC(today.getFullYear(), today.getMonth(), today.getDate(), hours, minutes));

  // Convert to user's timezone
  const localTime = utcDate.toLocaleString("en-US", {
    timeZone: timezone,
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  });

  return localTime;
}

/**
 * Format a timestamp to display in user's local timezone
 * @param timestamp - ISO timestamp string or Date object
 * @param userTimezone - Optional user timezone, defaults to browser timezone
 * @param includeTimezone - Whether to include timezone abbreviation in the result
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
 * Get user-friendly timezone display name
 * @param userTimezone - Optional user timezone, defaults to browser timezone
 * @returns Human-readable timezone name (e.g., "PST", "EST")
 */
export function getUserTimezoneDisplay(userTimezone?: string): string {
  const timezone = userTimezone || getUserTimezone();

  // Get timezone abbreviation
  const now = new Date();
  const timeZoneName =
    now
      .toLocaleDateString("en-US", {
        timeZone: timezone,
        timeZoneName: "long",
      })
      .split(", ")
      .pop() || timezone;

  return timeZoneName;
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
