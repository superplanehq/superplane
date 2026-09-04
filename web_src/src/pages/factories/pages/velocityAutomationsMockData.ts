import type { VelocityPeriodDays } from "../lib/factoryVelocityReport";

/** One automation, summed over the selected period. */
export interface AutomationRunRow {
  /** Canvas id of the automation, used to link to its detail page. */
  id: string;
  name: string;
  runs: number;
  failed: number;
  averageDurationHours: number;
  averageCostUsd: number;
  totalCostUsd: number;
}

interface AutomationSeed {
  id: string;
  name: string;
  /** Runs per day, before the period multiplies them. */
  runsPerDay: number;
  failureRate: number;
  averageDurationHours: number;
  averageCostUsd: number;
}

/** Ids and names match the automations of the fixture workspace. */
const AUTOMATION_SEEDS: AutomationSeed[] = [
  {
    id: "app-refund-planner",
    name: "Refund Planner",
    runsPerDay: 4.1,
    failureRate: 0.04,
    averageDurationHours: 0.35,
    averageCostUsd: 1.42,
  },
  {
    id: "app-refund-implementer",
    name: "Refund Implementer",
    runsPerDay: 3.4,
    failureRate: 0.09,
    averageDurationHours: 1.65,
    averageCostUsd: 2.86,
  },
  {
    id: "app-refund-verifier",
    name: "Refund Verifier",
    runsPerDay: 3.1,
    failureRate: 0.13,
    averageDurationHours: 0.62,
    averageCostUsd: 0.74,
  },
  {
    id: "app-pr-closure",
    name: "PR Closure",
    runsPerDay: 2.4,
    failureRate: 0.01,
    averageDurationHours: 0.05,
    averageCostUsd: 0.08,
  },
  {
    id: "app-github-issues-intake",
    name: "GitHub issue intake",
    runsPerDay: 5.2,
    failureRate: 0.02,
    averageDurationHours: 0.03,
    averageCostUsd: 0.04,
  },
];

/** Busiest automation first, so the rows that carry the spend read first. */
function buildRows(periodDays: VelocityPeriodDays): AutomationRunRow[] {
  return AUTOMATION_SEEDS.map((seed) => {
    const runs = Math.round(seed.runsPerDay * periodDays);
    return {
      id: seed.id,
      name: seed.name,
      runs,
      failed: Math.round(runs * seed.failureRate),
      averageDurationHours: seed.averageDurationHours,
      averageCostUsd: seed.averageCostUsd,
      totalCostUsd: Math.round(runs * seed.averageCostUsd * 100) / 100,
    };
  }).sort((left, right) => right.runs - left.runs);
}

/**
 * Mock automation runs by period, until the velocity API reports them.
 * Mirrors `lineVelocityMockData.ts`, which backs the Automations detail page.
 */
export const AUTOMATION_RUNS_BY_PERIOD: Record<VelocityPeriodDays, AutomationRunRow[]> = {
  14: buildRows(14),
  30: buildRows(30),
};
