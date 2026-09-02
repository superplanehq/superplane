import type { FactoriesDescribeFactoryVelocityResponse } from "@/api-client";

/**
 * Storybook payloads for the workspace Velocity report. The series drift on
 * purpose: SuperPlane output climbs while review time grows, so the page shows
 * a mixed result instead of every metric improving at once.
 */

const PEOPLE_AUTHORS = [
  { id: "user-darko", name: "Darko Fabijan", email: "darko@superplane.com", accountId: 20469, share: 0.24 },
  { id: "user-aleksandar", name: "Aleksandar Mitrovic", email: "alex@superplane.com", accountId: 61409859, share: 0.2 },
  { id: "user-igor", name: "Igor Šarčević", email: "igor@superplane.com", accountId: 1779493, share: 0.18 },
  { id: "user-andre", name: "André Calil", email: "andre@superplane.com", accountId: 1105923, share: 0.14 },
  { id: "user-pedro", name: "Pedro Leão", email: "pedro@superplane.com", accountId: 60622592, share: 0.13 },
  { id: "user-marko", name: "Marko Anastasov", email: "marko@superplane.com", accountId: 8651, share: 0.11 },
];

function githubAvatar(accountId: number): string {
  return `https://avatars.githubusercontent.com/u/${accountId}?v=4&s=64`;
}

const INTAKE_SHARES = [
  { key: "github-issues", label: "GitHub issue", share: 0.43 },
  { key: "sentry-exceptions", label: "Sentry exception", share: 0.27 },
  { key: "manual", label: "Manually created", share: 0.18 },
  { key: "automation", label: "Automation", share: 0.12 },
];

interface VelocityDaySeed {
  index: number;
  periodDays: number;
  people: number;
  superplane: number;
  waste: number;
}

function dayLabel(index: number, periodDays: number): string {
  const dayNumber = index + 1;
  if (periodDays <= 14) return String(dayNumber);
  if (dayNumber === 1 || index % 5 === 0 || index === periodDays - 1) return String(dayNumber);
  return "";
}

/** Splits a day's merges across intake sources, largest share first. */
function intakeCounts(superplane: number) {
  let left = superplane;
  return INTAKE_SHARES.map((source, index) => {
    const isLast = index === INTAKE_SHARES.length - 1;
    const merged = isLast ? Math.max(0, left) : Math.round(superplane * source.share);
    left -= merged;
    return { key: source.key, merged: Math.max(0, merged) };
  }).filter((count) => count.merged > 0);
}

function buildDay({ index, periodDays, people, superplane, waste }: VelocityDaySeed) {
  const costCents = Math.round(superplane * 214 + waste * 185);
  return {
    day: dayLabel(index, periodDays),
    superplaneMerged: superplane,
    peopleMerged: people,
    waste,
    intake: intakeCounts(superplane),
    costCents: String(costCents),
    tokens: String(superplane * 18_500 + waste * 12_400),
    wasteCostCents: String(waste * 185),
  };
}

function buildPoints(periodDays: number, offset = 0) {
  return Array.from({ length: periodDays }, (_, index) => {
    const day = index + offset;
    const people = Math.max(1, Math.round(8 + 2.6 * Math.sin(day * 0.8)));
    const superplane = Math.max(1, Math.round(4 + 1.8 * Math.sin(day * 0.55 + 1) + day * 0.09));
    const waste = day % 6 === 2 ? Math.max(1, Math.round(superplane * 0.35)) : day % 4 === 0 ? 1 : 0;
    return buildDay({ index, periodDays, people, superplane, waste });
  });
}

type Point = ReturnType<typeof buildDay>;

function sumTotals(points: Point[]) {
  const superplaneMerged = points.reduce((total, point) => total + point.superplaneMerged, 0);
  const peopleMerged = points.reduce((total, point) => total + point.peopleMerged, 0);
  const waste = points.reduce((total, point) => total + point.waste, 0);
  const costCents = points.reduce((total, point) => total + Number(point.costCents), 0);
  const tokens = points.reduce((total, point) => total + Number(point.tokens), 0);
  const wasteCostCents = points.reduce((total, point) => total + Number(point.wasteCostCents), 0);
  const merged = superplaneMerged + peopleMerged;

  return {
    superplaneMerged,
    peopleMerged,
    waste,
    superplaneSharePct: merged > 0 ? Math.round((superplaneMerged / merged) * 100) : 0,
    wastePct: superplaneMerged + waste > 0 ? Math.round((waste / (superplaneMerged + waste)) * 100) : 0,
    costCents: String(costCents),
    tokens: String(tokens),
    wasteCostCents: String(wasteCostCents),
    // Every pull request of these payloads comes from a task of its own.
    tasksClosed: superplaneMerged + waste,
    tasksWaste: waste,
  };
}

/** Distributes a period total across authors so the table adds up to the totals. */
function distribute(total: number, shares: number[]): number[] {
  const exact = shares.map((share) => total * share);
  const floors = exact.map((value) => Math.floor(value));
  let remainder = total - floors.reduce((sum, value) => sum + value, 0);

  const order = exact
    .map((value, index) => ({ index, fraction: value - Math.floor(value) }))
    .sort((left, right) => right.fraction - left.fraction);

  const result = [...floors];
  for (const { index } of order) {
    if (remainder <= 0) break;
    result[index] = (result[index] ?? 0) + 1;
    remainder -= 1;
  }
  return result;
}

