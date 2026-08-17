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

const WEEK_TREND: FactoryVelocityFlowTrendPoint[] = [
  { day: "Fri", runningHours: 16, waitingHours: 12 },
  { day: "Sat", runningHours: 14, waitingHours: 16 },
  { day: "Sun", runningHours: 12, waitingHours: 20 },
  { day: "Mon", runningHours: 12, waitingHours: 22 },
  { day: "Tue", runningHours: 12, waitingHours: 26 },
  { day: "Wed", runningHours: 12, waitingHours: 28 },
  { day: "Thu", runningHours: 14, waitingHours: 22 },
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
    medianCycleHours: 36,
    medianRunningHours: 14,
    medianWaitingHours: 22,
    runningShareOfCyclePct: 39,
    waitingShareOfCyclePct: 61,
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
