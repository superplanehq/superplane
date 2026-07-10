import { useEffect, useMemo, type ComponentProps, type CSSProperties, type ReactNode } from "react";
import { LineChart as LineChartIcon, Loader2 } from "lucide-react";
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
  Rectangle,
  XAxis,
  YAxis,
  type BarShapeProps,
} from "recharts";

import { TimestampDetails } from "@/components/Timestamp";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { useTheme } from "@/contexts/useTheme";

import { applyFilters, applySort, buildChartData, distinctSeriesKeys } from "./widgetData";
import { resolveChartColor } from "./chartColors";
import { formatPercentOfTotal, formatSeriesValue } from "./chartFormat";
import { WidgetEmptyState } from "../WidgetEmptyState";
import {
  buildXAxisTickShowIndices,
  estimateYAxisWidth,
  formatXAxisTick,
  formatXTooltipLabel,
  formatYTick,
  resolveCartesianYFormat,
} from "./widgetChartAxis";
import { useInteractiveChartTooltip } from "./useInteractiveChartTooltip";
import { coerceWidgetTimestamp } from "./widgetFormat";
import type { WidgetChartLegendMode, WidgetChartRender, WidgetChartSeries, WidgetColumnFormat } from "./types";

const TIMESTAMP_X_FORMATS = new Set<WidgetColumnFormat>(["date", "datetime", "relative"]);
const TOOLTIP_WRAPPER_STYLE: CSSProperties = { transition: "none" };

interface WidgetChartProps {
  render: WidgetChartRender;
  rows: unknown[];
  isLoading: boolean;
}

const STACK_ID = "stack";

/** Ignore Recharts/row props and always paint bars with the resolved series color. */
function barShapeWithColor(seriesColor: string) {
  return (props: BarShapeProps) => {
    const { x, y, width, height, radius } = props;
    return (
      <Rectangle
        x={x}
        y={y}
        width={width}
        height={height}
        radius={radius}
        isAnimationActive={false}
        isUpdateAnimationActive={false}
        fill={seriesColor}
        style={{ fill: seriesColor }}
      />
    );
  };
}

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
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const filtered = useMemo(() => applyFilters(rows, render.filters), [rows, render.filters]);
  const sorted = useMemo(() => applySort(filtered, render.sort), [filtered, render.sort]);
  const seriesField = render.seriesField?.trim();
  const valueSeries = render.series[0];
  // When `seriesField` is set, the chart pivots: one series per distinct
  // value in that field, sharing the numeric `field` (and formatting) of the
  // first configured series. Without `seriesField` we keep the historical
  // behavior of one series per `render.series` entry. Split memos so row
  // updates do not recreate the configured series array (and downstream
  // chartConfig / Recharts layers) when only data values change.
  const configuredSeries = useMemo<ChartSeries[]>(
    () =>
      render.series.map((s, idx) => {
        const key = s.label ?? s.field ?? `series-${idx}`;
        return {
          ...s,
          key,
          color: resolveChartColor(key, idx, isDark),
        };
      }),
    [render.series, isDark],
  );
  const pivotedSeries = useMemo<ChartSeries[]>(() => {
    if (!seriesField) return [];
    const distinct = distinctSeriesKeys(sorted, seriesField);
    return distinct.map((key, idx) => ({
      ...valueSeries,
      key,
      label: key,
      color: resolveChartColor(key, idx, isDark),
    }));
  }, [seriesField, sorted, valueSeries, isDark]);
  const series = seriesField ? pivotedSeries : configuredSeries;
  const data = useMemo(() => {
    const built = buildChartData(
      sorted,
      render.xField,
      seriesField ? [{ key: "value", field: valueSeries?.field }] : series,
      seriesField ? { seriesField } : undefined,
    );
    if (render.limit) return built.slice(0, render.limit);
    return built;
  }, [sorted, render.xField, render.limit, series, seriesField, valueSeries]);

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-500" />
      </div>
    );
  }
  if (data.length === 0) {
    return <WidgetEmptyState icon={LineChartIcon} message="No data to display." testId="widget-chart-empty" />;
  }

  return (
    <div className="flex h-full min-h-0 flex-col gap-1 p-3" data-testid="widget-chart">
      {render.title ? (
        <div className="text-xs font-medium text-slate-600 dark:text-gray-400">{render.title}</div>
      ) : null}
      <div className="min-h-0 flex-1">
        {render.type === "donut" ? (
          <DonutChartView data={data} series={series[0]} legendMode={render.legend ?? "auto"} isDark={isDark} />
        ) : (
          <CartesianChartView
            type={render.type}
            data={data}
            series={series}
            legendMode={render.legend ?? "auto"}
            xFormat={render.xFormat}
            yLabel={render.yLabel}
            yFormat={render.yFormat}
          />
        )}
      </div>
    </div>
  );
}

