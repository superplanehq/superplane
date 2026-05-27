import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeAll, describe, it, expect } from "vitest";

import type { SuperplaneComponentsNode } from "@/api-client";

import { DashboardContextProvider } from "./DashboardContextProvider";
import type { TablePanelContent } from "./panelTypes";
import { TablePanelForm } from "./TablePanelForm";
import { EXECUTIONS_FIELDS, RUNS_FIELDS } from "./widget/staticFieldCatalogs";

const START_NODE: SuperplaneComponentsNode = {
  id: "start-id",
  name: "start",
  type: "TYPE_TRIGGER",
};

function makeInitial(kind: TablePanelContent["dataSource"]["kind"]): TablePanelContent {
  const dataSource =
    kind === "memory"
      ? { kind: "memory" as const, namespace: "" }
      : kind === "executions"
        ? { kind: "executions" as const, limit: 50 }
        : { kind: "runs" as const, limit: 50 };
  return {
    title: "",
    dataSource,
    render: { kind: "table", columns: [] },
  };
}

function Harness({ initial }: { initial: TablePanelContent }) {
  const [value, setValue] = useState<TablePanelContent>(initial);
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <DashboardContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[START_NODE]} canRunNodes>
          <TablePanelForm value={value} onChange={setValue} />
          <pre data-testid="harness-columns">{JSON.stringify(value.render.columns)}</pre>
        </DashboardContextProvider>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("TablePanelForm field catalog", () => {
  beforeAll(() => {
    Element.prototype.scrollIntoView ??= () => {};
  });

  it("surfaces execution quick-add buttons in alphabetical order", () => {
    render(<Harness initial={makeInitial("executions")} />);
    const quickAdd = screen.getByTestId("table-field-quick-add");
    expect(quickAdd).toBeInTheDocument();
    const chipLabels = Array.from(quickAdd.querySelectorAll("button")).map((b) => b.textContent ?? "");
    // Spot-check a few of the derived + raw execution fields are present.
    for (const expected of ["status", "nodeName", "durationMs", "createdAt", "result"]) {
      expect(chipLabels).toContain(expected);
    }
    expect(chipLabels).toEqual([...chipLabels].sort((a, b) => a.localeCompare(b)));
    // And the same alphabetical order is reflected in the catalog itself.
    expect(EXECUTIONS_FIELDS.map((f) => f.field)).toEqual(
      [...EXECUTIONS_FIELDS.map((f) => f.field)].sort((a, b) => a.localeCompare(b)),
    );
  });

  it("surfaces run quick-add buttons in alphabetical order", () => {
    render(<Harness initial={makeInitial("runs")} />);
    const quickAdd = screen.getByTestId("table-field-quick-add");
    const chipLabels = Array.from(quickAdd.querySelectorAll("button")).map((b) => b.textContent ?? "");
    for (const expected of ["state", "result", "createdAt", "finishedAt", "versionId"]) {
      expect(chipLabels).toContain(expected);
    }
    expect(chipLabels).toEqual([...chipLabels].sort((a, b) => a.localeCompare(b)));
    expect(RUNS_FIELDS.map((f) => f.field)).toEqual(
      [...RUNS_FIELDS.map((f) => f.field)].sort((a, b) => a.localeCompare(b)),
    );
  });

  it("adds an executions field to the columns list when its quick-add button is clicked", () => {
    render(<Harness initial={makeInitial("executions")} />);
    const button = screen
      .getAllByRole("button", { name: "status" })
      .find((b) => b.closest('[data-testid="table-field-quick-add"]'));
    expect(button).toBeTruthy();
    fireEvent.click(button!);

    const columns = JSON.parse(screen.getByTestId("harness-columns").textContent ?? "[]") as Array<
      Record<string, unknown>
    >;
    expect(columns).toHaveLength(1);
    expect(columns[0]?.field).toBe("status");
    // `suggestColumnFormat` maps "status" -> "status" — the same heuristic memory uses.
    expect(columns[0]?.format).toBe("status");
  });

  it("adds every executions field at once via the Add all fields button", () => {
    render(<Harness initial={makeInitial("executions")} />);
    fireEvent.click(screen.getByTestId("table-add-all-columns"));

    const columns = JSON.parse(screen.getByTestId("harness-columns").textContent ?? "[]") as Array<
      Record<string, unknown>
    >;
    expect(columns).toHaveLength(EXECUTIONS_FIELDS.length);
    expect(columns.map((c) => c.field)).toEqual(EXECUTIONS_FIELDS.map((f) => f.field));
  });

  it("adds every runs field at once via the Add all fields button", () => {
    render(<Harness initial={makeInitial("runs")} />);
    fireEvent.click(screen.getByTestId("table-add-all-columns"));

    const columns = JSON.parse(screen.getByTestId("harness-columns").textContent ?? "[]") as Array<
      Record<string, unknown>
    >;
    expect(columns).toHaveLength(RUNS_FIELDS.length);
    expect(columns.map((c) => c.field)).toEqual(RUNS_FIELDS.map((f) => f.field));
  });

  it("does not render quick-add buttons for memory sources without a discovered namespace", () => {
    render(<Harness initial={makeInitial("memory")} />);
    expect(screen.queryByTestId("table-field-quick-add")).toBeNull();
    expect(screen.queryByTestId("table-add-all-columns")).toBeNull();
  });
});
