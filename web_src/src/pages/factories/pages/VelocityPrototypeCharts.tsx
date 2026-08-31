import { Area, AreaChart, Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";

import {
  BREAKDOWN_SERIES,
  COST_SERIES_COLORS,
  TIME_SERIES_COLORS,
  type Breakdown,
  type VelocityPoint,
} from "./velocityPrototypeData";

const flowChartConfig = {
  runningHours: { label: "Time running", color: TIME_SERIES_COLORS.running },
  waitingHours: { label: "Time waiting", color: TIME_SERIES_COLORS.waiting },
} satisfies ChartConfig;

const costChartConfig = {
  tokenCostUsd: { label: "Model tokens", color: COST_SERIES_COLORS.tokens },
  computeCostUsd: { label: "Execution compute", color: COST_SERIES_COLORS.compute },
} satisfies ChartConfig;

/** Both area charts sit side by side, so they must read as one visual family. */
const AREA_FILL_OPACITY = 0.3;
const AREA_STROKE_WIDTH = 1.5;

export function DeliveryChart({ points, breakdown }: { points: VelocityPoint[]; breakdown: Breakdown }) {
  const series = BREAKDOWN_SERIES[breakdown];
  const config = Object.fromEntries(
    series.map((item) => [item.key, { label: item.label, color: item.color }]),
  ) satisfies ChartConfig;

  return (
    <ChartContainer
      config={config}
      className="aspect-auto h-[260px] w-full"
      initialDimension={{ width: 760, height: 260 }}
    >
      <BarChart data={points} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
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

export function FlowChart({ points }: { points: VelocityPoint[] }) {
  return (
    <ChartContainer
      config={flowChartConfig}
      className="aspect-auto h-[180px] w-full"
      initialDimension={{ width: 500, height: 180 }}
    >
      <AreaChart data={points} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
        <XAxis dataKey="day" tickLine={false} axisLine={false} tickMargin={8} interval={0} className="text-[11px]" />
        <YAxis
          tickLine={false}
          axisLine={false}
          width={36}
          className="text-[11px]"
          tickFormatter={(value: number) => `${value}h`}
        />
        <ChartTooltip content={<ChartTooltipContent />} />
        {/* No legend: the labeled split above the chart names both bands. */}
        <Area
          type="monotone"
          dataKey="runningHours"
          stackId="time"
          stroke={TIME_SERIES_COLORS.running}
          fill={TIME_SERIES_COLORS.running}
          fillOpacity={AREA_FILL_OPACITY}
          strokeWidth={AREA_STROKE_WIDTH}
        />
        <Area
          type="monotone"
          dataKey="waitingHours"
          stackId="time"
          stroke={TIME_SERIES_COLORS.waiting}
          fill={TIME_SERIES_COLORS.waiting}
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
          width={34}
          className="text-[11px]"
          tickFormatter={(value: number) => `$${value}`}
        />
        <ChartTooltip content={<ChartTooltipContent />} />
        {/* No legend: the labeled cost split above the chart already names both
            bands, and its dots use these colors. */}
        <Area
          type="monotone"
          dataKey="tokenCostUsd"
          stackId="cost"
          stroke={COST_SERIES_COLORS.tokens}
          fill={COST_SERIES_COLORS.tokens}
          fillOpacity={AREA_FILL_OPACITY}
          strokeWidth={AREA_STROKE_WIDTH}
        />
        <Area
          type="monotone"
          dataKey="computeCostUsd"
          stackId="cost"
          stroke={COST_SERIES_COLORS.compute}
          fill={COST_SERIES_COLORS.compute}
          fillOpacity={AREA_FILL_OPACITY}
          strokeWidth={AREA_STROKE_WIDTH}
        />
      </AreaChart>
    </ChartContainer>
  );
}
