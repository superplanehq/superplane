import { useMemo, useState } from "react";
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { SegmentedNav } from "@/ui/SegmentedNav";

export type VelocityPeriodDays = 7 | 30 | 90;

type VelocityPeriodStats = {
  days: VelocityPeriodDays;
  label: string;
  runs: number;
  succeeded: number;
  failed: number;
  /** Average tokens per completed run in the period. */
  tokensPerRun: number;
  /** Estimated token spend per completed run (USD). */
  tokenSpendPerRun: number;
  /** Estimated VM spend per completed run (USD). */
  vmSpendPerRun: number;
};

const PERIOD_OPTIONS: { value: string; label: string; days: VelocityPeriodDays }[] = [
  { value: "7", label: "7d", days: 7 },
  { value: "30", label: "30d", days: 30 },
  { value: "90", label: "90d", days: 90 },
];

/** Storybook / UI mock stats by selected window. */
export const VELOCITY_BY_PERIOD: Record<VelocityPeriodDays, VelocityPeriodStats> = {
  7: {
    days: 7,
    label: "Last 7 days",
    runs: 42,
    succeeded: 28,
    failed: 5,
    tokensPerRun: 18400,
    tokenSpendPerRun: 0.42,
    vmSpendPerRun: 1.18,
  },
  30: {
    days: 30,
    label: "Last 30 days",
    runs: 168,
    succeeded: 121,
    failed: 22,
    tokensPerRun: 17200,
    tokenSpendPerRun: 0.39,
    vmSpendPerRun: 1.05,
  },
  90: {
    days: 90,
    label: "Last 90 days",
    runs: 512,
    succeeded: 381,
    failed: 64,
    tokensPerRun: 16900,
    tokenSpendPerRun: 0.37,
    vmSpendPerRun: 0.98,
  },
};

const dailyVelocityChartConfig = {
  succeeded: { label: "Succeeded", color: "#10b981" },
  failed: { label: "Failed", color: "#ef4444" },
  inProgress: { label: "Still in progress", color: "#60a5fa" },
} satisfies ChartConfig;

type DailyVelocityPoint = {
  day: string;
  succeeded: number;
  failed: number;
  inProgress: number;
};

function formatTokens(value: number) {
  if (value >= 1000) {
    const thousands = value / 1000;
    return `${thousands % 1 === 0 ? thousands.toFixed(0) : thousands.toFixed(1)}k`;
  }
  return String(value);
}

function formatUsd(value: number) {
  return `$${value.toFixed(2)}`;
}

function formatPct(part: number, whole: number) {
  if (whole <= 0) return "0%";
  return `${Math.round((part / whole) * 100)}%`;
}

function buildDayLabels(days: VelocityPeriodDays): string[] {
  if (days === 7) return ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

  const labels: string[] = [];
  for (let index = 0; index < days; index += 1) {
    const dayNumber = index + 1;
    if (days === 30) {
      labels.push(dayNumber % 5 === 1 || dayNumber === days ? String(dayNumber) : "");
      continue;
    }
    labels.push(dayNumber % 15 === 1 || dayNumber === days ? String(dayNumber) : "");
  }
  return labels;
}

/**
 * Build stacked daily outcomes for the selected window.
 * Still-in-progress only on the last two days when the period has open work.
 */
