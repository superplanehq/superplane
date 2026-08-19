import type { ReactNode } from "react";

import type { FactoryVelocityFlow, FactoryVelocityFlowPeriodDays } from "../lib/factoryVelocityFlow";
import { useVelocityDurationFormat } from "../lib/velocityDurationFormatSlot";
import {
  CostSparkline,
  DailyOutputChart,
  SourceSplitChart,
  TimeTrendChart,
  type CostSparklinePoint,
} from "./VelocityCharts";

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
  if (value >= 1000) {
    const thousands = value / 1000;
    return `${thousands % 1 === 0 ? thousands.toFixed(0) : thousands.toFixed(1)}k`;
  }
  return String(value);
}

function formatPct(value: number) {
  return `${value}%`;
}

function MetricCell({ value, label, hint }: { value: string; label: string; hint?: string }) {
  return (
    <div className="min-w-0">
      <p className="text-[12px] text-muted-foreground">{label}</p>
      <p className="mt-2 text-[32px] leading-none font-semibold tracking-[-0.04em] tabular-nums text-foreground">
        {value}
      </p>
      {hint ? <p className="mt-2 min-h-[1rem] text-[12px] text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

interface YesterdayCardProps {
  snapshot: VelocityYesterday;
  cost?: { costUsd: number; tokens: number; costPerMerged: number };
}

function YesterdayCard({ snapshot, cost }: YesterdayCardProps) {
  const metrics: { value: string; label: string; hint: string }[] = [
    { value: String(snapshot.merged), label: "Merged PRs", hint: "Productive SuperPlane work" },
    { value: String(snapshot.waste), label: "Waste", hint: `${formatPct(snapshot.wastePct)} of SuperPlane output` },
  ];
  if (cost) {
    metrics.push(
      { value: formatUsd(cost.costUsd), label: "Cost", hint: `${formatTokens(cost.tokens)} tokens` },
      { value: formatUsd(cost.costPerMerged), label: "Cost per merged PR", hint: "Average spend" },
    );
  }

  const gridColsClass = cost ? "sm:grid-cols-4" : "sm:grid-cols-2";

  return (
    <section
      className="rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5"
      data-testid="velocity-yesterday"
    >
      <p className="text-[12px] font-medium text-foreground">{snapshot.dateLabel}</p>
      <div className={`mt-5 grid grid-cols-2 gap-x-8 gap-y-5 ${gridColsClass}`}>
        {metrics.map((metric) => (
          <MetricCell key={metric.label} value={metric.value} label={metric.label} hint={metric.hint} />
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
  const gridColsClass = cost ? "sm:grid-cols-3" : "sm:grid-cols-2";
  const costPoints: CostSparklinePoint[] = cost
    ? points.map((point, index) => ({ day: point.day, costUsd: cost.seriesUsd[index] ?? 0 }))
    : [];

  return (
    <section className="rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5" data-testid="velocity-trend">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-[14px] font-medium tracking-[-0.01em] text-foreground">SuperPlane output</h2>
          <p className="mt-0.5 text-[12px] text-muted-foreground">
            {cost ? "Merged, waste, and cost by day." : "Merged and waste by day."}
          </p>
        </div>
        <p className="text-[12px] text-muted-foreground">{periodLabel}</p>
      </div>

      <div className={`mt-5 grid grid-cols-2 gap-x-8 gap-y-5 ${gridColsClass}`}>
        <MetricCell value={String(totals.merged)} label="Merged PRs" />
        <MetricCell
          value={String(totals.waste)}
          label="Waste"
          hint={`${formatPct(totals.wastePct)} of SuperPlane output`}
        />
        {cost ? <MetricCell value={formatUsd(cost.totalCostUsd)} label="Cost" /> : null}
      </div>

      <div className="mt-6">
        <DailyOutputChart points={points} days={periodDays} />
      </div>

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
  return (
    <section
      className="rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5"
      data-testid="velocity-source-split"
    >
      <div>
        <h2 className="text-[14px] font-medium tracking-[-0.01em] text-foreground">Merged PRs by source</h2>
        {config.hasPeopleCohort ? (
          <p className="mt-0.5 text-[12px] text-muted-foreground">
            SuperPlane authored {formatPct(totals.superplaneSharePct)} of merged PRs in
            {config.repositoryLabel ? ` ${config.repositoryLabel}` : " this repository"} in this period.
          </p>
        ) : (
          <p className="mt-0.5 text-[12px] text-muted-foreground">Select a repository to see People vs SuperPlane.</p>
        )}
      </div>

      {config.hasPeopleCohort ? (
        <>
          <div className="mt-5 grid grid-cols-2 gap-x-8 gap-y-5">
            <MetricCell value={String(totals.peopleMerged)} label="People" />
            <MetricCell value={String(totals.superplaneMerged)} label="SuperPlane" />
          </div>

          <div className="mt-6">
            <SourceSplitChart points={points} days={periodDays} />
          </div>
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

export function WorkOrderFlowCard({ periodLabel, periodDays, config }: WorkOrderFlowCardProps) {
  const flow = config.flow;
  const emptyLabel = config.emptyLabel ?? "No work orders closed in this period.";

  return (
    <section
      className="rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5"
      data-testid="velocity-work-order-flow"
    >
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-[14px] font-medium tracking-[-0.01em] text-foreground">Work order time</h2>
          <p className="mt-0.5 text-[12px] text-muted-foreground">
            Median times for work orders that closed in this period. Cycle time is time running plus time in Waiting
            after the work order leaves Draft.
          </p>
        </div>
        <p className="text-[12px] text-muted-foreground">{periodLabel}</p>
      </div>

      {flow && flow.sampleSize > 0 ? (
        <WorkOrderFlowBody flow={flow} periodDays={periodDays} />
      ) : (
        <p className="mt-5 text-[13px] text-muted-foreground">{emptyLabel}</p>
      )}
    </section>
  );
}

function WorkOrderFlowBody({ flow, periodDays }: { flow: FactoryVelocityFlow; periodDays: VelocityPeriodDays }) {
  const durationFormat = useVelocityDurationFormat();

  return (
    <>
      <div className="mt-5">
        <div className="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3">
          <MetricCell
            value={durationFormat.formatDuration(flow.medianCycleHours)}
            label="Cycle time"
            hint="From start to close"
          />
          <MetricCell
            value={durationFormat.formatDuration(flow.medianRunningHours)}
            label="Time running"
            hint={`${flow.runningShareOfCyclePct}% of cycle time`}
          />
          <MetricCell
            value={durationFormat.formatDuration(flow.medianWaitingHours)}
            label="Time in Waiting"
            hint={`${flow.waitingShareOfCyclePct}% of cycle time`}
          />
        </div>
        <p className="mt-3 text-[12px] text-muted-foreground">
          Time running is time on a line. Time in Waiting is review or a pause before the next dispatch.
        </p>
      </div>

      <div className="mt-6 border-t border-border pt-5">
        <h3 className="text-[13px] font-medium text-foreground">Time running and Time in Waiting by day</h3>
        <p className="mb-3 mt-0.5 text-[12px] text-muted-foreground">
          Median for work orders that closed on that day. The stacked area is the two parts of cycle time.
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
