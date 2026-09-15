import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { CONFIDENCE_ANALYZING_TOOLTIP } from "./ConfidenceMeter";
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

function renderCard(props: { isAnalyzing?: boolean; confidenceScore?: number; hasAgentQuestion?: boolean }) {
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

function liveThinkingCopy(testId: string): string | undefined {
  return [...screen.getByTestId(testId).querySelectorAll(".t-think-text")].find(
    (el) => !el.classList.contains("is-exit"),
  )?.textContent;
}

describe("Confidence score on a backlog card", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows that analysis runs while the score is not ready", () => {
    renderCard({ isAnalyzing: true });

    const indicator = screen.getByTestId("work-order-card-analyzing-wo-1");
    expect(liveThinkingCopy("work-order-card-analyzing-wo-1")).toBe("Analyzing");
    expect(indicator.querySelector(".t-matrix")).not.toBeNull();
    expect(screen.queryByTestId("work-order-card-score-wo-1")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Start" })).not.toBeInTheDocument();
  });

  it("explains the full analysis step on hover", async () => {
    const user = userEvent.setup();
    renderCard({ isAnalyzing: true });

    await user.hover(screen.getByTestId("work-order-card-analyzing-wo-1"));

    const tip = await screen.findByRole("tooltip");
    expect(tip).toHaveTextContent(CONFIDENCE_ANALYZING_TOOLTIP);
    expect(tip).not.toHaveTextContent("Confidence score");
    expect(tip).not.toHaveTextContent("SuperPlane");
  });

  it("cycles thinking copy while analysis runs", () => {
    vi.useFakeTimers();
    renderCard({ isAnalyzing: true });

    expect(liveThinkingCopy("work-order-card-analyzing-wo-1")).toBe("Analyzing");
    act(() => {
      vi.advanceTimersByTime(2050);
    });
    expect(liveThinkingCopy("work-order-card-analyzing-wo-1")).toBe("Refining");
  });

  it("keeps thinking states while the agent still works after a score arrives", () => {
    renderCard({ isAnalyzing: true, confidenceScore: 4 });

    expect(liveThinkingCopy("work-order-card-analyzing-wo-1")).toBe("Analyzing");
    expect(screen.queryByTestId("work-order-card-score-wo-1")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Start" })).not.toBeInTheDocument();
  });

  it("shows the meter after analysis finishes", () => {
    renderCard({ confidenceScore: 4 });

    expect(screen.queryByTestId("work-order-card-analyzing-wo-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-order-card-score-wo-1")).toHaveAttribute("aria-valuenow", "4");
    expect(screen.getByRole("button", { name: "Start" })).toBeInTheDocument();
  });

  it("stays quiet when no automation analyzes the task", () => {
    renderCard({});

    expect(screen.queryByTestId("work-order-card-analyzing-wo-1")).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-card-score-wo-1")).not.toBeInTheDocument();
  });

  it("uses the same gray-blue hover fill as the factory run list", () => {
    renderCard({});

    expect(screen.getByTestId("work-order-card-wo-1").className).toContain("hover:bg-slate-100");
  });

  it("shows Agent question when the analysis waits for an answer", () => {
    renderCard({ isAnalyzing: true, hasAgentQuestion: true, confidenceScore: 4 });

    expect(screen.getByTestId("work-order-card-agent-question-wo-1")).toHaveTextContent("Agent question");
    expect(screen.queryByTestId("work-order-card-analyzing-wo-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-order-card-score-wo-1")).toHaveAttribute("aria-valuenow", "4");
    expect(screen.getByRole("button", { name: "Start" })).toBeInTheDocument();
  });

  it("hides Agent question after the task leaves the backlog", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <WorkOrderCard
            entry={buildWorkOrderListEntry({ ...order, state: "STATE_OPEN" }, factory)}
            organizationId="org-1"
            factoryKey="RF"
            factoryLines={[{ id: "line-a", name: "hotfix" }]}
            canDispatch
            canAssign
            dispatchingOrderIds={new Set()}
            isAssigneesSaving={false}
            onDispatch={vi.fn().mockResolvedValue(undefined)}
            onAssigneesSave={vi.fn().mockResolvedValue(undefined)}
            isAnalyzing
            hasAgentQuestion
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.queryByTestId("work-order-card-agent-question-wo-1")).not.toBeInTheDocument();
  });
});
