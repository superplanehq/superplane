import type { VelocityPerson } from "./VelocityPeopleTable";

export type PeriodDays = 14 | 30;
export type Breakdown = "origin" | "outcome" | "intake";

/** Workspace setup pins one app repository, so Velocity reports on that repository. */
export const WORKSPACE_REPOSITORY = "acme/refunds";

export interface VelocityPoint {
  day: string;
  people: number;
  superplane: number;
  merged: number;
  waste: number;
  githubIssues: number;
  sentryExceptions: number;
  manual: number;
  api: number;
  runningHours: number;
  waitingHours: number;
  costUsd: number;
  tokenCostUsd: number;
  computeCostUsd: number;
  wasteCostUsd: number;
  tokens: number;
}

export interface BreakdownSeries {
  key: keyof VelocityPoint;
  label: string;
  color: string;
}

export const PERIOD_OPTIONS = [
  { value: "14", label: "14d" },
  { value: "30", label: "30d" },
];

export const BREAKDOWN_OPTIONS = [
  { value: "origin", label: "Origin" },
  { value: "outcome", label: "Outcome" },
  { value: "intake", label: "Intake source" },
];

export const BREAKDOWN_SERIES: Record<Breakdown, BreakdownSeries[]> = {
  origin: [
    { key: "people", label: "People", color: "#64748b" },
    { key: "superplane", label: "SuperPlane", color: "#10b981" },
  ],
  outcome: [
    { key: "merged", label: "Merged", color: "#10b981" },
    { key: "waste", label: "Closed without merge", color: "#ef4444" },
  ],
  intake: [
    { key: "githubIssues", label: "GitHub issue", color: "#3b82f6" },
    { key: "sentryExceptions", label: "Sentry exception", color: "#8b5cf6" },
    { key: "manual", label: "Manually created", color: "#f59e0b" },
    { key: "api", label: "API", color: "#06b6d4" },
  ],
};

export const BREAKDOWN_COPY: Record<Breakdown, { title: string; description: string }> = {
  origin: {
    title: "Merged pull requests by origin",
    description: "Team output split between people and SuperPlane.",
  },
  outcome: {
    title: "Pull requests by outcome",
    description: "Merged pull requests and Factory pull requests closed without merge.",
  },
  intake: {
    title: "Factory merges by intake source",
    description: "Merged pull requests grouped by how their tasks entered the Factory.",
  },
};

export const COST_SERIES_COLORS = {
  tokens: "#64748b",
  compute: "#6366f1",
} as const;

export const TIME_SERIES_COLORS = {
  running: "#60a5fa",
  waiting: "#f59e0b",
} as const;

function githubAvatar(accountId: number): string {
  return `https://avatars.githubusercontent.com/u/${accountId}?v=4&s=64`;
}

/**
 * Per-member share of the period. Avatars use the GitHub account id, which is
 * how the connected provider serves them in production.
 */
const PEOPLE_SHARE: Array<
  Omit<VelocityPerson, "id" | "authoredMerged" | "factoryMerged" | "factoryWaste" | "costUsd"> & {
    id: string;
    authoredShare: number;
    factoryShare: number;
    wasteShare: number;
  }
> = [
  {
    id: "darkofabijan",
    name: "Darko Fabijan",
    email: "darko@superplane.com",
    avatarUrl: githubAvatar(20469),
    authoredShare: 0.24,
    factoryShare: 0.32,
    wasteShare: 0.18,
    medianCycleHours: 16,
  },
  {
    id: "AleksandarCole",
    name: "Aleksandar Mitrovic",
    email: "alex@superplane.com",
    avatarUrl: githubAvatar(61409859),
    authoredShare: 0.2,
    factoryShare: 0.2,
    wasteShare: 0.14,
    medianCycleHours: 21,
  },
  {
    id: "shiroyasha",
    name: "Igor Šarčević",
    email: "igor@superplane.com",
    avatarUrl: githubAvatar(1779493),
    authoredShare: 0.16,
    factoryShare: 0.16,
    wasteShare: 0.15,
    medianCycleHours: 19,
  },
  {
    id: "andrecalil",
    name: "André Calil",
    email: "andre@superplane.com",
    avatarUrl: githubAvatar(1105923),
    authoredShare: 0.14,
    factoryShare: 0.13,
    wasteShare: 0.2,
    medianCycleHours: 28,
  },
  {
    id: "forestileao",
    name: "Pedro Leão",
    email: "pedro@superplane.com",
    avatarUrl: githubAvatar(60622592),
    authoredShare: 0.12,
    factoryShare: 0.1,
    wasteShare: 0.18,
    medianCycleHours: 24,
  },
  {
    id: "markoa",
    name: "Marko Anastasov",
    email: "marko@superplane.com",
    avatarUrl: githubAvatar(8651),
    authoredShare: 0.08,
    factoryShare: 0.06,
    wasteShare: 0.1,
    medianCycleHours: 31,
  },
  {
    id: "lucaspin",
    name: "Lucas Pinheiro",
    email: "lucas@superplane.com",
    avatarUrl: githubAvatar(12387728),
    authoredShare: 0.06,
    factoryShare: 0.03,
    wasteShare: 0.05,
    medianCycleHours: 12,
  },
];

/**
 * `windowOffset` shifts the mock series along the timeline, so the current and
 * previous windows are different slices. Higher offset is more recent.
 *
 * The series drifts on purpose: Factory output ramps up while review time grows.
 * That gives the prototype a mixed result instead of every metric improving.
 */
