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

function renderCard(order: FactoriesWorkOrderSummary = draft, isAnalyzing = false, creditLabel?: string) {
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
            creditLabel={creditLabel}
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

  it("shows No credit when backlog analysis stopped for hosted credit", () => {
    renderCard(draft, false, "No credit");

    expect(screen.getByText("No credit")).toBeInTheDocument();
  });

  it("hides the analysis credit label after the task leaves the backlog", () => {
    renderCard({ ...draft, state: "STATE_CLOSED", result: "RESULT_COMPLETED" }, false, "No credit");

    expect(screen.queryByText("No credit")).not.toBeInTheDocument();
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

  it("shows merge confidence on a verify card and does not show Confidence", () => {
    renderCard({
      ...draft,
      id: "wo-verify",
      state: "STATE_OPEN",
      title: "Ship refund retries",
      checkScores: [
        { key: "risk-review", name: "Blast radius", score: 1, maxScore: 5 },
        { key: "drift-review", name: "Drift", score: 1, maxScore: 5 },
        { key: "code-coverage", name: "Code quality", score: 80, maxScore: 100 },
      ],
    });

    expect(screen.getByRole("img", { name: "Merge confidence 5 of 5" })).toHaveTextContent(/Merge\s*5\/5/);
    expect(screen.queryByText("Confidence")).not.toBeInTheDocument();
    expect(screen.queryByText("Blast radius")).not.toBeInTheDocument();
  });

  it("shows the weakest flipped merge confidence check", () => {
    renderCard({
      ...draft,
      id: "wo-verify",
      state: "STATE_OPEN",
      checkScores: [
        { key: "performance-review", name: "Performance", score: 5, maxScore: 5 },
        { key: "risk-review", name: "Blast radius", score: 3, maxScore: 5 },
      ],
    });

    expect(screen.getByRole("img", { name: "Merge confidence 3 of 5" })).toHaveTextContent(/Merge\s*3\/5/);
  });

  it("hides the merge chip when no merge confidence check exists", () => {
    renderCard({
      ...draft,
      id: "wo-open",
      state: "STATE_OPEN",
      checkScores: [{ key: "code-coverage", name: "Code quality", score: 80, maxScore: 100 }],
    });

    expect(screen.queryByRole("img", { name: /Merge confidence/ })).not.toBeInTheDocument();
  });

  it("shows merge confidence on a done card", () => {
    renderCard({
      ...draft,
      id: "wo-done",
      state: "STATE_CLOSED",
      result: "RESULT_COMPLETED",
      checkScores: [{ key: "security-review", name: "Security", score: 4, maxScore: 5 }],
    });

    expect(screen.getByRole("img", { name: "Merge confidence 4 of 5" })).toHaveTextContent(/Merge\s*4\/5/);
  });

  it("shows Clarity and Confidence only on a draft", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <FactoriesLayoutContext.Provider
            value={{
              ...layout,
              factory: {
                id: "factory-1",
                name: "Refunds",
                key: "RF",
                planning: { enabled: true, clarity: true, confidence: true },
              },
            }}
          >
            <LineBoardOrderCard
              order={{
                ...draft,
                checkScores: [
                  { key: "clarity", name: "Clarity score", score: 5, maxScore: 5 },
                  { key: "confidence", name: "Confidence score", score: 3, maxScore: 5 },
                ],
              }}
              workOrderCardContext={cardContext}
              onOpenWorkOrder={vi.fn()}
            />
          </FactoriesLayoutContext.Provider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByText("Clarity")).toBeInTheDocument();
    expect(screen.getByText("Confidence")).toBeInTheDocument();
    expect(screen.queryByRole("img", { name: /Merge confidence/ })).not.toBeInTheDocument();
  });

  it("keeps merge confidence when planning hides Clarity and Confidence", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <FactoriesLayoutContext.Provider
            value={{
              ...layout,
              factory: {
                id: "factory-1",
                name: "Refunds",
                key: "RF",
                planning: { enabled: true, clarity: false, confidence: false },
              },
            }}
          >
            <LineBoardOrderCard
              order={{
                ...draft,
                id: "wo-hidden-planning",
                checkScores: [
                  { key: "clarity", name: "Clarity score", score: 5, maxScore: 5 },
                  { key: "confidence", name: "Confidence score", score: 2, maxScore: 5 },
                  { key: "reversibility-review", name: "Reversibility", score: 4, maxScore: 5 },
                ],
              }}
              workOrderCardContext={cardContext}
              onOpenWorkOrder={vi.fn()}
            />
          </FactoriesLayoutContext.Provider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByRole("img", { name: "Merge confidence 4 of 5" })).toBeInTheDocument();
    expect(screen.queryByText("Clarity")).not.toBeInTheDocument();
    expect(screen.queryByText("Confidence")).not.toBeInTheDocument();
  });
});
