import type {
  FactoriesDescribeFactoryVelocityDay,
  FactoriesDescribeFactoryVelocityResponse,
  FactoriesDescribeFactoryVelocityTotals,
} from "@/api-client";

import { VELOCITY_ORIGIN_COLORS, VELOCITY_OUTCOME_COLORS, velocityIntakeColor } from "./velocitySeriesColors";
import { parseWorkOrderMetric } from "./workOrderUsage";

export type VelocityPeriodDays = 14 | 30;

/** How the delivery chart splits merged pull requests. */
export type VelocityBreakdown = "origin" | "outcome" | "intake";

export const VELOCITY_PERIOD_OPTIONS: { value: string; label: string }[] = [
  { value: "14", label: "14d" },
  { value: "30", label: "30d" },
];

export const VELOCITY_BREAKDOWN_OPTIONS: { value: VelocityBreakdown; label: string }[] = [
  { value: "origin", label: "Who created" },
  { value: "outcome", label: "Outcome" },
  { value: "intake", label: "Intake source" },
];

export const VELOCITY_BREAKDOWN_COPY: Record<VelocityBreakdown, { title: string; description: string }> = {
  origin: {
    title: "Merged pull requests by who created them",
    description: "Merged pull requests from people, next to pull requests SuperPlane created.",
  },
  outcome: {
    title: "Pull requests by outcome",
    description: "Merged pull requests and SuperPlane pull requests closed without merge.",
  },
  intake: {
    title: "SuperPlane merges by intake source",
    description: "Merged pull requests grouped by how their tasks reached the workspace.",
  },
};

export interface VelocityPoint {
  day: string;
  people: number;
  superplane: number;
  /** People plus SuperPlane merges of the day. */
  merged: number;
  waste: number;
  costUsd: number;
  wasteCostUsd: number;
  tokens: number;
  /** Merged SuperPlane pull requests of the day, keyed by intake source. */
  intake: Record<string, number>;
}

export interface VelocityTotals {
  merged: number;
  peopleMerged: number;
  superplaneMerged: number;
  waste: number;
  costUsd: number;
  wasteCostUsd: number;
  tokens: number;
  /**
   * Tasks that closed in the window, with or without a merge. A task counts
   * once, however many pull requests it opened, so this differs from the pull
   * request counts above.
   */
  tasksClosed: number;
  /** Part of `tasksClosed` that closed without a merge. */
  tasksWaste: number;
  /** Waste as a share of closed tasks, 0-100. */
  taskWasteRate: number;
  /** Tracked model spend divided by the tasks that closed. */
  costPerTask: number;
}

export interface VelocityIntakeSeries {
  key: string;
  label: string;
  color: string;
  merged: number;
}

export interface VelocityPerson {
  id: string;
  name: string;
  email: string;
  avatarUrl?: string;
  /** Pull requests the person authored outside SuperPlane. */
  authoredMerged: number;
  /** Merged pull requests from SuperPlane tasks this person opened. */
  factoryMerged: number;
  /** Tasks this person opened that closed without a merge. */
  factoryWaste: number;
  medianCycleHours: number;
  costUsd: number;
}

export interface VelocityReport {
  totals: VelocityTotals;
  /** Totals of the window before this one, when it holds comparable output. */
  previous?: VelocityTotals;
  points: VelocityPoint[];
  intakeSeries: VelocityIntakeSeries[];
  people: VelocityPerson[];
  hasPeopleCohort: boolean;
  /** When the background sync last stored repository merges. */
  peopleSyncedAt?: Date;
  /** True while the first repository sync has not stored any merges yet. */
  peopleSyncPending: boolean;
  repository?: string;
}

function centsToUsd(value: string | number | undefined): number {
  return parseWorkOrderMetric(value) / 100;
}

/** Rounded share of `whole` that `part` holds, 0-100. */
function sharePct(part: number, whole: number): number {
  if (whole <= 0) return 0;
  return Math.round((part / whole) * 100);
}