export function buildVelocityPoints(periodDays: PeriodDays, windowOffset = 0): VelocityPoint[] {
  return Array.from({ length: periodDays }, (_, index) => {
    const day = index + windowOffset;
    const people = Math.max(1, Math.round(8 + 2.6 * Math.sin(day * 0.8)));
    const superplane = Math.max(1, Math.round(4 + 1.8 * Math.sin(day * 0.55 + 1) + day * 0.09));
    const merged = people + superplane;
    const waste = day % 6 === 2 ? Math.max(1, Math.round(superplane * 0.35)) : day % 4 === 0 ? 1 : 0;
    const githubIssues = Math.round(superplane * 0.43);
    const sentryExceptions = Math.round(superplane * 0.27);
    const manual = Math.round(superplane * 0.18);
    const api = Math.max(0, superplane - githubIssues - sentryExceptions - manual);
    const runningHours = 8 + 3 * Math.sin(day * 0.48 + 0.5);
    const waitingHours = 13 + 5 * Math.sin(day * 0.35 + 1.8) + day * 0.18;
    const mergedCostUsd = superplane * 2.14;
    const wasteCostUsd = waste * 1.85;
    const costUsd = mergedCostUsd + wasteCostUsd;
    const tokenCostUsd = Math.round((superplane * 1.67 + waste * 1.44) * 100) / 100;
    const computeCostUsd = Math.round((costUsd - tokenCostUsd) * 100) / 100;

    return {
      day: periodDays === 14 ? `${index + 1}` : index % 5 === 0 || index === periodDays - 1 ? `${index + 1}` : "",
      people,
      superplane,
      merged,
      waste,
      githubIssues,
      sentryExceptions,
      manual,
      api,
      runningHours: Math.max(2, Math.round(runningHours * 10) / 10),
      waitingHours: Math.max(3, Math.round(waitingHours * 10) / 10),
      costUsd: Math.round(costUsd * 100) / 100,
      tokenCostUsd,
      computeCostUsd,
      wasteCostUsd: Math.round(wasteCostUsd * 100) / 100,
      tokens: superplane * 18_500 + waste * 12_400,
    };
  });
}

export function summarizePoints(points: VelocityPoint[]) {
  const merged = sum(points, "merged");
  const superplaneMerged = sum(points, "superplane");
  const waste = sum(points, "waste");
  const cost = sum(points, "costUsd");
  const cycleHours = median(points.map((point) => point.runningHours + point.waitingHours));
  const wasteRate = superplaneMerged + waste > 0 ? Math.round((waste / (superplaneMerged + waste)) * 100) : 0;
  const costPerMerge = superplaneMerged > 0 ? cost / superplaneMerged : 0;

  return {
    merged,
    peopleMerged: sum(points, "people"),
    superplaneMerged,
    waste,
    wasteRate,
    cycleHours,
    runningHours: median(points.map((point) => point.runningHours)),
    waitingHours: median(points.map((point) => point.waitingHours)),
    cost,
    tokenCost: sum(points, "tokenCostUsd"),
    computeCost: sum(points, "computeCostUsd"),
    wasteCost: sum(points, "wasteCostUsd"),
    tokens: sum(points, "tokens"),
    costPerMerge,
  };
}

export function buildPeople(totals: {
  peopleMerged: number;
  superplaneMerged: number;
  waste: number;
  costUsd: number;
}): VelocityPerson[] {
  const authored = distribute(
    totals.peopleMerged,
    PEOPLE_SHARE.map((person) => person.authoredShare),
  );
  const factoryMerged = distribute(
    totals.superplaneMerged,
    PEOPLE_SHARE.map((person) => person.factoryShare),
  );
  const factoryWaste = distribute(
    totals.waste,
    PEOPLE_SHARE.map((person) => person.wasteShare),
  );

  return PEOPLE_SHARE.map((person, index) => ({
    id: person.id,
    name: person.name,
    email: person.email,
    avatarUrl: person.avatarUrl,
    authoredMerged: authored[index] ?? 0,
    factoryMerged: factoryMerged[index] ?? 0,
    factoryWaste: factoryWaste[index] ?? 0,
    medianCycleHours: person.medianCycleHours,
    costUsd: totals.costUsd * person.factoryShare,
  }));
}

/**
 * Splits a period total across members with the largest-remainder method, so
 * the People table always adds up to the totals shown above it.
 */
function distribute(total: number, shares: number[]): number[] {
  const exact = shares.map((share) => total * share);
  const floors = exact.map((value) => Math.floor(value));
  let remainder = total - floors.reduce((sum, value) => sum + value, 0);

  const order = exact
    .map((value, index) => ({ index, fraction: value - Math.floor(value) }))
    .sort((a, b) => b.fraction - a.fraction);

  const result = [...floors];
  for (const { index } of order) {
    if (remainder <= 0) break;
    result[index] = (result[index] ?? 0) + 1;
    remainder -= 1;
  }
  return result;
}

function sum(points: VelocityPoint[], field: keyof VelocityPoint): number {
  return points.reduce((total, point) => {
    const value = point[field];
    return total + (typeof value === "number" ? value : 0);
  }, 0);
}

function median(values: number[]): number {
  const sorted = [...values].sort((a, b) => a - b);
  const middle = Math.floor(sorted.length / 2);
  if (sorted.length % 2 === 0) return ((sorted[middle - 1] ?? 0) + (sorted[middle] ?? 0)) / 2;
  return sorted[middle] ?? 0;
}
