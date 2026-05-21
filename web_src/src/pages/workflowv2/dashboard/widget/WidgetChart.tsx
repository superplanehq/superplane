import { useMemo } from "react";
import { Loader2 } from "lucide-react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Line,
  LineChart,
  Pie,
  PieChart,
  XAxis,
  YAxis,
} from "recharts";

import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";

import { applyFilters, buildChartData } from "./widgetData";
import { formatPercentOfTotal, formatSeriesValue } from "./chartFormat";
import type { WidgetChartLegendMode, WidgetChartRender, WidgetChartSeries } from "./types";

interface WidgetChartProps {
  render: WidgetChartRender;
  rows: unknown[];
  isLoading: boolean;
}

const DEFAULT_PALETTE = ["#0284c7", "#16a34a", "#dc2626", "#a855f7", "#f59e0b", "#0ea5e9"];
const STACK_ID = "stack";

interface ChartSeries extends WidgetChartSeries {
  key: string;
  color: string;
}

/**
 * Dashboard chart renderer powered by Recharts via the project's shadcn
 * `ChartContainer`. Supports bar (grouped/stacked), line, area, and donut
 * charts, with hover tooltips and a configurable legend. Each series can
 * declare a `format` / `prefix` / `suffix` so currencies, durations, or
 * percentages render consistently in the tooltip.
 */
export function WidgetChart({ render, rows, isLoading }: WidgetChartProps) {
  const filtered = useMemo(() => applyFilters(rows, render.filters), [rows, render.filters]);
  const series = useMemo<ChartSeries[]>(
    () =>
      render.series.map((s, idx) => ({
        ...s,
        key: s.label ?? s.field ?? `series-${idx}`,
        color: s.color ?? DEFAULT_PALETTE[idx % DEFAULT_PALETTE.length],
      })),
    [render.series],
  );
  const data = useMemo(() => {
    const built = buildChartData(filtered, render.xField, series);
    if (render.limit) return built.slice(0, render.limit);
    return built;
  }, [filtered, render.xField, render.limit, series]);

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <Loader2 className="size-4 animate-spin text-slate-400" />
      </div>
    );
  }
  if (data.length === 0) {
    return (
      <div className="p-4 text-center text-xs text-slate-500" data-testid="widget-chart-empty">
        No data to display.
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col gap-1 p-3" data-testid="widget-chart">
      {render.title ? <div className="text-xs font-medium text-slate-600">{render.title}</div> : null}
      <div className="min-h-0 flex-1">
        {render.type === "donut" ? (
          <DonutChartView data={data} series={series[0]} legendMode={render.legend ?? "auto"} />
        ) : (
          <CartesianChartView type={render.type} data={data} series={series} legendMode={render.legend ?? "auto"} />
        )}
      </div>
    </div>
  );
}

const CHART_MARGIN = { top: 8, right: 8, left: 0, bottom: 0 } as const;

function CartesianChartView({
  type,
  data,
  series,
  legendMode,
}: {
  type: Exclude<WidgetChartRender["type"], "donut">;
  data: Array<Record<string, unknown>>;
  series: ChartSeries[];
  legendMode: WidgetChartLegendMode;
}) {
  const chartConfig = useMemo<ChartConfig>(() => {
    const config: ChartConfig = {};
    for (const s of series) {
      config[s.key] = { label: s.label ?? s.field ?? s.key, color: s.color };
    }
    return config;
  }, [series]);
  const seriesByKey = useMemo(() => new Map(series.map((s) => [s.key, s])), [series]);
  const showLegend = legendMode === "show" || (legendMode === "auto" && series.length > 1);
  const stacked = type === "stacked-bar";

  const sharedAxes = <CartesianFrame stacked={stacked || type === "bar"} seriesByKey={seriesByKey} />;
  const legend = showLegend ? <ChartLegend content={<ChartLegendContent />} verticalAlign="bottom" /> : null;

  return (
    <ChartContainer config={chartConfig} className="aspect-auto h-full w-full">
      {type === "area" ? (
        <AreaChart data={data} margin={CHART_MARGIN}>
          {sharedAxes}
          {legend}
          {series.map((s) => (
            <Area
              key={s.key}
              type="monotone"
              dataKey={s.key}
              stroke={s.color}
              fill={s.color}
              fillOpacity={0.2}
              strokeWidth={2}
            />
          ))}
        </AreaChart>
      ) : type === "line" ? (
        <LineChart data={data} margin={CHART_MARGIN}>
          {sharedAxes}
          {legend}
          {series.map((s) => (
            <Line
              key={s.key}
              type="monotone"
              dataKey={s.key}
              stroke={s.color}
              strokeWidth={2}
              dot={{ r: 2.5, fill: s.color }}
              activeDot={{ r: 4 }}
            />
          ))}
        </LineChart>
      ) : (
        <BarChart data={data} margin={CHART_MARGIN}>
          {sharedAxes}
          {legend}
          {series.map((s) => (
            <Bar
              key={s.key}
              dataKey={s.key}
              fill={s.color}
              stackId={stacked ? STACK_ID : undefined}
              radius={stacked ? 0 : [3, 3, 0, 0]}
            />
          ))}
        </BarChart>
      )}
    </ChartContainer>
  );
}

