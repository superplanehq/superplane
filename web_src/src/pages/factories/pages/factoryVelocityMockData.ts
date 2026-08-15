export type FactoryVelocityPeriodDays = 7 | 30;

export type FactoryVelocityDay = {
  day: string;
  merged: number;
  waste: number;
  costUsd: number;
  tokens: number;
  superplaneMerged: number;
  peopleMerged: number;
};

export type FactoryVelocityYesterday = {
  dateLabel: string;
  merged: number;
  waste: number;
  wastePct: number;
  costUsd: number;
  tokens: number;
  costPerMergedPr: number;
};

export type FactoryVelocityTotals = {
  merged: number;
  waste: number;
  wastePct: number;
  costUsd: number;
  superplaneMerged: number;
  peopleMerged: number;
  superplaneSharePct: number;
};

export type FactoryVelocityPeriod = {
  days: FactoryVelocityPeriodDays;
  label: string;
  points: FactoryVelocityDay[];
  totals: FactoryVelocityTotals;
};

function pct(part: number, whole: number) {
  if (whole <= 0) return 0;
  return Math.round((part / whole) * 100);
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

function totalsFrom(points: FactoryVelocityDay[]): FactoryVelocityTotals {
  const merged = points.reduce((sum, point) => sum + point.merged, 0);
  const waste = points.reduce((sum, point) => sum + point.waste, 0);
  const costUsd = Math.round(points.reduce((sum, point) => sum + point.costUsd, 0) * 100) / 100;
  const superplaneMerged = points.reduce((sum, point) => sum + point.superplaneMerged, 0);
  const peopleMerged = points.reduce((sum, point) => sum + point.peopleMerged, 0);
  return {
    merged,
    waste,
    wastePct: pct(waste, merged + waste),
    costUsd,
    superplaneMerged,
    peopleMerged,
    superplaneSharePct: pct(superplaneMerged, superplaneMerged + peopleMerged),
  };
}

function peopleMergedForDay(index: number, isWeekend: boolean) {
  const base = isWeekend ? 20 : 25;
  return Math.max(16, Math.round(base + 3 * Math.sin(index * 0.7)));
}

/** SuperPlane starts at 0, then ramps to about 20–30% of people volume (3–10 PRs). */
function superplaneMergedForDay(index: number, peopleMerged: number) {
  if (index < 2) return 0;
  if (index === 2) return 2;
  if (index === 3) return 3;
  if (index === 4) return 4;

  const share = 0.24 + 0.05 * Math.sin(index * 0.45);
  return clamp(Math.round(peopleMerged * share), 3, 10);
}

function buildDay(index: number, day: string, weekday: string): FactoryVelocityDay {
  const isWeekend = weekday === "Sat" || weekday === "Sun";
  const peopleMerged = peopleMergedForDay(index, isWeekend);
  const superplaneMerged = superplaneMergedForDay(index, peopleMerged);
  const waste = superplaneMerged === 0 ? 0 : Math.max(0, Math.round(superplaneMerged * (isWeekend ? 0.3 : 0.18)));
  const costUsd = superplaneMerged === 0 ? 0 : Math.round(superplaneMerged * 1.85 * 100) / 100;

  return {
    day,
    merged: superplaneMerged,
    waste,
    costUsd,
    tokens: superplaneMerged * 18500,
    superplaneMerged,
    peopleMerged,
  };
}

const WEEKDAYS = ["Fri", "Sat", "Sun", "Mon", "Tue", "Wed", "Thu"] as const;

const WEEK_POINTS: FactoryVelocityDay[] = WEEKDAYS.map((weekday, index) => buildDay(index, weekday, weekday));

function buildMonthPoints(): FactoryVelocityDay[] {
  return Array.from({ length: 30 }, (_, index) => {
    const weekday = WEEKDAYS[index % 7] ?? "Mon";
    const dayNumber = index + 1;
    const day = dayNumber % 5 === 1 || dayNumber === 30 ? String(dayNumber) : "";
    return buildDay(index, day, weekday);
  });
}

const MONTH_POINTS = buildMonthPoints();

/** Last complete day. Fixed, not affected by the period pills. */
export const FACTORY_VELOCITY_YESTERDAY: FactoryVelocityYesterday = (() => {
  const yesterday = WEEK_POINTS[WEEK_POINTS.length - 1];
  if (!yesterday) {
    return {
      dateLabel: "Yesterday",
      merged: 0,
      waste: 0,
      wastePct: 0,
      costUsd: 0,
      tokens: 0,
      costPerMergedPr: 0,
    };
  }
  return {
    dateLabel: "Yesterday",
    merged: yesterday.merged,
    waste: yesterday.waste,
    wastePct: pct(yesterday.waste, yesterday.merged + yesterday.waste),
    costUsd: yesterday.costUsd,
    tokens: yesterday.tokens,
    costPerMergedPr: yesterday.merged > 0 ? Math.round((yesterday.costUsd / yesterday.merged) * 100) / 100 : 0,
  };
})();

export const FACTORY_VELOCITY_BY_PERIOD: Record<FactoryVelocityPeriodDays, FactoryVelocityPeriod> = {
  7: {
    days: 7,
    label: "Last 7 days",
    points: WEEK_POINTS,
    totals: totalsFrom(WEEK_POINTS),
  },
  30: {
    days: 30,
    label: "Last 30 days",
    points: MONTH_POINTS,
    totals: totalsFrom(MONTH_POINTS),
  },
};
