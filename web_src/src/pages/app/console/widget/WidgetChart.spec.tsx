import type { ReactNode } from "react";
import { isValidElement } from "react";
import type * as Recharts from "recharts";
import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";

vi.mock("recharts", async () => {
  const actual = (await vi.importActual("recharts")) as typeof Recharts;
  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children: ReactNode }) => (
      <div data-testid="responsive-container" style={{ width: 600, height: 400 }}>
        <actual.ResponsiveContainer width={600} height={400}>
          {children as never}
        </actual.ResponsiveContainer>
      </div>
    ),
  };
});

const tooltipContentProps = vi.hoisted(() => ({ value: null as Record<string, unknown> | null }));

vi.mock("@/components/ui/chart", async () => {
  const actual = (await vi.importActual("@/components/ui/chart")) as Record<string, unknown>;
  const ChartTooltipContent = (props: Record<string, unknown>) => {
    tooltipContentProps.value = props;
    const Original = actual.ChartTooltipContent as (p: unknown) => ReactNode;
    return <Original {...props} />;
  };
  return { ...actual, ChartTooltipContent };
});

import { WidgetChart } from "./WidgetChart";
import type { WidgetChartRender } from "./types";

beforeAll(() => {
  if (typeof globalThis.ResizeObserver === "undefined") {
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver;
  }
});

const ROWS = [
  { service: "ec2", cost: 1200, errors: 4 },
  { service: "s3", cost: 350, errors: 1 },
  { service: "rds", cost: 600, errors: 2 },
];

function renderChart(render_: WidgetChartRender, options: { rows?: unknown[]; isLoading?: boolean } = {}) {
  return render(<WidgetChart render={render_} rows={options.rows ?? ROWS} isLoading={options.isLoading ?? false} />, {
    wrapper: ThemeProvider,
  });
}

describe("WidgetChart states", () => {
  it("renders a loading spinner when isLoading", () => {
    const { container } = renderChart(
      { kind: "chart", type: "bar", xField: "service", series: [{ field: "cost", label: "Cost" }] },
      { isLoading: true },
    );
    expect(container.querySelector(".animate-spin")).not.toBeNull();
    expect(screen.queryByTestId("widget-chart")).toBeNull();
  });

  it("renders an empty-state message when there are no rows", () => {
    renderChart(
      { kind: "chart", type: "bar", xField: "service", series: [{ field: "cost", label: "Cost" }] },
      { rows: [] },
    );
    expect(screen.getByTestId("widget-chart-empty")).toBeInTheDocument();
  });

  it("renders the chart container when rows are present", () => {
    renderChart({ kind: "chart", type: "bar", xField: "service", series: [{ field: "cost", label: "Cost" }] });
    expect(screen.getByTestId("widget-chart")).toBeInTheDocument();
  });

  it("renders the optional chart title", () => {
    renderChart({
      kind: "chart",
      type: "bar",
      xField: "service",
      series: [{ field: "cost", label: "Cost" }],
      title: "AWS spend",
    });
    expect(screen.getByText("AWS spend")).toBeInTheDocument();
  });
});

