import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import type { FactoriesWorkOrderSummary } from "@/api-client";

import { FactoriesLayoutContext, type FactoriesLayoutContextValue } from "../layout/factoriesLayoutContext";
import type { WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { LineBoardOrderCard } from "./LineBoardOrderCard";

const layout: FactoriesLayoutContextValue = {
  organizationId: "org-1",
  factoryId: "factory-1",
  factoryKey: "RF",
  factory: { id: "factory-1", name: "Refunds", key: "RF" },
  factories: [],
  openCreateWorkOrder: () => undefined,
};

const cardContext: WorkOrderCardContext = {
  organizationId: "org-1",
  factoryId: "factory-1",
  factoryKey: "RF",
  factoryLines: [{ id: "line-a", name: "hotfix" }],
  canDispatch: true,
  canAssign: true,
  dispatchingOrderIds: new Set(),
  isAssigneesSaving: false,
  onDispatch: vi.fn().mockResolvedValue(undefined),
  onAssigneesSave: vi.fn().mockResolvedValue(undefined),
};

const draft: FactoriesWorkOrderSummary = {
  id: "wo-1",
  number: "1",
  title: "The site feels weird lately",
  state: "STATE_DRAFT",
  createdAt: "2026-08-28T10:00:00Z",
  updatedAt: "2026-08-28T10:00:00Z",
  lineDispatches: [],
  assignees: [],
};

function renderCard(order: FactoriesWorkOrderSummary = draft, isAnalyzing = false) {
  const onOpenWorkOrder = vi.fn();
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <FactoriesLayoutContext.Provider value={layout}>
          <LineBoardOrderCard
            order={order}
            workOrderCardContext={cardContext}
            onOpenWorkOrder={onOpenWorkOrder}
            isAnalyzing={isAnalyzing}
          />
        </FactoriesLayoutContext.Provider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return onOpenWorkOrder;
}

function planningSessionRequests(fetchMock: ReturnType<typeof vi.spyOn>): string[] {
  const urls: string[] = [];
  for (const call of fetchMock.mock.calls) {
    const url = String(call[0]);
    if (url.includes("/planning-session")) {
      urls.push(url);
    }
  }
  return urls;
}

describe("LineBoardOrderCard", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("does not load a planning session for a draft on the board", () => {
    const fetchMock = vi.spyOn(globalThis, "fetch");
    renderCard();

    expect(planningSessionRequests(fetchMock)).toEqual([]);
    expect(screen.queryByTestId("work-order-card-analyzing-wo-1")).not.toBeInTheDocument();
  });

  it("shows a local backlog analysis without loading a planning session", () => {
    const fetchMock = vi.spyOn(globalThis, "fetch");
    renderCard(draft, true);

    expect(screen.getByTestId("work-order-card-analyzing-wo-1")).toBeInTheDocument();
    expect(planningSessionRequests(fetchMock)).toEqual([]);
  });

  it("shows that the agent is working from the planning session summary", () => {
    const fetchMock = vi.spyOn(globalThis, "fetch");
    renderCard({
      ...draft,
      planningSession: { id: "ps-1", state: "running", executionId: "exec-1" },
      checkScores: [{ key: "confidence", name: "Confidence score", score: 4, maxScore: 5 }],
    });

    expect(screen.getByTestId("work-order-card-analyzing-wo-1")).toBeInTheDocument();
    expect(planningSessionRequests(fetchMock)).toEqual([]);
  });

  it("shows an agent question from the planning session summary", () => {
    const fetchMock = vi.spyOn(globalThis, "fetch");
    renderCard({
      ...draft,
      planningSession: {
        id: "ps-1",
        state: "running",
        waitState: "pending",
        executionId: "exec-1",
        survey: { id: "survey-1", questions: [{ prompt: "Which API?", options: ["REST"] }] },
      },
    });

    expect(screen.getByTestId("work-order-card-agent-question-wo-1")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-card-analyzing-wo-1")).not.toBeInTheDocument();
    expect(planningSessionRequests(fetchMock)).toEqual([]);
  });
});
