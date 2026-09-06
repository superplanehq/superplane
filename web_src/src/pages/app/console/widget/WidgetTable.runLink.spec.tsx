import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi } from "vitest";

import { ConsoleContextProvider } from "../ConsoleContextProvider";
import { WidgetTable } from "./WidgetTable";
import { isRunIdColumn } from "./runLink";
import type { WidgetDataSourceKind, WidgetTableRender } from "./types";

const RUN_ID = "8f2b1c44-0a3e-4d21-9f77-2b6c5d0e1a99";

const RUN_ROWS = [{ id: RUN_ID, status: "passed", nodeName: "deploy-prod" }];

function renderTable({
  tableRender,
  rows = RUN_ROWS,
  dataSourceKind = "runs",
  initialEntry = "/canvas-1?view=console",
  onSelectRun,
  /** Whether the host offers run selection; false mirrors an open edit session. */
  hostSelectsRuns = true,
}: {
  tableRender: WidgetTableRender;
  rows?: unknown[];
  dataSourceKind?: WidgetDataSourceKind;
  initialEntry?: string;
  onSelectRun?: (runId: string) => void;
  hostSelectsRuns?: boolean;
}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <QueryClientProvider client={queryClient}>
        <ConsoleContextProvider
          canvasId="canvas-1"
          organizationId="org-1"
          nodes={[]}
          canRunNodes={false}
          onSelectRun={hostSelectsRuns ? (onSelectRun ?? (() => {})) : undefined}
        >
          <WidgetTable render={tableRender} rows={rows} isLoading={false} dataSourceKind={dataSourceKind} />
        </ConsoleContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

function tableWith(columns: WidgetTableRender["columns"]): WidgetTableRender {
  return { kind: "table", columns };
}

describe("isRunIdColumn", () => {
  it("matches a plain id column on the runs data source", () => {
    expect(isRunIdColumn({ field: "id" }, "runs")).toBe(true);
    expect(isRunIdColumn({ field: "id", format: "text" }, "runs")).toBe(true);
  });

  it("ignores id columns from data sources whose id is not a run id", () => {
    expect(isRunIdColumn({ field: "id" }, "memory")).toBe(false);
    expect(isRunIdColumn({ field: "id" }, "executions")).toBe(false);
    expect(isRunIdColumn({ field: "id" }, undefined)).toBe(false);
  });

  it("leaves other run fields alone", () => {
    expect(isRunIdColumn({ field: "status" }, "runs")).toBe(false);
    expect(isRunIdColumn({ field: "rootEvent.id" }, "runs")).toBe(false);
  });

  it("defers to an author's explicit href or presentational format", () => {
    expect(isRunIdColumn({ field: "id", href: "https://example.com/{id}" }, "runs")).toBe(false);
    expect(isRunIdColumn({ field: "id", format: "link" }, "runs")).toBe(false);
    expect(isRunIdColumn({ field: "id", format: "code" }, "runs")).toBe(false);
    expect(isRunIdColumn({ field: "id", format: "badge" }, "runs")).toBe(false);
  });
});

describe("WidgetTable run id links", () => {
  it("links a runs-table id cell to that run's detail view", () => {
    renderTable({ tableRender: tableWith([{ field: "id", label: "Run" }]) });

    const link = screen.getByTestId("widget-table-run-link");
    expect(link).toHaveTextContent(RUN_ID);
    // `run=<id>` set and the non-canvas `view=console` dropped.
    expect(link).toHaveAttribute("href", `/canvas-1?run=${RUN_ID}`);
  });

  it("preserves unrelated search params already on the URL", () => {
    renderTable({
      tableRender: tableWith([{ field: "id" }]),
      initialEntry: "/canvas-1?view=console&appId=app-9",
    });

    const href = screen.getByTestId("widget-table-run-link").getAttribute("href");
    expect(href).toContain("appId=app-9");
    expect(href).toContain(`run=${RUN_ID}`);
    expect(href).not.toContain("view=console");
  });

  it("renders id as plain text for non-runs data sources", () => {
    renderTable({
      tableRender: tableWith([{ field: "id" }]),
      rows: [{ id: "mem-1" }],
      dataSourceKind: "memory",
    });

    expect(screen.queryByTestId("widget-table-run-link")).toBeNull();
    expect(screen.getByText("mem-1")).toBeInTheDocument();
  });

  it("keeps an author's explicit href instead of hijacking the cell", () => {
    renderTable({
      tableRender: tableWith([{ field: "id", href: "https://example.com/runs/{id}" }]),
    });

    expect(screen.queryByTestId("widget-table-run-link")).toBeNull();
    expect(screen.getByRole("link")).toHaveAttribute("href", `https://example.com/runs/${RUN_ID}`);
  });

  it("routes a plain click through the host's run selection", () => {
    const onSelectRun = vi.fn();
    renderTable({ tableRender: tableWith([{ field: "id" }]), onSelectRun });

    fireEvent.click(screen.getByTestId("widget-table-run-link"), { button: 0 });

    expect(onSelectRun).toHaveBeenCalledWith(RUN_ID);
  });

  it("leaves modified clicks to the browser so new-tab still works", () => {
    const onSelectRun = vi.fn();
    renderTable({ tableRender: tableWith([{ field: "id" }]), onSelectRun });

    fireEvent.click(screen.getByTestId("widget-table-run-link"), { button: 0, metaKey: true });

    expect(onSelectRun).not.toHaveBeenCalled();
  });

  it("stays plain text when the host offers no run selection", () => {
    renderTable({ tableRender: tableWith([{ field: "id" }]), hostSelectsRuns: false });

    expect(screen.queryByTestId("widget-table-run-link")).toBeNull();
    expect(screen.getByText(RUN_ID)).toBeInTheDocument();
  });

  it("renders plain text when a run row has no id", () => {
    renderTable({
      tableRender: tableWith([{ field: "id" }]),
      rows: [{ id: undefined, status: "passed" }],
    });

    expect(screen.queryByTestId("widget-table-run-link")).toBeNull();
  });
});
