import { createContext, createElement, isValidElement, useContext, type ReactNode } from "react";

import { toChartColorVarName } from "@/components/ui/chartColorVarName";

export type ChartRow = Record<string, unknown>;
export type ChartConfig = Record<string, { label?: ReactNode; color?: string }>;

export const tooltipContentProps = { value: null as Record<string, unknown> | null };
export const tooltipWrapperProps = { value: null as Record<string, unknown> | null };

const ChartDataContext = createContext<ChartRow[]>([]);
const ChartConfigContext = createContext<ChartConfig>({});

function chartFrame({ children, data }: { children?: ReactNode; data?: ChartRow[] }) {
  return createElement(
    "div",
    { className: "recharts-wrapper" },
    createElement(ChartDataContext.Provider, { value: data ?? [] }, children),
  );
}

function seriesLayer({ className, fill, children }: { className: string; fill?: string; children?: ReactNode }) {
  return createElement(
    "div",
    { className },
    createElement(
      "div",
      { className: "recharts-bar-rectangle" },
      createElement("path", { fill, style: { fill: fill ?? "" } }),
    ),
    children,
  );
}

export function rechartsTestDoubles() {
  return {
    ResponsiveContainer: ({ children }: { children: ReactNode }) =>
      createElement("div", { "data-testid": "responsive-container", style: { width: 600, height: 400 } }, children),
    BarChart: ({ children, data }: { children?: ReactNode; data?: ChartRow[] }) =>
      createElement(chartFrame, { data }, children),
    AreaChart: ({ children, data }: { children?: ReactNode; data?: ChartRow[] }) =>
      createElement(chartFrame, { data }, children),
    LineChart: ({ children, data }: { children?: ReactNode; data?: ChartRow[] }) =>
      createElement(chartFrame, { data }, children),
    PieChart: ({ children }: { children?: ReactNode }) => createElement(chartFrame, {}, children),
    Bar: ({ fill, children }: { fill?: string; children?: ReactNode }) =>
      createElement(seriesLayer, { className: "recharts-bar", fill }, children),
    Area: ({ fill }: { fill?: string }) => createElement(seriesLayer, { className: "recharts-area", fill }),
    Line: ({ stroke }: { stroke?: string }) => createElement(seriesLayer, { className: "recharts-line", fill: stroke }),
    Pie: ({ children }: { children?: ReactNode }) => createElement("div", { className: "recharts-pie" }, children),
    Cell: ({ fill }: { fill?: string }) => createElement("path", { fill, style: { fill: fill ?? "" } }),
    Rectangle: () => null,
    CartesianGrid: () => null,
    XAxis: ({ tickFormatter }: { tickFormatter?: (value: unknown, index: number) => string }) => {
      const data = useContext(ChartDataContext);
      return createElement(
        "div",
        { className: "recharts-xAxis-tick-labels" },
        data.map((row, index) =>
          createElement(
            "span",
            { key: index, className: "recharts-cartesian-axis-tick-value" },
            tickFormatter ? tickFormatter(row.x, index) : String(row.x ?? ""),
          ),
        ),
      );
    },
    YAxis: ({ label, tickFormatter }: { label?: { value?: string }; tickFormatter?: (value: number) => string }) => {
      const data = useContext(ChartDataContext);
      const values = data.flatMap((row) =>
        Object.entries(row)
          .filter(([key]) => key !== "x")
          .map(([, value]) => Number(value))
          .filter((value) => !Number.isNaN(value)),
      );
      return createElement(
        "div",
        { className: "recharts-yAxis" },
        label?.value ? createElement("span", { className: "recharts-label" }, label.value) : null,
        createElement(
          "div",
          { className: "recharts-yAxis-tick-labels" },
          (values.length > 0 ? values : [90]).map((value, index) =>
            createElement(
              "span",
              { key: index, className: "recharts-cartesian-axis-tick-value" },
              tickFormatter ? tickFormatter(value) : String(value),
            ),
          ),
        ),
      );
    },
    Legend: ({ content }: { content?: ReactNode }) =>
      createElement("div", { className: "recharts-legend-wrapper" }, content),
  };
}

export function chartUiTestDoubles() {
  function chartStyle({ id, config }: { id: string; config: Record<string, { color?: string }> }) {
    const colorConfig = Object.entries(config).filter(([, item]) => item.color);
    if (!colorConfig.length) {
      return null;
    }
    const css = colorConfig.map(([key, item]) => `  --color-${toChartColorVarName(key)}: ${item.color};`).join("\n");
    return createElement("style", null, `[data-chart=${id}] {\n${css}\n}`);
  }

  function chartContainer({ children, config }: { children: ReactNode; config: ChartConfig }) {
    return createElement(
      ChartConfigContext.Provider,
      { value: config },
      createElement(
        "div",
        { "data-chart": "widget-chart" },
        createElement(chartStyle, { id: "widget-chart", config }),
        children,
      ),
    );
  }

  return {
    ChartContainer: chartContainer,
    ChartTooltip: (props: Record<string, unknown>) => {
      tooltipWrapperProps.value = props;
      const content = props.content;
      if (isValidElement(content)) {
        tooltipContentProps.value = content.props as Record<string, unknown>;
      }
      return null;
    },
    ChartTooltipContent: (props: Record<string, unknown>) => {
      tooltipContentProps.value = props;
      return null;
    },
    ChartLegend: ({ content }: { content?: ReactNode }) =>
      createElement("div", { className: "recharts-legend-wrapper" }, content),
    ChartLegendContent: () => {
      const config = useContext(ChartConfigContext);
      return createElement(
        "div",
        null,
        Object.values(config).map((item) => createElement("span", { key: String(item.label) }, item.label)),
      );
    },
  };
}