function buildPeople(totals: ReturnType<typeof sumTotals>) {
  const shares = PEOPLE_AUTHORS.map((person) => person.share);
  const authored = distribute(totals.peopleMerged, shares);
  const factoryMerged = distribute(totals.superplaneMerged, shares);
  const factoryWaste = distribute(totals.waste, shares);
  const costCents = Number(totals.costCents);

  return PEOPLE_AUTHORS.map((person, index) => ({
    id: person.id,
    name: person.name,
    email: person.email,
    avatarUrl: githubAvatar(person.accountId),
    authoredMerged: authored[index] ?? 0,
    factoryMerged: factoryMerged[index] ?? 0,
    factoryWaste: factoryWaste[index] ?? 0,
    medianCycleHours: 14 + index * 3,
    costCents: String(Math.round(costCents * person.share)),
  }));
}

function intakeSources(points: Point[]) {
  const totals = new Map<string, number>();
  for (const point of points) {
    for (const count of point.intake) {
      totals.set(count.key, (totals.get(count.key) ?? 0) + count.merged);
    }
  }
  return INTAKE_SHARES.filter((source) => (totals.get(source.key) ?? 0) > 0).map((source) => ({
    key: source.key,
    label: source.label,
    merged: totals.get(source.key) ?? 0,
  }));
}

function buildReport(periodDays: number, withComparison: boolean): FactoriesDescribeFactoryVelocityResponse {
  const points = buildPoints(periodDays, periodDays);
  const totals = sumTotals(points);
  const previous = sumTotals(buildPoints(periodDays, 0));
  const yesterday = points[points.length - 2] ?? points[points.length - 1]!;

  return {
    yesterday: { superplaneMerged: yesterday.superplaneMerged, waste: yesterday.waste },
    totals,
    points,
    repository: "acme/refunds",
    hasPeopleCohort: true,
    peopleSyncedAt: new Date().toISOString(),
    peopleSyncPending: false,
    previousTotals: previous,
    hasPreviousWindow: withComparison,
    intakeSources: intakeSources(points),
    people: buildPeople(totals),
  };
}

export const DEFAULT_FACTORY_VELOCITY: Record<number, FactoriesDescribeFactoryVelocityResponse> = {
  14: buildReport(14, true),
  30: buildReport(30, true),
};

/** A new workspace: nothing merged, nothing closed, nothing spent. */
export const EMPTY_FACTORY_VELOCITY: FactoriesDescribeFactoryVelocityResponse = {
  yesterday: { superplaneMerged: 0, waste: 0 },
  totals: {
    superplaneMerged: 0,
    peopleMerged: 0,
    waste: 0,
    superplaneSharePct: 0,
    wastePct: 0,
    costCents: "0",
    tokens: "0",
    wasteCostCents: "0",
    tasksClosed: 0,
    tasksWaste: 0,
  },
  points: Array.from({ length: 14 }, (_, index) => ({
    day: dayLabel(index, 14),
    superplaneMerged: 0,
    peopleMerged: 0,
    waste: 0,
  })),
  hasPeopleCohort: false,
  hasPreviousWindow: false,
  intakeSources: [],
  people: [],
};

/**
 * A workspace a few hours old. The repository already holds people merges over
 * the whole period, because Velocity reads repository history, but SuperPlane
 * has one day of output. There is no earlier period to compare with.
 */
export const EARLY_USAGE_FACTORY_VELOCITY: FactoriesDescribeFactoryVelocityResponse = (() => {
  const report = buildReport(14, false);
  const points = (report.points ?? []).map((point, index, all) => {
    const isLastDay = index === all.length - 1;
    if (isLastDay) return point;
    return {
      ...point,
      superplaneMerged: 0,
      waste: 0,
      intake: [],
      costCents: "0",
      tokens: "0",
      wasteCostCents: "0",
    };
  });

  const totals = sumTotals(points as Point[]);
  return {
    ...report,
    points,
    totals,
    previousTotals: undefined,
    hasPreviousWindow: false,
    intakeSources: intakeSources(points as Point[]),
    people: buildPeople(totals),
  };
})();

/**
 * A workspace that just connected a repository. SuperPlane output is already
 * there, but the background sync has not stored the repository history yet, so
 * the People series and the SuperPlane share are withheld.
 */
export const PEOPLE_SYNC_PENDING_FACTORY_VELOCITY: FactoriesDescribeFactoryVelocityResponse = (() => {
  const report = buildReport(14, true);
  const points = (report.points ?? []).map((point) => ({ ...point, peopleMerged: 0 }));
  const totals = sumTotals(points as Point[]);

  return {
    ...report,
    points,
    totals: { ...totals, peopleMerged: 0, superplaneSharePct: 0 },
    hasPeopleCohort: false,
    peopleSyncedAt: undefined,
    peopleSyncPending: true,
    people: buildPeople(totals).map((person) => ({ ...person, authoredMerged: 0 })),
  };
})();