describe("WidgetChart bar variants", () => {
  it("uses a shared stackId when type is stacked-bar", () => {
    const { container } = renderChart({
      kind: "chart",
      type: "stacked-bar",
      xField: "service",
      series: [
        { field: "cost", label: "Cost" },
        { field: "errors", label: "Errors" },
      ],
    });
    const layers = container.querySelectorAll(".recharts-bar");
    expect(layers.length).toBeGreaterThanOrEqual(2);
  });

  it("renders multiple bar series side-by-side for grouped bars", () => {
    const { container } = renderChart({
      kind: "chart",
      type: "bar",
      xField: "service",
      series: [
        { field: "cost", label: "Cost" },
        { field: "errors", label: "Errors" },
      ],
    });
    const layers = container.querySelectorAll(".recharts-bar");
    expect(layers.length).toBe(2);
  });

  it("pivots long-format rows into one bar layer per distinct seriesField value", () => {
    const COST_ROWS = [
      { date: "2026-05-26", service: "ec2", cost_usd: 58.45 },
      { date: "2026-05-26", service: "s3", cost_usd: 0.0034 },
      { date: "2026-05-26", service: "rds", cost_usd: 0.42 },
      { date: "2026-05-27", service: "ec2", cost_usd: 60.0 },
      { date: "2026-05-27", service: "s3", cost_usd: 0.01 },
      { date: "2026-05-27", service: "rds", cost_usd: 0.43 },
    ];
    const { container } = renderChart(
      {
        kind: "chart",
        type: "stacked-bar",
        xField: "date",
        seriesField: "service",
        series: [{ field: "cost_usd", label: "Cost", prefix: "$" }],
      },
      { rows: COST_ROWS },
    );
    const layers = container.querySelectorAll(".recharts-bar");
    expect(layers.length).toBe(3);
  });

  it("applies resolved palette hex fills on pivoted stacked bars", () => {
    const rows = [
      { date: "May 01", model: "Claude Haiku 4.5", cost: 100 },
      { date: "May 01", model: "Claude Sonnet 4.6", cost: 200 },
    ];
    const { container } = renderChart(
      {
        kind: "chart",
        type: "stacked-bar",
        xField: "date",
        seriesField: "model",
        series: [{ field: "cost", label: "Cost" }],
      },
      { rows },
    );

    const style = container.querySelector("style")?.textContent ?? "";
    expect(style).toContain("--color-claude-haiku-4-5: #0284c7");
    expect(style).toContain("--color-claude-sonnet-4-6: #0ea5e9");

    const paths = [...container.querySelectorAll(".recharts-bar-rectangle path")];
    const fills = paths.map((el) => el.getAttribute("fill"));
    expect(fills).toContain("#0284c7");
    expect(fills).toContain("#0ea5e9");
    expect(fills.every((fill) => !fill?.startsWith("var("))).toBe(true);
    // Theme fallback seen in DevTools when palette colors fail to apply.
    expect(fills.every((fill) => fill?.toLowerCase() !== "#c4627d")).toBe(true);
    for (const path of paths) {
      expect(path.getAttribute("style")).toContain("fill:");
    }
  });

  it("ignores stored series colors and uses the default palette for multi-series stacked bars", () => {
    const rows = [
      {
        date: "May 01",
        claude_sonnet_46: 200,
        claude_opus_47: 100,
        claude_haiku_45: 50,
      },
    ];
    const { container } = renderChart(
      {
        kind: "chart",
        type: "stacked-bar",
        xField: "date",
        series: [
          { field: "claude_sonnet_46", label: "Claude Sonnet 4.6", color: "#c4627d" },
          { field: "claude_opus_47", label: "Claude Opus 4.7", color: "#c5c6e0" },
          { field: "claude_haiku_45", label: "Claude Haiku 4.5", color: "#e0d6c5" },
        ],
      },
      { rows },
    );

    const fills = [...container.querySelectorAll(".recharts-bar-rectangle path")].map((el) =>
      el.getAttribute("fill")?.toLowerCase(),
    );
    expect(fills).toContain("#0284c7");
    expect(fills).toContain("#0ea5e9");
    expect(fills).toContain("#38bdf8");
    expect(fills.every((fill) => fill !== "#c4627d")).toBe(true);
  });

  it("renders a bar layer for rows with a missing seriesField value", () => {
    const rows = [
      { date: "2026-05-26", service: "ec2", cost_usd: 10 },
      { date: "2026-05-26", cost_usd: 3 },
    ];
    const { container } = renderChart(
      {
        kind: "chart",
        type: "bar",
        xField: "date",
        seriesField: "service",
        series: [{ field: "cost_usd", label: "Cost" }],
      },
      { rows },
    );
    expect(container.querySelectorAll(".recharts-bar").length).toBe(2);
  });
});

describe("WidgetChart legend visibility", () => {
  it("hides the legend by default for a single-series cartesian chart", () => {
    const { container } = renderChart({
      kind: "chart",
      type: "bar",
      xField: "service",
      series: [{ field: "cost", label: "Cost" }],
    });
    expect(container.querySelector(".recharts-legend-wrapper")).toBeNull();
  });

  it("shows the legend automatically when there are 2+ series", () => {
    const { container } = renderChart({
      kind: "chart",
      type: "bar",
      xField: "service",
      series: [
        { field: "cost", label: "Cost" },
        { field: "errors", label: "Errors" },
      ],
    });
    expect(container.querySelector(".recharts-legend-wrapper")).not.toBeNull();
  });

  it("renders a legend even for a single-series chart when legend is forced to show", () => {
    const { container } = renderChart({
      kind: "chart",
      type: "bar",
      xField: "service",
      series: [{ field: "cost", label: "Cost" }],
      legend: "show",
    });
    expect(container.querySelector(".recharts-legend-wrapper")).not.toBeNull();
  });

  it("hides the legend when legend is set to hide", () => {
    const { container } = renderChart({
      kind: "chart",
      type: "bar",
      xField: "service",
      series: [
        { field: "cost", label: "Cost" },
        { field: "errors", label: "Errors" },
      ],
      legend: "hide",
    });
    expect(container.querySelector(".recharts-legend-wrapper")).toBeNull();
  });
});

