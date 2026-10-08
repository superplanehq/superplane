import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation, useSearchParams } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
  REFUND_LINE_HOTFIX_ID,
  REFUND_LINE_PLAN_ID,
  DRAFT_WORK_ORDER,
  RUNNING_WORK_ORDER,
  CLOSED_WORK_ORDER,
  factoryWithPlanning,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { MOBILE_TASK_COPY } from "./mobileCopy";
import { MobileTaskDetailPage } from "./MobileTaskDetailPage";

const useWorkOrder = vi.fn((): { data: FactoriesWorkOrder | undefined; isLoading: boolean; isError: boolean } => ({
  data: undefined,
  isLoading: true,
  isError: false,
}));

const {
  onDispatch,
  dispatchingIds,
  canUpdateWorkOrder,
  runnerModelCalls,
  updateAssignees,
  findPlanningSession,
  sendPlanningMessage,
  answerPlanningSurvey,
  liveCanvas,
} = vi.hoisted(() => ({
  onDispatch: vi.fn(),
  dispatchingIds: { current: new Set<string>() },
  canUpdateWorkOrder: { current: true },
  runnerModelCalls: [] as unknown[][],
  updateAssignees: vi.fn(),
  findPlanningSession: vi.fn(),
  sendPlanningMessage: vi.fn(),
  answerPlanningSurvey: vi.fn(),
  liveCanvas: { current: undefined as unknown },
}));

vi.mock("../pages/planningSessionClient", () => ({
  findPlanningSessionByWorkOrder: (...args: unknown[]) => findPlanningSession(...args),
  sendPlanningSessionMessage: (...args: unknown[]) => sendPlanningMessage(...args),
  answerPlanningSessionSurvey: (...args: unknown[]) => answerPlanningSurvey(...args),
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
  useUpdateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: updateAssignees, isPending: false }),
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useFactoryPRFeedbackHandlers: () => ({ data: [] }),
}));

vi.mock("@/hooks/useWorkOrderCardActions", () => ({
  useWorkOrderCardActions: () => ({
    dispatchingOrderIds: dispatchingIds.current,
    isAssigneesSaving: false,
    onDispatch,
    onAssigneesSave: vi.fn(),
  }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => canUpdateWorkOrder.current, currentUserId: "user-1", isLoading: false }),
}));

vi.mock("@/hooks/useFactoryLineRunnerModels", () => ({
  useFactoryLineRunnerModels: (...args: unknown[]) => {
    runnerModelCalls.push(args);
    return {
      data: [
        { id: "claude-opus-4-6", name: "claude-opus-4-6" },
        { id: "long-model", name: "Anthropic Claude Opus 4.6 with extended context" },
      ],
      isLoading: false,
    };
  },
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({
    data: [
      {
        metadata: { id: "user-1", email: "casey@example.com" },
        spec: { displayName: "Casey Reviewer" },
      },
    ],
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({ data: [] }),
}));

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="mobile-task-location">{`${location.pathname}${location.search}`}</div>;
}

function TestLineSwitch() {
  const [, setSearchParams] = useSearchParams();
  return (
    <button
      type="button"
      data-testid="mobile-task-test-line"
      onClick={() => setSearchParams({ lineId: REFUND_LINE_HOTFIX_ID })}
    >
      Switch test line
    </button>
  );
}

function renderTask(
  entry: string | { pathname: string; search?: string; state?: unknown },
  factory: FactoriesFactory = REFUND_FACTORY,
) {
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
                factory,
                factories: [factory],
                openCreateWorkOrder: vi.fn(),
              }}
            >
              <TestLineSwitch />
              <Routes>
                <Route path="/org-1/workspaces/:factoryKey/task/:orderNumber" element={<MobileTaskDetailPage />} />
                <Route path="/org-1/workspaces/:factoryKey/lines/:lineId" element={<LocationProbe />} />
              </Routes>
            </FactoriesLayoutContext.Provider>
          </MemoryRouter>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>,
  );
}

