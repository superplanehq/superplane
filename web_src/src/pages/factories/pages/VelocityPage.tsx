import { useState } from "react";
import { Area, AreaChart, Bar, BarChart, CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { cn } from "@/lib/utils";
import { SegmentedNav } from "@/ui/SegmentedNav";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import {
  FACTORY_VELOCITY_BY_PERIOD,
  FACTORY_VELOCITY_YESTERDAY,
  type FactoryVelocityDay,
  type FactoryVelocityPeriodDays,
  type FactoryVelocityYesterday,
} from "./factoryVelocityMockData";
import { factoryContentBodyClassName } from "./factoryPageLayoutStyles";

const PERIOD_OPTIONS: { value: string; label: string }[] = [
  { value: "7", label: "7d" },
  { value: "30", label: "30d" },
];

const outputChartConfig = {
  merged: { label: "Merged", color: "#10b981" },
  waste: { label: "Waste", color: "#ef4444" },
} satisfies ChartConfig;

const costChartConfig = {
  costUsd: { label: "Cost", color: "#64748b" },
} satisfies ChartConfig;

const sourceChartConfig = {
  peopleMerged: { label: "People", color: "#64748b" },
  superplaneMerged: { label: "SuperPlane", color: "#10b981" },
} satisfies ChartConfig;

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

type YesterdayMetric = {
  value: string;
  label: string;
  hint: string;
};

function yesterdayMetrics(snapshot: FactoryVelocityYesterday): YesterdayMetric[] {
  return [
    { value: String(snapshot.merged), label: "Merged PRs", hint: "Productive SuperPlane work" },
    { value: String(snapshot.waste), label: "Waste", hint: `${formatPct(snapshot.wastePct)} of SuperPlane output` },
    { value: formatUsd(snapshot.costUsd), label: "Cost", hint: `${formatTokens(snapshot.tokens)} tokens` },
    { value: formatUsd(snapshot.costPerMergedPr), label: "Cost per merged PR", hint: "Average spend" },
  ];
}

function YesterdayCard({ snapshot }: { snapshot: FactoryVelocityYesterday }) {
  const metrics = yesterdayMetrics(snapshot);

  return (
    <section
      className="rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5"
      data-testid="velocity-yesterday"
    >
      <p className="text-[12px] font-medium text-foreground">{snapshot.dateLabel}</p>
      <div className="mt-5 grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-4">
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

function DailyOutputChart({ points, days }: { points: FactoryVelocityDay[]; days: FactoryVelocityPeriodDays }) {
  const height = days === 7 ? 240 : 220;

  return (
    <ChartContainer
      config={outputChartConfig}
      className="aspect-auto w-full"
      style={{ height }}
      initialDimension={{ width: 720, height }}
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
        <Bar dataKey="merged" stackId="day" fill="var(--color-merged)" maxBarSize={days === 7 ? 36 : 12} />
        <Bar
          dataKey="waste"
          stackId="day"
          fill="var(--color-waste)"
          radius={[3, 3, 0, 0]}
          maxBarSize={days === 7 ? 36 : 12}
        />
      </BarChart>
    </ChartContainer>
  );
}

function CostSparkline({ points, days }: { points: FactoryVelocityDay[]; days: FactoryVelocityPeriodDays }) {
  const height = days === 7 ? 180 : 160;

  return (
    <ChartContainer
      config={costChartConfig}
      className="aspect-auto w-full"
      style={{ height }}
      initialDimension={{ width: 720, height }}
    >
      <AreaChart data={points} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
        <XAxis dataKey="day" tickLine={false} axisLine={false} tickMargin={8} interval={0} className="text-[11px]" />
        <YAxis
          tickLine={false}
          axisLine={false}
          width={44}
          tickMargin={4}
          className="text-[11px]"
          tickFormatter={(value: number) => `$${Number(value).toFixed(0)}`}
        />
        <ChartTooltip content={<ChartTooltipContent />} />
        <Area
          type="monotone"
          dataKey="costUsd"
          stroke="var(--color-costusd)"
          fill="var(--color-costusd)"
          fillOpacity={0.12}
          strokeWidth={1.5}
        />
      </AreaChart>
    </ChartContainer>
  );
}

function TrendCard({ periodDays }: { periodDays: FactoryVelocityPeriodDays }) {
  const period = FACTORY_VELOCITY_BY_PERIOD[periodDays];

  return (
    <section className="rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5" data-testid="velocity-trend">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-[14px] font-medium tracking-[-0.01em] text-foreground">SuperPlane output</h2>
          <p className="mt-0.5 text-[12px] text-muted-foreground">Merged, waste, and cost by day.</p>
        </div>
        <p className="text-[12px] text-muted-foreground">{period.label}</p>
      </div>

      <div className="mt-5 grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3">
        <MetricCell value={String(period.totals.merged)} label="Merged PRs" />
        <MetricCell
          value={String(period.totals.waste)}
          label="Waste"
          hint={`${formatPct(period.totals.wastePct)} of SuperPlane output`}
        />
        <MetricCell value={formatUsd(period.totals.costUsd)} label="Cost" />
      </div>

      <div className="mt-6">
        <DailyOutputChart points={period.points} days={period.days} />
      </div>

      <div className="mt-4 border-t border-border pt-3">
        <p className="mb-1 text-[12px] text-muted-foreground">Cost</p>
        <CostSparkline points={period.points} days={period.days} />
      </div>
    </section>
  );
}

function SourceSplitCard({ periodDays }: { periodDays: FactoryVelocityPeriodDays }) {
  const period = FACTORY_VELOCITY_BY_PERIOD[periodDays];
  const height = period.days === 7 ? 200 : 180;

  return (
    <section
      className="rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5"
      data-testid="velocity-source-split"
    >
      <div>
        <h2 className="text-[14px] font-medium tracking-[-0.01em] text-foreground">Merged PRs by source</h2>
        <p className="mt-0.5 text-[12px] text-muted-foreground">
          SuperPlane authored {formatPct(period.totals.superplaneSharePct)} of merged PRs in this period.
        </p>
      </div>

      <div className="mt-5 grid grid-cols-2 gap-x-8 gap-y-5">
        <MetricCell value={String(period.totals.peopleMerged)} label="People" />
        <MetricCell value={String(period.totals.superplaneMerged)} label="SuperPlane" />
      </div>

      <div className="mt-6">
        <ChartContainer
          config={sourceChartConfig}
          className="aspect-auto w-full"
          style={{ height }}
          initialDimension={{ width: 720, height }}
        >
          <LineChart data={period.points} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
            <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
            <XAxis
              dataKey="day"
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              interval={0}
              className="text-[11px]"
            />
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
            <Line
              type="monotone"
              dataKey="peopleMerged"
              stroke="var(--color-peoplemerged)"
              strokeWidth={2}
              dot={false}
            />
            <Line
              type="monotone"
              dataKey="superplaneMerged"
              stroke="var(--color-superplanemerged)"
              strokeWidth={2}
              dot={false}
            />
          </LineChart>
        </ChartContainer>
      </div>
    </section>
  );
}

export function VelocityPage() {
  const [periodDays, setPeriodDays] = useState<FactoryVelocityPeriodDays>(7);

  return (
    <>
      <WorkspacePageHeader
        title="Velocity"
        subtitle="Merged pull requests from SuperPlane, waste, and cost."
        actions={
          <SegmentedNav
            ariaLabel="Velocity period in days"
            size="xs"
            value={String(periodDays)}
            onValueChange={(value) => {
              const next = Number(value);
              if (next === 7 || next === 30) setPeriodDays(next);
            }}
            options={PERIOD_OPTIONS}
          />
        }
      />

      <div className={cn(factoryContentBodyClassName, "space-y-6")} data-testid="factory-velocity-page">
        <YesterdayCard snapshot={FACTORY_VELOCITY_YESTERDAY} />
        <TrendCard periodDays={periodDays} />
        <SourceSplitCard periodDays={periodDays} />
      </div>
    </>
  );
}
