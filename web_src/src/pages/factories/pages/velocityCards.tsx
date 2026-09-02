import type { ReactNode } from "react";
import { ArrowDownRight, ArrowUpRight, Info } from "lucide-react";

import { formatCompactTokenValue } from "@/lib/formatTokenCount";
import { cn } from "@/lib/utils";
import { SegmentedNav } from "@/ui/SegmentedNav";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/ui/tooltip";

import { formatDurationHours, type FactoryVelocityFlow } from "../lib/factoryVelocityFlow";
import {
  VELOCITY_BREAKDOWN_COPY,
  VELOCITY_BREAKDOWN_OPTIONS,
  type VelocityBreakdown,
  type VelocityIntakeSeries,
  type VelocityPoint,
  type VelocityTotals,
} from "../lib/factoryVelocityReport";
import { VELOCITY_TIME_COLORS } from "../lib/velocitySeriesColors";
import { CostChart, DeliveryChart, FlowChart } from "./VelocityCharts";

export const velocityCardClassName = "rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5";
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

function formatUsd(value: number) {
  return `$${value.toFixed(2)}`;
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
  tooltip,
  change,
}: {
  label: string;
  value: string;
  hint?: string;
  tooltip?: string;
  change?: MetricChange;
}) {
  return (
    <div className="min-w-0">
      <div className="flex items-center gap-1">
        <p className="text-[12px] text-muted-foreground">{label}</p>
        {tooltip ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className="shrink-0 rounded-sm text-muted-foreground hover:text-foreground"
                aria-label={`About ${label}`}
              >
                <Info className="size-3.5" aria-hidden />
              </button>
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-xs">
              {tooltip}
            </TooltipContent>
          </Tooltip>
        ) : null}
      </div>
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
 * below, so the chart needs no legend of its own.
 *
 * There is no share column: medians of the parts do not add up to the median of
 * the whole, so a percentage here would not be true.
 */
function SplitRow({ color, label, value }: { color: string; label: string; value: string }) {
  return (
    <div className="flex items-baseline gap-2 py-1.5">
      <span className="size-1.5 shrink-0 rounded-full" style={{ backgroundColor: color }} />
      <span className="text-[13px] text-foreground">{label}</span>
      <span className="ml-auto text-[13px] font-medium tabular-nums text-foreground">{value}</span>
    </div>
  );
}

/** Empty note used where a chart would otherwise draw an axis with no data. */
function ChartEmptyNote({ children }: { children: ReactNode }) {
  return <p className="mt-5 text-[13px] text-muted-foreground">{children}</p>;
}

/**
 * Deltas against the previous window. Each field is absent when the window
 * before this one holds no comparable sample.
 */
export interface VelocityComparison {
  tasksClosed?: number;
  taskWasteRate?: number;
  cycleHours?: number;
  costPerTask?: number;
}

export function SummaryCard({
  totals,
  caption,
  medianCycleHours,
  comparison,
}: {
  totals: VelocityTotals;
  caption: string;
  /** Median cycle time of the tasks that closed in this window. */
  medianCycleHours?: number;
  comparison?: VelocityComparison;
}) {
  return (
    <section className={velocityCardClassName} data-testid="velocity-summary">
      <p className="text-[12px] text-muted-foreground">{caption}</p>
      <div className="mt-4 grid grid-cols-2 gap-x-8 gap-y-6 lg:grid-cols-4">
        <Metric
          label="Tasks closed"
          value={String(totals.tasksClosed)}
          tooltip="Finished, with or without a result"
          change={
            comparison?.tasksClosed === undefined
              ? undefined
              : buildChange(comparison.tasksClosed, "up", (magnitude) => String(magnitude))
          }
        />
        <Metric
          label="Task waste"
          value={`${totals.taskWasteRate}%`}
          tooltip={`${totals.tasksWaste} ${totals.tasksWaste === 1 ? "task" : "tasks"} closed without a merge`}
          change={
            comparison?.taskWasteRate === undefined
              ? undefined
              : buildChange(comparison.taskWasteRate, "down", (magnitude) => `${magnitude} pp`)
          }
        />
        <Metric
          label="Median cycle time"
          value={medianCycleHours === undefined ? "—" : formatDurationHours(medianCycleHours)}
          tooltip="From task start to close"
          change={
            comparison?.cycleHours === undefined
              ? undefined
              : buildChange(comparison.cycleHours, "down", (magnitude) => formatDurationHours(magnitude))
          }
        />
        <Metric
          label="Cost per task"
          value={formatUsd(totals.costPerTask)}
          tooltip="Tracked model spend"
          change={
            comparison?.costPerTask === undefined
              ? undefined
              : buildChange(comparison.costPerTask, "down", (magnitude) => `$${magnitude.toFixed(2)}`)
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
  intakeSeries,
  hasOutput,
}: {
  points: VelocityPoint[];
  breakdown: VelocityBreakdown;
  onBreakdownChange: (breakdown: VelocityBreakdown) => void;
  intakeSeries: VelocityIntakeSeries[];
  hasOutput: boolean;
}) {
  const copy = VELOCITY_BREAKDOWN_COPY[breakdown];
  const canSplitByIntake = intakeSeries.length > 0;
  const options = VELOCITY_BREAKDOWN_OPTIONS.filter((option) => option.value !== "intake" || canSplitByIntake);

  return (
    <section className={velocityCardClassName} data-testid="velocity-delivery">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className={cardTitleClassName}>{copy.title}</h2>
          <p className={cardSubtitleClassName}>{copy.description}</p>
        </div>
        <SegmentedNav
          ariaLabel="Group merged pull requests by"
          size="xs"
          value={breakdown}
          onValueChange={(value) => onBreakdownChange(value as VelocityBreakdown)}
          options={options}
        />
      </div>
      {hasOutput ? (
        <div className="mt-5">
          <DeliveryChart points={points} breakdown={breakdown} intakeSeries={intakeSeries} />
        </div>
      ) : (
        <ChartEmptyNote>No pull requests merged or closed in this period.</ChartEmptyNote>
      )}
    </section>
  );
}

export function TaskTimeCard({
  flow,
  emptyLabel,
}: {
  flow: FactoryVelocityFlow | null;
  /** Shown instead of the numbers when no task closed, or when loading failed. */
  emptyLabel?: string;
}) {
  const hasSample = flow !== null && flow.sampleSize > 0;

  return (
    <section className={velocityCardClassName} data-testid="velocity-task-time">
      <h2 className={cardTitleClassName}>Task time</h2>
      <p className={cardSubtitleClassName}>Median time for SuperPlane tasks that closed in this period.</p>

      {hasSample ? (
        <>
          <div className="mt-5">
            <Metric
              label="Cycle time"
              value={formatDurationHours(flow.medianCycleHours)}
              hint={`From ${flow.sampleSize} ${flow.sampleSize === 1 ? "task" : "tasks"} closed in this period`}
            />
          </div>

          <div className="mt-4 border-t border-border pt-2">
            <SplitRow
              color={VELOCITY_TIME_COLORS.running}
              label="Time running"
              value={formatDurationHours(flow.medianRunningHours)}
            />
            <SplitRow
              color={VELOCITY_TIME_COLORS.waiting}
              label="Time in Waiting"
              value={formatDurationHours(flow.medianWaitingHours)}
            />
            <p className="mt-1.5 text-[12px] text-muted-foreground">
              Time in Waiting is review or a pause before the next dispatch, not time an agent runs.
            </p>
          </div>

          <div className="mt-5 border-t border-border pt-4">
            <FlowChart trend={flow.timeTrend} />
          </div>
        </>
      ) : (
        <ChartEmptyNote>{emptyLabel ?? "No tasks closed in this period."}</ChartEmptyNote>
      )}
    </section>
  );
}

export function CostCard({ totals, points }: { totals: VelocityTotals; points: VelocityPoint[] }) {
  const hasCost = totals.costUsd > 0;

  return (
    <section className={velocityCardClassName} data-testid="velocity-cost">
      <h2 className={cardTitleClassName}>Tracked SuperPlane cost</h2>
      <p className={cardSubtitleClassName}>Model spend of the tasks that closed. Third-party charges are excluded.</p>

      {hasCost ? (
        <>
          <div className="mt-5">
            <Metric
              label="Total cost"
              value={formatUsd(totals.costUsd)}
              hint={`${formatCompactTokenValue(totals.tokens)} tokens`}
            />
          </div>

          <div className="mt-4 border-t border-border pt-2">
            <p className="text-[12px] text-muted-foreground">
              {formatUsd(totals.wasteCostUsd)} of this went to tasks that closed without a merge.
            </p>
          </div>

          <div className="mt-5 border-t border-border pt-4">
            <CostChart points={points} />
          </div>
        </>
      ) : (
        <ChartEmptyNote>No tracked model spend in this period.</ChartEmptyNote>
      )}
    </section>
  );
}
