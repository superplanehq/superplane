import type { ReactNode } from "react";
import type * as Recharts from "recharts";
import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

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
  return render(<WidgetChart render={render_} rows={options.rows ?? ROWS} isLoading={options.isLoading ?? false} />);
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
