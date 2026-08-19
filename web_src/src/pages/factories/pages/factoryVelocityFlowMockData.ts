import type { FactoryVelocityPeriodDays } from "./factoryVelocityMockData";

export type FactoryVelocityFlowTrendPoint = {
  day: string;
  /** Median hours in Running for work orders that closed that day. */
  runningHours: number;
  /** Median hours in Waiting for work orders that closed that day. */
  waitingHours: number;
};

export type FactoryVelocityFlowPeriod = {
  days: FactoryVelocityPeriodDays;
  label: string;
  /** Median hours from leaving Draft to closed. */
  medianCycleHours: number;
  /** Median hours spent in Running before close. */
  medianRunningHours: number;
  /** Median hours spent in Waiting before close. */
  medianWaitingHours: number;
  runningShareOfCyclePct: number;
  waitingShareOfCyclePct: number;
  timeTrend: FactoryVelocityFlowTrendPoint[];
};

// Every work order in the last 7 days closed inside the hour, so the whole
// week is sub-hour: the chart axis should read in minutes, not repeat "0h".
const WEEK_TREND: FactoryVelocityFlowTrendPoint[] = [
  { day: "Fri", runningHours: 0.15, waitingHours: 0.1 },
  { day: "Sat", runningHours: 0.2, waitingHours: 0.15 },
  { day: "Sun", runningHours: 0.1, waitingHours: 0.2 },
  { day: "Mon", runningHours: 0.25, waitingHours: 0.15 },
  { day: "Tue", runningHours: 0.2, waitingHours: 0.25 },
  { day: "Wed", runningHours: 0.15, waitingHours: 0.3 },
  { day: "Thu", runningHours: 0.2, waitingHours: 0.2 },
];

function buildMonthTimeTrend(): FactoryVelocityFlowTrendPoint[] {
  return Array.from({ length: 30 }, (_, index) => {
    const dayNumber = index + 1;
    const day = dayNumber % 5 === 1 || dayNumber === 30 ? String(dayNumber) : "";

    // Mix a genuine zero day (no work orders closed) and a sub-hour day into
    // an otherwise hour-scale month, so the picked "h" axis unit still needs
    // sub-hour-aware tooltips on some points.
    if (index === 3) return { day, runningHours: 0, waitingHours: 0 };
    if (index === 10) return { day, runningHours: 0.2, waitingHours: 0.15 };

    const runningHours = Math.max(10, Math.round(14 + 2 * Math.sin(index * 0.5)));
    const waitingHours = Math.max(10, Math.round(16 + index * 0.25 + 4 * Math.sin(index * 0.35)));
    return { day, runningHours, waitingHours };
  });
}

export const FACTORY_VELOCITY_FLOW_BY_PERIOD: Record<FactoryVelocityPeriodDays, FactoryVelocityFlowPeriod> = {
  7: {
    days: 7,
    label: "Last 7 days",
    medianCycleHours: 0.45,
    medianRunningHours: 0.2,
    medianWaitingHours: 0.25,
    runningShareOfCyclePct: 44,
    waitingShareOfCyclePct: 56,
    timeTrend: WEEK_TREND,
  },
  30: {
    days: 30,
    label: "Last 30 days",
    medianCycleHours: 42,
    medianRunningHours: 16,
    medianWaitingHours: 26,
    runningShareOfCyclePct: 38,
    waitingShareOfCyclePct: 62,
    timeTrend: buildMonthTimeTrend(),
  },
};
