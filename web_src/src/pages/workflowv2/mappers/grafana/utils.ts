import { formatTimestampInUserTimezone } from "@/utils/timezone";

export function formatTimestamp(value?: string): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return formatTimestampInUserTimezone(date);
}