function CartesianFrame({ stacked, seriesByKey }: { stacked: boolean; seriesByKey: Map<string, ChartSeries> }) {
  const tooltipFormatter = (value: unknown, name: unknown) => {
    const key = String(name ?? "");
    const s = seriesByKey.get(key);
    const label = s?.label ?? s?.field ?? key;
    const formatted = formatSeriesValue(value, { format: s?.format, prefix: s?.prefix, suffix: s?.suffix });
    return (
      <div className="flex w-full items-center justify-between gap-3">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-mono font-medium text-foreground tabular-nums">{formatted}</span>
      </div>
    );
  };
  return (
    <>
      <CartesianGrid vertical={false} strokeDasharray="3 3" />
      <XAxis
        dataKey="x"
        tickLine={false}
        axisLine={false}
        fontSize={11}
        interval="preserveStartEnd"
        minTickGap={16}
        tickFormatter={(v: unknown) => String(v ?? "")}
      />
      <YAxis tickLine={false} axisLine={false} fontSize={11} width={36} tickFormatter={yTickFormatter} />
      <ChartTooltip
        cursor={stacked ? { fill: "rgba(148, 163, 184, 0.12)" } : true}
        content={<ChartTooltipContent formatter={tooltipFormatter} indicator="dot" />}
      />
    </>
  );
}

function yTickFormatter(value: number) {
  if (!Number.isFinite(value)) return String(value);
  if (Math.abs(value) >= 1000) return value.toLocaleString();
  return String(value);
}

function DonutChartView({
  data,
  series,
  legendMode,
}: {
  data: Array<Record<string, unknown>>;
  series: ChartSeries | undefined;
  legendMode: WidgetChartLegendMode;
}) {
  const seriesKey = series?.key ?? "";
  const sliceData = useMemo(() => {
    if (!seriesKey) return [];
    return data.map((row, idx) => ({
      x: String(row.x ?? ""),
      value: Number(row[seriesKey]) || 0,
      color: DEFAULT_PALETTE[idx % DEFAULT_PALETTE.length],
    }));
  }, [data, seriesKey]);

  const total = useMemo(() => sliceData.reduce((acc, slice) => acc + slice.value, 0), [sliceData]);

  const chartConfig = useMemo<ChartConfig>(() => {
    const config: ChartConfig = {};
    for (const slice of sliceData) {
      config[slice.x || "(empty)"] = { label: slice.x || "(empty)", color: slice.color };
    }
    return config;
  }, [sliceData]);

  if (!series) return null;
  if (total === 0) {
    return <div className="p-4 text-center text-xs text-slate-500">No data</div>;
  }

  const showLegend = legendMode !== "hide";

  const tooltipFormatter = (value: unknown, _name: unknown, item: { payload?: { x?: string } }) => {
    const sliceName = item.payload?.x ?? String(_name ?? "");
    const formatted = formatSeriesValue(value, {
      format: series.format,
      prefix: series.prefix,
      suffix: series.suffix,
    });
    const pct = formatPercentOfTotal(value, total);
    return (
      <div className="flex w-full items-center justify-between gap-3">
        <span className="text-muted-foreground">{sliceName}</span>
        <span className="font-mono font-medium text-foreground tabular-nums">
          {formatted}
          {pct}
        </span>
      </div>
    );
  };

  return (
    <ChartContainer config={chartConfig} className="aspect-auto h-full w-full">
      <PieChart>
        <ChartTooltip content={<ChartTooltipContent nameKey="x" hideLabel formatter={tooltipFormatter} />} />
        {showLegend ? <ChartLegend content={<ChartLegendContent nameKey="x" />} verticalAlign="bottom" /> : null}
        <Pie
          data={sliceData}
          dataKey="value"
          nameKey="x"
          cx="50%"
          cy="50%"
          innerRadius="55%"
          outerRadius="85%"
          paddingAngle={1}
          stroke="#fff"
          strokeWidth={1.5}
        >
          {sliceData.map((slice) => (
            <Cell key={slice.x} fill={slice.color} />
          ))}
        </Pie>
      </PieChart>
    </ChartContainer>
  );
}
