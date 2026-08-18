import type { FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";

export type FactoryVelocityFlowPeriodDays = 7 | 30;

export const VELOCITY_PERIOD_OPTIONS: { value: string; label: string }[] = [
  { value: "7", label: "7d" },
  { value: "30", label: "30d" },
];

export function factoryVelocityPeriodLabel(days: FactoryVelocityFlowPeriodDays): string {
  return days === 7 ? "Last 7 days" : "Last 30 days";
}

export function formatDurationHours(hours: number) {
  if (hours < 48) {
    return `${Math.round(hours)}h`;
  }
  const days = hours / 24;
  return `${days % 1 === 0 ? days.toFixed(0) : days.toFixed(1)}d`;
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
const WEEKDAY_SHORT = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"] as const;

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

function cycleEndTimestamp(order: FactoriesWorkOrder, executions: FactoriesWorkOrderExecution[]): number | null {
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
  return parseTimestamp(order.updatedAt);
}

function toCycleSample(order: FactoriesWorkOrder, windowStart: number, windowEnd: number): CycleSample | null {
  if (order.state !== "STATE_CLOSED") return null;

  const executions = order.executions ?? [];
  if (executions.length === 0) return null;

  const start = firstExecutionStart(executions);
  const end = cycleEndTimestamp(order, executions);
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

function dayLabel(dayIndex: number, totalDays: number, weekday: number): string {
  if (totalDays <= 7) {
    return WEEKDAY_SHORT[weekday];
  }
  const dayNumber = dayIndex + 1;
  const isLast = dayIndex === totalDays - 1;
  if (dayNumber === 1 || dayIndex % 5 === 0 || isLast) {
    return String(dayNumber);
  }
  return "";
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
    const date = new Date(key);
    const dayIndex = periodDays - 1 - offset;
    buckets.push({
      key,
      label: dayLabel(dayIndex, periodDays, date.getDay()),
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
