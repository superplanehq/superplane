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
import { formatCompactTokenValue } from "@/lib/formatTokenCount";
import { SegmentedNav } from "@/ui/SegmentedNav";
import { buildDailyVelocity } from "./buildDailyVelocity";
import { VELOCITY_BY_PERIOD, type VelocityPeriodDays, type VelocityPeriodStats } from "./lineVelocityMockData";

const PERIOD_OPTIONS: { value: string; label: string; days: VelocityPeriodDays }[] = [
  { value: "7", label: "7d", days: 7 },
  { value: "30", label: "30d", days: 30 },
  { value: "90", label: "90d", days: 90 },
];

const dailyVelocityChartConfig = {
  succeeded: { label: "Succeeded", color: "#10b981" },
  failed: { label: "Failed", color: "#ef4444" },
  inProgress: { label: "Still in progress", color: "#60a5fa" },
} satisfies ChartConfig;

function formatTokens(value: number) {
  return formatCompactTokenValue(value);
}

function formatUsd(value: number) {
  return `$${value.toFixed(2)}`;
}

function formatPct(part: number, whole: number) {
  if (whole <= 0) return "0%";
  return `${Math.round((part / whole) * 100)}%`;
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
      {succeededPct > 0 ? <span className="h-full bg-[#10b981]" style={{ width: `${succeededPct}%` }} /> : null}
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
    <section
      className="rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5"
      data-testid="velocity-run-stats"
    >
      <div className="flex flex-wrap items-end gap-x-8 gap-y-4">
        <MetricCell value={String(stats.runs)} label="Runs" hint={stats.label} emphasize />
        <MetricCell value={String(stats.succeeded)} label="Succeeded" hint={formatPct(stats.succeeded, stats.runs)} />
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
