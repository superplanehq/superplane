import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import type * as canvasData from "@/hooks/useCanvasData";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { unmockedSrc } from "@/test/unmockedModule";

vi.mock("@monaco-editor/react", () => {
  function MockMonacoEditor({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) {
    return <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />;
  }
  return { default: MockMonacoEditor, Editor: MockMonacoEditor };
});
import { TooltipProvider } from "@/ui/tooltip";
import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { BOARD_DONE_REJECTED_ORDER } from "../__fixtures__/lineMetricsBoardOrders";
import { withPlanLinePhases } from "../__fixtures__/lineMetricsPlanLine";
import { LINE_COLUMN_SORT_STORAGE_KEY } from "../lib/lineColumnSort";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { LinesPage } from "./LinesPage";

const useFactoryWorkOrders = vi.fn(() => ({ data: [] as FactoriesWorkOrder[] }));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => useFactoryWorkOrders(),
  useFactoryAutomations: () => ({ data: [] }),
  useCreateFactoryLine: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateFactoryLine: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useWorkOrder: () => ({ data: undefined }),
  useWorkOrderEvents: () => ({ data: { pages: [] } }),
  useWorkOrderArtifacts: () => ({ data: [] }),
  useFactoryPullRequests: () => ({ data: [] }),
  useCreateFactoryAutomation: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteFactoryAutomation: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCloseWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCreateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({ data: [] }),
  useFactoryIntakeRuns: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn() }),
  useCreateFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false, error: null }),
  useSearchFactoryIntakeItems: () => ({ data: [], isLoading: false, isError: false }),
  useImportFactoryIntakeItem: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useRefreshBacklog: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useWorkOrderCardActions", () => ({
  useWorkOrderCardActions: () => ({
    dispatchingOrderIds: new Set<string>(),
    isAssigneesSaving: false,
    onDispatch: vi.fn(),
    onAssigneesSave: vi.fn(),
  }),
}));

vi.mock("@/pages/home/useInstallFactory", () => ({
  useInstallFactory: () => ({ installFactory: vi.fn(), isInstalling: false }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: () => undefined,
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: { id: "storybook-user" } }),
}));

vi.mock("@/hooks/useWorkOrderChecks", () => ({
  useWorkOrderChecks: () => ({ data: [], refetch: vi.fn() }),
  useWorkOrderChecksForOrders: () => [],
  ANALYZING_WORK_ORDER_CHECKS_POLL_MS: 1500,
}));

vi.mock("./useWorkOrderPlanningSurvey", () => ({
  useWorkOrderPlanningSurvey: () => false,
  useWorkOrderPlanningActivity: (
    _organizationId: string,
    _factoryId: string,
    _workOrderId: string,
    enabled: boolean,
    backlogAnalyzing = false,
  ) => ({
    hasAgentQuestion: false,
    isWaiting: false,
    isWorking: false,
    isAgentWorking: Boolean(enabled && backlogAnalyzing),
  }),
  workOrderPlanningSessionQueryKey: (organizationId: string, factoryId: string, workOrderId: string) => [
    "planning-session-by-work-order",
    organizationId,
    factoryId,
    workOrderId,
  ],
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useFactoryPRFeedbackHandlers: () => ({ data: [] }),
  useCreateFactoryPRFeedbackHandler: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useCanvasData", () => {
  const actual = unmockedSrc<typeof canvasData>("hooks/useCanvasData");
  return {
    ...actual,
    useCanvas: () => ({ data: { spec: { nodes: [] } }, isPending: false, isError: false }),
    useUpdateCanvasVersion: () => ({ mutateAsync: vi.fn(), isPending: false }),
    useCommitCanvasStaging: () => ({ mutateAsync: vi.fn(), isPending: false }),
  };
});

function renderBoard(factory: FactoriesFactory = REFUND_FACTORY) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <ThemeProvider>
        <TooltipProvider>
          <MemoryRouter initialEntries={[`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`]}>
            <FactoriesLayoutContext.Provider
              value={{
                organizationId: "org-1",
                factoryId: factory.id ?? PRIMARY_FACTORY_ID,
                factoryKey: factory.key ?? PRIMARY_FACTORY_KEY,
                factory,
                factories: [factory],
                openCreateWorkOrder: vi.fn(),
              }}
            >
              <Routes>
                <Route path="/org-1/workspaces/:factoryKey/lines/:lineId" element={<LinesPage />} />
              </Routes>
            </FactoriesLayoutContext.Provider>
          </MemoryRouter>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>,
  );
}

