import type { ReactNode } from "react";
import { ArrowDownRight, ArrowUpRight } from "lucide-react";

import { formatCompactTokenValue } from "@/lib/formatTokenCount";
import { cn } from "@/lib/utils";
import { SegmentedNav } from "@/ui/SegmentedNav";

import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { factoryCenteredSectionBodyClassName, factoryCenteredSectionHeaderClassName } from "./factoryPageLayoutStyles";
import { CostChart, DeliveryChart, FlowChart } from "./VelocityPrototypeCharts";
import {
  BREAKDOWN_COPY,
  BREAKDOWN_OPTIONS,
  COST_SERIES_COLORS,
  PERIOD_OPTIONS,
  TIME_SERIES_COLORS,
  WORKSPACE_REPOSITORY,
  type Breakdown,
  type PeriodDays,
  type VelocityPoint,
  type VelocitySummary,
} from "./velocityPrototypeData";

export const cardClassName = "rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5";
const cardTitleClassName = "text-[14px] font-medium tracking-[-0.01em] text-foreground";
const cardSubtitleClassName = "mt-0.5 text-[12px] text-muted-foreground";

type ChangeBetter = "up" | "down";

/** The arrow shows which way the number moved. The color shows whether that is good. */
interface MetricChange {
  text: string;
  tone: "better" | "worse" | "same";
  direction: "up" | "down" | "flat";
}

const CHANGE_TONE_CLASS: Record<MetricChange["tone"], string> = {
  better: "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400",
  worse: "bg-rose-50 text-rose-700 dark:bg-rose-950/60 dark:text-rose-400",
  same: "bg-muted text-muted-foreground",
};

function formatHours(value: number) {
  return `${Math.round(value)}h`;
}

function formatUsd(value: number) {
  return `$${value.toFixed(2)}`;
}

function costShare(part: number, total: number): number {
  if (total <= 0) return 0;
  return Math.round((part / total) * 100);
}

function buildChange(delta: number, better: ChangeBetter, format: (magnitude: number) => string): MetricChange {
  if (delta === 0) return { text: "No change", tone: "same", direction: "flat" };

  const improved = better === "up" ? delta > 0 : delta < 0;
  return {
    text: format(Math.abs(delta)),
    tone: improved ? "better" : "worse",
    direction: delta > 0 ? "up" : "down",
  };
}

function ChangeChip({ change }: { change: MetricChange }) {
  const Icon = change.direction === "up" ? ArrowUpRight : ArrowDownRight;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-[11px] font-medium tabular-nums",
        CHANGE_TONE_CLASS[change.tone],
      )}
    >
      {change.direction === "flat" ? null : <Icon className="size-3" aria-hidden />}
      {change.text}
    </span>
  );
}

