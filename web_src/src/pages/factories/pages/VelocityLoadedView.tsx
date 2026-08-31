import type { ReactNode } from "react";

import { formatCompactTokenValue } from "@/lib/formatTokenCount";

import {
  formatDurationHours,
  type FactoryVelocityFlow,
  type FactoryVelocityFlowPeriodDays,
} from "../lib/factoryVelocityFlow";
import { VELOCITY_SERIES_COLORS } from "../lib/velocitySeriesColors";
import {
  CostSparkline,
  DailyOutputChart,
  SourceSplitChart,
  TimeTrendChart,
  type CostSparklinePoint,
} from "./VelocityCharts";

/** Every Velocity card shares one frame, so the page reads as one report. */
const cardClassName = "rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5";

const cardTitleClassName = "text-[14px] font-medium tracking-[-0.01em] text-foreground";

const cardSubtitleClassName = "mt-0.5 text-[12px] text-muted-foreground";

/**
 * Metric rows stop short of the full card width. Two or three numbers spread
 * over a wide card drift apart and stop reading as one group.
 */
const metricRowClassName = "mt-5 grid grid-cols-2 gap-x-8 gap-y-5";

export type VelocityPeriodDays = FactoryVelocityFlowPeriodDays;

export interface VelocityYesterday {
  dateLabel: string;
  merged: number;
  waste: number;
  wastePct: number;
}

export interface VelocityTotals {
  merged: number;
  waste: number;
  wastePct: number;
  superplaneMerged: number;
  peopleMerged: number;
  superplaneSharePct: number;
}

export interface VelocityDayPoint {
  day: string;
  merged: number;
  waste: number;
  peopleMerged: number;
  superplaneMerged: number;
}

export interface VelocityData {
  yesterday: VelocityYesterday;
  totals: VelocityTotals;
  points: VelocityDayPoint[];
}

export interface VelocitySourceSplitConfig {
  hasPeopleCohort: boolean;
  repositoryLabel?: string;
  emptyState?: ReactNode;
}

export interface VelocityWorkOrderFlowConfig {
  flow: FactoryVelocityFlow | null;
  emptyLabel?: string;
}

export interface VelocityCostConfig {
  yesterdayCostUsd: number;
  totalCostUsd: number;
  seriesUsd: number[];
  yesterdayTokens: number;
  yesterdayCostPerMerged: number;
}

export interface VelocityLoadedViewProps {
  periodLabel: string;
  periodDays: VelocityPeriodDays;
  data: VelocityData;
  sourceSplit: VelocitySourceSplitConfig;
  workOrderFlow?: VelocityWorkOrderFlowConfig;
  cost?: VelocityCostConfig;
}

function formatUsd(value: number) {
  return `$${value.toFixed(2)}`;
}

function formatTokens(value: number) {
  return formatCompactTokenValue(value);
}

function formatPct(value: number) {
  return `${value}%`;
}

interface MetricCellProps {
  value: string;
  label: string;
  hint?: string;
  /** Series color, when the number also appears as a line or bar in a chart. */
  seriesColor?: string;
}

