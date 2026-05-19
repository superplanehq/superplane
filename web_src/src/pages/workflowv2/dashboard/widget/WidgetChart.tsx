import { useMemo } from "react";
import { Loader2 } from "lucide-react";

import { applyFilters, buildChartData } from "./widgetData";
import type { WidgetChartRender } from "./types";

interface WidgetChartProps {
  render: WidgetChartRender;
  rows: unknown[];
  isLoading: boolean;
}

const DEFAULT_PALETTE = ["#0284c7", "#16a34a", "#dc2626", "#a855f7", "#f59e0b", "#0ea5e9"];

/**
 * Compact SVG-based chart renderer used by Dashboard widget blocks.
 *
 * Kept dependency-free on purpose: the canvas already ships Recharts but that
 * library expects to live inside its own ResponsiveContainer, which makes
 * reliable embedding inside a dashboard cell tricky. The chart shapes covered
 * here (bar, stacked-bar, line, area, donut) all map cleanly onto a small set
 * of SVG primitives so the renderer stays predictable.
 */
export function WidgetChart({ render, rows, isLoading }: WidgetChartProps) {
  const filtered = useMemo(() => applyFilters(rows, render.filters), [rows, render.filters]);
  const seriesKeys = useMemo(
    () =>
      render.series.map((s, idx) => ({
        key: s.label ?? s.field ?? `series-${idx}`,
        field: s.field,
        color: s.color ?? DEFAULT_PALETTE[idx % DEFAULT_PALETTE.length],
      })),
    [render.series],
  );
  const data = useMemo(() => {
    const built = buildChartData(filtered, render.xField, seriesKeys);
    if (render.limit) return built.slice(0, render.limit);
    return built;
  }, [filtered, render.xField, render.limit, seriesKeys]);

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
    <div className="flex h-full flex-col gap-2 p-3" data-testid="widget-chart">
      {render.title ? <div className="text-xs font-medium text-slate-600">{render.title}</div> : null}
      <div className="min-h-[120px] flex-1">
        {render.type === "donut" ? (
          <DonutChart data={data} series={seriesKeys[0]} />
        ) : (
          <CartesianChart type={render.type} data={data} series={seriesKeys} />
        )}
      </div>
      <Legend series={seriesKeys} />
    </div>
  );
}

interface SeriesKey {
  key: string;
  field?: string;
  color: string;
}