function resetPlanningSession() {
  findPlanningSession.mockReset();
  findPlanningSession.mockResolvedValue(null);
  sendPlanningMessage.mockReset();
  answerPlanningSurvey.mockReset();
  liveCanvas.current = undefined;
}

function renderDraft(factory: FactoriesFactory = REFUND_FACTORY, search = `?lineId=${REFUND_LINE_PLAN_ID}`) {
  useWorkOrder.mockReturnValue({ data: DRAFT_WORK_ORDER, isLoading: false, isError: false });
  return renderTask(
    `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/${DRAFT_WORK_ORDER.number}${search}`,
    factory,
  );
}

const WAITING_SESSION = {
  id: "session-1",
  state: "running",
  executionId: "execution-1",
  waitState: "pending",
  messages: [{ id: "agent-1", role: "agent", text: "Which refunds are in scope?" }],
};

const SURVEY_SESSION = {
  ...WAITING_SESSION,
  survey: { id: "survey-1", questions: [{ prompt: "Priority?", options: ["High", "Low"] }] },
};

describe("MobileTaskDetailPage refine chat", () => {
  beforeEach(() => {
    useWorkOrder.mockReset();
    onDispatch.mockReset();
    dispatchingIds.current = new Set();
    canUpdateWorkOrder.current = true;
    runnerModelCalls.length = 0;
    resetPlanningSession();
  });

  it("shows the composer, one Start, and one model control on a Planning draft", async () => {
    findPlanningSession.mockResolvedValue(WAITING_SESSION);
    renderDraft();

    expect(await screen.findByTestId("split-run-intent-transcript")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-composer")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Start" })).toHaveLength(1);
    expect(screen.getAllByTestId("split-run-draft-model")).toHaveLength(1);
    expect(screen.getByTestId("mobile-task-detail")).toBeInTheDocument();
  });

  it("does not show the activity list or a phase log in the refine chat, like the desktop popup", async () => {
    findPlanningSession.mockResolvedValue(WAITING_SESSION);
    renderDraft();

    expect(await screen.findByTestId("split-run-intent-transcript")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: MOBILE_TASK_COPY.activity })).not.toBeInTheDocument();
    expect(screen.queryByTestId(/^mobile-task-phase-/)).not.toBeInTheDocument();
  });

  it("sends a text reply through the planning session message path", async () => {
    findPlanningSession.mockResolvedValue(WAITING_SESSION);
    sendPlanningMessage.mockResolvedValue(WAITING_SESSION);
    const user = userEvent.setup();
    renderDraft();

    const composer = await screen.findByTestId("split-run-intent-composer");
    await waitFor(() => expect(composer).toBeEnabled());
    await user.type(composer, "Only card refunds.");
    await user.click(screen.getByTestId("split-run-intent-composer-send"));

    await waitFor(() => {
      expect(sendPlanningMessage).toHaveBeenCalledWith("org-1", PRIMARY_FACTORY_ID, "session-1", "Only card refunds.");
    });
    expect(answerPlanningSurvey).not.toHaveBeenCalled();
  });

  it("answers a pending question through the survey answer path", async () => {
    findPlanningSession.mockResolvedValue(SURVEY_SESSION);
    answerPlanningSurvey.mockResolvedValue({ ...SURVEY_SESSION, survey: null });
    const user = userEvent.setup();
    renderDraft();

    expect(await screen.findByTestId("create-with-agent-survey")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /High/ }));
    await user.click(screen.getByRole("button", { name: "Send answers" }));

    await waitFor(() => {
      expect(answerPlanningSurvey).toHaveBeenCalledWith(
        "org-1",
        PRIMARY_FACTORY_ID,
        "session-1",
        expect.stringContaining("High"),
      );
    });
    expect(sendPlanningMessage).not.toHaveBeenCalled();
  });

  it("shows the recovery sentence when the session does not load", async () => {
    findPlanningSession.mockRejectedValue(new Error("boom"));
    renderDraft();

    expect(
      await screen.findByText("The refinement session did not load. Refresh the page to try again."),
    ).toBeInTheDocument();
  });

  it("lets a viewer read the chat but not send, answer, or start", async () => {
    canUpdateWorkOrder.current = false;
    findPlanningSession.mockResolvedValue(WAITING_SESSION);
    renderDraft();

    expect(await screen.findByTestId("split-run-intent-transcript")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-composer")).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Start" })).not.toBeInTheDocument();
  });

  it("keeps the description and Start without a composer when Planning is off", () => {
    renderDraft(factoryWithPlanning(REFUND_FACTORY, { enabled: false, clarity: false, confidence: false }));

    expect(screen.getByRole("heading", { name: "Description" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-composer")).not.toBeInTheDocument();
    expect(findPlanningSession).not.toHaveBeenCalled();
  });
});

describe("MobileTaskDetailPage phase log", () => {
  beforeEach(() => {
    useWorkOrder.mockReset();
    canUpdateWorkOrder.current = true;
    resetPlanningSession();
  });

  it("shows the desktop console card for a task that is not a draft", async () => {
    liveCanvas.current = {
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: { key: "live", title: "Implementation", nodes: [], edges: [], statuses: {} },
      stream: [],
    };
    useWorkOrder.mockReturnValue({ data: RUNNING_WORK_ORDER, isLoading: false, isError: false });
    renderTask(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/${RUNNING_WORK_ORDER.number}?lineId=${REFUND_LINE_PLAN_ID}`,
    );

    expect(await screen.findByTestId("mobile-task-activity")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-task-column-implement")).toHaveTextContent("Implement");
    expect(screen.getByTestId("mobile-task-column-implement")).toHaveTextContent("Implementation");
    expect(screen.getByText("1 agent run")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-composer")).not.toBeInTheDocument();
    expect(findPlanningSession).not.toHaveBeenCalled();
  });
});

describe("MobileTaskDetailPage owner", () => {
  beforeEach(() => {
    useWorkOrder.mockReset();
    onDispatch.mockReset();
    dispatchingIds.current = new Set();
    canUpdateWorkOrder.current = true;
    updateAssignees.mockReset();
  });

  function renderRunning() {
    useWorkOrder.mockReturnValue({ data: RUNNING_WORK_ORDER, isLoading: false, isError: false });
    return renderTask(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/${RUNNING_WORK_ORDER.number}?lineId=${REFUND_LINE_PLAN_ID}`,
    );
  }

  it("assigns the task to the current user from the header", async () => {
    const user = userEvent.setup();
    updateAssignees.mockResolvedValue({
      assignees: [{ id: "user-1", name: "Casey Reviewer" }],
    });
    renderRunning();

    await user.click(screen.getByTestId("popup-edit-owner"));
    await user.click(screen.getByRole("option", { name: "Casey Reviewer" }));

    await waitFor(() =>
      expect(updateAssignees).toHaveBeenCalledWith({
        orderId: RUNNING_WORK_ORDER.id,
        assigneeIds: ["user-1"],
      }),
    );
    expect(screen.getByRole("button", { name: "Owner: Casey Reviewer" })).toBeInTheDocument();
  });

  it("keeps the owner as text without task update permission", () => {
    canUpdateWorkOrder.current = false;
    renderRunning();

    expect(screen.queryByTestId("popup-edit-owner")).not.toBeInTheDocument();
    expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("Leonardo DiCaprio");
  });

  it("does not offer an owner change on a completed task", () => {
    useWorkOrder.mockReturnValue({ data: CLOSED_WORK_ORDER, isLoading: false, isError: false });
    renderTask(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/${CLOSED_WORK_ORDER.number}?lineId=${REFUND_LINE_PLAN_ID}`,
    );

    expect(screen.queryByTestId("popup-edit-owner")).not.toBeInTheDocument();
    expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("Leonardo DiCaprio");
  });
});
