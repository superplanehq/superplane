import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
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
import { MOBILE_TASK_COPY } from "./mobileCopy";
import { MobileTaskDetailPage } from "./MobileTaskDetailPage";

const useWorkOrder = vi.fn((): { data: FactoriesWorkOrder | undefined; isLoading: boolean; isError: boolean } => ({
  data: undefined,
  isLoading: true,
  isError: false,
}));

const { findPlanningSession, liveCanvas } = vi.hoisted(() => ({
  findPlanningSession: vi.fn(),
  liveCanvas: { current: undefined as unknown },
}));

vi.mock("../pages/planningSessionClient", () => ({
  findPlanningSessionByWorkOrder: (...args: unknown[]) => findPlanningSession(...args),
  sendPlanningSessionMessage: vi.fn(),
  answerPlanningSessionSurvey: vi.fn(),
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
    liveCanvas.current = undefined;
  });

  it("keeps the description and explains why replies are not available", async () => {
    useWorkOrder.mockReturnValue({ data: DRAFT_WORK_ORDER, isLoading: false, isError: false });
    renderTask(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/${DRAFT_WORK_ORDER.number}?lineId=${REFUND_LINE_PLAN_ID}`,
    );

    expect(await screen.findByText(MOBILE_TASK_COPY.noRefinementSession)).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Description" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-composer")).not.toBeInTheDocument();
  });
});

describe("MobileTaskDetailPage stored phase log", () => {
  beforeEach(() => {
    findPlanningSession.mockReset();
    findPlanningSession.mockResolvedValue(null);
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
