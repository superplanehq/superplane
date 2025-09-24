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
  const [hours, minutes] = timeString.split(':').map(Number);

  // Create a date in the user's timezone
  const localDate = new Date(today.getFullYear(), today.getMonth(), today.getDate(), hours, minutes);

  // Get the timezone offset for the user's timezone
  const tempDate = new Date(localDate.toLocaleString("en-US", { timeZone: timezone }));
  const offset = localDate.getTime() - tempDate.getTime();

  // Adjust the date to UTC
  const utcDate = new Date(localDate.getTime() - offset);

  const utcHours = utcDate.getUTCHours().toString().padStart(2, '0');
  const utcMinutes = utcDate.getUTCMinutes().toString().padStart(2, '0');

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
  const [hours, minutes] = utcTimeString.split(':').map(Number);

  // Create a UTC date
  const utcDate = new Date(Date.UTC(today.getFullYear(), today.getMonth(), today.getDate(), hours, minutes));

  // Convert to user's timezone
  const localTime = utcDate.toLocaleString('en-US', {
    timeZone: timezone,
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
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
export function formatTimestampInUserTimezone(
  timestamp: string | Date,
  userTimezone?: string,
  includeTimezone = true
): string {
  const timezone = userTimezone || getUserTimezone();
  const date = typeof timestamp === 'string' ? new Date(timestamp) : timestamp;

  const options: Intl.DateTimeFormatOptions = {
    timeZone: timezone,
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false // Use 24-hour format instead of AM/PM
  };

  if (includeTimezone) {
    options.timeZoneName = 'short';
  }

  return date.toLocaleDateString('en-US', options);
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
  const timeZoneName = now.toLocaleDateString('en-US', {
    timeZone: timezone,
    timeZoneName: 'long'
  }).split(', ').pop() || timezone;

  return timeZoneName;
}