import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect } from "vitest";

import { ConsoleContextProvider } from "../ConsoleContextProvider";
import { WidgetTable } from "./WidgetTable";
import type { WidgetTableRender } from "./types";

describe("WidgetTable trend columns", () => {
  function renderTrend({
    render: tableRender,
    rows,
    hasMore,
  }: {
    render: WidgetTableRender;
    rows: unknown[];
    hasMore?: boolean;
  }) {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
            <WidgetTable render={tableRender} rows={rows} isLoading={false} hasMore={hasMore} />
          </ConsoleContextProvider>
        </QueryClientProvider>
      </MemoryRouter>,
    );
  }

  const TREND_RENDER: WidgetTableRender = {
    kind: "table",
    columns: [
      { field: "id", label: "Deploy" },
      {
        field: "durationMs",
        label: "Trend",
        format: "trend",
        trendBetter: "down",
        trendDisplay: "percent",
      },
    ],
  };

  const TREND_ROWS = [
    { id: "d-3", durationMs: 900 },
    { id: "d-2", durationMs: 1000 },
    { id: "d-1", durationMs: 1000 },
  ];

  it("shows a signed percent and 'better' color when the field decreased and down is better", () => {
    const view = renderTrend({ render: TREND_RENDER, rows: TREND_ROWS });
    const cells = view.container.querySelectorAll('[data-testid="widget-trend-cell"]');
    expect(cells).toHaveLength(3);
    expect(cells[0].getAttribute("data-trend-kind")).toBe("changed");
    expect(cells[0].getAttribute("data-trend-direction")).toBe("down");
    expect(cells[0].getAttribute("data-trend-polarity")).toBe("better");
    expect(cells[0].textContent).toContain("-10%");
    expect(cells[0].className).toContain("text-emerald-600");
    view.unmount();
  });

  it("renders '- 0' for a flat delta", () => {
    const view = renderTrend({ render: TREND_RENDER, rows: TREND_ROWS });
    const flatCell = view.container.querySelectorAll('[data-testid="widget-trend-cell"]')[1];
    expect(flatCell.getAttribute("data-trend-kind")).toBe("flat");
    expect(flatCell.textContent).toContain("0");
    view.unmount();
  });

  it("renders '- 0' for the last row when no more data is available", () => {
    const view = renderTrend({ render: TREND_RENDER, rows: TREND_ROWS });
    const lastCell = view.container.querySelectorAll('[data-testid="widget-trend-cell"]')[2];
    expect(lastCell.getAttribute("data-trend-kind")).toBe("no-baseline");
    expect(lastCell.textContent).toContain("0");
    view.unmount();
  });

  it("renders '...' on the last row when more data is still loading", () => {
    const view = renderTrend({ render: TREND_RENDER, rows: TREND_ROWS, hasMore: true });
    const cells = view.container.querySelectorAll('[data-testid="widget-trend-cell"]');
    expect(cells[cells.length - 1].getAttribute("data-trend-kind")).toBe("pending");
    expect(cells[cells.length - 1].textContent).toContain("...");
    view.unmount();
  });

  it("renders incomparable '-' when the row below exists but the field is missing", () => {
    const view = renderTrend({
      render: TREND_RENDER,
      rows: [{ id: "d-2", durationMs: 900 }, { id: "d-1" }],
    });
    const cells = view.container.querySelectorAll('[data-testid="widget-trend-cell"]');
    expect(cells[0].getAttribute("data-trend-kind")).toBe("incomparable");
    expect(cells[0].textContent).not.toContain("0");
    expect(cells[1].getAttribute("data-trend-kind")).toBe("no-baseline");
    view.unmount();
  });

  it("renders muted '-' with no arrow when percent mode has previous value 0", () => {
    const view = renderTrend({
      render: TREND_RENDER,
      rows: [
        { id: "d-2", durationMs: 900 },
        { id: "d-1", durationMs: 0 },
      ],
    });
    const cell = view.container.querySelectorAll('[data-testid="widget-trend-cell"]')[0];
    expect(cell.getAttribute("data-trend-kind")).toBe("incomparable");
    expect(cell.getAttribute("data-trend-direction")).toBeNull();
    expect(cell.getAttribute("data-trend-polarity")).toBeNull();
    expect(cell.className).toContain("text-slate-400");
    expect(cell.className).not.toContain("text-emerald");
    expect(cell.className).not.toContain("text-red");
    expect(cell.textContent).toBe("-");
    view.unmount();
  });

  it("re-evaluates a CEL field on the row below (same aggregation on both rows)", () => {
    const view = renderTrend({
      render: {
        kind: "table",
        columns: [
          { field: "id" },
          {
            field: "{{ durationMs / 1000 }}",
            label: "Trend (s)",
            format: "trend",
            trendBetter: "down",
            trendDisplay: "value",
          },
        ],
      },
      rows: [
        { id: "row-a", durationMs: 4000 },
        { id: "row-b", durationMs: 5000 },
      ],
    });
    const cells = view.container.querySelectorAll('[data-testid="widget-trend-cell"]');
    expect(cells[0].getAttribute("data-trend-kind")).toBe("changed");
    expect(cells[0].textContent).toContain("-1");
    view.unmount();
  });
});
