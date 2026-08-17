import { FACTORY_VELOCITY_BY_PERIOD } from "./factoryVelocityMockData";

/** Same 7-day period totals the Velocity page shows, so the numbers stay consistent. */
export type OverviewVelocitySummary = {
  merged: number;
  waste: number;
  wastePct: number;
  costUsd: number;
  tokens: number;
  superplaneSharePct: number;
};

/** Storybook mock: last-7-days totals from the Velocity page mock data, plus a token sum for the cost hint. */
export function buildOverviewVelocitySummary(): OverviewVelocitySummary {
  const period = FACTORY_VELOCITY_BY_PERIOD[7];
  return {
    merged: period.totals.merged,
    waste: period.totals.waste,
    wastePct: period.totals.wastePct,
    costUsd: period.totals.costUsd,
    tokens: period.points.reduce((sum, point) => sum + point.tokens, 0),
    superplaneSharePct: period.totals.superplaneSharePct,
  };
}

function formatUsd(value: number) {
  return `$${value.toFixed(2)}`;
}

function formatTokens(value: number) {
  if (value >= 1000) {
    const thousands = value / 1000;
    return `${thousands % 1 === 0 ? thousands.toFixed(0) : thousands.toFixed(1)}k`;
  }
  return String(value);
}

export type OverviewScorecardMetric = { label: string; value: string; hint: string };

/** Four scorecard cells for the Overview velocity row. `null` renders placeholders (no data yet). */
export function overviewScorecardMetrics(summary: OverviewVelocitySummary | null): OverviewScorecardMetric[] {
  if (!summary) {
    return [
      { label: "Merged PRs", value: "—", hint: "No velocity data yet" },
      { label: "Waste", value: "—", hint: "No velocity data yet" },
      { label: "Cost", value: "—", hint: "No velocity data yet" },
      { label: "SuperPlane share", value: "—", hint: "No velocity data yet" },
    ];
  }
  return [
    { label: "Merged PRs", value: String(summary.merged), hint: "Merged in the last 7 days" },
    { label: "Waste", value: String(summary.waste), hint: `${summary.wastePct}% of SuperPlane output` },
    { label: "Cost", value: formatUsd(summary.costUsd), hint: `${formatTokens(summary.tokens)} tokens` },
    { label: "SuperPlane share", value: `${summary.superplaneSharePct}%`, hint: "Share of merged PRs" },
  ];
}
