import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useNavigate } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import type * as canvasData from "@/hooks/useCanvasData";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { WORKSPACE_LOADING_TEST_ID } from "@/lib/workspaceLoadingCopy";
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
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
  REFUND_LINE_HOTFIX_ID,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { factoryRouteSegment, workOrderDetailPath } from "../lib/factoryPagePaths";
import { BOARD_DONE_REJECTED_ORDER } from "../__fixtures__/lineMetricsBoardOrders";
import { withPlanLinePhases } from "../__fixtures__/lineMetricsPlanLine";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { LinesPage } from "./LinesPage";

const idleBoardPage = () => ({ hasNextPage: false, isFetchingNextPage: false, fetchNextPage: vi.fn() });
const useFactoryWorkOrders = vi.fn(() => ({ data: [] as FactoriesWorkOrder[] }));
const useWorkOrder = vi.fn((..._args: unknown[]) => ({
  data: undefined as FactoriesWorkOrder | undefined,
  isLoading: false,
}));
function boardPageResult(isLoading = false, workOrders: FactoriesWorkOrder[] = []) {
  return {
    workOrders,
    isLoading,
    isPlaceholderData: false,
    backlog: idleBoardPage(),
    open: idleBoardPage(),
    done: idleBoardPage(),
  };
}
const useFactoryBoardWorkOrders = vi.fn((..._args: unknown[]) =>
  boardPageResult(false, useFactoryWorkOrders().data ?? []),
);

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => useFactoryWorkOrders(),
  useFactoryBoardWorkOrders: (...args: unknown[]) => useFactoryBoardWorkOrders(...args),
  useFactoryAutomations: () => ({ data: [] }),
  useCreateFactoryLine: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateFactoryLine: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useWorkOrder: (...args: unknown[]) => useWorkOrder(...args),
  useWorkOrderEvents: () => ({ data: { pages: [] } }),
  useWorkOrderArtifacts: () => ({ data: [] }),
  useCreateFactoryAutomation: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteFactoryAutomation: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCloseWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useSendWorkOrderToBacklog: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useFactoryWorkOrdersPage: () => ({
    orders: [],
    isLoading: false,
    isPlaceholderData: false,
    hasNextPage: false,
    fetchNextPage: vi.fn(),
    isFetchingNextPage: false,
  }),
  useCreateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({ data: [] }),
  useFactoryIntakeRuns: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn() }),
  useCreateFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false, error: null }),
  useDeleteFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false, error: null }),
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
  usePermissions: () => ({ canAct: () => true, currentUserId: "storybook-user", isLoading: false }),
}));

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: () => undefined,
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: { id: "storybook-user" } }),
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

function OpenTaskRouteButton({ href }: { href: string }) {
  const navigate = useNavigate();
  return (
    <button type="button" data-testid="open-task-route" onClick={() => navigate(href)}>
      Open task
    </button>
  );
}

