import {
  Bar,
  BarChart,
  Line,
  LineChart,
  Area,
  AreaChart,
  Pie,
  PieChart,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
} from "recharts";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  ChartLegend,
  ChartLegendContent,
  type ChartConfig as ShadcnChartConfig,
} from "@/components/ui/chart";
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

  const chartConfig: ShadcnChartConfig = {};
  series.forEach((s, i) => {
    chartConfig[s.name] = {
      label: s.name,
      color: s.color || DEFAULT_COLORS[i % DEFAULT_COLORS.length],
    };
  });

  const ChartComponent = type === "bar" ? BarChart : type === "area" ? AreaChart : LineChart;

  return (
    <div className="my-4 w-full min-w-0 rounded-lg border border-slate-200 bg-white p-3">
      {title && <p className="text-xs font-medium text-slate-700 mb-2">{title}</p>}
      <ChartContainer config={chartConfig} className="h-[200px] w-full">
        <ChartComponent data={data} margin={{ top: 5, right: 5, left: -10, bottom: 5 }}>
          <CartesianGrid vertical={false} />
          <XAxis dataKey="x" tickLine={false} axisLine={false} fontSize={11} />
          <YAxis tickLine={false} axisLine={false} fontSize={11} />
          <ChartTooltip content={<ChartTooltipContent />} />
          <ChartLegend content={<ChartLegendContent />} />
          {series.map((s, i) => {
            const color = s.color || DEFAULT_COLORS[i % DEFAULT_COLORS.length];
            if (type === "bar") {
              return <Bar key={s.name} dataKey={s.name} fill={color} radius={[4, 4, 0, 0]} />;
            }
            if (type === "area") {
              return (
                <Area
                  key={s.name}
                  type="monotone"
                  dataKey={s.name}
                  stroke={color}
                  fill={color}
                  fillOpacity={0.2}
                  strokeWidth={2}
                />
              );
            }
            return (
              <Line
                key={s.name}
                type="monotone"
                dataKey={s.name}
                stroke={color}
                strokeWidth={2}
                dot={{ r: 3, fill: color }}
              />
            );
          })}
        </ChartComponent>
      </ChartContainer>
    </div>
  );
}

function PieChartWidget({ config }: { config: ChartConfig }) {
  const { title, data } = config;
  if (!data?.length) {
    return <div className="text-xs text-slate-400 my-2">Pie chart: missing data</div>;
  }

  const chartConfig: ShadcnChartConfig = {};
  data.forEach((d, i) => {
    chartConfig[d.name] = {
      label: d.name,
      color: d.color || DEFAULT_COLORS[i % DEFAULT_COLORS.length],
    };
  });

  return (
    <div className="my-4 w-full min-w-0 rounded-lg border border-slate-200 bg-white p-3">
      {title && <p className="text-xs font-medium text-slate-700 mb-2">{title}</p>}
      <ChartContainer config={chartConfig} className="h-[200px] w-full">
        <PieChart>
          <ChartTooltip content={<ChartTooltipContent />} />
          <Pie
            data={data}
            dataKey="value"
            nameKey="name"
            cx="50%"
            cy="50%"
            outerRadius={70}
            label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}
            labelLine={{ strokeWidth: 1 }}
            fontSize={11}
          >
            {data.map((entry, i) => (
              <Cell key={entry.name} fill={entry.color || DEFAULT_COLORS[i % DEFAULT_COLORS.length]} />
            ))}
          </Pie>
        </PieChart>
      </ChartContainer>
    </div>
  );
}
