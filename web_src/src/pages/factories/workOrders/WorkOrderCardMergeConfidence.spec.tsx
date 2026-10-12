import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard } from "./WorkOrderCard";

const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: "RF" };

function verifyOrder(id: string): FactoriesWorkOrder {
  return {
    id,
    number: "12",
    title: "Ship refund retries",
    state: "STATE_OPEN",
    createdAt: "2024-06-01T00:00:00Z",
    updatedAt: "2024-06-02T00:00:00Z",
    lineDispatches: [],
    assignees: [{ id: "user-1", name: "Ada Lovelace" }],
  };
}

function renderCard(
  order: FactoriesWorkOrder,
  mergeConfidence: {
    score: number;
    maxScore: number;
    checks?: { key: string; name: string; score: number; maxScore: number }[];
  },
) {
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
          onDispatch={vi.fn()}
          onAssigneesSave={vi.fn()}
          showClarity={false}
          showConfidenceScore={false}
          clarityScore={5}
          confidenceScore={3}
          mergeConfidence={mergeConfidence}
          onOpen={vi.fn()}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderCard merge confidence", () => {
  it("shows three bars in the footer and keeps planning scores hidden", async () => {
    const user = userEvent.setup();
    const order = verifyOrder("wo-verify");
    renderCard(order, {
      score: 4,
      maxScore: 5,
      checks: [
        { key: "risk-review", name: "Blast radius", score: 1, maxScore: 5 },
        { key: "security-review", name: "Security", score: 4, maxScore: 5 },
      ],
    });

    const chip = screen.getByRole("img", { name: "Merge confidence 4 of 5" });
    expect(chip).toHaveTextContent("Confidence");
    expect(chip).not.toHaveTextContent("Merge");
    expect(chip).not.toHaveTextContent(/\d/);
    expect(chip.className).not.toContain("rounded-full");
    const meter = screen.getByTestId("work-order-card-merge-wo-verify-meter");
    expect(meter.querySelectorAll("[data-filled]")).toHaveLength(3);
    expect(meter.querySelectorAll("[data-filled='true']")).toHaveLength(3);
    expect(meter.querySelector("[data-filled='true']")).toHaveClass("bg-emerald-500");
    expect(screen.queryByText("Clarity")).not.toBeInTheDocument();
    const footer = screen.getByTestId("work-order-card-footer-leading-wo-verify").parentElement;
    expect(footer).toContainElement(chip);
    expect(screen.getByTestId("work-order-card-title-trailing-wo-verify")).toContainElement(
      screen.getByTestId("work-order-row-assignees-wo-verify"),
    );

    await user.hover(chip);
    const popup = await screen.findByRole("tooltip");
    expect(popup).toHaveAttribute("data-side", "right");
    expect(popup).toHaveTextContent("Merge confidence");
    expect(popup).toHaveTextContent("Blast radius");
    expect(popup).toHaveTextContent("Low");
    expect(popup).toHaveTextContent("Security");
    expect(popup).toHaveTextContent("Met");
    expect(popup).not.toHaveTextContent("4/5");
  });

  it("uses the medium band when the score is 3", () => {
    renderCard(verifyOrder("wo-verify-mid"), { score: 3, maxScore: 5 });

    const chip = screen.getByRole("img", { name: "Merge confidence 3 of 5" });
    expect(chip).not.toHaveTextContent("Merge");
    expect(chip).not.toHaveTextContent(/\d/);
    const meter = screen.getByTestId("work-order-card-merge-wo-verify-mid-meter");
    expect(meter.querySelectorAll("[data-filled]")).toHaveLength(3);
    expect(meter.querySelectorAll("[data-filled='true']")).toHaveLength(2);
    expect(meter.querySelector("[data-filled='true']")).toHaveClass("bg-orange-500");
  });
});
