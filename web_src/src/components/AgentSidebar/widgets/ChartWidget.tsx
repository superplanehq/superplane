import {
  LineChart,
  Line,
  BarChart,
  Bar,
  AreaChart,
  Area,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import type { ChartConfig } from "./parser";

const DEFAULT_COLORS = ["#8b5cf6", "#06b6d4", "#22c55e", "#f59e0b", "#ef4444", "#ec4899"];

interface ChartWidgetProps {
  config: ChartConfig;
}

export function ChartWidget({ config }: ChartWidgetProps) {
  if (config.type === "pie") {
    return <PieChartWidget config={config} />;
  }
  return <XYChartWidget config={config} />;
}

function XYChartWidget({ config }: { config: ChartConfig }) {
  const { type, title, x, series } = config;
  if (!x || !series?.length) {
    return <div className="text-xs text-slate-400 my-2">Chart: missing data</div>;
  }

  const data = x.map((label, i) => {
    const point: Record<string, string | number> = { x: label };
    for (const s of series) {
      point[s.name] = s.data[i] ?? 0;
    }
    return point;
  });

  const ChartComponent = type === "bar" ? BarChart : type === "area" ? AreaChart : LineChart;
  const DataComponent = type === "bar" ? Bar : type === "area" ? Area : Line;

  return (
    <div className="my-2">
      {title && <p className="text-xs font-medium text-slate-700 mb-1">{title}</p>}
      <div className="h-[200px] w-full">
        <ResponsiveContainer width="100%" height="100%">
          <ChartComponent data={data} margin={{ top: 5, right: 5, left: -20, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
            <XAxis dataKey="x" tick={{ fontSize: 11 }} stroke="#94a3b8" />
            <YAxis tick={{ fontSize: 11 }} stroke="#94a3b8" />
            <Tooltip contentStyle={{ fontSize: 12 }} />
            <Legend wrapperStyle={{ fontSize: 12 }} />
            {series.map((s, i) => (
              <DataComponent
                key={s.name}
                type="monotone"
                dataKey={s.name}
                stroke={s.color || DEFAULT_COLORS[i % DEFAULT_COLORS.length]}
                fill={s.color || DEFAULT_COLORS[i % DEFAULT_COLORS.length]}
                fillOpacity={type === "area" ? 0.3 : 1}
                strokeWidth={2}
                dot={type === "line" ? { r: 3 } : undefined}
              />
            ))}
          </ChartComponent>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

function PieChartWidget({ config }: { config: ChartConfig }) {
  const { title, data } = config;
  if (!data?.length) {
    return <div className="text-xs text-slate-400 my-2">Pie chart: missing data</div>;
  }

  return (
    <div className="my-2">
      {title && <p className="text-xs font-medium text-slate-700 mb-1">{title}</p>}
      <div className="h-[200px] w-full">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={data}
              dataKey="value"
              nameKey="name"
              cx="50%"
              cy="50%"
              outerRadius={70}
              label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
              labelLine={{ strokeWidth: 1 }}
              fontSize={11}
            >
              {data.map((entry, i) => (
                <Cell key={entry.name} fill={entry.color || DEFAULT_COLORS[i % DEFAULT_COLORS.length]} />
              ))}
            </Pie>
            <Tooltip contentStyle={{ fontSize: 12 }} />
          </PieChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