function toTotals(totals: FactoriesDescribeFactoryVelocityTotals | undefined): VelocityTotals {
  const superplaneMerged = totals?.superplaneMerged ?? 0;
  const peopleMerged = totals?.peopleMerged ?? 0;
  const waste = totals?.waste ?? 0;
  const costUsd = centsToUsd(totals?.costCents);
  const tasksClosed = totals?.tasksClosed ?? 0;
  const tasksWaste = totals?.tasksWaste ?? 0;

  return {
    merged: peopleMerged + superplaneMerged,
    peopleMerged,
    superplaneMerged,
    waste,
    costUsd,
    wasteCostUsd: centsToUsd(totals?.wasteCostCents),
    tokens: parseWorkOrderMetric(totals?.tokens),
    tasksClosed,
    tasksWaste,
    taskWasteRate: sharePct(tasksWaste, tasksClosed),
    costPerTask: tasksClosed > 0 ? costUsd / tasksClosed : 0,
  };
}

function toPoint(point: FactoriesDescribeFactoryVelocityDay): VelocityPoint {
  const superplane = point.superplaneMerged ?? 0;
  const people = point.peopleMerged ?? 0;

  const intake: Record<string, number> = {};
  for (const count of point.intake ?? []) {
    if (count.key) intake[count.key] = count.merged ?? 0;
  }

  return {
    day: point.day ?? "",
    people,
    superplane,
    merged: people + superplane,
    waste: point.waste ?? 0,
    costUsd: centsToUsd(point.costCents),
    wasteCostUsd: centsToUsd(point.wasteCostCents),
    tokens: parseWorkOrderMetric(point.tokens),
    intake,
  };
}

export function toVelocityReport(response: FactoriesDescribeFactoryVelocityResponse): VelocityReport {
  const intakeSeries: VelocityIntakeSeries[] = (response.intakeSources ?? [])
    .filter((source) => Boolean(source.key))
    .map((source, index) => ({
      key: source.key!,
      label: source.label || source.key!,
      color: velocityIntakeColor(source.key!, index),
      merged: source.merged ?? 0,
    }));

  const people: VelocityPerson[] = (response.people ?? []).map((person) => ({
    id: person.id ?? "",
    name: person.name || person.email || "Unknown member",
    email: person.email ?? "",
    avatarUrl: person.avatarUrl || undefined,
    authoredMerged: person.authoredMerged ?? 0,
    factoryMerged: person.factoryMerged ?? 0,
    factoryWaste: person.factoryWaste ?? 0,
    medianCycleHours: person.medianCycleHours ?? 0,
    costUsd: centsToUsd(person.costCents),
  }));

  return {
    totals: toTotals(response.totals),
    previous: response.hasPreviousWindow ? toTotals(response.previousTotals) : undefined,
    points: (response.points ?? []).map(toPoint),
    intakeSeries,
    people,
    hasPeopleCohort: Boolean(response.hasPeopleCohort),
    peopleSyncedAt: response.peopleSyncedAt ? new Date(response.peopleSyncedAt) : undefined,
    peopleSyncPending: Boolean(response.peopleSyncPending),
    repository: response.repository || undefined,
  };
}

/** hasVelocityOutput reports whether the window holds anything worth charting. */
export function hasVelocityOutput(report: VelocityReport): boolean {
  const { merged, waste, costUsd } = report.totals;
  return merged > 0 || waste > 0 || costUsd > 0;
}

export interface VelocityBreakdownSeries {
  key: string;
  label: string;
  color: string;
}

/**
 * The bands of the delivery chart. Intake bands come from the response, so a
 * workspace only sees the sources it actually uses.
 */
export function velocityBreakdownSeries(
  breakdown: VelocityBreakdown,
  intakeSeries: VelocityIntakeSeries[],
): VelocityBreakdownSeries[] {
  if (breakdown === "origin") {
    return [
      { key: "people", label: "Manual work", color: VELOCITY_ORIGIN_COLORS.people },
      { key: "superplane", label: "Automated via SuperPlane", color: VELOCITY_ORIGIN_COLORS.superplane },
    ];
  }
  if (breakdown === "outcome") {
    return [
      { key: "merged", label: "Merged", color: VELOCITY_OUTCOME_COLORS.merged },
      { key: "waste", label: "Closed without merge", color: VELOCITY_OUTCOME_COLORS.waste },
    ];
  }
  return intakeSeries.map((series) => ({ key: series.key, label: series.label, color: series.color }));
}