function buildDailyVelocity(stats: VelocityPeriodStats): DailyVelocityPoint[] {
  const open = Math.max(0, stats.runs - stats.succeeded - stats.failed);
  const dayCount = stats.days;
  const labels = buildDayLabels(stats.days);

  const intakeWeights = Array.from({ length: dayCount }, (_, index) => {
    const wave = 0.85 + 0.3 * Math.sin((index / dayCount) * Math.PI * 2);
    return wave;
  });
  const weightSum = intakeWeights.reduce((sum, weight) => sum + weight, 0);
  const normalized = intakeWeights.map((weight) => weight / weightSum);

  const distribute = (total: number) => {
    const values = normalized.map((weight) => Math.round(total * weight));
    const drift = total - values.reduce((sum, value) => sum + value, 0);
    values[values.length - 1] = (values[values.length - 1] ?? 0) + drift;
    return values.map((value) => Math.max(0, value));
  };

  const entered = distribute(stats.runs);
  const closedTotal = stats.succeeded + stats.failed;
  const successShareOfClosed = closedTotal > 0 ? stats.succeeded / closedTotal : 1;
  const openDayIndexes = [dayCount - 2, dayCount - 1];

  const points: DailyVelocityPoint[] = labels.map((day, index) => {
    const dayEntered = entered[index] ?? 0;
    const isOpenDay = open > 0 && openDayIndexes.includes(index);
    let dayInProgress = isOpenDay ? Math.min(dayEntered, Math.round(dayEntered * (index === dayCount - 1 ? 0.55 : 0.35))) : 0;
    let closed = dayEntered - dayInProgress;
    let daySucceeded = Math.round(closed * successShareOfClosed);
    let dayFailed = closed - daySucceeded;
    if (dayFailed < 0) {
      daySucceeded = closed;
      dayFailed = 0;
    }
    const used = daySucceeded + dayFailed + dayInProgress;
    if (used !== dayEntered) {
      if (isOpenDay) dayInProgress = Math.max(0, dayInProgress + (dayEntered - used));
      else daySucceeded = Math.max(0, daySucceeded + (dayEntered - used));
    }
    return {
      day: day || "·",
      succeeded: daySucceeded,
      failed: dayFailed,
      inProgress: dayInProgress,
    };
  });

  const sum = (key: keyof Omit<DailyVelocityPoint, "day">) => points.reduce((total, point) => total + point[key], 0);

  const shift = (
    from: "succeeded" | "failed" | "inProgress",
    to: "succeeded" | "failed" | "inProgress",
    amount: number,
    dayIndexes: number[],
  ) => {
    let remaining = amount;
    for (const index of dayIndexes) {
      if (remaining <= 0) break;
      const point = points[index];
      if (!point) continue;
      const move = Math.min(remaining, point[from]);
      if (move <= 0) continue;
      point[from] -= move;
      point[to] += move;
      remaining -= move;
    }
  };

  const closedDays = Array.from({ length: dayCount }, (_, index) => index).filter(
    (index) => !openDayIndexes.includes(index),
  );

  let openDelta = open - sum("inProgress");
  if (openDelta > 0) {
    shift("succeeded", "inProgress", openDelta, openDayIndexes);
    openDelta = open - sum("inProgress");
    if (openDelta > 0) shift("failed", "inProgress", openDelta, openDayIndexes);
  } else if (openDelta < 0) {
    shift("inProgress", "succeeded", -openDelta, openDayIndexes);
  }

  for (const index of closedDays) {
    const point = points[index];
    if (!point || point.inProgress === 0) continue;
    point.succeeded += point.inProgress;
    point.inProgress = 0;
  }

  let successDelta = stats.succeeded - sum("succeeded");
  if (successDelta > 0) shift("failed", "succeeded", successDelta, closedDays);
  else if (successDelta < 0) shift("succeeded", "failed", -successDelta, closedDays);

  if (open > 0) {
    for (const index of openDayIndexes) {
      const point = points[index];
      if (!point || point.inProgress > 0) continue;
      if (point.succeeded > 0) {
        point.succeeded -= 1;
        point.inProgress += 1;
      } else if (point.failed > 0) {
        point.failed -= 1;
        point.inProgress += 1;
      }
    }
  }

  // Restore blank tick labels for sparse axes (chart still needs a stable key).
  return points.map((point, index) => ({
    ...point,
    day: labels[index] || "",
  }));
}

function DailyVelocityChart({ stats }: { stats: VelocityPeriodStats }) {
  const points = useMemo(() => buildDailyVelocity(stats), [stats]);
  const chartHeight = stats.days === 7 ? 260 : 240;

  return (
    <ChartContainer
      config={dailyVelocityChartConfig}
      className="aspect-auto w-full"
      style={{ height: chartHeight }}
      initialDimension={{ width: 720, height: chartHeight }}
    >
      <BarChart data={points} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
        <XAxis dataKey="day" tickLine={false} axisLine={false} tickMargin={8} interval={0} className="text-[11px]" />
        <YAxis
          allowDecimals={false}
          tickLine={false}
          axisLine={false}
          width={28}
          tickMargin={4}
          className="text-[11px]"
        />
        <ChartTooltip content={<ChartTooltipContent />} />
        <ChartLegend content={<ChartLegendContent />} verticalAlign="bottom" />
        <Bar dataKey="succeeded" stackId="day" fill="var(--color-succeeded)" maxBarSize={stats.days === 7 ? 36 : 12} />
        <Bar dataKey="failed" stackId="day" fill="var(--color-failed)" maxBarSize={stats.days === 7 ? 36 : 12} />
        <Bar
          dataKey="inProgress"
          stackId="day"
          fill="var(--color-inprogress)"
          radius={[3, 3, 0, 0]}
          maxBarSize={stats.days === 7 ? 36 : 12}
        />
      </BarChart>
    </ChartContainer>
  );
}

function OutcomeSplitBar({ stats }: { stats: VelocityPeriodStats }) {
  const open = Math.max(0, stats.runs - stats.succeeded - stats.failed);
  if (stats.runs <= 0) return null;

  const succeededPct = (stats.succeeded / stats.runs) * 100;
  const failedPct = (stats.failed / stats.runs) * 100;
  const openPct = (open / stats.runs) * 100;

  return (
    <div
      className="flex h-1.5 w-full overflow-hidden rounded-full bg-muted"
      role="img"
      aria-label={`Succeeded ${formatPct(stats.succeeded, stats.runs)}, failed ${formatPct(stats.failed, stats.runs)}${open > 0 ? `, still in progress ${formatPct(open, stats.runs)}` : ""}`}
    >
      {succeededPct > 0 ? (
        <span className="h-full bg-[#10b981]" style={{ width: `${succeededPct}%` }} />
      ) : null}
      {failedPct > 0 ? <span className="h-full bg-[#ef4444]" style={{ width: `${failedPct}%` }} /> : null}
      {openPct > 0 ? <span className="h-full bg-[#60a5fa]" style={{ width: `${openPct}%` }} /> : null}
    </div>
  );
}

