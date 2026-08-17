import { ArrowUpRight } from "lucide-react";
import { Link } from "react-router";

import { cn } from "@/lib/utils";
import { factoryVelocityPath } from "../lib/factoryPagePaths";
import { factoryCardClassName } from "./factoryPageLayoutStyles";
import {
  buildOverviewVelocitySummary,
  overviewScorecardMetrics,
  type OverviewVelocitySummary,
} from "./overviewVelocitySummary";

/**
 * Scorecard row prototype for the workspace Overview: four velocity cards
 * (Merged PRs, Waste, Cost, SuperPlane share) plus a link out to the full
 * Velocity page. Mirrors the `MetricCell` styling on `VelocityPage`. With no
 * `summary`, cards render as dashes but the row still shows — Overview never
 * hides it.
 */
export function OverviewMetricsScorecardRow({
  organizationId,
  factoryKey,
  summary,
}: {
  organizationId: string;
  factoryKey: string;
  summary: OverviewVelocitySummary | null;
}) {
  const metrics = overviewScorecardMetrics(summary);

  return (
    <section className={cn("overflow-hidden", factoryCardClassName)} data-testid="overview-metrics-row">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div>
          <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Velocity</h2>
          <p className="text-[12px] text-muted-foreground">Factory output over the last 7 days.</p>
        </div>
        <Link
          to={factoryVelocityPath(organizationId, factoryKey)}
          className="inline-flex shrink-0 items-center gap-1 text-[12px] font-medium text-muted-foreground hover:text-foreground"
          data-testid="overview-metrics-view-velocity"
        >
          View velocity
          <ArrowUpRight className="h-3.5 w-3.5" aria-hidden />
        </Link>
      </div>

      <div className="grid grid-cols-2 gap-x-8 gap-y-5 px-4 py-5 sm:grid-cols-4">
        {metrics.map((metric) => (
          <div key={metric.label} className="min-w-0">
            <p className="text-[12px] text-muted-foreground">{metric.label}</p>
            <p className="mt-2 text-[32px] leading-none font-semibold tracking-[-0.04em] tabular-nums text-foreground">
              {metric.value}
            </p>
            <p className="mt-2 min-h-[1rem] text-[12px] text-muted-foreground">{metric.hint}</p>
          </div>
        ))}
      </div>
    </section>
  );
}

/** Storybook entry: populated velocity numbers, same totals as the Velocity page. */
export function OverviewMetricsScorecardRowPopulated({
  organizationId,
  factoryKey,
}: {
  organizationId: string;
  factoryKey: string;
}) {
  return (
    <OverviewMetricsScorecardRow
      organizationId={organizationId}
      factoryKey={factoryKey}
      summary={buildOverviewVelocitySummary()}
    />
  );
}

/** Storybook entry: no velocity data yet. Row still renders, values are placeholders. */
export function OverviewMetricsScorecardRowEmpty({
  organizationId,
  factoryKey,
}: {
  organizationId: string;
  factoryKey: string;
}) {
  return <OverviewMetricsScorecardRow organizationId={organizationId} factoryKey={factoryKey} summary={null} />;
}
