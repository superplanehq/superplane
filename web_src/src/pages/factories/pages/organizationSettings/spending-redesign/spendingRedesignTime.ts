export const DAY_MS = 24 * 60 * 60 * 1000;
export const HOUR_MS = 60 * 60 * 1000;

export function startOfUtcDay(value: Date): Date {
  return new Date(Date.UTC(value.getUTCFullYear(), value.getUTCMonth(), value.getUTCDate()));
}

export function hourKey(value: Date): string {
  return `${dayKey(value)}T${String(value.getUTCHours()).padStart(2, "0")}`;
}

export function dayKey(value: Date): string {
  return `${value.getUTCFullYear()}-${String(value.getUTCMonth() + 1).padStart(2, "0")}-${String(value.getUTCDate()).padStart(2, "0")}`;
}

export function monthKey(value: Date): string {
  return `${value.getUTCFullYear()}-${String(value.getUTCMonth() + 1).padStart(2, "0")}`;
}

export function formatUtcDay(value: Date): string {
  return value.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric", timeZone: "UTC" });
}

export function formatHourLabel(value: Date): string {
  return `${String(value.getUTCHours()).padStart(2, "0")}:00`;
}

export function formatDayTick(value: Date): string {
  return value.toLocaleDateString("en-US", { month: "short", day: "numeric", timeZone: "UTC" });
}

export function formatMonthTick(value: Date): string {
  return value.toLocaleDateString("en-US", { month: "short", year: "2-digit", timeZone: "UTC" });
}
