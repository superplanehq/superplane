/**
 * Storybook-only sub-hour-aware duration labels for Velocity's work-order
 * time section. The live app keeps `formatDurationHours` (whole-hour
 * rounding); this module is only ever reached through
 * `VelocityDurationFormatSlotContext`, provided by `FactoriesHarness`.
 */

export type VelocityDurationUnit = "m" | "h" | "d";

export interface VelocityChartUnit {
  /** The unit chosen for a whole chart's Y axis, from its largest value. */
  unit: VelocityDurationUnit;
  /** Formats a raw hours value (an axis tick) in the chosen unit. */
  formatTick: (hours: number) => string;
}

export interface VelocityDurationFormat {
  /** Formats a single duration for scorecards and chart tooltips. */
  formatDuration: (hours: number) => string;
  /** Picks one axis unit for a whole chart from its largest value. */
  pickChartUnit: (hoursValues: number[]) => VelocityChartUnit;
}

function formatMinutes(hours: number): string {
  return `${Math.round(hours * 60)}m`;
}

function formatHours(hours: number): string {
  return `${Math.round(hours)}h`;
}

function formatDays(hours: number): string {
  const days = hours / 24;
  return `${days % 1 === 0 ? days.toFixed(0) : days.toFixed(1)}d`;
}

/**
 * Minutes below one hour (12m), hours from one hour up to two days (5h),
 * days above that (2.5d). A genuine zero still reads as 0h rather than 0m,
 * since there is no sub-hour value being hidden.
 */
export function formatSubHourDuration(hours: number): string {
  if (hours <= 0) return "0h";
  if (hours < 1) return formatMinutes(hours);
  if (hours < 48) return formatHours(hours);
  return formatDays(hours);
}

function unitForMax(maxHours: number): VelocityDurationUnit {
  if (maxHours <= 0) return "h";
  if (maxHours < 1) return "m";
  if (maxHours < 48) return "h";
  return "d";
}

function tickFormatterForUnit(unit: VelocityDurationUnit): (hours: number) => string {
  if (unit === "m") return formatMinutes;
  if (unit === "d") return formatDays;
  return formatHours;
}

/** Picks one unit for the whole chart from the largest value in the period. */
function pickChartUnit(hoursValues: number[]): VelocityChartUnit {
  const max = hoursValues.reduce((acc, value) => Math.max(acc, value), 0);
  const unit = unitForMax(max);
  return { unit, formatTick: tickFormatterForUnit(unit) };
}

export const subHourVelocityDurationFormat: VelocityDurationFormat = {
  formatDuration: formatSubHourDuration,
  pickChartUnit,
};
