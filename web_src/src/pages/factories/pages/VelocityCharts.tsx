import { Area, AreaChart, Bar, BarChart, CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";

import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";

import {
  formatDurationHours,
  pickVelocityChartUnit,
  type FactoryVelocityFlow,
  type FactoryVelocityFlowPeriodDays,
} from "../lib/factoryVelocityFlow";

export type VelocityChartsPeriodDays = FactoryVelocityFlowPeriodDays;

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

const timeTrendChartConfig = {
  runningHours: { label: "Time running", color: "#60a5fa" },
  waitingHours: { label: "Time in Waiting", color: "#f59e0b" },
} satisfies ChartConfig;

function formatTimeTrendTooltip(value: unknown, name: unknown) {
  const hours = Array.isArray(value) ? Number(value[0]) : Number(value);
  const seriesKey = String(name);
  const label =
    seriesKey in timeTrendChartConfig
      ? timeTrendChartConfig[seriesKey as keyof typeof timeTrendChartConfig].label
      : seriesKey;

  return (
    <div className="flex w-full items-center justify-between gap-8">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-mono font-medium text-foreground tabular-nums">
        {Number.isFinite(hours) ? formatDurationHours(hours) : String(value)}
      </span>
    </div>
  );
}

export interface DailyOutputPoint {
  day: string;
  merged: number;
  waste: number;
}

export function DailyOutputChart({ points, days }: { points: DailyOutputPoint[]; days: VelocityChartsPeriodDays }) {
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

export interface CostSparklinePoint {
  day: string;
  costUsd: number;
}

export function CostSparkline({ points, days }: { points: CostSparklinePoint[]; days: VelocityChartsPeriodDays }) {
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

export interface SourceSplitPoint {
  day: string;
  peopleMerged: number;
  superplaneMerged: number;
}

export function SourceSplitChart({ points, days }: { points: SourceSplitPoint[]; days: VelocityChartsPeriodDays }) {
  const height = days === 7 ? 200 : 180;

  return (
    <ChartContainer
      config={sourceChartConfig}
      className="aspect-auto w-full"
      style={{ height }}
      initialDimension={{ width: 720, height }}
    >
      <LineChart data={points} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
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
        <Line type="monotone" dataKey="peopleMerged" stroke="var(--color-peoplemerged)" strokeWidth={2} dot={false} />
        <Line
          type="monotone"
          dataKey="superplaneMerged"
          stroke="var(--color-superplanemerged)"
          strokeWidth={2}
          dot={false}
        />
      </LineChart>
    </ChartContainer>
  );
}

export function TimeTrendChart({
  trend,
  days,
}: {
  trend: FactoryVelocityFlow["timeTrend"];
  days: VelocityChartsPeriodDays;
}) {
  const height = days === 7 ? 240 : 220;
  const chartUnit = pickVelocityChartUnit(trend.map((point) => point.runningHours + point.waitingHours));

  return (
    <ChartContainer
      config={timeTrendChartConfig}
      className="aspect-auto w-full"
      style={{ height }}
      initialDimension={{ width: 720, height }}
    >
      <AreaChart data={trend} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
        <XAxis dataKey="day" tickLine={false} axisLine={false} tickMargin={8} interval={0} className="text-[11px]" />
        <YAxis
          tickLine={false}
          axisLine={false}
          width={40}
          tickMargin={4}
          className="text-[11px]"
          tickFormatter={(value: number) => chartUnit.formatTick(Number(value))}
        />
        <ChartTooltip content={<ChartTooltipContent formatter={formatTimeTrendTooltip} />} />
        <ChartLegend content={<ChartLegendContent />} verticalAlign="bottom" />
        <Area
          type="monotone"
          dataKey="runningHours"
          stackId="day"
          stroke="var(--color-runninghours)"
          fill="var(--color-runninghours)"
          fillOpacity={0.35}
          strokeWidth={1.5}
        />
        <Area
          type="monotone"
          dataKey="waitingHours"
          stackId="day"
          stroke="var(--color-waitinghours)"
          fill="var(--color-waitinghours)"
          fillOpacity={0.35}
          strokeWidth={1.5}
        />
      </AreaChart>
    </ChartContainer>
  );
}
