import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesWorkOrder } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
  DRAFT_WORK_ORDER,
  RUNNING_WORK_ORDER,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { MobileTaskDetailPage } from "./MobileTaskDetailPage";

const useWorkOrder = vi.fn((): { data: FactoriesWorkOrder | undefined; isLoading: boolean; isError: boolean } => ({
  data: undefined,
  isLoading: true,
  isError: false,
}));

const { findPlanningSession, sendWorkOrderPlanningMessage, analyzingOrderIds, liveCanvas } = vi.hoisted(() => ({
  findPlanningSession: vi.fn(),
  sendWorkOrderPlanningMessage: vi.fn(),
  analyzingOrderIds: { current: new Set<string>() },
  liveCanvas: { current: undefined as unknown },
}));

vi.mock("../pages/planningSessionClient", () => ({
  findPlanningSessionByWorkOrder: (...args: unknown[]) => findPlanningSession(...args),
  sendPlanningSessionMessage: vi.fn(),
  sendWorkOrderPlanningMessage: (...args: unknown[]) => sendWorkOrderPlanningMessage(...args),
  answerPlanningSessionSurvey: vi.fn(),
}));

vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: () => ({ data: { activeTaskCount: 0, parallelTaskLimit: 1 } }),
}));

vi.mock("@/hooks/useBacklogAnalysisRuns", () => ({
  useFactoryBacklogAnalysis: () => ({ analyzingOrderIds: analyzingOrderIds.current, runsByWorkOrder: new Map() }),
}));

vi.mock("../pages/usePlanningSessionLiveRun", () => ({
  usePlanningSessionLiveRun: (_organizationId: string, view: unknown) => view,
}));

vi.mock("../pages/work-order-split-run/useSplitRunLiveCanvas", () => ({
  useSplitRunLiveCanvas: () =>
    liveCanvas.current ?? { enabled: false, isError: false, isLoading: false, canvas: undefined, stream: [] },
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useWorkOrder: () => useWorkOrder(),
  useWorkOrderArtifacts: () => ({ data: [] }),
  useFactoryAutomations: () => ({ data: [] }),
  useWorkOrderEvents: () => ({ data: { pages: [] } }),
  useCloseWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useFactoryPRFeedbackHandlers: () => ({ data: [] }),
}));

vi.mock("@/hooks/useWorkOrderCardActions", () => ({
  useWorkOrderCardActions: () => ({
    dispatchingOrderIds: new Set<string>(),
    isAssigneesSaving: false,
    onDispatch: vi.fn(),
    onAssigneesSave: vi.fn(),
  }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, currentUserId: "user-1", isLoading: false }),
}));

vi.mock("@/hooks/useFactoryLineRunnerModels", () => ({
  useFactoryLineRunnerModels: () => ({ data: [], isLoading: false }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({ data: [], isLoading: false }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({ data: [] }),
}));

beforeAll(() => {
  Element.prototype.scrollIntoView ??= () => {};
});

function renderTask(entry: string) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ThemeProvider>
        <TooltipProvider>
          <MemoryRouter initialEntries={[entry]}>
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
                <Route path="/org-1/workspaces/:factoryKey/task/:orderNumber" element={<MobileTaskDetailPage />} />
              </Routes>
            </FactoriesLayoutContext.Provider>
          </MemoryRouter>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>,
  );
}

function renderRunningTask() {
  useWorkOrder.mockReturnValue({ data: RUNNING_WORK_ORDER, isLoading: false, isError: false });
  renderTask(
    `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/${RUNNING_WORK_ORDER.number}?lineId=${REFUND_LINE_PLAN_ID}`,
  );
}

describe("MobileTaskDetailPage missing refinement session", () => {
  beforeEach(() => {
    findPlanningSession.mockReset();
    findPlanningSession.mockResolvedValue(null);
    sendWorkOrderPlanningMessage.mockReset();
    sendWorkOrderPlanningMessage.mockImplementation(() => new Promise(() => {}));
    analyzingOrderIds.current = new Set();
    liveCanvas.current = undefined;
  });

  it("lets a dequeued draft restart analysis before it has a planning session", async () => {
    const user = userEvent.setup();
    useWorkOrder.mockReturnValue({ data: DRAFT_WORK_ORDER, isLoading: false, isError: false });
    renderTask(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/${DRAFT_WORK_ORDER.number}?lineId=${REFUND_LINE_PLAN_ID}`,
    );

    const analyze = await screen.findByRole("button", { name: "Analyze" });
    expect(screen.getByTestId("split-run-intent-composer")).toBeEnabled();
    await user.click(analyze);

    await waitFor(() =>
      expect(sendWorkOrderPlanningMessage).toHaveBeenCalledWith(
        "org-1",
        PRIMARY_FACTORY_ID,
        DRAFT_WORK_ORDER.id,
        "Analyze this task.",
      ),
    );
  });

  it("shows the analysis queue position while it waits for a slot", async () => {
    analyzingOrderIds.current = new Set([DRAFT_WORK_ORDER.id ?? ""]);
    useWorkOrder.mockReturnValue({
      data: {
        ...DRAFT_WORK_ORDER,
        queuedTasks: [{ id: "analysis-queue", name: "Task analysis", position: 2 }],
      },
      isLoading: false,
      isError: false,
    });
    renderTask(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/${DRAFT_WORK_ORDER.number}?lineId=${REFUND_LINE_PLAN_ID}`,
    );

    await screen.findByTestId("split-run-intent-composer");
    expect(screen.getByRole("status")).toHaveTextContent("Queued #2");
    expect(screen.getByTestId("split-run-intent-composer")).toBeDisabled();
    expect(screen.queryByText("Analyzing task…")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^(Analyze|Queue analysis)$/ })).not.toBeInTheDocument();
  });
});

describe("MobileTaskDetailPage stored phase log", () => {
  beforeEach(() => {
    findPlanningSession.mockReset();
    findPlanningSession.mockResolvedValue(null);
  });

  it("keeps running status when another automation is queued on an active task", () => {
    useWorkOrder.mockReturnValue({
      data: {
        ...RUNNING_WORK_ORDER,
        queuedTasks: [{ id: "queue-follow-up", name: "Review", position: 2, taskActive: true }],
      },
      isLoading: false,
      isError: false,
    });
    renderTask(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/${RUNNING_WORK_ORDER.number}?lineId=${REFUND_LINE_PLAN_ID}`,
    );

    expect(screen.getByTestId("mobile-task-status")).toHaveTextContent("Running");
    expect(screen.getByRole("status")).toHaveTextContent("Queued #2");
  });

  it.each([
    ["not ready", { enabled: true, isError: false, isLoading: true, stream: [] }],
    ["failed", { enabled: true, isError: true, isLoading: false, stream: [] }],
  ])("keeps the stored summary when the live log is %s", async (_label, canvas) => {
    liveCanvas.current = canvas;
    renderRunningTask();

    const activity = await screen.findByTestId("mobile-task-activity");
    expect(within(activity).getByText("Implementation")).toBeInTheDocument();
    expect(screen.queryByText("Implement From Task Description")).not.toBeInTheDocument();
  });
});
