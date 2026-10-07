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
  factoryWithPlanning,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
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
  findPlanningSession,
  sendPlanningMessage,
  answerPlanningSurvey,
  liveCanvas,
} = vi.hoisted(() => ({
  onDispatch: vi.fn(),
  dispatchingIds: { current: new Set<string>() },
  canUpdateWorkOrder: { current: true },
  runnerModelCalls: [] as unknown[][],
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

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({ data: [], isLoading: false }),
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

describe("MobileTaskDetailPage back link", () => {
  beforeEach(() => {
    resetPlanningSession();
    useWorkOrder.mockReset();
    useWorkOrder.mockReturnValue({ data: undefined, isLoading: true, isError: false });
    onDispatch.mockReset();
    dispatchingIds.current = new Set();
    canUpdateWorkOrder.current = true;
    runnerModelCalls.length = 0;
  });

  it("returns to the clicked line when the task URL has no line id", async () => {
    const user = userEvent.setup();
    renderTask({
      pathname: `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42`,
      state: { lineId: REFUND_LINE_HOTFIX_ID },
    });

    await user.click(screen.getByTestId("mobile-task-back"));

    expect(screen.getByTestId("mobile-task-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_HOTFIX_ID}`,
    );
  });

  it("returns to the line in the task URL when that line still exists", async () => {
    const user = userEvent.setup();
    renderTask(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42?lineId=${REFUND_LINE_HOTFIX_ID}`);

    await user.click(screen.getByTestId("mobile-task-back"));

    expect(screen.getByTestId("mobile-task-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_HOTFIX_ID}`,
    );
  });

  it("does not open the first line while a refreshed task is loading", async () => {
    const user = userEvent.setup();
    renderTask(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42`);

    const back = screen.getByTestId("mobile-task-back");
    expect(back).toBeDisabled();
    await user.click(back);

    expect(screen.queryByTestId("mobile-task-location")).toBeNull();
    expect(screen.getByText("Loading task…")).toBeTruthy();
  });

  it("returns to the task dispatch line after a refresh", async () => {
    useWorkOrder.mockReturnValue({
      data: {
        id: "wo-42",
        number: "42",
        title: "Fix refund rounding",
        lineDispatches: [
          { id: "dispatch-hotfix", createdAt: "2026-09-28T00:00:00.000Z", line: { id: REFUND_LINE_HOTFIX_ID } },
        ],
      },
      isLoading: false,
      isError: false,
    });
    const user = userEvent.setup();
    renderTask(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42`);

    await user.click(screen.getByTestId("mobile-task-back"));

    expect(screen.getByTestId("mobile-task-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_HOTFIX_ID}`,
    );
  });

  it("ignores a deleted line id and uses the task dispatch line", async () => {
    useWorkOrder.mockReturnValue({
      data: {
        id: "wo-42",
        number: "42",
        title: "Fix refund rounding",
        lineDispatches: [
          { id: "dispatch-plan", createdAt: "2026-09-28T00:00:00.000Z", line: { id: REFUND_LINE_PLAN_ID } },
        ],
      },
      isLoading: false,
      isError: false,
    });
    const user = userEvent.setup();
    renderTask(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42?lineId=deleted-line`);

    await user.click(screen.getByTestId("mobile-task-back"));

    expect(screen.getByTestId("mobile-task-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`,
    );
  });
});

const PLAN_LINE_NAME = "plan-and-implement";

function renderDraft(factory: FactoriesFactory = REFUND_FACTORY, search = `?lineId=${REFUND_LINE_PLAN_ID}`) {
  useWorkOrder.mockReturnValue({ data: DRAFT_WORK_ORDER, isLoading: false, isError: false });
  return renderTask(
    `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/${DRAFT_WORK_ORDER.number}${search}`,
    factory,
  );
}

describe("MobileTaskDetailPage model select", () => {
  beforeEach(() => {
    resetPlanningSession();
    useWorkOrder.mockReset();
    onDispatch.mockReset();
    dispatchingIds.current = new Set();
    canUpdateWorkOrder.current = true;
    runnerModelCalls.length = 0;
  });

  it("shows a model control next to Start on a draft when planning is on", () => {
    renderDraft();

    expect(screen.getByRole("button", { name: "Model: Auto Medium" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Start" })).toBeEnabled();
  });

  it("opens model and thinking choices in one menu and updates the closed label", async () => {
    const user = userEvent.setup();
    renderDraft();

    await user.click(screen.getByRole("button", { name: "Model: Auto Medium" }));

    expect(screen.getByRole("menuitem", { name: "Auto" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "claude-opus-4-6" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Low" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Medium" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "High" })).toBeInTheDocument();

    await user.click(screen.getByRole("menuitem", { name: "claude-opus-4-6" }));

    expect(screen.getByRole("button", { name: "Model: claude-opus-4-6 Medium" })).toBeInTheDocument();
  });

  it("keeps the full model name in the accessible label", async () => {
    const user = userEvent.setup();
    renderDraft();

    await user.click(screen.getByRole("button", { name: "Model: Auto Medium" }));
    await user.click(screen.getByRole("menuitem", { name: "Anthropic Claude Opus 4.6 with extended context" }));

    expect(
      screen.getByRole("button", { name: "Model: Anthropic Claude Opus 4.6 with extended context Medium" }),
    ).toBeInTheDocument();
  });

  it("starts on Auto without a model id and sends the thinking level", async () => {
    const user = userEvent.setup();
    renderDraft();

    await user.click(screen.getByRole("button", { name: "Start" }));

    expect(onDispatch).toHaveBeenCalledWith(DRAFT_WORK_ORDER.id, {
      lineName: PLAN_LINE_NAME,
      model: undefined,
      thinkingLevel: "medium",
    });
  });

  it("sends the chosen model and thinking level when Start runs", async () => {
    const user = userEvent.setup();
    renderDraft();

    await user.click(screen.getByRole("button", { name: "Model: Auto Medium" }));
    await user.click(screen.getByRole("menuitem", { name: "High" }));
    await user.click(screen.getByRole("button", { name: "Model: Auto High" }));
    await user.click(screen.getByRole("menuitem", { name: "claude-opus-4-6" }));
    await user.click(screen.getByRole("button", { name: "Start" }));

    expect(onDispatch).toHaveBeenCalledWith(DRAFT_WORK_ORDER.id, {
      lineName: PLAN_LINE_NAME,
      model: "claude-opus-4-6",
      thinkingLevel: "high",
    });
  });

  it("does not start with a model from the previous line", async () => {
    const user = userEvent.setup();
    renderDraft();

    await user.click(screen.getByRole("button", { name: "Model: Auto Medium" }));
    await user.click(screen.getByRole("menuitem", { name: "High" }));
    await user.click(screen.getByRole("button", { name: "Model: Auto High" }));
    await user.click(screen.getByRole("menuitem", { name: "claude-opus-4-6" }));
    await user.click(screen.getByTestId("mobile-task-test-line"));

    expect(screen.getByRole("button", { name: "Model: Auto High" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Start" }));

    expect(onDispatch).toHaveBeenCalledWith(DRAFT_WORK_ORDER.id, {
      lineName: "hotfix",
      model: undefined,
      thinkingLevel: "high",
    });
  });

  it("hides the model control on a running task", () => {
    useWorkOrder.mockReturnValue({ data: RUNNING_WORK_ORDER, isLoading: false, isError: false });
    renderTask(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/${RUNNING_WORK_ORDER.number}?lineId=${REFUND_LINE_PLAN_ID}`,
    );

    expect(screen.queryByTestId("split-run-draft-model")).not.toBeInTheDocument();
  });

  it("hides the model control and omits thinking when planning is off", async () => {
    const user = userEvent.setup();
    renderDraft(factoryWithPlanning(REFUND_FACTORY, { enabled: false, clarity: false, confidence: false }));

    expect(screen.queryByTestId("split-run-draft-model")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Start" }));

    expect(onDispatch).toHaveBeenCalledWith(DRAFT_WORK_ORDER.id, {
      lineName: PLAN_LINE_NAME,
      model: undefined,
      thinkingLevel: undefined,
    });
  });

  it("disables the model control while start is busy", () => {
    dispatchingIds.current = new Set([DRAFT_WORK_ORDER.id ?? ""]);
    renderDraft();

    expect(screen.getByRole("button", { name: "Model: Auto Medium" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Start" })).toBeDisabled();
  });

  it("keeps Start disabled and does not list models from another line when the start line has no name", () => {
    renderDraft(
      {
        ...REFUND_FACTORY,
        lines: [
          { id: REFUND_LINE_PLAN_ID, name: " " },
          { id: REFUND_LINE_HOTFIX_ID, name: "hotfix" },
        ],
      },
      `?lineId=${REFUND_LINE_PLAN_ID}`,
    );

    expect(screen.getByRole("button", { name: "Start" })).toBeDisabled();
    expect(runnerModelCalls.map((call) => call[2])).not.toContain("hotfix");
    expect(runnerModelCalls.every((call) => call[2] == null || call[2] === "")).toBe(true);
  });
});

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

  it("shows notes and command output in the open phase of a task that is not a draft", async () => {
    liveCanvas.current = {
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: { key: "live", title: "Implementation", nodes: [], edges: [], statuses: {} },
      stream: [
        { id: "node-1", nodeId: "node-1", at: "", componentName: "Run tests", status: "failed", kind: "action" },
        {
          id: "step-1",
          nodeId: "node-1",
          at: "",
          note: true,
          componentType: "bash",
          componentName: "npm test",
          detail: "FAIL refund.spec.ts",
          status: "failed",
        },
        {
          id: "note-1",
          nodeId: "node-1",
          at: "",
          note: true,
          noteParentId: "step-1",
          componentType: "note",
          componentName: "The refund case still fails.",
          status: "passed",
        },
      ],
    };
    useWorkOrder.mockReturnValue({ data: RUNNING_WORK_ORDER, isLoading: false, isError: false });
    renderTask(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/${RUNNING_WORK_ORDER.number}?lineId=${REFUND_LINE_PLAN_ID}`,
    );

    expect(await screen.findByText("The refund case still fails.")).toBeInTheDocument();
    expect(screen.getByText("FAIL refund.spec.ts")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-composer")).not.toBeInTheDocument();
    expect(findPlanningSession).not.toHaveBeenCalled();
  });
});