function MetricCell({ value, label, hint, seriesColor }: MetricCellProps) {
  return (
    <div className="min-w-0">
      <p className="flex items-center gap-1.5 text-[12px] text-muted-foreground">
        {seriesColor ? (
          <span
            className="size-1.5 shrink-0 rounded-full"
            style={{ backgroundColor: seriesColor }}
            aria-hidden="true"
          />
        ) : null}
        {label}
      </p>
      <p className="mt-2 text-[32px] leading-none font-semibold tracking-[-0.04em] tabular-nums text-foreground">
        {value}
      </p>
      {hint ? <p className="mt-2 min-h-[1rem] text-[12px] text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

/** Empty note used where a chart would otherwise draw an axis with no data. */
function ChartEmptyNote({ children }: { children: ReactNode }) {
  return <p className="mt-5 text-[13px] text-muted-foreground">{children}</p>;
}

interface YesterdayCardProps {
  snapshot: VelocityYesterday;
  cost?: { costUsd: number; tokens: number; costPerMerged: number };
}

function YesterdayCard({ snapshot, cost }: YesterdayCardProps) {
  const metrics: MetricCellProps[] = [
    {
      value: String(snapshot.merged),
      label: "Merged PRs",
      hint: "Productive SuperPlane work",
      seriesColor: VELOCITY_SERIES_COLORS.merged,
    },
    {
      value: String(snapshot.waste),
      label: "Waste",
      hint: `${formatPct(snapshot.wastePct)} of SuperPlane output`,
      seriesColor: VELOCITY_SERIES_COLORS.waste,
    },
  ];
  if (cost) {
    metrics.push(
      {
        value: formatUsd(cost.costUsd),
        label: "Cost",
        hint: `${formatTokens(cost.tokens)} tokens`,
        seriesColor: VELOCITY_SERIES_COLORS.cost,
      },
      { value: formatUsd(cost.costPerMerged), label: "Cost per merged PR", hint: "Average spend" },
    );
  }

  const gridColsClass = cost ? "sm:grid-cols-4" : "sm:grid-cols-2 sm:max-w-2xl";

  return (
    <section className={cardClassName} data-testid="velocity-yesterday">
      <h2 className={cardTitleClassName}>{snapshot.dateLabel}</h2>
      <div className={`${metricRowClassName} ${gridColsClass}`}>
        {metrics.map((metric) => (
          <MetricCell
            key={metric.label}
            value={metric.value}
            label={metric.label}
            hint={metric.hint}
            seriesColor={metric.seriesColor}
          />
        ))}
      </div>
    </section>
  );
}

interface TrendCardProps {
  periodLabel: string;
  periodDays: VelocityPeriodDays;
  totals: VelocityTotals;
  points: VelocityDayPoint[];
  cost?: { totalCostUsd: number; seriesUsd: number[] };
}

function TrendCard({ periodLabel, periodDays, totals, points, cost }: TrendCardProps) {
  const gridColsClass = cost ? "sm:grid-cols-3" : "sm:grid-cols-2 sm:max-w-2xl";
  const costPoints: CostSparklinePoint[] = cost
    ? points.map((point, index) => ({ day: point.day, costUsd: cost.seriesUsd[index] ?? 0 }))
    : [];
  const hasOutput = totals.merged > 0 || totals.waste > 0;

  return (
    <section className={cardClassName} data-testid="velocity-trend">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className={cardTitleClassName}>SuperPlane output</h2>
          <p className={cardSubtitleClassName}>
            {cost ? "Merged, waste, and cost by day." : "Merged and waste by day."}
          </p>
        </div>
        <p className="text-[12px] text-muted-foreground">{periodLabel}</p>
      </div>

      <div className={`${metricRowClassName} ${gridColsClass}`}>
        <MetricCell value={String(totals.merged)} label="Merged PRs" seriesColor={VELOCITY_SERIES_COLORS.merged} />
        <MetricCell
          value={String(totals.waste)}
          label="Waste"
          hint={`${formatPct(totals.wastePct)} of SuperPlane output`}
          seriesColor={VELOCITY_SERIES_COLORS.waste}
        />
        {cost ? (
          <MetricCell value={formatUsd(cost.totalCostUsd)} label="Cost" seriesColor={VELOCITY_SERIES_COLORS.cost} />
        ) : null}
      </div>

      {hasOutput ? (
        <div className="mt-6">
          <DailyOutputChart points={points} days={periodDays} />
        </div>
      ) : (
        <ChartEmptyNote>No pull requests merged or closed in this period.</ChartEmptyNote>
      )}

      {cost ? (
        <div className="mt-4 border-t border-border pt-3">
          <p className="mb-1 text-[12px] text-muted-foreground">Cost</p>
          <CostSparkline points={costPoints} days={periodDays} />
        </div>
      ) : null}
    </section>
  );
}

interface SourceSplitCardProps {
  periodDays: VelocityPeriodDays;
  totals: VelocityTotals;
  points: VelocityDayPoint[];
  config: VelocitySourceSplitConfig;
}

function SourceSplitCard({ periodDays, totals, points, config }: SourceSplitCardProps) {
  const hasMerges = totals.peopleMerged > 0 || totals.superplaneMerged > 0;

  return (
    <section className={cardClassName} data-testid="velocity-source-split">
      <div>
        <h2 className={cardTitleClassName}>Merged PRs by source</h2>
        {config.hasPeopleCohort ? (
          <p className={cardSubtitleClassName}>
            SuperPlane authored {formatPct(totals.superplaneSharePct)} of merged PRs in
            {config.repositoryLabel ? ` ${config.repositoryLabel}` : " this repository"} in this period.
          </p>
        ) : (
          <p className={cardSubtitleClassName}>People and SuperPlane merges in the workspace repository.</p>
        )}
      </div>

      {config.hasPeopleCohort ? (
        <>
          <div className={`${metricRowClassName} sm:max-w-2xl`}>
            <MetricCell
              value={String(totals.peopleMerged)}
              label="People"
              seriesColor={VELOCITY_SERIES_COLORS.people}
            />
            <MetricCell
              value={String(totals.superplaneMerged)}
              label="SuperPlane"
              seriesColor={VELOCITY_SERIES_COLORS.superplane}
            />
          </div>

          {hasMerges ? (
            <div className="mt-6">
              <SourceSplitChart points={points} days={periodDays} />
            </div>
          ) : (
            <ChartEmptyNote>No pull requests merged in this period.</ChartEmptyNote>
          )}
        </>
      ) : (
        <div className="mt-4">{config.emptyState}</div>
      )}
    </section>
  );
}

interface WorkOrderFlowCardProps {
  periodLabel: string;
  periodDays: VelocityPeriodDays;
  config: VelocityWorkOrderFlowConfig;
}

function WorkOrderFlowCard({ periodLabel, periodDays, config }: WorkOrderFlowCardProps) {
  const flow = config.flow;
  const emptyLabel = config.emptyLabel ?? "No tasks closed in this period.";

  return (
    <section className={cardClassName} data-testid="velocity-work-order-flow">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className={cardTitleClassName}>Task time</h2>
          <p className={cardSubtitleClassName}>
            Median times for tasks that closed in this period. Cycle time is time running plus time in Waiting after the
            task leaves Draft.
          </p>
        </div>
        <p className="text-[12px] text-muted-foreground">{periodLabel}</p>
      </div>

      {flow && flow.sampleSize > 0 ? (
        <WorkOrderFlowBody flow={flow} periodDays={periodDays} />
      ) : (
        <ChartEmptyNote>{emptyLabel}</ChartEmptyNote>
      )}
    </section>
  );
}

function WorkOrderFlowBody({ flow, periodDays }: { flow: FactoryVelocityFlow; periodDays: VelocityPeriodDays }) {
  return (
    <>
      <div className="mt-5">
        <div className="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3">
          <MetricCell
            value={formatDurationHours(flow.medianCycleHours)}
            label="Cycle time"
            hint="From start to close"
          />
          <MetricCell
            value={formatDurationHours(flow.medianRunningHours)}
            label="Time running"
            hint={`${flow.runningShareOfCyclePct}% of cycle time`}
            seriesColor={VELOCITY_SERIES_COLORS.running}
          />
          <MetricCell
            value={formatDurationHours(flow.medianWaitingHours)}
            label="Time in Waiting"
            hint={`${flow.waitingShareOfCyclePct}% of cycle time`}
            seriesColor={VELOCITY_SERIES_COLORS.waiting}
          />
        </div>
        <p className="mt-3 text-[12px] text-muted-foreground">
          Time running is time on a line. Time in Waiting is review or a pause before the next dispatch.
        </p>
      </div>

      <div className="mt-6 border-t border-border pt-5">
        <h3 className="text-[13px] font-medium text-foreground">Time running and Time in Waiting by day</h3>
        <p className="mb-3 mt-0.5 text-[12px] text-muted-foreground">
          Median for tasks that closed on that day. The stacked area is the two parts of cycle time.
        </p>
        <TimeTrendChart trend={flow.timeTrend} days={periodDays} />
      </div>
    </>
  );
}

export function VelocityLoadedView({
  periodLabel,
  periodDays,
  data,
  sourceSplit,
  workOrderFlow,
  cost,
}: VelocityLoadedViewProps) {
  return (
    <>
      <YesterdayCard
        snapshot={data.yesterday}
        cost={
          cost
            ? {
                costUsd: cost.yesterdayCostUsd,
                tokens: cost.yesterdayTokens,
                costPerMerged: cost.yesterdayCostPerMerged,
              }
            : undefined
        }
      />
      <TrendCard
        periodLabel={periodLabel}
        periodDays={periodDays}
        totals={data.totals}
        points={data.points}
        cost={cost ? { totalCostUsd: cost.totalCostUsd, seriesUsd: cost.seriesUsd } : undefined}
      />
      <SourceSplitCard periodDays={periodDays} totals={data.totals} points={data.points} config={sourceSplit} />
      {workOrderFlow ? (
        <WorkOrderFlowCard periodLabel={periodLabel} periodDays={periodDays} config={workOrderFlow} />
      ) : null}
    </>
  );
}