function CartesianChart({
  type,
  data,
  series,
}: {
  type: WidgetChartRender["type"];
  data: Array<Record<string, unknown>>;
  series: SeriesKey[];
}) {
  const width = 320;
  const height = 160;
  const padding = { top: 8, right: 8, bottom: 24, left: 32 };
  const innerW = width - padding.left - padding.right;
  const innerH = height - padding.top - padding.bottom;

  const values = data.flatMap((row) => series.map((s) => toNumber(row[s.key])));
  const stacked = type === "stacked-bar";
  const stackTotals = stacked
    ? data.map((row) => series.reduce((acc, s) => acc + Math.max(0, toNumber(row[s.key]) ?? 0), 0))
    : null;
  const max = stacked
    ? Math.max(...(stackTotals ?? [0]), 0)
    : Math.max(0, ...values.filter((v): v is number => v != null));
  const safeMax = max === 0 ? 1 : max;

  const slotW = innerW / data.length;
  const groupGap = 0.2 * slotW;

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="h-full w-full" role="img" aria-label="chart">
      {/* axes */}
      <line
        x1={padding.left}
        x2={padding.left}
        y1={padding.top}
        y2={padding.top + innerH}
        stroke="#cbd5e1"
        strokeWidth={1}
      />
      <line
        x1={padding.left}
        x2={padding.left + innerW}
        y1={padding.top + innerH}
        y2={padding.top + innerH}
        stroke="#cbd5e1"
        strokeWidth={1}
      />
      {data.map((row, i) => {
        const x = padding.left + slotW * i + groupGap / 2;
        const xLabel = padding.left + slotW * (i + 0.5);
        if (type === "bar" || type === "stacked-bar") {
          const barAreaW = slotW - groupGap;
          if (type === "stacked-bar") {
            let cumulative = 0;
            return (
              <g key={i}>
                {series.map((s) => {
                  const v = Math.max(0, toNumber(row[s.key]) ?? 0);
                  const h = (v / safeMax) * innerH;
                  const y = padding.top + innerH - cumulative - h;
                  cumulative += h;
                  return <rect key={s.key} x={x} y={y} width={barAreaW} height={h} fill={s.color} />;
                })}
                <text x={xLabel} y={height - 6} textAnchor="middle" fontSize={9} fill="#64748b">
                  {String(row.x ?? "")}
                </text>
              </g>
            );
          }
          const perBarW = barAreaW / series.length;
          return (
            <g key={i}>
              {series.map((s, si) => {
                const v = toNumber(row[s.key]) ?? 0;
                const h = (Math.max(0, v) / safeMax) * innerH;
                const y = padding.top + innerH - h;
                return (
                  <rect
                    key={s.key}
                    x={x + si * perBarW}
                    y={y}
                    width={Math.max(0, perBarW - 1)}
                    height={h}
                    fill={s.color}
                  />
                );
              })}
              <text x={xLabel} y={height - 6} textAnchor="middle" fontSize={9} fill="#64748b">
                {String(row.x ?? "")}
              </text>
            </g>
          );
        }
        // line/area: handled below as polylines
        return (
          <text key={`label-${i}`} x={xLabel} y={height - 6} textAnchor="middle" fontSize={9} fill="#64748b">
            {String(row.x ?? "")}
          </text>
        );
      })}
      {(type === "line" || type === "area") &&
        series.map((s) => {
          const points = data.map((row, i) => {
            const v = toNumber(row[s.key]) ?? 0;
            const x = padding.left + slotW * (i + 0.5);
            const y = padding.top + innerH - (Math.max(0, v) / safeMax) * innerH;
            return `${x.toFixed(1)},${y.toFixed(1)}`;
          });
          const linePath = points.join(" ");
          if (type === "area") {
            const baseY = padding.top + innerH;
            const firstX = padding.left + slotW * 0.5;
            const lastX = padding.left + slotW * (data.length - 0.5);
            const areaPath = `${firstX},${baseY} ${linePath} ${lastX},${baseY}`;
            return (
              <g key={s.key}>
                <polygon points={areaPath} fill={s.color} fillOpacity={0.18} />
                <polyline points={linePath} fill="none" stroke={s.color} strokeWidth={1.5} />
              </g>
            );
          }
          return <polyline key={s.key} points={linePath} fill="none" stroke={s.color} strokeWidth={1.5} />;
        })}
    </svg>
  );
}

function DonutChart({ data, series }: { data: Array<Record<string, unknown>>; series: SeriesKey | undefined }) {
  if (!series) return null;
  const radius = 56;
  const width = radius * 2 + 24;
  const height = radius * 2 + 24;
  const cx = width / 2;
  const cy = height / 2;
  const total = data.reduce((acc, row) => acc + (toNumber(row[series.key]) ?? 0), 0);
  if (total === 0) return <div className="p-4 text-center text-xs text-slate-500">No data</div>;
  let cumulative = 0;
  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="mx-auto h-full" role="img" aria-label="donut">
      {data.map((row, i) => {
        const value = toNumber(row[series.key]) ?? 0;
        if (value === 0) return null;
        const start = (cumulative / total) * Math.PI * 2;
        cumulative += value;
        const end = (cumulative / total) * Math.PI * 2;
        const largeArc = end - start > Math.PI ? 1 : 0;
        const x1 = cx + radius * Math.sin(start);
        const y1 = cy - radius * Math.cos(start);
        const x2 = cx + radius * Math.sin(end);
        const y2 = cy - radius * Math.cos(end);
        const color = DEFAULT_PALETTE[i % DEFAULT_PALETTE.length];
        return (
          <path
            key={i}
            d={`M ${cx} ${cy} L ${x1.toFixed(1)} ${y1.toFixed(1)} A ${radius} ${radius} 0 ${largeArc} 1 ${x2.toFixed(1)} ${y2.toFixed(1)} Z`}
            fill={color}
            stroke="#fff"
            strokeWidth={1.5}
          />
        );
      })}
      <circle cx={cx} cy={cy} r={radius * 0.55} fill="#fff" />
    </svg>
  );
}

function Legend({ series }: { series: SeriesKey[] }) {
  if (series.length <= 1) return null;
  return (
    <div className="flex flex-wrap items-center gap-3 text-[11px] text-slate-600">
      {series.map((s) => (
        <div key={s.key} className="inline-flex items-center gap-1">
          <span className="inline-block size-2 rounded-full" style={{ backgroundColor: s.color }} />
          {s.key}
        </div>
      ))}
    </div>
  );
}

function toNumber(value: unknown): number | null {
  const n = typeof value === "number" ? value : Number(value);
  return Number.isFinite(n) ? n : null;
}