const CHART_MARGIN = { top: 8, right: 8, left: 4, bottom: 0 } as const;
// Recharts wraps the tooltip in an absolutely positioned div with a default
// `transform 400ms ease` transition. That transition makes the tooltip slide
// in from the chart origin (top-left) the first time it appears, which feels
// confusing. We disable the wrapper transition and add a quick fade-in on the
// content so the tooltip appears in place.
const TOOLTIP_CONTENT_CLASS = "animate-in fade-in duration-150";

/** Bridges Recharts' active point into the interactive-tooltip hook without doing work in render. */
function RechartsActiveBridge({
  active,
  activeKey,
  forceContentActive,
  onActiveChange,
}: {
  active?: boolean;
  activeKey?: string;
  /** Re-sync when force releases so a still-hovered point re-arms `wasActive`. */
  forceContentActive: boolean;
  onActiveChange: (active: boolean, activeKey?: string) => void;
}) {
  useEffect(() => {
    onActiveChange(Boolean(active), activeKey);
  }, [active, activeKey, forceContentActive, onActiveChange]);
  return null;
}

type ChartTooltipContentProps = ComponentProps<typeof ChartTooltipContent>;

/**
 * Chart tooltip content that can receive pointer events (CopyButton) and stays
 * mounted briefly after the pointer leaves the chart point.
 */
function InteractiveChartTooltipContent({
  forceContentActive,
  syncRechartsActive,
  onTooltipEnter,
  onTooltipLeave,
  ...tooltipProps
}: ChartTooltipContentProps & {
  forceContentActive: boolean;
  syncRechartsActive: (active: boolean, activeKey?: string) => void;
  onTooltipEnter: () => void;
  onTooltipLeave: () => void;
}) {
  const activeKey = tooltipProps.label == null ? undefined : String(tooltipProps.label);
  return (
    <div onMouseEnter={onTooltipEnter} onMouseLeave={onTooltipLeave}>
      <RechartsActiveBridge
        active={tooltipProps.active}
        activeKey={activeKey}
        forceContentActive={forceContentActive}
        onActiveChange={syncRechartsActive}
      />
      <ChartTooltipContent {...tooltipProps} active={Boolean(tooltipProps.active || forceContentActive)} />
    </div>
  );
}

function CartesianChartView({
  type,
  data,
  series,
  legendMode,
  xFormat,
  yLabel,
  yFormat,
}: {
  type: Exclude<WidgetChartRender["type"], "donut">;
  data: Array<Record<string, unknown>>;
  series: ChartSeries[];
  legendMode: WidgetChartLegendMode;
  xFormat?: WidgetColumnFormat;
  yLabel?: string;
  yFormat?: WidgetColumnFormat;
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
  const effectiveYFormat = resolveCartesianYFormat(yFormat, series);
  const yAxisWidth = useMemo(
    () =>
      estimateYAxisWidth(
        data,
        series.map((s) => s.key),
        effectiveYFormat,
        Boolean(yLabel?.trim()),
      ),
    [data, series, effectiveYFormat, yLabel],
  );

  const sharedAxes = (
    <CartesianFrame
      data={data}
      stacked={stacked || type === "bar"}
      seriesByKey={seriesByKey}
      xFormat={xFormat}
      yLabel={yLabel}
      yFormat={effectiveYFormat}
      yAxisWidth={yAxisWidth}
    />
  );
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
              isAnimationActive={false}
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
              isAnimationActive={false}
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
              shape={barShapeWithColor(s.color)}
              stackId={stacked ? STACK_ID : undefined}
              radius={stacked ? 0 : [3, 3, 0, 0]}
              isAnimationActive={false}
            >
              {data.map((_, idx) => (
                <Cell key={`${s.key}-${idx}`} fill={s.color} />
              ))}
            </Bar>
          ))}
        </BarChart>
      )}
    </ChartContainer>
  );
}