function MetricCell({
  value,
  label,
  hint,
  emphasize,
}: {
  value: string;
  label: string;
  hint?: string;
  emphasize?: boolean;
}) {
  return (
    <div className="min-w-0">
      <p
        className={
          emphasize
            ? "text-[28px] leading-none font-semibold tracking-[-0.03em] tabular-nums text-foreground"
            : "text-[22px] leading-none font-semibold tracking-[-0.02em] tabular-nums text-foreground"
        }
      >
        {value}
      </p>
      <p className="mt-1.5 text-[12px] text-muted-foreground">{label}</p>
      {hint ? <p className="mt-0.5 text-[12px] tabular-nums text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

/** Cursor Usage–style hero metrics + daily chart in one card. */
function VelocityOverviewCard({ stats }: { stats: VelocityPeriodStats }) {
  const open = Math.max(0, stats.runs - stats.succeeded - stats.failed);

  return (
    <section className="rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5" data-testid="velocity-run-stats">
      <div className="flex flex-wrap items-end gap-x-8 gap-y-4">
        <MetricCell value={String(stats.runs)} label="Runs" hint={stats.label} emphasize />
        <MetricCell
          value={String(stats.succeeded)}
          label="Succeeded"
          hint={formatPct(stats.succeeded, stats.runs)}
        />
        <MetricCell value={String(stats.failed)} label="Failed" hint={formatPct(stats.failed, stats.runs)} />
        {open > 0 ? (
          <MetricCell value={String(open)} label="Still in progress" hint={formatPct(open, stats.runs)} />
        ) : null}
      </div>

      <div className="mt-4">
        <OutcomeSplitBar stats={stats} />
      </div>

      <div className="mt-6 border-t border-border pt-4">
        <div className="mb-3">
          <h2 className="text-[14px] font-medium tracking-[-0.01em] text-foreground">Daily velocity</h2>
          <p className="mt-0.5 text-[12px] text-muted-foreground">Succeeded, failed, and still in progress by day.</p>
        </div>
        <DailyVelocityChart stats={stats} />
      </div>
    </section>
  );
}

/** Cursor Spending–style cost rows. */
function CostPerRun({ stats }: { stats: VelocityPeriodStats }) {
  return (
    <section className="space-y-3" data-testid="velocity-cost-per-run">
      <div>
        <h2 className="text-[14px] font-medium tracking-[-0.01em] text-foreground">Cost per run</h2>
        <p className="mt-0.5 text-[12px] text-muted-foreground">Average for completed runs in this period.</p>
      </div>
      <div className="overflow-hidden rounded-xl border border-border bg-card">
        <div className="flex items-start justify-between gap-4 px-4 py-3.5 sm:px-5">
          <div className="min-w-0">
            <p className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Tokens</p>
            <p className="mt-0.5 text-[12px] text-muted-foreground">Model usage for the run</p>
          </div>
          <div className="shrink-0 text-right">
            <p className="text-[15px] font-semibold tracking-[-0.02em] tabular-nums text-foreground">
              {formatTokens(stats.tokensPerRun)}
            </p>
            <p className="mt-0.5 text-[12px] tabular-nums text-muted-foreground">{formatUsd(stats.tokenSpendPerRun)}</p>
          </div>
        </div>
        <div className="flex items-start justify-between gap-4 border-t border-border px-4 py-3.5 sm:px-5">
          <div className="min-w-0">
            <p className="text-[13px] font-medium tracking-[-0.01em] text-foreground">VM spend</p>
            <p className="mt-0.5 text-[12px] text-muted-foreground">Compute for the run</p>
          </div>
          <div className="shrink-0 text-right">
            <p className="text-[15px] font-semibold tracking-[-0.02em] tabular-nums text-foreground">
              {formatUsd(stats.vmSpendPerRun)}
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}

export function LineVelocityPanel({
  intro = "Run throughput for this automation.",
  defaultPeriodDays = 7,
}: {
  intro?: string;
  defaultPeriodDays?: VelocityPeriodDays;
}) {
  const [periodDays, setPeriodDays] = useState<VelocityPeriodDays>(defaultPeriodDays);
  const stats = VELOCITY_BY_PERIOD[periodDays];

  return (
    <div className="space-y-6" data-testid="line-velocity-panel">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-[15px] font-semibold tracking-[-0.01em] text-foreground">Velocity</h1>
          <p className="mt-0.5 text-[13px] text-muted-foreground">{intro}</p>
        </div>
        <SegmentedNav
          ariaLabel="Velocity period in days"
          size="xs"
          value={String(periodDays)}
          onValueChange={(value) => {
            const next = Number(value);
            if (next === 7 || next === 30 || next === 90) setPeriodDays(next);
          }}
          options={PERIOD_OPTIONS.map(({ value, label }) => ({ value, label }))}
        />
      </div>

      <VelocityOverviewCard stats={stats} />
      <CostPerRun stats={stats} />
    </div>
  );
}
