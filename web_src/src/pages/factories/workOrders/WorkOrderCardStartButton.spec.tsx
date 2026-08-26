import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrdersBoardView } from "./WorkOrdersBoardView";

const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: "RF" };

function draft(id: string, title: string) {
  return buildWorkOrderListEntry(
    {
      id,
      number: id,
      title,
      state: "STATE_DRAFT",
      createdAt: "2024-06-01T00:00:00Z",
      updatedAt: "2024-06-02T00:00:00Z",
      lineDispatches: [],
      assignees: [],
    } satisfies FactoriesWorkOrder,
    factory,
  );
}

function renderBoard(dispatchingOrderIds: ReadonlySet<string>) {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <WorkOrdersBoardView
          entries={[draft("wo-1", "Reconcile refund batch"), draft("wo-2", "Handle duplicate charges")]}
          organizationId="org-1"
          factoryKey="RF"
          factoryLines={[{ id: "line-a", name: "hotfix" }]}
          canDispatch
          canAssign
          dispatchingOrderIds={dispatchingOrderIds}
          isAssigneesSaving={false}
          onDispatch={vi.fn().mockResolvedValue(undefined)}
          onAssigneesSave={vi.fn().mockResolvedValue(undefined)}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("Start button on draft cards", () => {
  it("shows the busy state only on the work order that dispatches", () => {
    renderBoard(new Set(["wo-1"]));

    const starting = screen.getByTestId("work-order-card-start-wo-1");
    expect(starting).toHaveTextContent("Starting...");
    expect(starting).toBeDisabled();

    const idle = screen.getByTestId("work-order-card-start-wo-2");
    expect(idle).toHaveTextContent("Start");
    expect(idle).toBeEnabled();
  });

  it("leaves every card ready when no dispatch runs", () => {
    renderBoard(new Set());

    for (const id of ["wo-1", "wo-2"]) {
      const start = screen.getByTestId(`work-order-card-start-${id}`);
      expect(start).toHaveTextContent("Start");
      expect(start).toBeEnabled();
    }
  });
});
