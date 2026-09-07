import { Area, AreaChart, Bar, BarChart, CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";

import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { useElementWidth } from "@/hooks/useElementWidth";

import { formatDurationHours, pickVelocityChartUnit, type FactoryVelocityFlow } from "../lib/factoryVelocityFlow";
import {
  velocityBreakdownSeries,
  type VelocityBreakdown,
  type VelocityCostMode,
  type VelocityIntakeSeries,
  type VelocityPoint,
} from "../lib/factoryVelocityReport";
import { pickVelocityAxisTicks } from "../lib/velocityAxisTicks";
import { VELOCITY_COST_COLORS, VELOCITY_TIME_COLORS } from "../lib/velocitySeriesColors";

const flowChartConfig = {
  runningHours: { label: "Time running", color: VELOCITY_TIME_COLORS.running },
  waitingHours: { label: "Time in Waiting", color: VELOCITY_TIME_COLORS.waiting },
} satisfies ChartConfig;

const costChartConfig = {
  modelCostUsd: { label: "Tokens", color: VELOCITY_COST_COLORS.model },
  computeCostUsd: { label: "Compute", color: VELOCITY_COST_COLORS.compute },
} satisfies ChartConfig;

/** Both area charts span a full row, one above the other, so they must read as one visual family. */
const AREA_FILL_OPACITY = 0.3;
const AREA_STROKE_WIDTH = 1.5;
const AREA_CHART_HEIGHT = 220;

/** Width assumed before the container reports its own, matching a full-row card. */
const AREA_CHART_FALLBACK_WIDTH = 1160;

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

  const { ref, width } = useElementWidth<HTMLDivElement>(760);
  const ticks = pickVelocityAxisTicks(
    points.map((point) => point.day),
    width,
  );

  return (
    <div ref={ref} className="w-full">
      <ChartContainer
        config={config}
        className="aspect-auto h-[260px] w-full"
        initialDimension={{ width: 760, height: 260 }}
      >
        <BarChart data={rows} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
          <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
          <XAxis
            dataKey="day"
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            interval={0}
            ticks={ticks}
            className="text-[11px]"
          />
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
    </div>
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
  const { ref, width } = useElementWidth<HTMLDivElement>(AREA_CHART_FALLBACK_WIDTH);
  const ticks = pickVelocityAxisTicks(
    trend.map((point) => point.day),
    width,
  );

  return (
    <div ref={ref} className="w-full">
      <ChartContainer
        config={flowChartConfig}
        className="aspect-auto h-[220px] w-full"
        initialDimension={{ width: AREA_CHART_FALLBACK_WIDTH, height: AREA_CHART_HEIGHT }}
      >
        <AreaChart data={trend} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
          <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
          <XAxis
            dataKey="day"
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            interval={0}
            ticks={ticks}
            className="text-[11px]"
          />
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
    </div>
  );
}

function formatCostTooltip(value: unknown, name: unknown) {
  const usd = Array.isArray(value) ? Number(value[0]) : Number(value);
  const key = String(name);
  const label = key in costChartConfig ? costChartConfig[key as keyof typeof costChartConfig].label : key;

  return (
    <div className="flex w-full items-center justify-between gap-8">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-mono font-medium text-foreground tabular-nums">
        {Number.isFinite(usd) ? formatUsd(usd) : String(value)}
      </span>
    </div>
  );
}

function formatUsd(value: number): string {
  return `$${value.toFixed(2)}`;
}

/**
 * Cents matter on a median task that cost a few dollars, whole dollars are
 * enough on a month of spend, so the axis follows the numbers it labels.
 */
function costTickFormatter(rows: CostRow[]): (value: number) => string {
  const max = Math.max(0, ...rows.map((row) => row.modelCostUsd + row.computeCostUsd));
  const digits = max < 10 ? 1 : 0;
  return (value: number) => `$${Number(value).toFixed(digits)}`;
}

/** Row of a cost chart: one day, split into both bands. */
interface CostRow {
  day: string;
  modelCostUsd: number;
  computeCostUsd: number;
}

/** Grid, axes, tooltip, and legend shared by both cost charts. */
function useCostChartAxes(rows: CostRow[]) {
  const { ref, width } = useElementWidth<HTMLDivElement>(AREA_CHART_FALLBACK_WIDTH);
  const ticks = pickVelocityAxisTicks(
    rows.map((row) => row.day),
    width,
  );

  const axes = (
    <>
      <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
      <XAxis
        dataKey="day"
        tickLine={false}
        axisLine={false}
        tickMargin={8}
        interval={0}
        ticks={ticks}
        className="text-[11px]"
      />
      <YAxis
        tickLine={false}
        axisLine={false}
        width={44}
        className="text-[11px]"
        tickFormatter={costTickFormatter(rows)}
      />
      <ChartTooltip content={<ChartTooltipContent formatter={formatCostTooltip} />} />
      <ChartLegend content={<ChartLegendContent />} verticalAlign="bottom" />
    </>
  );

  return { ref, axes };
}

const costChartContainerProps = {
  config: costChartConfig,
  className: "aspect-auto h-[220px] w-full",
  initialDimension: { width: AREA_CHART_FALLBACK_WIDTH, height: AREA_CHART_HEIGHT },
};

/** Running totals, so the reader sees what the period has cost by each day. */
function cumulativeCostRows(points: VelocityPoint[]): CostRow[] {
  let modelCostUsd = 0;
  let computeCostUsd = 0;

  return points.map((point) => {
    modelCostUsd += point.cost.modelCostUsd;
    computeCostUsd += point.cost.computeCostUsd;
    return { day: point.day, modelCostUsd, computeCostUsd };
  });
}

/** Spend, stacked, because the two bands add up to the total they sit under. */
export function CostChart({ points, mode }: { points: VelocityPoint[]; mode: VelocityCostMode }) {
  const rows: CostRow[] =
    mode === "cumulative" ? cumulativeCostRows(points) : points.map((point) => ({ day: point.day, ...point.cost }));
  const { ref, axes } = useCostChartAxes(rows);

  return (
    <div ref={ref} className="w-full">
      <ChartContainer {...costChartContainerProps}>
        <AreaChart data={rows} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
          {axes}
          <Area
            type="monotone"
            dataKey="computeCostUsd"
            stackId="cost"
            stroke={VELOCITY_COST_COLORS.compute}
            fill={VELOCITY_COST_COLORS.compute}
            fillOpacity={AREA_FILL_OPACITY}
            strokeWidth={AREA_STROKE_WIDTH}
          />
          <Area
            type="monotone"
            dataKey="modelCostUsd"
            stackId="cost"
            stroke={VELOCITY_COST_COLORS.model}
            fill={VELOCITY_COST_COLORS.model}
            fillOpacity={AREA_FILL_OPACITY}
            strokeWidth={AREA_STROKE_WIDTH}
          />
        </AreaChart>
      </ChartContainer>
    </div>
  );
}

/**
 * Median spend of one task, by day.
 *
 * Lines, not stacked bands: a median of the parts does not add up to the median
 * of the whole, so a stack here would draw a total that no task ever cost.
 */
export function TaskCostChart({ points }: { points: VelocityPoint[] }) {
  const rows: CostRow[] = points.map((point) => ({ day: point.day, ...point.medianTaskCost }));
  const { ref, axes } = useCostChartAxes(rows);

  return (
    <div ref={ref} className="w-full">
      <ChartContainer {...costChartContainerProps}>
        <LineChart data={rows} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
          {axes}
          <Line
            type="monotone"
            dataKey="modelCostUsd"
            stroke={VELOCITY_COST_COLORS.model}
            strokeWidth={AREA_STROKE_WIDTH}
            dot={false}
          />
          <Line
            type="monotone"
            dataKey="computeCostUsd"
            stroke={VELOCITY_COST_COLORS.compute}
            strokeWidth={AREA_STROKE_WIDTH}
            dot={false}
          />
        </LineChart>
      </ChartContainer>
    </div>
  );
}