function renderBoard(factory: FactoriesFactory = REFUND_FACTORY) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <ThemeProvider>
        <TooltipProvider>
          <MemoryRouter
            initialEntries={[`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`]}
          >
            <FactoriesLayoutContext.Provider
              value={{
                organizationId: "org-1",
                factoryId: factory.id ?? PRIMARY_FACTORY_ID,
                factoryKey: factory.key ?? PRIMARY_FACTORY_KEY,
                routeSegment: factoryRouteSegment(factory) || PRIMARY_FACTORY_ROUTE_SEGMENT,
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
    useWorkOrder.mockReturnValue({ data: undefined, isLoading: false });
    useFactoryBoardWorkOrders.mockImplementation((..._args: unknown[]) =>
      boardPageResult(false, useFactoryWorkOrders().data ?? []),
    );
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

  it("puts a completed task in Done instead of the last stage", () => {
    useFactoryWorkOrders.mockReturnValue({
      data: [
        {
          id: "wo-completed",
          title: "Publish refund SLA dashboard",
          state: "STATE_CLOSED",
          result: "RESULT_COMPLETED",
          lineDispatches: [{ id: "dispatch-1", line: { id: REFUND_LINE_PLAN_ID } }],
        },
      ],
    });
    renderBoard();

    expect(
      within(screen.getByTestId("lines-done-column")).getByText("Publish refund SLA dashboard"),
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-phase-column-0")).queryByText("Publish refund SLA dashboard"),
    ).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-phase-column-1")).queryByText("Publish refund SLA dashboard"),
    ).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("lines-verify-column")).queryByText("Publish refund SLA dashboard"),
    ).not.toBeInTheDocument();
  });

  it("keeps a rejected task out of Done", () => {
    useFactoryWorkOrders.mockReturnValue({
      data: [BOARD_DONE_REJECTED_ORDER],
    });
    renderBoard();

    expect(screen.getByTestId("lines-done-column")).toHaveTextContent("No tasks in Done.");
    expect(screen.queryByText("Replace the refund batch exporter")).not.toBeInTheDocument();
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

  it("asks the Done page for this line when no status filter is set", () => {
    renderBoard();

    expect(useFactoryBoardWorkOrders).toHaveBeenLastCalledWith(
      "org-1",
      PRIMARY_FACTORY_ID,
      expect.objectContaining({
        done: {
          lineId: REFUND_LINE_PLAN_ID,
          results: ["RESULT_COMPLETED", "RESULT_FAILED"],
        },
      }),
    );
  });

  it("asks the Done page for this line's failed tasks when Failed is selected", () => {
    window.localStorage.setItem(
      `sp:work-orders:filters:${PRIMARY_FACTORY_ID}`,
      JSON.stringify({
        statuses: ["failed"],
        labels: [],
        lineIds: [],
        sourceIds: [],
        assigneeIds: [],
      }),
    );
    renderBoard();

    expect(useFactoryBoardWorkOrders).toHaveBeenLastCalledWith(
      "org-1",
      PRIMARY_FACTORY_ID,
      expect.objectContaining({
        done: {
          lineId: REFUND_LINE_PLAN_ID,
          results: ["RESULT_FAILED"],
        },
      }),
    );
  });

  it("collects failed tasks in the Done column", () => {
    useFactoryWorkOrders.mockReturnValue({
      data: [
        {
          id: "wo-failed",
          title: "Fix refund dispatcher timeout loop",
          state: "STATE_CLOSED",
          result: "RESULT_FAILED",
          lineDispatches: [{ id: "dispatch-failed", line: { id: REFUND_LINE_PLAN_ID } }],
        },
      ] as FactoriesWorkOrder[],
    });
    renderBoard();

    const done = screen.getByTestId("lines-done-column");
    expect(within(done).getByRole("button", { name: "Open Fix refund dispatcher timeout loop" })).toBeInTheDocument();
    expect(screen.getByTestId("lines-phase-column-1")).toHaveTextContent("Nothing here.");
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

  it("keeps the Done line and the board when a card opens for the first time", async () => {
    const user = userEvent.setup();
    const taskHref = workOrderDetailPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, "101", REFUND_LINE_PLAN_ID);
    useFactoryBoardWorkOrders.mockImplementation((...args: unknown[]) => {
      const options = args[2] as { done?: { lineId?: string } } | undefined;
      return boardPageResult(!options?.done?.lineId, useFactoryWorkOrders().data ?? []);
    });

    render(
      <QueryClientProvider client={new QueryClient()}>
        <ThemeProvider>
          <TooltipProvider>
            <MemoryRouter
              initialEntries={[`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`]}
            >
              <FactoriesLayoutContext.Provider
                value={{
                  organizationId: "org-1",
                  factoryId: PRIMARY_FACTORY_ID,
                  factoryKey: PRIMARY_FACTORY_KEY,
                  routeSegment: PRIMARY_FACTORY_ROUTE_SEGMENT,
                  factory: REFUND_FACTORY,
                  factories: [REFUND_FACTORY],
                  openCreateWorkOrder: vi.fn(),
                }}
              >
                <Routes>
                  <Route path="/org-1/workspaces/:factoryKey/lines/:lineId" element={<LinesPage />} />
                  <Route path="/org-1/workspaces/:factoryKey/task/:orderNumber" element={<LinesPage />} />
                </Routes>
                <OpenTaskRouteButton href={taskHref} />
              </FactoriesLayoutContext.Provider>
            </MemoryRouter>
          </TooltipProvider>
        </ThemeProvider>
      </QueryClientProvider>,
    );

    expect(screen.queryByTestId(WORKSPACE_LOADING_TEST_ID)).not.toBeInTheDocument();

    await user.click(screen.getByTestId("open-task-route"));

    expect(useFactoryBoardWorkOrders).toHaveBeenLastCalledWith(
      "org-1",
      PRIMARY_FACTORY_ID,
      expect.objectContaining({
        done: {
          lineId: REFUND_LINE_PLAN_ID,
          results: ["RESULT_COMPLETED", "RESULT_FAILED"],
        },
      }),
    );
    expect(screen.queryByTestId(WORKSPACE_LOADING_TEST_ID)).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-done-column")).toBeInTheDocument();
  });

  it("scopes Done to the displayed line when the task URL line was deleted", () => {
    const order = {
      id: "wo-done-hotfix",
      number: "77",
      title: "Ship the hotfix",
      state: "STATE_CLOSED",
      result: "RESULT_COMPLETED",
      lineDispatches: [
        {
          id: "dispatch-hotfix",
          createdAt: "2026-09-28T00:00:00.000Z",
          line: { id: REFUND_LINE_HOTFIX_ID },
        },
      ],
    } as FactoriesWorkOrder;
    useWorkOrder.mockReturnValue({ data: order, isLoading: false });

    render(
      <QueryClientProvider client={new QueryClient()}>
        <ThemeProvider>
          <TooltipProvider>
            <MemoryRouter
              initialEntries={[`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/77?lineId=deleted-line`]}
            >
              <FactoriesLayoutContext.Provider
                value={{
                  organizationId: "org-1",
                  factoryId: PRIMARY_FACTORY_ID,
                  factoryKey: PRIMARY_FACTORY_KEY,
                  routeSegment: PRIMARY_FACTORY_ROUTE_SEGMENT,
                  factory: REFUND_FACTORY,
                  factories: [REFUND_FACTORY],
                  openCreateWorkOrder: vi.fn(),
                }}
              >
                <Routes>
                  <Route path="/org-1/workspaces/:factoryKey/task/:orderNumber" element={<LinesPage />} />
                </Routes>
              </FactoriesLayoutContext.Provider>
            </MemoryRouter>
          </TooltipProvider>
        </ThemeProvider>
      </QueryClientProvider>,
    );

    expect(useFactoryBoardWorkOrders).toHaveBeenLastCalledWith(
      "org-1",
      PRIMARY_FACTORY_ID,
      expect.objectContaining({
        done: {
          lineId: REFUND_LINE_HOTFIX_ID,
          results: ["RESULT_COMPLETED", "RESULT_FAILED"],
        },
      }),
    );
    for (const call of useFactoryBoardWorkOrders.mock.calls) {
      const options = call[2] as { done?: { lineId?: string } };
      expect(options.done?.lineId).not.toBe("deleted-line");
    }
    expect(screen.getByTestId("lines-done-column")).toBeInTheDocument();
  });

  it("does not show the first line while a task link is still loading", () => {
    useWorkOrder.mockReturnValue({ data: undefined, isLoading: true });
    useFactoryBoardWorkOrders.mockClear();

    render(
      <QueryClientProvider client={new QueryClient()}>
        <ThemeProvider>
          <TooltipProvider>
            <MemoryRouter initialEntries={[`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/77`]}>
              <FactoriesLayoutContext.Provider
                value={{
                  organizationId: "org-1",
                  factoryId: PRIMARY_FACTORY_ID,
                  factoryKey: PRIMARY_FACTORY_KEY,
                  routeSegment: PRIMARY_FACTORY_ROUTE_SEGMENT,
                  factory: REFUND_FACTORY,
                  factories: [REFUND_FACTORY],
                  openCreateWorkOrder: vi.fn(),
                }}
              >
                <Routes>
                  <Route path="/org-1/workspaces/:factoryKey/task/:orderNumber" element={<LinesPage />} />
                </Routes>
              </FactoriesLayoutContext.Provider>
            </MemoryRouter>
          </TooltipProvider>
        </ThemeProvider>
      </QueryClientProvider>,
    );

    expect(screen.getByText("Loading task…")).toBeInTheDocument();
    expect(screen.queryByTestId("lines-done-column")).not.toBeInTheDocument();
    expect(useFactoryBoardWorkOrders).toHaveBeenCalled();
    for (const call of useFactoryBoardWorkOrders.mock.calls) {
      const options = call[2] as { done?: { lineId?: string; enabled?: boolean } };
      expect(options.done?.lineId).toBeUndefined();
      expect(options.done?.enabled).toBe(false);
    }
  });
});
