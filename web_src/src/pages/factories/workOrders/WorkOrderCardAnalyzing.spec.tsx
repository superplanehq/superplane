import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard } from "./WorkOrderCard";

const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: "RF" };

const order: FactoriesWorkOrder = {
  id: "wo-1",
  number: "1",
  title: "The site feels weird lately",
  state: "STATE_DRAFT",
  createdAt: "2026-08-28T10:00:00Z",
  updatedAt: "2026-08-28T10:00:00Z",
  lineDispatches: [],
  assignees: [],
};

function renderCard(props: { isAnalyzing?: boolean; confidenceScore?: number }) {
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
          {...props}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("Confidence score on a backlog card", () => {
  it("shows that analysis runs while the score is not ready", () => {
    renderCard({ isAnalyzing: true });

    expect(screen.getByTestId("work-order-card-analyzing-wo-1")).toHaveTextContent("Analyzing");
    expect(screen.queryByTestId("work-order-card-score-wo-1")).not.toBeInTheDocument();
  });

  it("replaces the spinner with the meter once the score arrives", () => {
    renderCard({ isAnalyzing: true, confidenceScore: 4 });

    expect(screen.queryByTestId("work-order-card-analyzing-wo-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-order-card-score-wo-1")).toHaveAttribute("aria-valuenow", "4");
  });

  it("stays quiet when no automation analyzes the task", () => {
    renderCard({});

    expect(screen.queryByTestId("work-order-card-analyzing-wo-1")).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-card-score-wo-1")).not.toBeInTheDocument();
  });
});