function Metric({
  label,
  value,
  hint,
  change,
}: {
  label: string;
  value: string;
  hint?: string;
  change?: MetricChange;
}) {
  return (
    <div className="min-w-0">
      <p className="text-[12px] text-muted-foreground">{label}</p>
      <div className="mt-2 flex flex-wrap items-baseline gap-x-2 gap-y-1">
        <p className="text-[30px] leading-none font-semibold tracking-[-0.04em] tabular-nums text-foreground">
          {value}
        </p>
        {change ? <ChangeChip change={change} /> : null}
      </div>
      {hint ? <p className="mt-2 text-[12px] text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

/**
 * One line of a card split. The dot ties the number to its band in the chart
 * below, so neither chart needs its own legend.
 *
 * `share` is only for splits that partition a total. Medians do not sum to the
 * median of the total, so the time split leaves it out.
 */
function SplitRow({
  color,
  label,
  value,
  share,
  hint,
}: {
  color: string;
  label: string;
  value: string;
  share?: number;
  hint?: string;
}) {
  return (
    <div className="flex items-baseline gap-2 py-1.5">
      <span className="size-1.5 shrink-0 rounded-full" style={{ backgroundColor: color }} />
      <span className="text-[13px] text-foreground">{label}</span>
      {hint ? <span className="text-[12px] text-muted-foreground">{hint}</span> : null}
      <span className="ml-auto text-[13px] font-medium tabular-nums text-foreground">{value}</span>
      {share === undefined ? null : (
        <span className="w-9 text-right text-[12px] tabular-nums text-muted-foreground">{share}%</span>
      )}
    </div>
  );
}

export function VelocityPrototypeChrome({
  periodDays,
  onPeriodDaysChange,
  children,
}: {
  periodDays: PeriodDays;
  onPeriodDaysChange: (days: PeriodDays) => void;
  children: ReactNode;
}) {
  return (
    /* Gray page, white cards. Dark mode keeps its darker page, because the
       factories theme paints the sidebar and cards the same color. */
    <div className="min-h-full bg-sidebar dark:bg-background">
      <WorkspacePageHeader
        className={cn(factoryCenteredSectionHeaderClassName, "bg-transparent")}
        title="Velocity"
        subtitle={`What ${WORKSPACE_REPOSITORY} ships, how long the work takes, and what it costs.`}
        actions={
          <SegmentedNav
            ariaLabel="Velocity period"
            size="xs"
            value={String(periodDays)}
            onValueChange={(value) => onPeriodDaysChange(value === "30" ? 30 : 14)}
            options={PERIOD_OPTIONS}
          />
        }
      />

      <div className={cn(factoryCenteredSectionBodyClassName, "space-y-5 pb-10")} data-testid="factory-velocity-page">
        {children}
      </div>
    </div>
  );
}

/** Deltas against the previous window. Omit them when there is no history. */
export interface VelocityComparison {
  merged: number;
  wasteRate: number;
  cycleHours: number;
  costPerMerge: number;
}

export function SummaryCard({
  summary,
  caption,
  comparison,
}: {
  summary: VelocitySummary;
  caption: string;
  comparison?: VelocityComparison;
}) {
  return (
    <section className={cardClassName}>
      <p className="text-[12px] text-muted-foreground">{caption}</p>
      <div className="mt-4 grid grid-cols-2 gap-x-8 gap-y-6 lg:grid-cols-4">
        <Metric
          label="Merged PRs"
          value={String(summary.merged)}
          hint="People and SuperPlane"
          change={comparison ? buildChange(comparison.merged, "up", (magnitude) => String(magnitude)) : undefined}
        />
        <Metric
          label="Factory waste"
          value={`${summary.wasteRate}%`}
          hint={`${summary.waste} PRs closed without merge`}
          change={comparison ? buildChange(comparison.wasteRate, "down", (magnitude) => `${magnitude} pp`) : undefined}
        />
        <Metric
          label="Median cycle time"
          value={formatHours(summary.cycleHours)}
          hint="From start to close"
          change={comparison ? buildChange(comparison.cycleHours, "down", (magnitude) => `${magnitude}h`) : undefined}
        />
        <Metric
          label="Cost per Factory merge"
          value={formatUsd(summary.costPerMerge)}
          hint="Tokens and execution compute"
          change={
            comparison
              ? buildChange(comparison.costPerMerge, "down", (magnitude) => `$${magnitude.toFixed(2)}`)
              : undefined
          }
        />
      </div>
    </section>
  );
}

export function DeliveryCard({
  points,
  breakdown,
  onBreakdownChange,
}: {
  points: VelocityPoint[];
  breakdown: Breakdown;
  onBreakdownChange: (breakdown: Breakdown) => void;
}) {
  const copy = BREAKDOWN_COPY[breakdown];

  return (
    <section className={cardClassName}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className={cardTitleClassName}>{copy.title}</h2>
          <p className={cardSubtitleClassName}>{copy.description}</p>
        </div>
        <SegmentedNav
          ariaLabel="Group merged pull requests by"
          size="xs"
          value={breakdown}
          onValueChange={(value) => onBreakdownChange(value as Breakdown)}
          options={BREAKDOWN_OPTIONS}
        />
      </div>
      <div className="mt-5">
        <DeliveryChart points={points} breakdown={breakdown} />
      </div>
    </section>
  );
}

export function TaskTimeCard({
  summary,
  points,
  sampleNote,
}: {
  summary: VelocitySummary;
  points: VelocityPoint[];
  /** Names the sample behind the median when it is too small to trust alone. */
  sampleNote?: string;
}) {
  return (
    <section className={cardClassName}>
      <h2 className={cardTitleClassName}>Task time</h2>
      <p className={cardSubtitleClassName}>Median time for Factory tasks that closed in this period.</p>

      <div className="mt-5">
        <Metric label="Cycle time" value={formatHours(summary.cycleHours)} hint={sampleNote} />
      </div>

      <div className="mt-4 border-t border-border pt-2">
        <SplitRow color={TIME_SERIES_COLORS.running} label="Time running" value={formatHours(summary.runningHours)} />
        <SplitRow color={TIME_SERIES_COLORS.waiting} label="Time waiting" value={formatHours(summary.waitingHours)} />
        <p className="mt-1.5 text-[12px] text-muted-foreground">
          Waiting is review and queue time, not time the agent runs.
        </p>
      </div>

      <div className="mt-5 border-t border-border pt-4">
        <FlowChart points={points} />
      </div>
    </section>
  );
}

export function CostCard({
  summary,
  points,
  sampleNote,
}: {
  summary: VelocitySummary;
  points: VelocityPoint[];
  sampleNote?: string;
}) {
  return (
    <section className={cardClassName}>
      <h2 className={cardTitleClassName}>Tracked Factory cost</h2>
      <p className={cardSubtitleClassName}>Model tokens and execution compute. Third-party charges are excluded.</p>

      <div className="mt-5">
        <Metric label="Total cost" value={formatUsd(summary.cost)} hint={sampleNote} />
      </div>

      <div className="mt-4 border-t border-border pt-2">
        <SplitRow
          color={COST_SERIES_COLORS.tokens}
          label="Model tokens"
          hint={formatCompactTokenValue(summary.tokens)}
          value={formatUsd(summary.tokenCost)}
          share={costShare(summary.tokenCost, summary.cost)}
        />
        <SplitRow
          color={COST_SERIES_COLORS.compute}
          label="Execution compute"
          value={formatUsd(summary.computeCost)}
          share={costShare(summary.computeCost, summary.cost)}
        />
        <p className="mt-1.5 text-[12px] text-muted-foreground">
          {formatUsd(summary.wasteCost)} of this went to Factory work that closed without a merge.
        </p>
      </div>

      <div className="mt-5 border-t border-border pt-4">
        <CostChart points={points} />
      </div>
    </section>
  );
}