function CartesianFrame({
  data,
  stacked,
  seriesByKey,
  xFormat,
  yLabel,
  yFormat,
  yAxisWidth,
}: {
  data: Array<Record<string, unknown>>;
  stacked: boolean;
  seriesByKey: Map<string, ChartSeries>;
  xFormat?: WidgetColumnFormat;
  yLabel?: string;
  yFormat?: WidgetColumnFormat;
  yAxisWidth: number;
}) {
  const xTickShowIndices = useMemo(() => buildXAxisTickShowIndices(data, xFormat), [data, xFormat]);
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
  const xTickFormatter = (v: unknown, index: number) => {
    if (xTickShowIndices && !xTickShowIndices.has(index)) return "";
    return formatXAxisTick(v, xFormat);
  };
  const xTooltipLabelFormatter = (v: unknown): ReactNode => {
    if (xFormat && TIMESTAMP_X_FORMATS.has(xFormat)) {
      const date = coerceWidgetTimestamp(v);
      if (date) return <TimestampDetails date={date} copyTestId="chart-tooltip-timestamp-copy" />;
    }
    return formatXTooltipLabel(v, xFormat);
  };
  const yTick = (v: number) => formatYTick(v, yFormat);
  const trimmedYLabel = yLabel?.trim() ? yLabel.trim() : undefined;
  const interactiveTooltip = Boolean(xFormat && TIMESTAMP_X_FORMATS.has(xFormat));
  const { activeProp, forceContentActive, syncRechartsActive, onTooltipEnter, onTooltipLeave, wrapperStyle } =
    useInteractiveChartTooltip(interactiveTooltip);

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
        tickFormatter={xTickFormatter}
      />
      <YAxis
        tickLine={false}
        axisLine={false}
        fontSize={11}
        width={yAxisWidth}
        tickFormatter={yTick}
        label={
          trimmedYLabel
            ? {
                value: trimmedYLabel,
                angle: -90,
                position: "insideLeft",
                style: { textAnchor: "middle", fontSize: 11, fill: "currentColor" },
              }
            : undefined
        }
      />
      <ChartTooltip
        cursor={stacked ? { fill: "rgba(148, 163, 184, 0.12)" } : true}
        active={activeProp}
        wrapperStyle={wrapperStyle}
        content={
          interactiveTooltip ? (
            <InteractiveChartTooltipContent
              forceContentActive={forceContentActive}
              syncRechartsActive={syncRechartsActive}
              onTooltipEnter={onTooltipEnter}
              onTooltipLeave={onTooltipLeave}
              formatter={tooltipFormatter}
              labelFormatter={xTooltipLabelFormatter}
              indicator="dot"
              className={TOOLTIP_CONTENT_CLASS}
            />
          ) : (
            <ChartTooltipContent
              formatter={tooltipFormatter}
              labelFormatter={xTooltipLabelFormatter}
              indicator="dot"
              className={TOOLTIP_CONTENT_CLASS}
            />
          )
        }
      />
    </>
  );
}

function DonutChartView({
  data,
  series,
  legendMode,
  isDark,
}: {
  data: Array<Record<string, unknown>>;
  series: ChartSeries | undefined;
  legendMode: WidgetChartLegendMode;
  isDark: boolean;
}) {
  const seriesKey = series?.key ?? "";
  const sliceData = useMemo(() => {
    if (!seriesKey) return [];
    return data.map((row, idx) => {
      const x = String(row.x ?? "");
      return {
        x,
        value: Number(row[seriesKey]) || 0,
        color: resolveChartColor(x, idx, isDark),
      };
    });
  }, [data, seriesKey, isDark]);

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
    return <WidgetEmptyState icon={LineChartIcon} message="No data" />;
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
        <ChartTooltip
          wrapperStyle={TOOLTIP_WRAPPER_STYLE}
          content={
            <ChartTooltipContent nameKey="x" hideLabel formatter={tooltipFormatter} className={TOOLTIP_CONTENT_CLASS} />
          }
        />
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
          stroke={isDark ? "#111827" : "#fff"}
          strokeWidth={1.5}
        >
          {sliceData.map((slice, idx) => (
            <Cell key={`${slice.x}-${idx}`} fill={slice.color} />
          ))}
        </Pie>
      </PieChart>
    </ChartContainer>
  );
}
