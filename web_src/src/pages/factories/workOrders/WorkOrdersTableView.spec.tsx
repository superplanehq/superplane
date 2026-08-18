import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrdersTableView } from "./WorkOrdersTableView";

const organizationId = "org-1";
const factoryKey = "RF";
const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: factoryKey };

/** Has a real assignee and is dispatchable, so its Owner column renders an
 * avatar *and* a dispatch button — the widest realistic Owner content. */
const busyEntry = buildWorkOrderListEntry(
  {
    id: "wo-1",
    number: "1",
    title: "Reconcile refund batch",
    state: "STATE_OPEN",
    createdAt: "2024-06-01T00:00:00Z",
    updatedAt: "2024-06-02T00:00:00Z",
    executions: [
      {
        id: "e1",
        step: "verify",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        line: { id: "line-a", name: "hotfix" },
        totalTokens: "1200",
        costCents: "45",
      },
    ],
    assignees: [{ id: "user-1", name: "Ada Lovelace" }],
  } satisfies FactoriesWorkOrder,
  factory,
);

/** No assignees and not dispatchable, so its Owner column renders only the
 * dashed "Assign" chip — the narrowest realistic Owner content. */
const idleEntry = buildWorkOrderListEntry(
  {
    id: "wo-2",
    number: "2",
    title: "Archive stale draft",
    state: "STATE_CLOSED",
    createdAt: "2024-06-01T00:00:00Z",
    updatedAt: "2024-06-03T00:00:00Z",
    executions: [],
    assignees: [],
  } satisfies FactoriesWorkOrder,
  factory,
);

function renderTable() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <WorkOrdersTableView
          entries={[busyEntry, idleEntry]}
          organizationId={organizationId}
          factoryKey={factoryKey}
          factoryLines={[]}
          canDispatch={true}
          canAssign={true}
          isDispatching={false}
          isAssigneesSaving={false}
          onDispatch={vi.fn().mockResolvedValue(undefined)}
          onAssigneesSave={vi.fn().mockResolvedValue(undefined)}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

/** Pulls the `grid-cols-[...]` / `md:grid-cols-[...]` / `lg:grid-cols-[...]`
 * tokens out of a className string so we can compare grid templates without
 * being tripped up by unrelated classes (background, hover, etc.). */
function gridTemplateTokens(className: string): string[] {
  return className.split(/\s+/).filter((token) => /grid-cols-\[/.test(token));
}

describe("WorkOrdersTableView", () => {
  it("has a non-empty spend usage label to prove removal isn't just an empty-state coincidence", () => {
    expect(busyEntry.usageLabel).toBe("$0.45");
  });

  it("never renders a Spend column, header or cell", () => {
    renderTable();

    expect(screen.queryByText("Spend")).not.toBeInTheDocument();
    expect(screen.queryByText(busyEntry.usageLabel as string)).not.toBeInTheDocument();
  });

  it("keeps the header and every row on the exact same grid template, regardless of row content", () => {
    renderTable();

    const header = screen.getByTestId("work-orders-table").firstElementChild as HTMLElement;
    const busyRow = screen.getByTestId(`work-orders-table-row-${busyEntry.id}`);
    const idleRow = screen.getByTestId(`work-orders-table-row-${idleEntry.id}`);

    const headerTokens = gridTemplateTokens(header.className);
    expect(headerTokens.length).toBeGreaterThan(0);
    expect(gridTemplateTokens(busyRow.className)).toEqual(headerTokens);
    expect(gridTemplateTokens(idleRow.className)).toEqual(headerTokens);
  });

  it("never uses a content-sized (auto) grid track, so header/row widths can't drift apart", () => {
    renderTable();

    const header = screen.getByTestId("work-orders-table").firstElementChild as HTMLElement;
    expect(header.className).not.toMatch(/grid-cols-\[[^\]]*auto[^\]]*\]/);
  });
});