describe("LinesPage Done column", () => {
  beforeEach(() => {
    window.localStorage.clear();
    useFactoryWorkOrders.mockReturnValue({ data: [] });
  });

  it("always shows a Done column after the line stages", () => {
    renderBoard();

    expect(screen.getByTestId("lines-column-title-backlog")).toHaveTextContent("Backlog");
    expect(screen.getByTestId("lines-column-title-phase-0")).toBeInTheDocument();
    expect(screen.getByTestId("lines-column-title-phase-1")).toBeInTheDocument();
    expect(screen.getByTestId("lines-column-title-verify")).toHaveTextContent("Verify");
    expect(screen.getByTestId("lines-column-title-done")).toHaveTextContent("Done");
    expect(screen.getByTestId("lines-done-column")).toHaveTextContent("No tasks in Done.");
    expect(screen.getByTestId("lines-verify-column")).toHaveTextContent("No tasks in Verify.");
    expect(screen.getByTestId("lines-phase-column-1")).toHaveTextContent("Nothing here.");
    expect(screen.queryByTestId("lines-column-title-phase-2")).not.toBeInTheDocument();
  });

  it("puts a closed task in Done instead of the last stage", () => {
    useFactoryWorkOrders.mockReturnValue({
      data: [BOARD_DONE_REJECTED_ORDER],
    });
    renderBoard();

    expect(
      within(screen.getByTestId("lines-done-column")).getByText("Replace the refund batch exporter"),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-phase-column-0")).queryByText("Replace the refund batch exporter"),
    ).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-phase-column-1")).queryByText("Replace the refund batch exporter"),
    ).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-verify-column")).queryByText("Replace the refund batch exporter"),
    ).not.toBeInTheDocument();
  });

  it("collects finished tasks in the Done column", () => {
    useFactoryWorkOrders.mockReturnValue({
      data: [
        {
          id: "wo-completed",
          title: "Publish refund SLA dashboard",
          state: "STATE_CLOSED",
          result: "RESULT_COMPLETED",
          lineDispatches: [{ id: "dispatch-1", line: { id: REFUND_LINE_PLAN_ID } }],
        },
      ] as FactoriesWorkOrder[],
    });
    renderBoard();

    const done = screen.getByTestId("lines-done-column");
    expect(within(done).getByRole("button", { name: "Open Publish refund SLA dashboard" })).toBeInTheDocument();
    expect(screen.getByTestId("lines-phase-column-1")).toHaveTextContent("Nothing here.");
    expect(screen.getByTestId("lines-verify-column")).toHaveTextContent("No tasks in Verify.");
  });

  it("does not show a draft rejected out of the Backlog in Done — it archives off the board", () => {
    useFactoryWorkOrders.mockReturnValue({
      data: [
        {
          id: "wo-rejected-draft",
          title: "Retire the legacy refund webhook",
          state: "STATE_CLOSED",
          result: "RESULT_REJECTED",
          lineDispatches: [],
        },
      ] as FactoriesWorkOrder[],
    });
    renderBoard();

    expect(screen.getByTestId("lines-done-column")).toHaveTextContent("No tasks in Done.");
    expect(screen.queryByText("Retire the legacy refund webhook")).not.toBeInTheDocument();
  });

  it("keeps the bookend Done column when the line has its own Done automation", () => {
    const factory: FactoriesFactory = {
      ...REFUND_FACTORY,
      lines: (REFUND_FACTORY.lines ?? []).map(withPlanLinePhases),
    };
    renderBoard(factory);

    expect(screen.getByTestId("lines-done-column")).toBeInTheDocument();
    expect(screen.queryByTestId("lines-phase-column-2")).not.toBeInTheDocument();
  });

  it("reorders Done by result without changing Backlog", async () => {
    useFactoryWorkOrders.mockReturnValue({
      data: [
        {
          id: "wo-completed",
          title: "Completed task",
          state: "STATE_CLOSED",
          result: "RESULT_COMPLETED",
          createdAt: "2026-08-11T10:00:00.000Z",
          updatedAt: "2026-08-11T18:00:00.000Z",
          lineDispatches: [{ id: "dispatch-completed", line: { id: REFUND_LINE_PLAN_ID } }],
        },
        {
          id: "wo-failed",
          title: "Failed task",
          state: "STATE_CLOSED",
          result: "RESULT_FAILED",
          createdAt: "2026-08-11T11:00:00.000Z",
          updatedAt: "2026-08-11T17:00:00.000Z",
          lineDispatches: [{ id: "dispatch-failed", line: { id: REFUND_LINE_PLAN_ID } }],
        },
        {
          id: "wo-draft",
          title: "Draft task",
          state: "STATE_DRAFT",
          createdAt: "2026-08-11T09:00:00.000Z",
          updatedAt: "2026-08-11T19:00:00.000Z",
        },
      ] as FactoriesWorkOrder[],
    });
    const user = userEvent.setup();
    renderBoard();

    expect(doneCardIds()).toEqual(["work-order-card-wo-completed", "work-order-card-wo-failed"]);
    expect(within(screen.getByTestId("lines-backlog-column")).getByText("Draft task")).toBeInTheDocument();

    await user.click(screen.getByTestId("lines-done-menu"));
    expect(screen.queryByTestId("lines-done-menu-sort-confidence")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("lines-done-menu-sort-result"));

    expect(doneCardIds()).toEqual(["work-order-card-wo-failed", "work-order-card-wo-completed"]);
    expect(within(screen.getByTestId("lines-backlog-column")).getByText("Draft task")).toBeInTheDocument();
  });

  it("restores a stored Done sort after reload", () => {
    window.localStorage.setItem(
      LINE_COLUMN_SORT_STORAGE_KEY,
      JSON.stringify({ [REFUND_LINE_PLAN_ID]: { done: "result" } }),
    );
    useFactoryWorkOrders.mockReturnValue({
      data: [
        {
          id: "wo-completed",
          title: "Completed task",
          state: "STATE_CLOSED",
          result: "RESULT_COMPLETED",
          updatedAt: "2026-08-11T18:00:00.000Z",
          lineDispatches: [{ id: "dispatch-completed", line: { id: REFUND_LINE_PLAN_ID } }],
        },
        {
          id: "wo-failed",
          title: "Failed task",
          state: "STATE_CLOSED",
          result: "RESULT_FAILED",
          updatedAt: "2026-08-11T17:00:00.000Z",
          lineDispatches: [{ id: "dispatch-failed", line: { id: REFUND_LINE_PLAN_ID } }],
        },
      ] as FactoriesWorkOrder[],
    });
    renderBoard();

    expect(doneCardIds()).toEqual(["work-order-card-wo-failed", "work-order-card-wo-completed"]);
  });
});

function doneCardIds(): string[] {
  return [
    ...screen.getByTestId("lines-done-column-scroll").querySelectorAll("[data-testid^='work-order-card-wo-']"),
  ].map((node) => node.getAttribute("data-testid") ?? "");
}
