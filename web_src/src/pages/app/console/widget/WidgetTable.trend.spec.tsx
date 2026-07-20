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
    displayCount,
  }: {
    render: WidgetTableRender;
    rows: unknown[];
    hasMore?: boolean;
    displayCount?: number;
  }) {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
            <WidgetTable
              render={tableRender}
              rows={rows}
              isLoading={false}
              hasMore={hasMore}
              displayCount={displayCount}
            />
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

  it("compares against the next already-loaded row when hasMore is only the display window", () => {
    // Progressive tables set hasMore when more rows are loaded but still
    // hidden behind the display window. Pass the full loaded set + displayCount
    // so the last visible trend cell compares against the hidden neighbor.
    const view = renderTrend({
      render: TREND_RENDER,
      rows: [
        { id: "d-3", durationMs: 900 },
        { id: "d-2", durationMs: 1000 },
        { id: "d-1", durationMs: 1000 },
      ],
      hasMore: true,
      displayCount: 2,
    });
    const cells = view.container.querySelectorAll('[data-testid="widget-trend-cell"]');
    expect(cells).toHaveLength(2);
    expect(cells[1].getAttribute("data-trend-kind")).toBe("flat");
    expect(cells[1].textContent).toContain("0");
    expect(cells[1].getAttribute("data-trend-kind")).not.toBe("pending");
    view.unmount();
  });

  it("ignores a loaded baseline that fails where/filters", () => {
    // Loaded-but-hidden rows still go through where/filters. If they would
    // be hidden, trend must not use them as the "row below".
    const view = renderTrend({
      render: {
        ...TREND_RENDER,
        where: [{ field: "status", op: "eq", value: "ok" }],
      },
      rows: [
        { id: "d-3", durationMs: 900, status: "ok" },
        { id: "d-2", durationMs: 1000, status: "ok" },
        { id: "d-1", durationMs: 500, status: "fail" },
      ],
      hasMore: true,
      displayCount: 2,
    });
    const cells = view.container.querySelectorAll('[data-testid="widget-trend-cell"]');
    expect(cells).toHaveLength(2);
    expect(cells[1].getAttribute("data-trend-kind")).toBe("pending");
    expect(cells[1].textContent).toContain("...");
    view.unmount();
  });

  it("uses the next row after widget sort, not fetch order", () => {
    // Loaded order is ascending by name, but sort desc so the visible window's
    // last row's baseline is the next name in sorted order (c), not the next
    // in fetch order.
    const view = renderTrend({
      render: {
        ...TREND_RENDER,
        sort: { field: "id", order: "desc" },
      },
      rows: [
        { id: "a", durationMs: 1000 },
        { id: "b", durationMs: 800 },
        { id: "c", durationMs: 900 },
      ],
      displayCount: 2,
    });
    const cells = view.container.querySelectorAll('[data-testid="widget-trend-cell"]');
    expect(cells).toHaveLength(2);
    // Visible sorted: c (900), b (800); baseline for b is a (1000).
    // b vs a: 800 vs 1000 → down / better with trendBetter: down → -20%
    expect(cells[1].getAttribute("data-trend-kind")).toBe("changed");
    expect(cells[1].getAttribute("data-trend-direction")).toBe("down");
    expect(cells[1].textContent).toContain("-20%");
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

  it("renders formatted value beside the trend chip when showTrend is set", () => {
    const view = renderTrend({
      render: {
        kind: "table",
        columns: [
          { field: "id", label: "Deploy" },
          {
            field: "durationMs",
            label: "Duration",
            format: "duration",
            showTrend: true,
            trendBetter: "down",
            trendDisplay: "percent",
          },
        ],
      },
      rows: TREND_ROWS,
    });
    const combined = view.container.querySelectorAll('[data-testid="widget-value-with-trend"]');
    expect(combined).toHaveLength(3);
    expect(combined[0].textContent).toContain("900ms");
    expect(combined[0].textContent).toContain("-10%");
    const chip = combined[0].querySelector('[data-testid="widget-trend-cell"]');
    expect(chip?.getAttribute("data-trend-kind")).toBe("changed");
    expect(chip?.getAttribute("data-trend-direction")).toBe("down");
    expect(chip?.getAttribute("data-trend-polarity")).toBe("better");
    view.unmount();
  });

  it("keeps the value visible when the trend chip is pending", () => {
    const view = renderTrend({
      render: {
        kind: "table",
        columns: [
          {
            field: "passRate",
            label: "Pass rate",
            format: "percent",
            showTrend: true,
            trendBetter: "up",
          },
        ],
      },
      rows: [{ passRate: 0.9 }, { passRate: 0.8 }],
      hasMore: true,
    });
    const combined = view.container.querySelectorAll('[data-testid="widget-value-with-trend"]');
    const last = combined[combined.length - 1];
    expect(last.textContent).toContain("80%");
    const chip = last.querySelector('[data-testid="widget-trend-cell"]');
    expect(chip?.getAttribute("data-trend-kind")).toBe("pending");
    expect(chip?.textContent).toContain("...");
    view.unmount();
  });
});
