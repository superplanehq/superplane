import { formatDate, formatRelative, toDate, type TimestampInput } from "@/lib/datetime";

const RELATIVE_CREATED_TIME_WINDOW = 7 * 24 * 60 * 60 * 1000;

export function formatIntakeCreatedTime(
  value: TimestampInput | undefined,
  locale?: string,
  now: number = Date.now(),
): string {
  const date = toDate(value);
  if (!date) return "";

  if (now - date.getTime() < RELATIVE_CREATED_TIME_WINDOW) {
    return formatRelative(date, locale, now);
  }
  return formatDate(date, locale);
}
