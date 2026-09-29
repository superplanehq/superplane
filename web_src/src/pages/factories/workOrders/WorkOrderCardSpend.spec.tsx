import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard } from "./WorkOrderCard";

const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: "RF" };

const baseOrder: FactoriesWorkOrder = {
  id: "wo-1",
  number: "12",
  title: "Ship refund retries",
  state: "STATE_CLOSED",
  result: "RESULT_COMPLETED",
  createdAt: "2024-06-01T00:00:00Z",
  updatedAt: "2024-06-02T00:00:00Z",
  lineDispatches: [],
  assignees: [],
};

function renderCard(order: FactoriesWorkOrder) {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <WorkOrderCard
          entry={buildWorkOrderListEntry(order, factory)}
          organizationId="org-1"
          factoryKey="RF"
          factoryLines={[{ id: "line-a", name: "hotfix" }]}
          canDispatch
          canAssign
          dispatchingOrderIds={new Set()}
          isAssigneesSaving={false}
          onDispatch={vi.fn().mockResolvedValue(undefined)}
          onAssigneesSave={vi.fn().mockResolvedValue(undefined)}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("Spend on a task card", () => {
  it("shows the total spend for a finished task", () => {
    renderCard({ ...baseOrder, totalCostCents: "482", totalTokens: "12345" });

    const usage = screen.getByTestId("work-order-card-usage-wo-1");
    expect(usage).toHaveTextContent("$4.82");
    expect(usage).toHaveAttribute("title", expect.stringContaining("$4.82"));
  });

  it("stays quiet when the task has no recorded spend", () => {
    renderCard({ ...baseOrder, totalCostCents: "0", totalTokens: "0" });

    expect(screen.queryByTestId("work-order-card-usage-wo-1")).not.toBeInTheDocument();
  });
});
