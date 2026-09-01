import { Area, AreaChart, Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";

import { formatDurationHours, pickVelocityChartUnit, type FactoryVelocityFlow } from "../lib/factoryVelocityFlow";
import {
  velocityBreakdownSeries,
  type VelocityBreakdown,
  type VelocityIntakeSeries,
  type VelocityPoint,
} from "../lib/factoryVelocityReport";
import { VELOCITY_COST_COLOR, VELOCITY_TIME_COLORS } from "../lib/velocitySeriesColors";

const flowChartConfig = {
  runningHours: { label: "Time running", color: VELOCITY_TIME_COLORS.running },
  waitingHours: { label: "Time in Waiting", color: VELOCITY_TIME_COLORS.waiting },
} satisfies ChartConfig;

const costChartConfig = {
  costUsd: { label: "Tracked cost", color: VELOCITY_COST_COLOR },
} satisfies ChartConfig;

/** Both area charts sit side by side, so they must read as one visual family. */
const AREA_FILL_OPACITY = 0.3;
const AREA_STROKE_WIDTH = 1.5;

/** Row of the delivery chart: one day, one value per visible band. */
type DeliveryRow = Record<string, string | number>;

function deliveryValue(point: VelocityPoint, key: string, breakdown: VelocityBreakdown): number {
  if (breakdown === "intake") {
    return point.intake[key] ?? 0;
  }
  const value = point[key as keyof VelocityPoint];
  return typeof value === "number" ? value : 0;
}

export function DeliveryChart({
  points,
  breakdown,
  intakeSeries,
}: {
  points: VelocityPoint[];
  breakdown: VelocityBreakdown;
  intakeSeries: VelocityIntakeSeries[];
}) {
  const series = velocityBreakdownSeries(breakdown, intakeSeries);
  const config = Object.fromEntries(
    series.map((item) => [item.key, { label: item.label, color: item.color }]),
  ) satisfies ChartConfig;

  const rows: DeliveryRow[] = points.map((point) => {
    const row: DeliveryRow = { day: point.day };
    for (const item of series) {
      row[item.key] = deliveryValue(point, item.key, breakdown);
    }
    return row;
  });

  return (
    <ChartContainer
      config={config}
      className="aspect-auto h-[260px] w-full"
      initialDimension={{ width: 760, height: 260 }}
    >
      <BarChart data={rows} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
        <XAxis dataKey="day" tickLine={false} axisLine={false} tickMargin={8} interval={0} className="text-[11px]" />
        <YAxis allowDecimals={false} tickLine={false} axisLine={false} width={28} className="text-[11px]" />
        <ChartTooltip content={<ChartTooltipContent />} />
        <ChartLegend content={<ChartLegendContent />} verticalAlign="bottom" />
        {series.map((item, index) => (
          <Bar
            key={item.key}
            dataKey={item.key}
            name={item.label}
            stackId="day"
            fill={item.color}
            radius={index === series.length - 1 ? [3, 3, 0, 0] : undefined}
            maxBarSize={24}
          />
        ))}
      </BarChart>
    </ChartContainer>
  );
}

function formatFlowTooltip(value: unknown, name: unknown) {
  const hours = Array.isArray(value) ? Number(value[0]) : Number(value);
  const key = String(name);
  const label = key in flowChartConfig ? flowChartConfig[key as keyof typeof flowChartConfig].label : key;

  return (
    <div className="flex w-full items-center justify-between gap-8">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-mono font-medium text-foreground tabular-nums">
        {Number.isFinite(hours) ? formatDurationHours(hours) : String(value)}
      </span>
    </div>
  );
}

export function FlowChart({ trend }: { trend: FactoryVelocityFlow["timeTrend"] }) {
  const chartUnit = pickVelocityChartUnit(trend.map((point) => point.runningHours + point.waitingHours));

  return (
    <ChartContainer
      config={flowChartConfig}
      className="aspect-auto h-[180px] w-full"
      initialDimension={{ width: 500, height: 180 }}
    >
      <AreaChart data={trend} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
        <XAxis dataKey="day" tickLine={false} axisLine={false} tickMargin={8} interval={0} className="text-[11px]" />
        <YAxis
          tickLine={false}
          axisLine={false}
          width={40}
          className="text-[11px]"
          tickFormatter={(value: number) => chartUnit.formatTick(Number(value))}
        />
        <ChartTooltip content={<ChartTooltipContent formatter={formatFlowTooltip} />} />
        {/* No legend: the labeled split above the chart names both bands. */}
        <Area
          type="monotone"
          dataKey="runningHours"
          stackId="time"
          stroke={VELOCITY_TIME_COLORS.running}
          fill={VELOCITY_TIME_COLORS.running}
          fillOpacity={AREA_FILL_OPACITY}
          strokeWidth={AREA_STROKE_WIDTH}
        />
        <Area
          type="monotone"
          dataKey="waitingHours"
          stackId="time"
          stroke={VELOCITY_TIME_COLORS.waiting}
          fill={VELOCITY_TIME_COLORS.waiting}
          fillOpacity={AREA_FILL_OPACITY}
          strokeWidth={AREA_STROKE_WIDTH}
        />
      </AreaChart>
    </ChartContainer>
  );
}

export function CostChart({ points }: { points: VelocityPoint[] }) {
  return (
    <ChartContainer
      config={costChartConfig}
      className="aspect-auto h-[180px] w-full"
      initialDimension={{ width: 500, height: 180 }}
    >
      <AreaChart data={points} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
        <XAxis dataKey="day" tickLine={false} axisLine={false} tickMargin={8} interval={0} className="text-[11px]" />
        <YAxis
          tickLine={false}
          axisLine={false}
          width={38}
          className="text-[11px]"
          tickFormatter={(value: number) => `$${Number(value).toFixed(0)}`}
        />
        <ChartTooltip content={<ChartTooltipContent />} />
        {/* No legend: one band, already named by the total above the chart. */}
        <Area
          type="monotone"
          dataKey="costUsd"
          stroke={VELOCITY_COST_COLOR}
          fill={VELOCITY_COST_COLOR}
          fillOpacity={AREA_FILL_OPACITY}
          strokeWidth={AREA_STROKE_WIDTH}
        />
      </AreaChart>
    </ChartContainer>
  );
}
