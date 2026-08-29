import type { FactoryVelocityPeriodDays } from "./factoryVelocityMockData";

export type FactoryVelocityFlowTrendPoint = {
  day: string;
  /** Median hours in Running for tasks that closed that day. */
  runningHours: number;
  /** Median hours in Waiting for tasks that closed that day. */
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

// A sub-hour-heavy week: most days finish in minutes, so the chart axis and
// scorecards must show minute labels instead of a flat, repeated "0h".
const WEEK_TREND: FactoryVelocityFlowTrendPoint[] = [
  { day: "Fri", runningHours: 0.2, waitingHours: 0.1 },
  { day: "Sat", runningHours: 0.25, waitingHours: 0.15 },
  { day: "Sun", runningHours: 0.15, waitingHours: 0.2 },
  { day: "Mon", runningHours: 0.3, waitingHours: 0.25 },
  { day: "Tue", runningHours: 0.4, waitingHours: 0.3 },
  { day: "Wed", runningHours: 0.35, waitingHours: 0.35 },
  { day: "Thu", runningHours: 0.4, waitingHours: 0.3 },
];

function buildMonthTimeTrend(): FactoryVelocityFlowTrendPoint[] {
  return Array.from({ length: 30 }, (_, index) => {
    const dayNumber = index + 1;
    const day = dayNumber % 5 === 1 || dayNumber === 30 ? String(dayNumber) : "";
    const runningHours = Math.max(10, Math.round(14 + 2 * Math.sin(index * 0.5)));
    const waitingHours = Math.max(10, Math.round(16 + index * 0.25 + 4 * Math.sin(index * 0.35)));
    return { day, runningHours, waitingHours };
  });
}

export const FACTORY_VELOCITY_FLOW_BY_PERIOD: Record<FactoryVelocityPeriodDays, FactoryVelocityFlowPeriod> = {
  7: {
    days: 7,
    label: "Last 7 days",
    medianCycleHours: 0.5,
    medianRunningHours: 0.3,
    medianWaitingHours: 0.2,
    runningShareOfCyclePct: 60,
    waitingShareOfCyclePct: 40,
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
