import type { VelocityPeriodDays, VelocityPeriodStats } from "./lineVelocityMockData";

export type DailyVelocityPoint = {
  day: string;
  succeeded: number;
  failed: number;
  inProgress: number;
};

type OutcomeKey = "succeeded" | "failed" | "inProgress";

function buildDayLabels(days: VelocityPeriodDays): string[] {
  if (days === 7) return ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

  const labels: string[] = [];
  for (let index = 0; index < days; index += 1) {
    const dayNumber = index + 1;
    if (days === 30) {
      labels.push(dayNumber % 5 === 1 || dayNumber === days ? String(dayNumber) : "");
      continue;
    }
    labels.push(dayNumber % 15 === 1 || dayNumber === days ? String(dayNumber) : "");
  }
  return labels;
}

function distributeByWeights(total: number, normalized: number[]): number[] {
  const values = normalized.map((weight) => Math.round(total * weight));
  const drift = total - values.reduce((sum, value) => sum + value, 0);
  values[values.length - 1] = (values[values.length - 1] ?? 0) + drift;
  return values.map((value) => Math.max(0, value));
}

function buildIntakeWeights(dayCount: number): number[] {
  const intakeWeights = Array.from({ length: dayCount }, (_, index) => {
    return 0.85 + 0.3 * Math.sin((index / dayCount) * Math.PI * 2);
  });
  const weightSum = intakeWeights.reduce((sum, weight) => sum + weight, 0);
  return intakeWeights.map((weight) => weight / weightSum);
}

function buildInitialDailyPoint(args: {
  day: string;
  dayEntered: number;
  index: number;
  dayCount: number;
  open: number;
  openDayIndexes: number[];
  successShareOfClosed: number;
}): DailyVelocityPoint {
  const isOpenDay = args.open > 0 && args.openDayIndexes.includes(args.index);
  let dayInProgress = isOpenDay
    ? Math.min(args.dayEntered, Math.round(args.dayEntered * (args.index === args.dayCount - 1 ? 0.55 : 0.35)))
    : 0;
  const closed = args.dayEntered - dayInProgress;
  let daySucceeded = Math.round(closed * args.successShareOfClosed);
  let dayFailed = closed - daySucceeded;
  if (dayFailed < 0) {
    daySucceeded = closed;
    dayFailed = 0;
  }
  const used = daySucceeded + dayFailed + dayInProgress;
  if (used !== args.dayEntered) {
    if (isOpenDay) dayInProgress = Math.max(0, dayInProgress + (args.dayEntered - used));
    else daySucceeded = Math.max(0, daySucceeded + (args.dayEntered - used));
  }
  return {
    day: args.day || "·",
    succeeded: daySucceeded,
    failed: dayFailed,
    inProgress: dayInProgress,
  };
}

function sumOutcome(points: DailyVelocityPoint[], key: OutcomeKey): number {
  return points.reduce((total, point) => total + point[key], 0);
}

function shiftOutcome(
  points: DailyVelocityPoint[],
  from: OutcomeKey,
  to: OutcomeKey,
  amount: number,
  dayIndexes: number[],
) {
  let remaining = amount;
  for (const index of dayIndexes) {
    if (remaining <= 0) break;
    const point = points[index];
    if (!point) continue;
    const move = Math.min(remaining, point[from]);
    if (move <= 0) continue;
    point[from] -= move;
    point[to] += move;
    remaining -= move;
  }
}

function reconcileOpenWork(points: DailyVelocityPoint[], open: number, openDayIndexes: number[], closedDays: number[]) {
  let openDelta = open - sumOutcome(points, "inProgress");
  if (openDelta > 0) {
    shiftOutcome(points, "succeeded", "inProgress", openDelta, openDayIndexes);
    openDelta = open - sumOutcome(points, "inProgress");
    if (openDelta > 0) shiftOutcome(points, "failed", "inProgress", openDelta, openDayIndexes);
  } else if (openDelta < 0) {
    shiftOutcome(points, "inProgress", "succeeded", -openDelta, openDayIndexes);
  }

  for (const index of closedDays) {
    const point = points[index];
    if (!point || point.inProgress === 0) continue;
    point.succeeded += point.inProgress;
    point.inProgress = 0;
  }
}

function reconcileSuccessTotals(points: DailyVelocityPoint[], succeeded: number, closedDays: number[]) {
  const successDelta = succeeded - sumOutcome(points, "succeeded");
  if (successDelta > 0) shiftOutcome(points, "failed", "succeeded", successDelta, closedDays);
  else if (successDelta < 0) shiftOutcome(points, "succeeded", "failed", -successDelta, closedDays);
}

function ensureOpenDaysHaveProgress(points: DailyVelocityPoint[], open: number, openDayIndexes: number[]) {
  if (open <= 0) return;
  for (const index of openDayIndexes) {
    const point = points[index];
    if (!point || point.inProgress > 0) continue;
    if (point.succeeded > 0) {
      point.succeeded -= 1;
      point.inProgress += 1;
    } else if (point.failed > 0) {
      point.failed -= 1;
      point.inProgress += 1;
    }
  }
}

/**
 * Build stacked daily outcomes for the selected window.
 * Still-in-progress only on the last two days when the period has open work.
 */
export function buildDailyVelocity(stats: VelocityPeriodStats): DailyVelocityPoint[] {
  const open = Math.max(0, stats.runs - stats.succeeded - stats.failed);
  const dayCount = stats.days;
  const labels = buildDayLabels(stats.days);
  const normalized = buildIntakeWeights(dayCount);
  const entered = distributeByWeights(stats.runs, normalized);
  const closedTotal = stats.succeeded + stats.failed;
  const successShareOfClosed = closedTotal > 0 ? stats.succeeded / closedTotal : 1;
  const openDayIndexes = [dayCount - 2, dayCount - 1];

  const points = labels.map((day, index) =>
    buildInitialDailyPoint({
      day,
      dayEntered: entered[index] ?? 0,
      index,
      dayCount,
      open,
      openDayIndexes,
      successShareOfClosed,
    }),
  );

  const closedDays = Array.from({ length: dayCount }, (_, index) => index).filter(
    (index) => !openDayIndexes.includes(index),
  );

  reconcileOpenWork(points, open, openDayIndexes, closedDays);
  reconcileSuccessTotals(points, stats.succeeded, closedDays);
  ensureOpenDaysHaveProgress(points, open, openDayIndexes);

  // Restore blank tick labels for sparse axes (chart still needs a stable key).
  return points.map((point, index) => ({
    ...point,
    day: labels[index] || "",
  }));
}
