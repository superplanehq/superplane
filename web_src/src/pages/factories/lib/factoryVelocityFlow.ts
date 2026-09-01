import type { FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";
import { flattenWorkOrderExecutions } from "./workOrderExecutions";

export type FactoryVelocityFlowPeriodDays = 14 | 30;

export function factoryVelocityPeriodLabel(days: FactoryVelocityFlowPeriodDays): string {
  return `Last ${days} days`;
}

type VelocityDurationUnit = "m" | "h" | "d";

/** Classifies a raw (unrounded) hours value into the unit band it belongs to. */
function classifyVelocityDurationUnit(hours: number): VelocityDurationUnit {
  if (!Number.isFinite(hours) || hours <= 0) return "h";
  if (hours < 1) return "m";
  if (hours < 48) return "h";
  return "d";
}

/** Renders minutes, guarding the rollover where rounding pushes 59.5–59.99m to "60m". */
function formatMinutes(hours: number): string {
  const minutes = Math.round(hours * 60);
  if (minutes >= 60) return "1h";
  return `${Math.max(1, minutes)}m`;
}

function formatHours(hours: number): string {
  return `${Math.round(hours)}h`;
}

function formatDays(hours: number): string {
  const days = hours / 24;
  return `${days % 1 === 0 ? days.toFixed(0) : days.toFixed(1)}d`;
}

function formatVelocityDuration(hours: number, unit: VelocityDurationUnit): string {
  switch (unit) {
    case "m":
      return formatMinutes(hours);
    case "d":
      return formatDays(hours);
    case "h":
    default:
      return formatHours(hours);
  }
}

export function formatDurationHours(hours: number): string {
  const safeHours = Number.isFinite(hours) ? Math.max(0, hours) : 0;
  return formatVelocityDuration(safeHours, classifyVelocityDurationUnit(safeHours));
}

export interface VelocityChartUnit {
  unit: VelocityDurationUnit;
  formatTick: (hours: number) => string;
}

/**
 * Picks a single unit for a whole chart axis from the largest raw value in
 * the series, so ticks never mix units (e.g. a sub-hour period doesn't
 * repeat "0h" and a multi-day period doesn't show triple-digit hours).
 */
export function pickVelocityChartUnit(hoursValues: number[]): VelocityChartUnit {
  const safeValues = hoursValues.filter((value) => Number.isFinite(value) && value > 0);
  const max = safeValues.length > 0 ? Math.max(...safeValues) : 0;
  const unit = classifyVelocityDurationUnit(max);

  return {
    unit,
    formatTick: (hours: number) => formatVelocityDuration(Number.isFinite(hours) ? Math.max(0, hours) : 0, unit),
  };
}

export interface FactoryVelocityFlowTrendPoint {
  day: string;
  runningHours: number;
  waitingHours: number;
}

export interface FactoryVelocityFlow {
  days: FactoryVelocityFlowPeriodDays;
  label: string;
  sampleSize: number;
  medianCycleHours: number;
  medianRunningHours: number;
  medianWaitingHours: number;
  runningShareOfCyclePct: number;
  waitingShareOfCyclePct: number;
  timeTrend: FactoryVelocityFlowTrendPoint[];
}

const MS_PER_HOUR = 60 * 60 * 1000;
const yesterdayFormatter = new Intl.DateTimeFormat(undefined, {
  weekday: "short",
  month: "short",
  day: "numeric",
  timeZone: "UTC",
});

export function formatVelocityYesterdayLabel(iso?: string, now: number = Date.now()): string {
  const parsed = iso ? new Date(iso) : null;
  if (parsed && !Number.isNaN(parsed.getTime())) {
    return `Yesterday · ${yesterdayFormatter.format(parsed)}`;
  }

  const local = new Date(now);
  const utcNoon = Date.UTC(local.getFullYear(), local.getMonth(), local.getDate() - 1, 12, 0, 0);
  return `Yesterday · ${yesterdayFormatter.format(new Date(utcNoon))}`;
}

interface CycleSample {
  closedDay: number;
  cycleHours: number;
  runningHours: number;
  waitingHours: number;
}

function parseTimestamp(value: string | null | undefined): number | null {
  if (!value) return null;
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function localMidnight(timestamp: number): number {
  const date = new Date(timestamp);
  date.setHours(0, 0, 0, 0);
  return date.getTime();
}

function addLocalCalendarDays(timestamp: number, days: number): number {
  const date = new Date(timestamp);
  date.setDate(date.getDate() + days);
  return date.getTime();
}

function firstExecutionStart(executions: FactoriesWorkOrderExecution[]): number | null {
  let earliest = Number.POSITIVE_INFINITY;
  for (const execution of executions) {
    const at = parseTimestamp(execution.createdAt);
    if (at !== null && at < earliest) {
      earliest = at;
    }
  }
  return Number.isFinite(earliest) ? earliest : null;
}

function sumRunningMillis(executions: FactoriesWorkOrderExecution[]): number {
  let total = 0;
  for (const execution of executions) {
    if (execution.state !== "STATE_FINISHED") continue;
    const start = parseTimestamp(execution.createdAt);
    const finish = parseTimestamp(execution.finishedAt) ?? parseTimestamp(execution.updatedAt);
    if (start === null || finish === null) continue;
    const span = finish - start;
    if (span > 0) {
      total += span;
    }
  }
  return total;
}

function closeTimestamp(order: FactoriesWorkOrder, executions: FactoriesWorkOrderExecution[]): number | null {
  // The work-order API has no closed_at. updatedAt is the close instant for
  // STATE_CLOSED rows unless a later edit moved it. Windowing and day
  // buckets must follow that instant so the card matches "closed in this period".
  const closedAt = parseTimestamp(order.updatedAt);
  if (closedAt !== null) {
    return closedAt;
  }

  let latestFinish = -Infinity;
  for (const execution of executions) {
    if (execution.state !== "STATE_FINISHED") continue;
    const finish = parseTimestamp(execution.finishedAt) ?? parseTimestamp(execution.updatedAt);
    if (finish !== null && finish > latestFinish) {
      latestFinish = finish;
    }
  }
  if (Number.isFinite(latestFinish)) {
    return latestFinish;
  }
  return null;
}

function toCycleSample(order: FactoriesWorkOrder, windowStart: number, windowEnd: number): CycleSample | null {
  if (order.state !== "STATE_CLOSED") return null;

  const executions = flattenWorkOrderExecutions(order);
  if (executions.length === 0) return null;

  const start = firstExecutionStart(executions);
  const end = closeTimestamp(order, executions);
  if (start === null || end === null) return null;
  if (end < windowStart || end >= windowEnd) return null;

  const cycleMillis = Math.max(0, end - start);
  const runningMillis = Math.min(cycleMillis, sumRunningMillis(executions));
  const waitingMillis = Math.max(0, cycleMillis - runningMillis);

  return {
    closedDay: localMidnight(end),
    cycleHours: cycleMillis / MS_PER_HOUR,
    runningHours: runningMillis / MS_PER_HOUR,
    waitingHours: waitingMillis / MS_PER_HOUR,
  };
}

function median(values: number[]): number {
  if (values.length === 0) return 0;
  const sorted = [...values].sort((left, right) => left - right);
  const mid = sorted.length >> 1;
  if (sorted.length % 2 === 1) {
    return sorted[mid];
  }
  return (sorted[mid - 1] + sorted[mid]) / 2;
}

function pct(part: number, whole: number): number {
  if (whole <= 0) return 0;
  return Math.round((part / whole) * 100);
}

/** How many days separate labelled ticks on a month-long window. */
const MONTH_LABEL_STEP = 5;

/**
 * Names the axis tick of a day, matching how the velocity API labels its own
 * points so both charts on the page share one axis language.
 *
 * The weekday explains the gaps in the chart: a quiet Saturday reads as a
 * weekend rather than as an outage. The month appears only where the window
 * crosses into a new one, so no single tick reads as the odd one out.
 */
function dayLabel(date: number, dayIndex: number, totalDays: number): string {
  let step = 1;
  if (totalDays > 14) {
    step = MONTH_LABEL_STEP;
    const isLast = dayIndex === totalDays - 1;
    if (dayIndex !== 0 && dayIndex % step !== 0 && !isLast) return "";
  }

  const day = new Date(date);
  const previous = new Date(addLocalCalendarDays(date, -step));
  const startsNewMonth = day.getMonth() !== previous.getMonth();

  // Composed part by part: asking Intl for a weekday and a day together yields
  // "18 Tue" in en-US, which does not match the labels the API sends.
  const weekday = day.toLocaleDateString(undefined, { weekday: "short" });
  const datePart = startsNewMonth
    ? day.toLocaleDateString(undefined, { month: "short", day: "numeric" })
    : String(day.getDate());

  return `${weekday} ${datePart}`;
}

interface DayBucket {
  key: number;
  label: string;
  running: number[];
  waiting: number[];
}

function buildDayBuckets(periodDays: FactoryVelocityFlowPeriodDays, now: number): DayBucket[] {
  const buckets: DayBucket[] = [];
  const todayMidnight = localMidnight(now);

  for (let offset = periodDays - 1; offset >= 0; offset--) {
    const key = addLocalCalendarDays(todayMidnight, -offset);
    const dayIndex = periodDays - 1 - offset;
    buckets.push({
      key,
      label: dayLabel(key, dayIndex, periodDays),
      running: [],
      waiting: [],
    });
  }
  return buckets;
}

export function aggregateFactoryVelocityFlow(
  orders: FactoriesWorkOrder[],
  periodDays: FactoryVelocityFlowPeriodDays,
  now: number = Date.now(),
): FactoryVelocityFlow {
  const todayMidnight = localMidnight(now);
  const windowStart = addLocalCalendarDays(todayMidnight, 1 - periodDays);
  const windowEnd = addLocalCalendarDays(todayMidnight, 1);

  const samples: CycleSample[] = [];
  for (const order of orders) {
    const sample = toCycleSample(order, windowStart, windowEnd);
    if (sample) samples.push(sample);
  }

  const buckets = buildDayBuckets(periodDays, now);
  const bucketByKey = new Map<number, DayBucket>();
  for (const bucket of buckets) {
    bucketByKey.set(bucket.key, bucket);
  }

  for (const sample of samples) {
    const bucket = bucketByKey.get(sample.closedDay);
    if (!bucket) continue;
    bucket.running.push(sample.runningHours);
    bucket.waiting.push(sample.waitingHours);
  }

  const medianCycle = median(samples.map((s) => s.cycleHours));
  const medianRunning = median(samples.map((s) => s.runningHours));
  const medianWaiting = median(samples.map((s) => s.waitingHours));

  const timeTrend: FactoryVelocityFlowTrendPoint[] = buckets.map((bucket) => ({
    day: bucket.label,
    runningHours: median(bucket.running),
    waitingHours: median(bucket.waiting),
  }));

  return {
    days: periodDays,
    label: factoryVelocityPeriodLabel(periodDays),
    sampleSize: samples.length,
    medianCycleHours: medianCycle,
    medianRunningHours: medianRunning,
    medianWaitingHours: medianWaiting,
    runningShareOfCyclePct: pct(medianRunning, medianCycle),
    waitingShareOfCyclePct: pct(medianWaiting, medianCycle),
    timeTrend,
  };
}