describe("WidgetChart axis formatting", () => {
  function tickTexts(container: HTMLElement, axis: "xAxis" | "yAxis"): string[] {
    const labels = container.querySelectorAll(`.recharts-${axis}-tick-labels .recharts-cartesian-axis-tick-value`);
    return Array.from(labels).map((node) => node.textContent ?? "");
  }

  it("formats X-axis tick values when xFormat is set", () => {
    const TIME_ROWS = [
      { day: "2026-05-26T00:00:00Z", cost: 10 },
      { day: "2026-05-27T00:00:00Z", cost: 12 },
    ];
    const { container } = renderChart(
      { kind: "chart", type: "bar", xField: "day", xFormat: "date", series: [{ field: "cost", label: "Cost" }] },
      { rows: TIME_ROWS },
    );
    const ticks = tickTexts(container, "xAxis");
    expect(ticks.length).toBeGreaterThan(0);
    // No tick should still contain the raw ISO timestamp.
    expect(ticks.some((text) => text.includes("T00:00:00Z"))).toBe(false);
  });

  it("formats ISO timestamp X values as dates on the axis", () => {
    const TIME_ROWS = [
      { day: "2026-05-26T00:00:00Z", cost: 10 },
      { day: "2026-05-27T00:00:00Z", cost: 12 },
    ];
    const { container } = renderChart(
      { kind: "chart", type: "bar", xField: "day", series: [{ field: "cost", label: "Cost" }] },
      { rows: TIME_ROWS },
    );
    const ticks = tickTexts(container, "xAxis");
    expect(ticks.some((text) => text.includes("T00:00:00Z"))).toBe(false);
    expect(ticks.some((text) => /May/.test(text))).toBe(true);
    expect(ticks.every((text) => !/AM|PM/.test(text))).toBe(true);
  });

  it("renders a Y-axis title when yLabel is set", () => {
    const { container } = renderChart({
      kind: "chart",
      type: "bar",
      xField: "service",
      yLabel: "USD",
      series: [{ field: "cost", label: "Cost" }],
    });
    const labels = Array.from(container.querySelectorAll(".recharts-label")).map((node) => node.textContent ?? "");
    expect(labels).toContain("USD");
  });

  it("trims surrounding whitespace from yLabel before rendering it", () => {
    const { container } = renderChart({
      kind: "chart",
      type: "bar",
      xField: "service",
      yLabel: "  USD  ",
      series: [{ field: "cost", label: "Cost" }],
    });
    const labels = Array.from(container.querySelectorAll(".recharts-label")).map((node) => node.textContent ?? "");
    expect(labels).toContain("USD");
    expect(labels).not.toContain("  USD  ");
  });

  it("does not render a Y-axis label when yLabel is blank", () => {
    const { container } = renderChart({
      kind: "chart",
      type: "bar",
      xField: "service",
      yLabel: "   ",
      series: [{ field: "cost", label: "Cost" }],
    });
    expect(container.querySelector(".recharts-label")).toBeNull();
  });

  it("shows date-only axis ticks but a TimestampDetails tooltip label for xFormat datetime", () => {
    tooltipContentProps.value = null as Record<string, unknown> | null;
    const TIME_ROWS = [{ day: "2026-05-26T16:10:00Z", cost: 10 }];
    const { container } = renderChart(
      {
        kind: "chart",
        type: "bar",
        xField: "day",
        xFormat: "datetime",
        series: [{ field: "cost", label: "Cost" }],
      },
      { rows: TIME_ROWS },
    );
    const ticks = tickTexts(container, "xAxis");
    expect(ticks.every((text) => !/AM|PM/.test(text))).toBe(true);
    const props: Record<string, unknown> | null = tooltipContentProps.value;
    const labelFormatter = props?.labelFormatter as ((label: unknown, payload?: unknown[]) => ReactNode) | undefined;
    const formatted = labelFormatter?.("2026-05-26T16:10:00Z", []);
    expect(isValidElement(formatted)).toBe(true);
    const { container: labelContainer } = render(<>{formatted}</>);
    expect(labelContainer.textContent).toMatch(/Local/);
    expect(labelContainer.textContent).toMatch(/UTC/);
    expect(labelContainer.textContent).toMatch(/2026-05-26T16:10:00\.000Z/);
  });

  it("shows a date axis label only on the first bar of each calendar day", () => {
    const TIME_ROWS = [
      { day: "2026-07-05T10:00:00Z", cost: 10 },
      { day: "2026-07-05T14:00:00Z", cost: 11 },
      { day: "2026-07-05T18:00:00Z", cost: 12 },
      { day: "2026-07-06T10:00:00Z", cost: 13 },
      { day: "2026-07-06T14:00:00Z", cost: 14 },
      { day: "2026-07-06T18:00:00Z", cost: 15 },
    ];
    const { container } = renderChart(
      {
        kind: "chart",
        type: "bar",
        xField: "day",
        xFormat: "datetime",
        series: [{ field: "cost", label: "Cost" }],
      },
      { rows: TIME_ROWS },
    );
    const ticks = tickTexts(container, "xAxis");
    const visible = ticks.filter((text) => text.trim() !== "");
    expect(visible).toEqual(["Jul 5", "Jul 6"]);
  });

  it("passes a labelFormatter to the tooltip that mirrors xFormat date", () => {
    tooltipContentProps.value = null as Record<string, unknown> | null;
    renderChart(
      { kind: "chart", type: "bar", xField: "day", xFormat: "date", series: [{ field: "cost", label: "Cost" }] },
      { rows: [{ day: "2026-05-26T00:00:00Z", cost: 10 }] },
    );
    const props: Record<string, unknown> | null = tooltipContentProps.value;
    expect(props).not.toBeNull();
    const labelFormatter = props?.labelFormatter as ((label: unknown, payload?: unknown[]) => ReactNode) | undefined;
    expect(labelFormatter).toBeTypeOf("function");
    const formatted = labelFormatter?.("2026-05-26T00:00:00Z", []);
    expect(isValidElement(formatted)).toBe(true);
    const { container: labelContainer } = render(<>{formatted}</>);
    expect(labelContainer.textContent).toMatch(/2026-05-26T00:00:00\.000Z/);
    expect(labelContainer.textContent).toMatch(/Local/);
  });

  it("returns the raw category label from the tooltip formatter when xFormat is unset", () => {
    tooltipContentProps.value = null as Record<string, unknown> | null;
    renderChart(
      { kind: "chart", type: "bar", xField: "service", series: [{ field: "cost", label: "Cost" }] },
      { rows: [{ service: "ec2", cost: 1 }] },
    );
    const props: Record<string, unknown> | null = tooltipContentProps.value;
    const labelFormatter = props?.labelFormatter as ((label: unknown, payload?: unknown[]) => ReactNode) | undefined;
    expect(labelFormatter?.("ec2", [])).toBe("ec2");
  });

  it("formats Y-axis ticks with yFormat", () => {
    const { container } = renderChart(
      {
        kind: "chart",
        type: "bar",
        xField: "service",
        yFormat: "duration",
        series: [{ field: "ms", label: "Latency" }],
      },
      {
        rows: [
          { service: "ec2", ms: 4500 },
          { service: "s3", ms: 1200 },
        ],
      },
    );
    const ticks = tickTexts(container, "yAxis");
    expect(ticks.some((text) => /(ms|s)$/.test(text))).toBe(true);
    expect(ticks.every((text) => !/\s/.test(text))).toBe(true);
    expect(ticks.every((text) => text !== "1000" && text !== "1,000")).toBe(true);
  });

  it("inherits a single series format for Y-axis ticks when yFormat is omitted", () => {
    const { container } = renderChart(
      {
        kind: "chart",
        type: "bar",
        xField: "service",
        series: [{ field: "ms", label: "Latency", format: "duration" }],
      },
      {
        rows: [
          { service: "ec2", ms: 1_111_256 },
          { service: "s3", ms: 45_000 },
        ],
      },
    );
    const ticks = tickTexts(container, "yAxis");
    expect(ticks.some((text) => /(m|h)/.test(text))).toBe(true);
    expect(ticks.every((text) => !/\s/.test(text))).toBe(true);
    expect(ticks.every((text) => !/^\d{1,3}(,\d{3})+$/.test(text))).toBe(true);
  });
});

describe("WidgetChart donut", () => {
  it("renders a pie when type is donut", () => {
    const { container } = renderChart({
      kind: "chart",
      type: "donut",
      xField: "service",
      series: [{ field: "cost", label: "Cost", prefix: "$" }],
    });
    expect(container.querySelector(".recharts-pie")).not.toBeNull();
  });

  it("renders an empty-state message when every slice value is zero", () => {
    renderChart(
      {
        kind: "chart",
        type: "donut",
        xField: "service",
        series: [{ field: "cost", label: "Cost" }],
      },
      { rows: [{ service: "a", cost: 0 }] },
    );
    expect(screen.getByText("No data")).toBeInTheDocument();
  });

  it("shows a legend by default", () => {
    const { container } = renderChart({
      kind: "chart",
      type: "donut",
      xField: "service",
      series: [{ field: "cost", label: "Cost" }],
    });
    expect(container.querySelector(".recharts-legend-wrapper")).not.toBeNull();
  });
});
