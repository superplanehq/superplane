import { Link } from "react-router";

import { cn } from "@/lib/utils";

import { factoryCardClassName } from "../factoryPageLayoutStyles";
import type { HealthMetric, MetricTone } from "./overviewRedesignMocks";

const METRIC_TONE_CLASS: Record<MetricTone, string> = {
  positive: "text-emerald-600 dark:text-emerald-400",
  negative: "text-red-600 dark:text-red-400",
  neutral: "text-muted-foreground",
};

const PLACEHOLDER_LABELS = ["Merged PRs", "Waste", "Cost", "SuperPlane share"];

/**
 * Workspace health section at the top of the page body: a labeled header
 * row (title, period, and Velocity link) above a horizontal row of metric
 * scorecards. Without metrics (fresh workspace), the cards show
 * placeholders so the page keeps its structure.
 */
export function HealthScorecards({ metrics, velocityHref }: { metrics?: HealthMetric[]; velocityHref: string }) {
  return (
    <section data-testid="overview-health-scorecards">
      <div className="mb-2 flex items-baseline justify-between gap-3">
        <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Workspace health</h2>
        <div className="flex items-center gap-2 text-[12px] text-muted-foreground">
          <span>Last 7 days</span>
          <span aria-hidden className="text-border">
            ·
          </span>
          <Link to={velocityHref} className="font-medium hover:text-foreground" data-testid="overview-velocity-link">
            View velocity
          </Link>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {metrics
          ? metrics.map((metric) => (
              <div key={metric.id} className={cn(factoryCardClassName, "px-4 py-3")}>
                <p className="text-[12px] text-muted-foreground">{metric.label}</p>
                <p className="mt-1 text-[22px] font-semibold tabular-nums leading-tight text-foreground">
                  {metric.value}
                </p>
                <p className={cn("mt-0.5 text-[12px] tabular-nums", METRIC_TONE_CLASS[metric.tone])}>{metric.delta}</p>
              </div>
            ))
          : PLACEHOLDER_LABELS.map((label) => (
              <div key={label} className={cn(factoryCardClassName, "px-4 py-3")}>
                <p className="text-[12px] text-muted-foreground">{label}</p>
                <p className="mt-1 text-[22px] font-semibold tabular-nums leading-tight text-muted-foreground">—</p>
                <p className="mt-0.5 text-[12px] text-muted-foreground">No data yet</p>
              </div>
            ))}
      </div>
    </section>
  );
}
