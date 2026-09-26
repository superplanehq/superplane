import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import type {
  FactoriesFactory,
  FactoriesFactoryIntake,
  FactoriesWorkOrder,
  FactoriesWorkOrderSummary,
  FactoryAutomation,
} from "@/api-client";
import type * as canvasData from "@/hooks/useCanvasData";
import { resetFactoryBoardLaneScrollPositions } from "@/hooks/useFactoryBoardLaneScroll";
import {
  FEATURE_FACTORY_CUSTOM_AUTOMATIONS,
  FEATURE_FACTORY_DEPENDABOT_INTAKE,
  FEATURE_FACTORY_JIRA_INTAKE,
  FEATURE_FACTORY_PRODUCTIVE_INTAKE,
  FEATURE_FACTORY_SENTRY_INTAKE,
} from "@/lib/experimentalFeatures";
import { unmockedSrc } from "@/test/unmockedModule";

vi.mock("@monaco-editor/react", () => {
  function MockMonacoEditor({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) {
    return <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />;
  }
  return { default: MockMonacoEditor, Editor: MockMonacoEditor };
});
import {
  factoryAppConfigurePath,
  factoryColumnAutomationViewPath,
  factoryDependabotIntakeSetupPath,
  factoryHomePath,
  factoryJiraIntakeSetupPath,
  factoryPlanningPath,
  factoryProductiveIntakeSetupPath,
  factoryPlanningSetupPath,
  factoryPRFeedbackPath,
  factoryPRFeedbackSetupPath,
  factorySentryIntakeSetupPath,
  firstFactoryLineId,
} from "../lib/factoryPagePaths";
import {
  ACME_ONBOARDING_FACTORY,
  ACME_ONBOARDING_FACTORY_KEY,
  ACME_ONBOARDING_LINE_ID,
  DEFAULT_FACTORY_PLANNING,
  DRAFT_WORK_ORDER,
  factoryWithPlanning,
  GITHUB_ISSUES_INTAKE,
  GITHUB_ISSUES_INTAKE_APP,
  GITHUB_ISSUES_INTAKE_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_LINE_HOTFIX_ID,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import {
  BOARD_DONE_REJECTED_ORDER,
  BOARD_IMPLEMENT_FAILED_ORDER,
  BOARD_IMPLEMENT_NOTIFY_ORDER,
} from "../__fixtures__/lineMetricsBoardOrders";
import { planLineActiveDispatch } from "../__fixtures__/lineMetricsPlanLine";
import { DEFAULT_CHECKS_BY_ORDER_ID } from "../__fixtures__/workOrderCheckFixtures";
import { clearBacklogAnalysisPending, markBacklogAnalysisPending } from "../lib/backlogAnalysis";
import type { FactoryPreviewFlags } from "./factoryPreviewFlagsContext";
import { lineBoardColumnLaneProps } from "./lineBoardColumnColors";
import { LinesBoardSpecHarness } from "./linesPageSpecRender";
import { ADD_INTAKE_COPY } from "./lineIntakeModel";
import { canvasQuery, canvasWithoutAgent, implementerCanvas } from "./linesPageCanvasFixtures";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "./onboarding/first-run/reviewCandidates";

function withBoardChecks(orders: FactoriesWorkOrder[]): FactoriesWorkOrderSummary[] {
  return orders.map((order) => {
    const checks = DEFAULT_CHECKS_BY_ORDER_ID[order.id ?? ""] ?? order.checks;
    return {
      ...order,
      checkScores: checks?.map((check) => ({
        key: check.key,
        name: check.name,
        score: check.score,
        maxScore: check.maxScore,
      })),
    };
  });
}

const LANE_BANNERS: FactoryPreviewFlags = { addIntakeControl: false, columnAutomations: false };
const ICON_VIEW: FactoryPreviewFlags = {
  addIntakeControl: false,
  columnAutomations: true,
  columnAutomationRows: false,
};

function renderLinesBoard(
  path = `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`,
  openCreateWorkOrder = vi.fn(),
  factory: FactoriesFactory = REFUND_FACTORY,
  previewFlags: FactoryPreviewFlags | null = null,
) {
  return render(
    <LinesBoardSpecHarness
      path={path}
      openCreateWorkOrder={openCreateWorkOrder}
      factory={factory}
      previewFlags={previewFlags}
    />,
  );
}

const createFactoryLineMutateAsync = vi.fn();
const updateFactoryLineMutateAsync = vi.fn();
const updateLineIsPending = vi.hoisted(() => ({ value: false }));
const idleBoardPage = () => ({ hasNextPage: false, isFetchingNextPage: false, fetchNextPage: vi.fn() });
const useFactoryWorkOrders = vi.fn(() => ({ data: [] as FactoriesWorkOrderSummary[] }));
const useFactoryBoardWorkOrders = vi.fn(() => ({
  workOrders: useFactoryWorkOrders().data ?? [],
  isLoading: false,
  isPlaceholderData: false,
  backlog: idleBoardPage(),
  open: idleBoardPage(),
  done: idleBoardPage(),
}));
const useWorkOrder = vi.fn(() => ({ data: undefined as FactoriesWorkOrder | undefined }));
const useFactoryAutomations = vi.fn(() => ({ data: [] as FactoryAutomation[] }));
const useFactoryIntakes = vi.fn(() => ({ data: [] as FactoriesFactoryIntake[] }));
const createFactoryIntakeMutateAsync = vi.fn();
const createFactoryAutomationMutateAsync = vi.fn();
const deleteFactoryAutomationMutateAsync = vi.fn();
const useFactoryPRFeedbackHandlers = vi.fn(
  (): {
    data?: { id?: string; source?: string; healthy?: boolean }[];
    isPending?: boolean;
    isError?: boolean;
  } => ({
    data: [],
    isPending: false,
    isError: false,
  }),
);
const createFactoryPRFeedbackHandler = vi.fn();
const searchFactoryIntakeItems = vi.fn(() => ({
  data: [] as { id: string; key: string; title: string; body: string; url: string }[],
  isLoading: false,
  isError: false,
}));
const importFactoryIntakeItem = vi.fn();
const refreshBacklogMutateAsync = vi.fn();

const PLANNING_OPEN_FACTORY = factoryWithPlanning(REFUND_FACTORY, { ...DEFAULT_FACTORY_PLANNING });

const SENTRY_INTAKE_ID = "intake-sentry";
const PAGERDUTY_INTAKE_ID = "intake-pagerduty";

const CONFIGURED_INTAKES: FactoriesFactoryIntake[] = [
  GITHUB_ISSUES_INTAKE,
  {
    id: SENTRY_INTAKE_ID,
    canvasId: "app-sentry-intake",
    name: "Sentry exceptions",
    source: "SOURCE_SENTRY_EXCEPTIONS",
    healthy: true,
  },
  {
    id: PAGERDUTY_INTAKE_ID,
    canvasId: "app-pagerduty-intake",
    name: "PagerDuty incidents",
    source: "SOURCE_PAGERDUTY_INCIDENTS",
    healthy: true,
  },
];

vi.mock("@/hooks/useFactoryData", () => ({
  useFactory: () => ({
    data: { id: "factory-1", planning: { enabled: true, clarity: true, confidence: true } },
    isPending: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useUpdateFactory: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useFactoryWorkOrders: () => useFactoryWorkOrders(),
  useFactoryBoardWorkOrders: () => useFactoryBoardWorkOrders(),
  useFactoryAutomations: () => useFactoryAutomations(),
  useCreateFactoryLine: () => ({ mutateAsync: createFactoryLineMutateAsync, isPending: false }),
  useUpdateFactoryLine: () => ({
    mutateAsync: updateFactoryLineMutateAsync,
    get isPending() {
      return updateLineIsPending.value;
    },
  }),
  useWorkOrder: () => useWorkOrder(),
  useWorkOrderEvents: () => ({ data: { pages: [] } }),
  useWorkOrderArtifacts: () => ({ data: [] }),
  useCreateFactoryAutomation: () => ({ mutateAsync: createFactoryAutomationMutateAsync, isPending: false }),
  useDeleteFactoryAutomation: () => ({ mutateAsync: deleteFactoryAutomationMutateAsync, isPending: false }),
  useCloseWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCreateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => useFactoryIntakes(),
  useFactoryIntakeRuns: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn() }),
  useCreateFactoryIntake: () => ({ mutateAsync: createFactoryIntakeMutateAsync, isPending: false }),
  useUpdateFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false, error: null }),
  useDeleteFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false, error: null }),
  useSearchFactoryIntakeItems: () => searchFactoryIntakeItems(),
  useImportFactoryIntakeItem: () => ({ mutateAsync: importFactoryIntakeItem, isPending: false }),
  useRefreshBacklog: () => ({ mutateAsync: refreshBacklogMutateAsync, isPending: false }),
}));

vi.mock("@/hooks/useWorkOrderCardActions", () => ({
  useWorkOrderCardActions: () => ({
    dispatchingOrderIds: new Set<string>(),
    isAssigneesSaving: false,
    onDispatch: vi.fn(),
    onAssigneesSave: vi.fn(),
  }),
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useFactoryPRFeedbackHandlers: () => useFactoryPRFeedbackHandlers(),
  useCreateFactoryPRFeedbackHandler: () => ({ mutateAsync: createFactoryPRFeedbackHandler, isPending: false }),
  useUpdateFactoryPRFeedbackHandler: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteFactoryPRFeedbackHandler: () => ({ mutateAsync: vi.fn(), isPending: false }),
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

vi.mock("./planningSessionClient", () => ({
  findPlanningSessionByWorkOrder: vi.fn(async () => null),
  sendPlanningSessionMessage: vi.fn(),
  answerPlanningSessionSurvey: vi.fn(),
}));

const enabledExperimentalFeatures = new Set<string>();

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (featureId: string) => enabledExperimentalFeatures.has(featureId),
    enabledExperimentalFeatures: [...enabledExperimentalFeatures],
    isLoading: false,
  }),
}));

const useCanvasMock = vi.hoisted(() => vi.fn());
const updateCanvasVersionMutateAsync = vi.hoisted(() => vi.fn());
const commitCanvasStagingMutateAsync = vi.hoisted(() => vi.fn());
const canvasStagingRefetch = vi.hoisted(() => vi.fn());
const discardCanvasStagingMutateAsync = vi.hoisted(() => vi.fn());

vi.mock("@/hooks/useCanvasData", () => {
  const actual = unmockedSrc<typeof canvasData>("hooks/useCanvasData");
  return {
    ...actual,
    useCanvas: (organizationId: string, canvasId: string, options?: { enabled?: boolean }) =>
      useCanvasMock(organizationId, canvasId, options),
    useCanvasStaging: () => ({
      data: { hasStaging: false, stale: false },
      isPending: false,
      refetch: canvasStagingRefetch,
    }),
    useUpdateCanvasVersion: () => ({ mutateAsync: updateCanvasVersionMutateAsync, isPending: false }),
    useCommitCanvasStaging: () => ({ mutateAsync: commitCanvasStagingMutateAsync, isPending: false }),
    useDiscardCanvasStaging: () => ({ mutateAsync: discardCanvasStagingMutateAsync, isPending: false }),
  };
});

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

async function resetLinesBoardMocks() {
  window.localStorage.clear();
  resetFactoryBoardLaneScrollPositions();
  updateFactoryLineMutateAsync.mockReset();
  updateLineIsPending.value = false;
  useFactoryWorkOrders.mockReturnValue({ data: [] });
  useFactoryBoardWorkOrders.mockImplementation(() => ({
    workOrders: useFactoryWorkOrders().data ?? [],
    isLoading: false,
    isPlaceholderData: false,
    backlog: idleBoardPage(),
    open: idleBoardPage(),
    done: idleBoardPage(),
  }));
  useWorkOrder.mockReturnValue({ data: undefined });
  useFactoryAutomations.mockReturnValue({ data: [] });
  useFactoryIntakes.mockReturnValue({ data: [] });
  createFactoryIntakeMutateAsync.mockReset();
  createFactoryAutomationMutateAsync.mockReset();
  deleteFactoryAutomationMutateAsync.mockReset();
  createFactoryPRFeedbackHandler.mockReset();
  useFactoryPRFeedbackHandlers.mockReturnValue({ data: [], isPending: false });
  searchFactoryIntakeItems.mockReturnValue({ data: [], isLoading: false, isError: false });
  importFactoryIntakeItem.mockReset();
  refreshBacklogMutateAsync.mockReset();
  enabledExperimentalFeatures.clear();
  useCanvasMock.mockImplementation((_organizationId: string, canvasId: string, options?: { enabled?: boolean }) => {
    if (options?.enabled === false) {
      return { data: undefined, isPending: false, isError: false };
    }
    if (canvasId === "app-refund-implementer") {
      return canvasQuery(implementerCanvas);
    }
    return canvasQuery(canvasWithoutAgent);
  });
  updateCanvasVersionMutateAsync.mockReset().mockResolvedValue({});
  commitCanvasStagingMutateAsync.mockReset().mockResolvedValue({});
  canvasStagingRefetch.mockReset().mockResolvedValue({ data: { hasStaging: false, stale: false } });
  discardCanvasStagingMutateAsync.mockReset().mockResolvedValue({});
}

describe("LinesPage board", () => {
  beforeEach(async () => {
    await resetLinesBoardMocks();
  });

  it("does not show a back link to the lines list", () => {
    renderLinesBoard();

    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();
    expect(screen.queryByTestId("lines-back-to-list")).not.toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-drawer")).not.toBeInTheDocument();
  });

  it("holds the empty board while the first task pages load", () => {
    useFactoryBoardWorkOrders.mockReturnValue({
      workOrders: [],
      isLoading: true,
      isPlaceholderData: false,
      backlog: idleBoardPage(),
      open: idleBoardPage(),
      done: idleBoardPage(),
    });
    renderLinesBoard();

    expect(screen.getByRole("status", { name: "Loading the board" })).toBeInTheDocument();
    expect(screen.queryByTestId("lines-phase-board")).not.toBeInTheDocument();
    expect(screen.queryByText("Nothing here.")).not.toBeInTheDocument();
  });

  it("keeps the columns and shows task skeletons while filter pages are placeholders", () => {
    const fetchNextOpen = vi.fn();
    useFactoryWorkOrders.mockReturnValue({ data: [DRAFT_WORK_ORDER] });
    useFactoryBoardWorkOrders.mockReturnValue({
      workOrders: [DRAFT_WORK_ORDER],
      isLoading: false,
      isPlaceholderData: true,
      backlog: idleBoardPage(),
      open: { hasNextPage: true, isFetchingNextPage: false, fetchNextPage: fetchNextOpen },
      done: idleBoardPage(),
    });
    renderLinesBoard();

    expect(screen.getByTestId("lines-phase-board")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Loading the board" })).not.toBeInTheDocument();
    expect(screen.getAllByRole("status", { name: "Loading tasks" }).length).toBeGreaterThan(0);
    expect(screen.queryByText("Nothing here.")).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-card-wo-draft-refunds")).not.toBeInTheDocument();

    fireEvent.scroll(screen.getByTestId("lines-phase-column-scroll-0"));
    expect(fetchNextOpen).not.toHaveBeenCalled();
  });

  it("fades the board cards in after a filter placeholder stretch", () => {
    useFactoryBoardWorkOrders.mockReturnValue({
      workOrders: [DRAFT_WORK_ORDER],
      isLoading: false,
      isPlaceholderData: true,
      backlog: idleBoardPage(),
      open: idleBoardPage(),
      done: idleBoardPage(),
    });
    const view = renderLinesBoard();

    expect(screen.getAllByRole("status", { name: "Loading tasks" }).length).toBeGreaterThan(0);

    useFactoryBoardWorkOrders.mockReturnValue({
      workOrders: [DRAFT_WORK_ORDER],
      isLoading: false,
      isPlaceholderData: false,
      backlog: idleBoardPage(),
      open: idleBoardPage(),
      done: idleBoardPage(),
    });
    view.rerender(
      <LinesBoardSpecHarness path={`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`} />,
    );

    expect(screen.queryByRole("status", { name: "Loading tasks" })).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-column-scroll")).toHaveAttribute("data-reveal");
    expect(screen.getByTestId("work-order-card-wo-draft-refunds")).toBeInTheDocument();
  });

  it("sets a pastel colour on the backlog from circular swatches", async () => {
    const user = userEvent.setup();
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`);

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-color-lime"));

    expect(screen.getByTestId("lines-backlog-column").className).toContain("bg-lime-300");
  });

  it("shows a score on a draft card and hides it on other columns", () => {
    const draft = withBoardChecks(REVIEW_CANDIDATE_WORK_ORDERS)[0];
    useFactoryWorkOrders.mockReturnValue({
      data: [draft, { ...BOARD_IMPLEMENT_FAILED_ORDER, checkScores: draft.checkScores }],
    });
    renderLinesBoard();

    expect(screen.getByTestId("work-order-card-score-wo-review-pay-842")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-card-score-wo-board-implement-failed")).not.toBeInTheDocument();
  });

  it("shows a verdict on a review-candidate backlog card and opens the split run", async () => {
    const [candidate] = REVIEW_CANDIDATE_WORK_ORDERS;
    useFactoryWorkOrders.mockReturnValue({ data: withBoardChecks(REVIEW_CANDIDATE_WORK_ORDERS) });
    useWorkOrder.mockReturnValue({
      data: { ...candidate, checks: DEFAULT_CHECKS_BY_ORDER_ID[candidate.id ?? ""] },
    });
    const user = userEvent.setup();
    renderLinesBoard();

    const card = screen.getByTestId("work-order-card-wo-review-pay-842");
    const cardScore = within(card).getByTestId("work-order-card-score-wo-review-pay-842");
    expect(cardScore).toHaveAttribute("data-tone", "ready");
    expect(cardScore).toHaveTextContent("Clarity5Confidence5");
    expect(cardScore).toHaveAttribute(
      "aria-label",
      "This task is ready to start. Clarity score 5 of 5. Confidence score 5 of 5",
    );
    expect(within(card).queryByRole("button", { name: "Start" })).not.toBeInTheDocument();
    expect(within(card).queryByTestId("work-order-card-start-wo-review-pay-842")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));

    const dialog = screen.getByTestId("work-order-split-run");
    expect(within(dialog).getByTestId("popup-work-order-title")).toHaveTextContent(
      "Add retry handling to webhook delivery",
    );
    expect(within(dialog).queryByRole("tab", { name: "Plan" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("tab", { name: "Ticket" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("tab", { name: "Task" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("tab", { name: "Automations" })).not.toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-intent-document")).toBeInTheDocument();
    expect(within(dialog).getByTestId("popup-work-order-archive-button")).toBeInTheDocument();
    expect(screen.queryByTestId("review-candidate-modal")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY.toLowerCase()}/task/842?lineId=${REFUND_LINE_PLAN_ID}`,
    );
    expect(screen.getByTestId("lines-detail-page")).not.toHaveClass("animate-in");

    await user.click(within(dialog).getByRole("button", { name: "Close" }));
    await waitFor(() => {
      expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
    });
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY.toLowerCase()}/lines/${REFUND_LINE_PLAN_ID}`,
    );
    expect(screen.getByTestId("lines-detail-page")).not.toHaveClass("animate-in");
  });

  it("shows the analyzing state in the popup while a fresh draft awaits its run", async () => {
    const analyzingOrder = { ...REVIEW_CANDIDATE_WORK_ORDERS[0], id: "wo-fresh-analyzing" };
    useFactoryWorkOrders.mockReturnValue({ data: [analyzingOrder] });
    const analyzingOrderId = analyzingOrder.id;
    // The board optimistically knows this draft is analyzing before its Backlog
    // run appears through the live subscription. The popup must match the board card.
    markBacklogAnalysisPending(analyzingOrderId);
    try {
      const user = userEvent.setup();
      renderLinesBoard();

      await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));

      const dialog = screen.getByTestId("work-order-split-run");
      expect(within(dialog).queryByTestId("split-run-intent-decision-tip")).not.toBeInTheDocument();
      expect(within(dialog).getByTestId("split-run-intent-plan-updated")).toBeInTheDocument();
      expect(within(dialog).getByTestId("split-run-intent-verdict-analyzing")).toBeInTheDocument();
      expect(within(dialog).queryByRole("tab", { name: "Task" })).not.toBeInTheDocument();
      expect(within(dialog).queryByRole("tab", { name: "Automations" })).not.toBeInTheDocument();
      expect(within(dialog).queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
      expect(within(dialog).queryByRole("button", { name: "Refine" })).not.toBeInTheDocument();
      expect(within(dialog).getByTestId("popup-work-order-archive-button")).toBeInTheDocument();
      expect(within(dialog).getByRole("button", { name: "Start" })).toBeInTheDocument();
    } finally {
      clearBacklogAnalysisPending(analyzingOrderId);
    }
  });

  it("opens the split run from a task permalink", async () => {
    useFactoryWorkOrders.mockReturnValue({ data: REVIEW_CANDIDATE_WORK_ORDERS });
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/task/842`);

    const popup = await screen.findByTestId("work-order-split-run");
    expect(within(popup).getByTestId("popup-work-order-title")).toHaveTextContent(
      "Add retry handling to webhook delivery",
    );
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/task/842`,
    );
  });

  it("keeps the line board without a popup when the permalink is unknown", () => {
    useFactoryWorkOrders.mockReturnValue({ data: REVIEW_CANDIDATE_WORK_ORDERS });
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/task/999`);

    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
  });

  it("keeps the current line when a card opens from a second line", async () => {
    useFactoryWorkOrders.mockReturnValue({ data: REVIEW_CANDIDATE_WORK_ORDERS });
    const user = userEvent.setup();
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_HOTFIX_ID}`);

    await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY.toLowerCase()}/task/842?lineId=${REFUND_LINE_HOTFIX_ID}`,
    );

    await user.click(within(screen.getByTestId("work-order-split-run")).getByRole("button", { name: "Close" }));
    await waitFor(() => {
      expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
    });
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY.toLowerCase()}/lines/${REFUND_LINE_HOTFIX_ID}`,
    );
  });

  it("does not reopen the popup when Back is pressed after Close", async () => {
    useFactoryWorkOrders.mockReturnValue({ data: REVIEW_CANDIDATE_WORK_ORDERS });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));
    expect(screen.getByTestId("work-order-split-run")).toBeInTheDocument();

    await user.click(within(screen.getByTestId("work-order-split-run")).getByRole("button", { name: "Close" }));
    await waitFor(() => {
      expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
    });

    await user.click(screen.getByTestId("lines-test-back"));
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`,
    );
  });

  it("redirects to the first board when a missing line id is opened", async () => {
    const user = userEvent.setup();
    const missingLinePath = `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/line-missing`;
    render(
      <LinesBoardSpecHarness
        path={`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`}
        factory={REFUND_FACTORY}
        navigateTo={missingLinePath}
      />,
    );

    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();

    await user.click(screen.getByTestId("lines-test-navigate"));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryHomePath("org-1", PRIMARY_FACTORY_KEY, firstFactoryLineId(REFUND_FACTORY)),
    );
    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();
  });

  it("lists the intakes at the head of the Backlog column, without a drawer", () => {
    useFactoryIntakes.mockReturnValue({ data: CONFIGURED_INTAKES });
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    const backlog = screen.getByTestId("lines-backlog-column");
    expect(within(backlog).getByTestId(`line-intake-source-${GITHUB_ISSUES_INTAKE_ID}`)).toBeInTheDocument();
    expect(within(backlog).getByTestId(`line-intake-source-${SENTRY_INTAKE_ID}`)).toBeInTheDocument();
    expect(within(backlog).getByTestId(`line-intake-source-${PAGERDUTY_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-drawer")).not.toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-add")).not.toBeInTheDocument();
    expect(screen.queryByTestId("intake-source-settings")).not.toBeInTheDocument();
  });

  it("shows the automations menu on every column and hides lane banners", () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    useFactoryIntakes.mockReturnValue({ data: CONFIGURED_INTAKES });
    useFactoryPRFeedbackHandlers.mockReturnValue({
      data: [{ id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION", healthy: true }],
      isPending: false,
    });
    renderLinesBoard();

    const backlog = screen.getByTestId("lines-backlog-column");
    expect(within(backlog).queryByTestId(`line-intake-source-${GITHUB_ISSUES_INTAKE_ID}`)).not.toBeInTheDocument();
    expect(screen.queryByTestId("lines-verify-listener-handler-discussion")).not.toBeInTheDocument();
    expect(screen.queryByTestId("lines-verify-add-pr-feedback")).not.toBeInTheDocument();
    expect(screen.getByTestId(`lines-backlog-automation-rows-row-${GITHUB_ISSUES_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.getByTestId(`lines-backlog-automation-rows-row-${SENTRY_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.getByTestId(`lines-backlog-automation-rows-row-${PAGERDUTY_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.getByTestId("lines-phase-0-automation-rows-row-step-0-app-refund-implementer")).toBeInTheDocument();
    expect(screen.getByTestId("lines-verify-automation-rows-row-handler-discussion")).toBeInTheDocument();
    expect(screen.queryByTestId("lines-done-automations")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-board-view-menu")).toBeInTheDocument();
  });

  it("puts backlog automations before the create plus and column menu", () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, ICON_VIEW);

    const backlog = screen.getByTestId("lines-backlog-column");
    const automations = within(backlog).getByTestId("lines-backlog-automations");
    const create = within(backlog).getByTestId("lines-backlog-create");
    const menu = within(backlog).getByTestId("lines-backlog-menu");

    expect(automations.compareDocumentPosition(create) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(create.compareDocumentPosition(menu) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("opens intake settings on the first tab when an automation icon is clicked", async () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    const user = userEvent.setup();
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`);

    await user.click(screen.getByTestId(`lines-backlog-automation-rows-row-${GITHUB_ISSUES_INTAKE_ID}`));

    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(`intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`);
    expect(screen.getByTestId("lines-test-location")).not.toHaveTextContent("settings=automation");
    expect(screen.getByTestId("intake-source-settings")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "General" })).toHaveAttribute("data-state", "active");
  });

  it("opens an existing phase automation in the board view popup", async () => {
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-refund-implementer", name: "Implement" }] });
    const user = userEvent.setup();
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`);

    await user.click(screen.getByTestId("lines-phase-0-automation-rows-row-step-0-app-refund-implementer"));

    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    const location = screen.getByTestId("lines-test-location");
    expect(location).toHaveTextContent(
      factoryColumnAutomationViewPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID, "app-refund-implementer"),
    );
    expect(location).not.toHaveTextContent("/apps/");
    expect(location).not.toHaveTextContent("configure=1");
    expect(screen.getByTestId("column-automation-view")).toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-view-delete")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("column-automation-view")).getByRole("heading", { name: "Implement" }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("column-automation-view-tab-agent")).toHaveAttribute("data-state", "active");
    expect(screen.getByTestId("planning-review-editor")).toBeInTheDocument();

    await user.click(screen.getByTestId("column-automation-view-tab-automation"));
    expect(screen.getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      factoryAppConfigurePath("org-1", PRIMARY_FACTORY_KEY, "app-refund-implementer", {
        from: "lines",
        lineId: REFUND_LINE_PLAN_ID,
      }),
    );
  });

  it("opens Planning settings from the Task analysis row", async () => {
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-refund-backlog", name: "Ingest" }] });
    const user = userEvent.setup();
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`);

    await user.click(screen.getByTestId("lines-backlog-automation-rows-row-analysis-app-refund-backlog"));

    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryPlanningPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID),
    );
    expect(screen.getByTestId("planning-settings")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).not.toHaveTextContent("configure=1");
  });

  it("opens the Planning setup wizard from the Task analysis row until setup is confirmed", async () => {
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-refund-backlog", name: "Ingest" }] });
    const user = userEvent.setup();
    renderLinesBoard(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`,
      vi.fn(),
      PLANNING_OPEN_FACTORY,
    );

    await user.click(screen.getByTestId("lines-backlog-automation-rows-row-analysis-app-refund-backlog"));

    expect(screen.getByTestId("planning-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryPlanningSetupPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID),
    );
    expect(screen.queryByTestId("planning-settings")).not.toBeInTheDocument();
  });

  it("opens the phase automation view from the header icon", async () => {
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-refund-implementer", name: "Implement" }] });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-0-automation-rows-row-step-0-app-refund-implementer"));

    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    expect(screen.getByTestId("column-automation-view")).toBeInTheDocument();
  });

  it("hides Add automation on step column menus", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    const phaseMenus = screen.getAllByTestId(/lines-phase-menu-\d+$/);
    expect(phaseMenus.length).toBeGreaterThan(0);
    for (const menu of phaseMenus) {
      await user.click(menu);
      expect(screen.queryByRole("menuitem", { name: "Add automation" })).not.toBeInTheDocument();
      await user.keyboard("{Escape}");
    }
  });

  it("opens verify and done automations from the header icons", async () => {
    useFactoryPRFeedbackHandlers.mockReturnValue({
      data: [{ id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION", healthy: true }],
      isPending: false,
    });
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-pr-closure", name: "PR Closure" }] });
    const user = userEvent.setup();
    const { unmount } = renderLinesBoard();

    await user.click(screen.getByTestId("lines-verify-automation-rows-row-handler-discussion"));
    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryPRFeedbackPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID, undefined, "handler-discussion"),
    );

    unmount();
    renderLinesBoard();
    await user.click(screen.getByTestId("lines-done-automation-rows-row-closure-app-pr-closure"));
    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryColumnAutomationViewPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID, "app-pr-closure"),
    );
  });

  it("names each Verify listener from its source", async () => {
    useFactoryPRFeedbackHandlers.mockReturnValue({
      data: [
        { id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION", healthy: true },
        { id: "handler-checks", source: "SOURCE_PULL_REQUEST_CHECKS", healthy: true },
      ],
      isPending: false,
    });
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    const verify = screen.getByTestId("lines-verify-column");
    expect(within(verify).getByTestId("lines-verify-listener-handler-discussion")).toHaveTextContent(
      "Listening to pull request comments",
    );
    expect(within(verify).getByTestId("lines-verify-listener-handler-checks")).toHaveTextContent(
      "Monitoring pull request checks",
    );

    await user.click(within(verify).getByTestId("lines-verify-listener-handler-checks"));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY.toLowerCase()}/lines/${REFUND_LINE_PLAN_ID}?prFeedback=1&prFeedbackHandler=handler-checks`,
    );
  });

  it("opens the source picker from the Verify column header", async () => {
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    expect(screen.queryByTestId("lines-verify-listener-pr-feedback")).not.toBeInTheDocument();
    const verify = screen.getByTestId("lines-verify-column");
    const add = within(verify).getByTestId("lines-verify-add-pr-feedback");
    const menu = within(verify).getByTestId("lines-verify-menu");
    expect(add.closest("[data-testid='lines-verify-listeners']")).toBeNull();
    expect(add.compareDocumentPosition(menu) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    await user.click(add);
    expect(screen.getByTestId("add-pr-feedback-picker")).toBeInTheDocument();
    expect(screen.getByTestId("add-pr-feedback-template-discussion")).toHaveTextContent("Pull request discussion");
    expect(screen.getByTestId("add-pr-feedback-template-checks")).toHaveTextContent("Pull request checks");
  });

  it("opens the checks setup page from the Verify picker", async () => {
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    await user.click(screen.getByTestId("lines-verify-add-pr-feedback"));
    await user.click(screen.getByTestId("add-pr-feedback-template-checks"));

    expect(screen.getByTestId("checks-pr-feedback-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryPRFeedbackSetupPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID, "checks"),
    );
    expect(createFactoryPRFeedbackHandler).not.toHaveBeenCalled();
  });

  it("opens the comments setup page instead of creating the handler immediately", async () => {
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    await user.click(screen.getByTestId("lines-verify-add-pr-feedback"));
    await user.click(screen.getByTestId("add-pr-feedback-template-discussion"));

    expect(screen.getByTestId("discussion-pr-feedback-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryPRFeedbackSetupPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID, "comments"),
    );
    expect(createFactoryPRFeedbackHandler).not.toHaveBeenCalled();
  });
});

describe("LinesPage next steps", () => {
  beforeEach(async () => {
    await resetLinesBoardMocks();
  });

  it("shows comments as open after onboarding", () => {
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    expect(screen.getByTestId("workspace-next-steps")).toHaveTextContent(
      "How should pull request comments be handled?",
    );
    expect(screen.getByTestId("workspace-next-steps-progress")).toHaveTextContent("2/4");
    expect(screen.getByTestId("workspace-next-steps")).toHaveTextContent(
      "SuperPlane can implement tasks and open pull requests, but pull request reviews are not handled yet.",
    );
    expect(screen.getByTestId("workspace-next-step-cta-pr-comments-handler")).toHaveTextContent("Configure");
    expect(screen.queryByTestId("workspace-next-step-later")).not.toBeInTheDocument();
  });

  it("shows status checks after the comments handler is configured", () => {
    useFactoryPRFeedbackHandlers.mockReturnValue({
      data: [{ id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION", healthy: true }],
      isPending: false,
    });
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    expect(screen.getByTestId("workspace-next-steps")).toHaveTextContent(
      "How should failing status checks be handled?",
    );
    expect(screen.getByTestId("workspace-next-steps-progress")).toHaveTextContent("3/4");
    expect(screen.getByTestId("workspace-next-step-cta-pr-checks-handler")).toHaveTextContent("Configure");
    expect(screen.getByTestId("workspace-next-step-later")).toHaveTextContent("Later");
  });

  it("moves the status-checks banner into a header badge when the user chooses Later", async () => {
    useFactoryPRFeedbackHandlers.mockReturnValue({
      data: [{ id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION", healthy: true }],
    });
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    await user.click(screen.getByTestId("workspace-next-step-later"));

    expect(screen.queryByTestId("workspace-next-steps")).not.toBeInTheDocument();
    const restore = screen.getByTestId("workspace-next-steps-restore");
    expect(restore).toHaveTextContent("3/4");
    expect(restore).toHaveTextContent("Configure status checks");
    expect(screen.getByTestId("lines-detail-header")).toContainElement(restore);
    const trial = screen.queryByTestId("hosted-credit-header-kicker");
    if (trial) {
      expect(trial.compareDocumentPosition(restore) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    }

    await user.click(restore);

    expect(screen.getByTestId("workspace-next-steps")).toHaveTextContent(
      "How should failing status checks be handled?",
    );
    expect(screen.queryByTestId("workspace-next-steps-restore")).not.toBeInTheDocument();
  });

  it("keeps the deferred status-checks badge after a reload", async () => {
    useFactoryPRFeedbackHandlers.mockReturnValue({
      data: [{ id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION", healthy: true }],
    });
    const user = userEvent.setup();
    const { unmount } = renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);
    await user.click(screen.getByTestId("workspace-next-step-later"));
    expect(screen.getByTestId("workspace-next-steps-restore")).toHaveTextContent("3/4");
    expect(screen.getByTestId("workspace-next-steps-restore")).toHaveTextContent("Configure status checks");
    unmount();

    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    expect(screen.queryByTestId("workspace-next-steps")).not.toBeInTheDocument();
    expect(screen.getByTestId("workspace-next-steps-restore")).toHaveTextContent("3/4");
    expect(screen.getByTestId("workspace-next-steps-restore")).toHaveTextContent("Configure status checks");
  });

  it("keeps next steps hidden while PR feedback handlers load", () => {
    useFactoryPRFeedbackHandlers.mockReturnValue({ data: [], isPending: true });
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    expect(screen.queryByTestId("workspace-next-steps")).not.toBeInTheDocument();
    expect(screen.queryByTestId("workspace-next-steps-restore")).not.toBeInTheDocument();
  });

  it("keeps next steps hidden when PR feedback handlers fail to load", () => {
    useFactoryPRFeedbackHandlers.mockReturnValue({ isPending: false, isError: true });
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    expect(screen.queryByTestId("workspace-next-steps")).not.toBeInTheDocument();
    expect(screen.queryByTestId("workspace-next-steps-restore")).not.toBeInTheDocument();
  });

  it("hides next steps when both handlers are configured", () => {
    useFactoryPRFeedbackHandlers.mockReturnValue({
      data: [
        { id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION", healthy: true },
        { id: "handler-checks", source: "SOURCE_PULL_REQUEST_CHECKS", healthy: true },
      ],
    });
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    expect(screen.queryByTestId("workspace-next-steps")).not.toBeInTheDocument();
  });

  it("opens the comments setup page from the next-step CTA", async () => {
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    await user.click(screen.getByTestId("workspace-next-step-cta-pr-comments-handler"));

    expect(screen.getByTestId("discussion-pr-feedback-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryPRFeedbackSetupPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID, "comments"),
    );
    expect(createFactoryPRFeedbackHandler).not.toHaveBeenCalled();
  });

  it("opens the status-checks setup page from the next-step CTA", async () => {
    useFactoryPRFeedbackHandlers.mockReturnValue({
      data: [{ id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION", healthy: true }],
    });
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    await user.click(screen.getByTestId("workspace-next-step-cta-pr-checks-handler"));

    expect(screen.getByTestId("checks-pr-feedback-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryPRFeedbackSetupPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID, "checks"),
    );
    expect(createFactoryPRFeedbackHandler).not.toHaveBeenCalled();
  });
});

describe("LinesPage board extras", () => {
  beforeEach(async () => {
    await resetLinesBoardMocks();
  });

  it("hides Add automation on Backlog and shows it on Verify and Done", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    useFactoryPRFeedbackHandlers.mockReturnValue({
      data: [{ id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION", healthy: true }],
      isPending: false,
    });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    expect(screen.queryByRole("menuitem", { name: "Add automation" })).not.toBeInTheDocument();
    await user.keyboard("{Escape}");

    await user.click(screen.getByTestId("lines-verify-menu"));
    expect(screen.getByRole("menuitem", { name: "Add automation" })).toBeInTheDocument();
    await user.keyboard("{Escape}");

    await user.click(screen.getByTestId("lines-done-menu"));
    expect(screen.getByRole("menuitem", { name: "Add automation" })).toBeInTheDocument();
  });

  it("hides Add automation when custom automations are off", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-verify-menu"));
    expect(screen.queryByRole("menuitem", { name: "Add automation" })).not.toBeInTheDocument();
    await user.keyboard("{Escape}");

    await user.click(screen.getByTestId("lines-done-menu"));
    expect(screen.queryByRole("menuitem", { name: "Add automation" })).not.toBeInTheDocument();
  });

  it("keeps the Verify plus when custom automations are off", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    expect(screen.getByTestId("lines-verify-add-pr-feedback")).toBeInTheDocument();
    await user.click(screen.getByTestId("lines-verify-add-pr-feedback"));
    expect(screen.getByTestId("add-pr-feedback-picker")).toBeInTheDocument();
  });

  it("opens the name dialog when Verify only has custom automation left", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    useFactoryPRFeedbackHandlers.mockReturnValue({
      data: [
        { id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION", healthy: true },
        { id: "handler-checks", source: "SOURCE_PULL_REQUEST_CHECKS", healthy: true },
      ],
      isPending: false,
    });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-verify-menu"));
    await user.click(screen.getByTestId("lines-verify-menu-add-automation"));

    expect(screen.queryByTestId("add-column-automation-picker")).not.toBeInTheDocument();
    expect(screen.getByTestId("factory-app-name-input")).toBeInTheDocument();
  });

  it("creates a custom Verify automation and opens the editor", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    createFactoryAutomationMutateAsync.mockResolvedValueOnce({ id: "canvas-custom", name: "Destroy ephemeral env" });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-verify-menu"));
    await user.click(screen.getByTestId("lines-verify-menu-add-automation"));
    await user.click(screen.getByTestId("add-column-automation-template-custom"));
    await user.type(screen.getByTestId("factory-app-name-input"), "Destroy ephemeral env");
    await user.click(screen.getByTestId("factory-app-create-button"));

    await waitFor(() => {
      expect(createFactoryAutomationMutateAsync).toHaveBeenCalledWith({
        name: "Destroy ephemeral env",
        columnKey: "verify",
      });
    });
    await waitFor(() => {
      expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
        `/org-1/workspaces/${PRIMARY_FACTORY_KEY.toLowerCase()}/automations/canvas-custom`,
      );
    });
  });

  it("deletes a custom Verify automation from the view modal", async () => {
    useFactoryAutomations.mockReturnValue({
      data: [{ id: "app-create-env", name: "Create env", columnKey: "verify" }],
    });
    deleteFactoryAutomationMutateAsync.mockResolvedValueOnce(undefined);
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-verify-automation-rows-row-custom-app-create-env"));
    expect(screen.getByTestId("column-automation-view")).toBeInTheDocument();
    await user.click(screen.getByTestId("column-automation-view-delete"));
    await user.click(screen.getByTestId("column-automation-view-delete-confirm"));

    await waitFor(() => {
      expect(deleteFactoryAutomationMutateAsync).toHaveBeenCalledWith("app-create-env");
    });
  });

  it("lists two intakes on the same source", () => {
    useFactoryIntakes.mockReturnValue({
      data: [
        GITHUB_ISSUES_INTAKE,
        { id: "intake-triage", canvasId: "app-triage", name: "Triage issues", source: "SOURCE_GITHUB_ISSUES" },
      ],
    });
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    expect(screen.getByTestId(`line-intake-source-${GITHUB_ISSUES_INTAKE_ID}`)).toHaveTextContent(
      "Listening to GitHub issues",
    );
    expect(screen.getByTestId("line-intake-source-intake-triage")).toHaveTextContent("Listening to Triage issues");
  });

  it("creates an intake from the picker and opens its canvas", async () => {
    createFactoryIntakeMutateAsync.mockResolvedValueOnce({ id: "intake-new", canvasId: "canvas-new" });
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, { addIntakeControl: true, columnAutomations: false });

    await user.click(screen.getByTestId("line-intake-add"));
    await user.click(screen.getByTestId("add-intake-template-github-issues"));

    await waitFor(() => {
      expect(createFactoryIntakeMutateAsync).toHaveBeenCalledWith({ source: "SOURCE_GITHUB_ISSUES" });
    });
    await waitFor(() => {
      expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
        `/org-1/workspaces/${PRIMARY_FACTORY_KEY.toLowerCase()}/automations/canvas-new`,
      );
    });
  });

  it("offers Add intake from the overflow menu", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    expect(screen.getByTestId("lines-backlog-menu-add-intake")).toBeInTheDocument();
  });

  it("offers Refresh backlog when a readable intake exists", async () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    refreshBacklogMutateAsync.mockResolvedValueOnce({
      archivedCount: 1,
      failedItemCount: 0,
      failedSourceCount: 0,
    });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-refresh-backlog"));

    await waitFor(() => {
      expect(refreshBacklogMutateAsync).toHaveBeenCalledTimes(1);
    });
  });

  it("hides Refresh backlog when no readable intake exists", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    expect(screen.queryByTestId("lines-backlog-menu-refresh-backlog")).not.toBeInTheDocument();
  });

  it("marks flagged intake sources as coming soon when the organization feature is off", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-add-intake"));

    expect(screen.getByTestId("add-intake-template-github-issues")).toBeEnabled();
    expect(screen.getByTestId("add-intake-template-jira-issues")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("add-intake-template-sentry-exceptions")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("add-intake-template-productive-tasks")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("add-intake-template-datadog")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("add-intake-template-notion")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);

    await user.click(screen.getByTestId("add-intake-template-sentry-exceptions"));
    await user.click(screen.getByTestId("add-intake-template-jira-issues"));
    await user.click(screen.getByTestId("add-intake-template-productive-tasks"));

    expect(screen.queryByTestId("sentry-intake-setup")).not.toBeInTheDocument();
    expect(screen.queryByTestId("jira-intake-setup")).not.toBeInTheDocument();
    expect(screen.queryByTestId("productive-intake-setup")).not.toBeInTheDocument();
  });

  it("opens guided Sentry setup from the overflow menu", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_SENTRY_INTAKE);
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-add-intake"));

    expect(screen.getByTestId("add-intake-template-github-issues")).toBeInTheDocument();
    expect(screen.getByTestId("add-intake-template-sentry-exceptions")).toBeInTheDocument();
    expect(screen.getByTestId("add-intake-template-jira-issues")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("add-intake-template-datadog")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("add-intake-template-notion")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.queryByTestId("add-intake-template-pagerduty-incidents")).not.toBeInTheDocument();
    expect(screen.getByTestId("add-intake-template-productive-tasks")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);

    await user.click(screen.getByTestId("add-intake-template-sentry-exceptions"));

    expect(screen.getByTestId("sentry-intake-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factorySentryIntakeSetupPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID),
    );
    expect(createFactoryIntakeMutateAsync).not.toHaveBeenCalled();
  });

  it("opens guided Dependabot setup from the overflow menu", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_DEPENDABOT_INTAKE);
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-add-intake"));
    await user.click(screen.getByTestId("add-intake-template-dependabot-alerts"));

    expect(screen.getByTestId("dependabot-intake-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryDependabotIntakeSetupPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID),
    );
    expect(createFactoryIntakeMutateAsync).not.toHaveBeenCalled();
  });

  it("marks a configured GitHub intake as already set up", async () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-add-intake"));

    const github = screen.getByTestId("add-intake-template-github-issues");
    expect(github).toBeDisabled();
    expect(github).toHaveTextContent(ADD_INTAKE_COPY.sourceTaken);
  });

  it("hides the backlog Sentry setup banner", () => {
    renderLinesBoard();

    expect(screen.queryByTestId("lines-backlog-setup-sentry")).not.toBeInTheDocument();
    expect(screen.queryByTestId("lines-backlog-setup-jira")).not.toBeInTheDocument();
  });

  it("opens guided Jira setup from the overflow menu", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_JIRA_INTAKE);
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-add-intake"));

    expect(screen.getByTestId("add-intake-template-github-issues")).toBeInTheDocument();
    expect(screen.getByTestId("add-intake-template-jira-issues")).toBeInTheDocument();
    expect(screen.getByTestId("add-intake-template-sentry-exceptions")).toBeInTheDocument();
    expect(screen.getByTestId("add-intake-template-productive-tasks")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);

    await user.click(screen.getByTestId("add-intake-template-jira-issues"));
    expect(screen.getByTestId("jira-intake-setup")).toBeInTheDocument();
    expect(createFactoryIntakeMutateAsync).not.toHaveBeenCalled();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryJiraIntakeSetupPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID),
    );
  });

  it("opens guided Productive.io setup from the overflow menu", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_PRODUCTIVE_INTAKE);
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-add-intake"));

    const productive = screen.getByTestId("add-intake-template-productive-tasks");
    expect(productive).toBeEnabled();
    expect(screen.getByTestId("add-intake-template-jira-issues")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);

    await user.click(productive);

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryProductiveIntakeSetupPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID),
    );
    expect(createFactoryIntakeMutateAsync).not.toHaveBeenCalled();
  });

  it("sends a legacy Jira OAuth return to the setup page", () => {
    renderLinesBoard(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?jiraIntake=1&jiraIntegrationId=int-new`,
      vi.fn(),
      REFUND_FACTORY,
      LANE_BANNERS,
    );

    expect(screen.getByTestId("jira-intake-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryJiraIntakeSetupPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID, { integrationId: "int-new" }),
    );
  });

  it("shows only declared intakes", () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    renderLinesBoard(
      `/org-1/workspaces/${ACME_ONBOARDING_FACTORY_KEY}/lines/${ACME_ONBOARDING_LINE_ID}`,
      vi.fn(),
      ACME_ONBOARDING_FACTORY,
      LANE_BANNERS,
    );

    expect(screen.getByTestId(`line-intake-source-${GITHUB_ISSUES_INTAKE_ID}`)).toHaveTextContent(
      "Listening to GitHub issues",
    );
    expect(screen.queryByText("Listening to Sentry exceptions")).not.toBeInTheDocument();
    expect(screen.queryByTestId(`line-intake-source-${PAGERDUTY_INTAKE_ID}`)).not.toBeInTheDocument();
  });

  it("opens intake settings from the row and links to the factory canvas editor", async () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    await user.click(screen.getByRole("button", { name: `Open ${GITHUB_ISSUES_INTAKE.name} settings` }));
    await user.click(screen.getByRole("tab", { name: "Automation" }));

    expect(screen.getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      factoryAppConfigurePath("org-1", PRIMARY_FACTORY_KEY, GITHUB_ISSUES_INTAKE_APP.id!, {
        from: "lines",
        lineId: REFUND_LINE_PLAN_ID,
      }),
    );
  });

  it("opens the settings of the intake whose row was used", async () => {
    useFactoryIntakes.mockReturnValue({
      data: [
        GITHUB_ISSUES_INTAKE,
        { id: "intake-triage", canvasId: "app-triage", name: "Triage issues", source: "SOURCE_GITHUB_ISSUES" },
      ],
    });
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    await user.click(screen.getByRole("button", { name: "Open Triage issues settings" }));

    expect(
      within(screen.getByTestId("intake-source-settings")).getByRole("heading", { name: "Intake Triage issues" }),
    ).toBeInTheDocument();
  });

  it("does not list intake runs under the GitHub issues row", () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    renderLinesBoard(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`,
      vi.fn(),
      REFUND_FACTORY,
      LANE_BANNERS,
    );

    expect(screen.getByTestId(`line-intake-source-${GITHUB_ISSUES_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.queryByText("No intake runs in progress.")).not.toBeInTheDocument();
    expect(screen.queryByText("Handle duplicate refunds on retry")).not.toBeInTheDocument();
  });
});

describe("LinesPage board pull request", () => {
  beforeEach(async () => {
    await resetLinesBoardMocks();
  });

  it("shows an attached pull request on the task card", () => {
    useFactoryWorkOrders.mockReturnValue({
      data: [
        {
          ...BOARD_IMPLEMENT_FAILED_ORDER,
          pullRequests: [
            {
              id: "pr-106",
              workOrderId: BOARD_IMPLEMENT_FAILED_ORDER.id,
              number: "106",
              url: "https://github.com/acme/payments/pull/106",
              title: "Fix refund dispatcher timeout loop",
              state: "STATE_CLOSED",
            },
          ],
        },
      ],
    });
    renderLinesBoard();

    const card = screen.getByTestId("work-order-card-wo-board-implement-failed");
    const pill = within(card).getByRole("link", { name: "Closed pull request #106." });
    expect(pill).toHaveTextContent("Closed #106");
    expect(pill).toHaveAttribute("href", "https://github.com/acme/payments/pull/106");
  });
});

describe("LinesPage board editing", () => {
  beforeEach(async () => {
    await resetLinesBoardMocks();
  });

  it("renames the board title on Enter", async () => {
    updateFactoryLineMutateAsync.mockResolvedValueOnce({});
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-board-title"));
    const input = await screen.findByTestId("lines-board-title-input");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "Refund line");
    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(updateFactoryLineMutateAsync).toHaveBeenCalledWith({
        lineId: REFUND_LINE_PLAN_ID,
        name: "Refund line",
      });
    });
  });

  it("renames a column title on Enter", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-column-title-backlog"));
    const input = await screen.findByTestId("lines-column-title-backlog-input");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "Inbox");
    await user.keyboard("{Enter}");

    expect(screen.getByTestId("lines-column-title-backlog")).toHaveTextContent("Inbox");
  });

  it("opens backlog settings in a modal and does not open a canvas", async () => {
    const user = userEvent.setup();
    const linePath = `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`;
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-edit"));

    expect(screen.getByTestId("lines-backlog-settings")).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toHaveValue("Backlog");
    expect(screen.getByLabelText("Size")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(linePath);
    expect(screen.getByTestId("lines-test-location")).not.toHaveTextContent("configure=1");
  });

  it("saves the backlog name from the settings modal", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-edit"));
    const name = screen.getByLabelText("Name");
    await user.clear(name);
    await user.type(name, "Inbox");
    await user.click(screen.getByTestId("lines-backlog-settings-save"));

    expect(screen.queryByTestId("lines-backlog-settings")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-column-title-backlog")).toHaveTextContent("Inbox");
  });

  it("opens the automation view from the header icon without an Edit menu", async () => {
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-refund-implementer", name: "Implement" }] });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-0-automation-rows-row-step-0-app-refund-implementer"));
    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    expect(screen.getByTestId("column-automation-view")).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Edit Agent" })).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Edit Automation" })).not.toBeInTheDocument();
  });

  it("stages and commits the canvas agent on Save Agent", async () => {
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-refund-implementer", name: "Implement" }] });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-0-automation-rows-row-step-0-app-refund-implementer"));
    await user.click(screen.getByTestId("planning-review-step-toggle-0"));
    const stepName = screen.getByTestId("planning-review-step-name-0");
    await user.clear(stepName);
    await user.type(stepName, "Clone repository");
    await user.click(screen.getByTestId("planning-review-save"));

    await waitFor(() => {
      expect(updateCanvasVersionMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          versionId: "version-live",
          canvasYaml: expect.stringContaining("Clone repository"),
        }),
      );
    });
    expect(commitCanvasStagingMutateAsync).toHaveBeenCalledWith("Update agent");
    expect(screen.getByTestId("column-automation-view")).toBeInTheDocument();
  });

  it("keeps Backlog Edit as the only edit action", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    expect(screen.getByTestId("lines-backlog-menu-edit")).toHaveTextContent("Edit");
    expect(screen.queryByTestId("lines-backlog-menu-edit-agent")).not.toBeInTheDocument();
    expect(screen.queryByTestId("lines-backlog-menu-parallelism")).not.toBeInTheDocument();
    expect(screen.queryByTestId("lines-backlog-menu-edit-automation")).not.toBeInTheDocument();
  });

  it("opens Set parallelism and saves a new cap", async () => {
    updateFactoryLineMutateAsync.mockResolvedValueOnce({});
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    expect(screen.getByTestId("lines-phase-menu-0-parallelism")).toHaveTextContent("Set parallelism (10)");
    await user.click(screen.getByTestId("lines-phase-menu-0-parallelism"));

    const input = screen.getByTestId("lines-parallelism-input");
    await user.clear(input);
    await user.type(input, "20");
    await user.click(screen.getByTestId("lines-parallelism-save"));

    await waitFor(() => {
      expect(updateFactoryLineMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          lineId: REFUND_LINE_PLAN_ID,
          steps: expect.arrayContaining([expect.objectContaining({ maxParallelism: 20 })]),
        }),
      );
    });
  });

  it("hydrates a persisted column color on mount", async () => {
    renderLinesBoard();

    const backlogLane = screen.getByTestId("lines-backlog-column");
    for (const className of lineBoardColumnLaneProps("lime").surfaceClassName!.split(" ")) {
      expect(backlogLane).toHaveClass(className);
    }
  });

  it("picking a color saves it on the line and updates the lane immediately", async () => {
    updateFactoryLineMutateAsync.mockResolvedValueOnce({
      id: REFUND_LINE_PLAN_ID,
      columnColors: { backlog: "lime", "phase-0": "sky" },
    });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    await user.click(screen.getByTestId("lines-phase-menu-0-color-sky"));

    await waitFor(() => {
      expect(updateFactoryLineMutateAsync).toHaveBeenCalledWith({
        lineId: REFUND_LINE_PLAN_ID,
        columnColors: { backlog: "lime", "phase-0": "sky" },
      });
    });

    const phaseLane = screen.getByTestId("lines-phase-column-0");
    for (const className of lineBoardColumnLaneProps("sky").surfaceClassName!.split(" ")) {
      expect(phaseLane).toHaveClass(className);
    }
  });

  it("keeps the picked color when the save completes before the factory refetch", async () => {
    const view = renderLinesBoard();
    updateFactoryLineMutateAsync.mockImplementation(async (input) => {
      updateLineIsPending.value = true;
      view.rerender(
        <LinesBoardSpecHarness
          path={`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`}
          factory={REFUND_FACTORY}
        />,
      );
      await Promise.resolve();
      updateLineIsPending.value = false;
      view.rerender(
        <LinesBoardSpecHarness
          path={`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`}
          factory={REFUND_FACTORY}
        />,
      );
      return {
        id: REFUND_LINE_PLAN_ID,
        columnColors: input.columnColors,
      };
    });
    const user = userEvent.setup();

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    await user.click(screen.getByTestId("lines-phase-menu-0-color-sky"));

    await waitFor(() => {
      expect(updateFactoryLineMutateAsync).toHaveBeenCalled();
    });

    const phaseLane = screen.getByTestId("lines-phase-column-0");
    for (const className of lineBoardColumnLaneProps("sky").surfaceClassName!.split(" ")) {
      expect(phaseLane).toHaveClass(className);
    }
  });

  it("rolls back the color and shows an error toast when saving fails", async () => {
    const { showErrorToast } = await import("@/lib/toast");
    updateFactoryLineMutateAsync.mockRejectedValueOnce(new Error("network error"));
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    await user.click(screen.getByTestId("lines-phase-menu-0-color-sky"));

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalled();
    });

    const phaseLane = screen.getByTestId("lines-phase-column-0");
    for (const className of lineBoardColumnLaneProps("sky").surfaceClassName!.split(" ")) {
      expect(phaseLane).not.toHaveClass(className);
    }
  });

  it("hides Edit on the Done column", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-done-menu"));
    expect(screen.queryByTestId("lines-done-menu-edit")).not.toBeInTheDocument();
    expect(screen.queryByTestId("lines-done-create")).not.toBeInTheDocument();
  });

  it("shows one Backlog card when My is on and the same draft is in every column list", async () => {
    const user = userEvent.setup();
    useFactoryWorkOrders.mockReturnValue({
      data: [DRAFT_WORK_ORDER, DRAFT_WORK_ORDER, DRAFT_WORK_ORDER],
    });
    renderLinesBoard();

    await user.click(screen.getByTestId("work-orders-scope-my"));

    const cardId = `work-order-card-${DRAFT_WORK_ORDER.id}`;
    expect(within(screen.getByTestId("lines-backlog-column")).getAllByTestId(cardId)).toHaveLength(1);
    expect(screen.getAllByTestId(cardId)).toHaveLength(1);
  });

  it("hides the phase path and shows work-order filters", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    const header = screen.getByTestId("lines-detail-header");
    expect(within(header).queryByText(/→/)).not.toBeInTheDocument();
    const actions = within(header).getByTestId("workspace-page-header-actions");
    const scopeAll = within(actions).getByTestId("work-orders-scope-all");
    const filter = within(actions).getByTestId("work-orders-filter-trigger");
    const search = within(actions).getByTestId("work-orders-search-trigger");
    expect(within(actions).queryByTestId("work-orders-scope-active")).not.toBeInTheDocument();
    expect(scopeAll).toHaveTextContent("All");
    expect(within(actions).getByTestId("work-orders-scope-my")).toHaveTextContent("My");
    expect(scopeAll.className).toMatch(/rounded-full/);
    expect(filter).toHaveAccessibleName("Filter");
    expect(filter).not.toHaveTextContent("Filter");
    expect(within(filter).queryByText("F")).not.toBeInTheDocument();
    expect(scopeAll.compareDocumentPosition(filter) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(filter.compareDocumentPosition(search) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(header).queryByTestId("lines-edit-menu")).not.toBeInTheDocument();
    expect(within(header).queryByRole("button", { name: "Plan and Implement menu" })).not.toBeInTheDocument();
    expect(within(header).getByTestId("lines-board-view-menu")).toBeInTheDocument();
    expect(
      filter.compareDocumentPosition(within(header).getByTestId("lines-board-view-menu")) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(within(header).queryByTestId("work-order-list-create-button")).not.toBeInTheDocument();

    await user.click(within(header).getByTestId("work-orders-filter-trigger"));
    expect(screen.getByTestId("work-orders-filter-statuses")).toBeInTheDocument();
    expect(screen.getByTestId("work-orders-filter-labels")).toHaveTextContent("Label");
    expect(screen.queryByTestId("work-orders-filter-lineIds")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-orders-filter-sourceIds")).toBeInTheDocument();
    expect(screen.getByTestId("work-orders-filter-assigneeIds")).toBeInTheDocument();
  });

  it("treats a leftover Active scope as All and keeps every card", () => {
    window.localStorage.setItem(`sp:work-orders:scope:${PRIMARY_FACTORY_ID}`, "active");
    useFactoryWorkOrders.mockReturnValue({
      data: [DRAFT_WORK_ORDER, BOARD_IMPLEMENT_NOTIFY_ORDER],
    });
    renderLinesBoard();

    const actions = within(screen.getByTestId("lines-detail-header")).getByTestId("workspace-page-header-actions");
    expect(within(actions).queryByTestId("work-orders-scope-active")).not.toBeInTheDocument();
    expect(within(actions).getByTestId("work-orders-scope-all")).toHaveAttribute("aria-pressed", "true");
    expect(within(actions).getByTestId("work-orders-scope-my")).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByText("Draft: rework refund telemetry")).toBeInTheDocument();
    expect(screen.getByText("Notify on status change after a reopen")).toBeInTheDocument();
    expect(window.localStorage.getItem(`sp:work-orders:scope:${PRIMARY_FACTORY_ID}`)).toBe("active");
  });

  it("lists Source in the filter menu from configured intakes", async () => {
    const user = userEvent.setup();
    useFactoryIntakes.mockReturnValue({ data: CONFIGURED_INTAKES });
    renderLinesBoard();

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    expect(screen.getByTestId("work-orders-filter-sourceIds")).toHaveTextContent("Source");
    expect(screen.queryByTestId("work-orders-filter-lineIds")).not.toBeInTheDocument();
  });

  it("narrows the board when a Source filter is selected", async () => {
    const user = userEvent.setup();
    useFactoryIntakes.mockReturnValue({ data: CONFIGURED_INTAKES });
    useFactoryWorkOrders.mockReturnValue({
      data: [BOARD_IMPLEMENT_FAILED_ORDER, BOARD_IMPLEMENT_NOTIFY_ORDER],
    });
    renderLinesBoard();

    expect(screen.getByText("Fix refund dispatcher timeout loop")).toBeInTheDocument();
    expect(screen.getByText("Notify on status change after a reopen")).toBeInTheDocument();

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    await user.hover(screen.getByTestId("work-orders-filter-sourceIds"));
    fireEvent.click(await screen.findByTestId("work-orders-filter-sourceIds-github-issues"));

    const board = screen.getByTestId("lines-detail-page");
    expect(within(board).getByText("Source is GitHub issues")).toBeInTheDocument();
    expect(within(board).getByText("Fix refund dispatcher timeout loop")).toBeInTheDocument();
    expect(within(board).queryByText("Notify on status change after a reopen")).not.toBeInTheDocument();
  });

  it("narrows the board when a Review label filter is selected", async () => {
    const user = userEvent.setup();
    useFactoryWorkOrders.mockReturnValue({
      data: [
        {
          ...BOARD_IMPLEMENT_FAILED_ORDER,
          pullRequests: [
            {
              id: "pr-review",
              workOrderId: BOARD_IMPLEMENT_FAILED_ORDER.id,
              number: "106",
              url: "https://github.com/acme/payments/pull/106",
              title: "Fix refund dispatcher timeout loop",
              state: "STATE_OPEN",
            },
          ],
        },
        BOARD_IMPLEMENT_NOTIFY_ORDER,
      ],
    });
    renderLinesBoard();

    expect(screen.getByText("Fix refund dispatcher timeout loop")).toBeInTheDocument();
    expect(screen.getByText("Notify on status change after a reopen")).toBeInTheDocument();

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    await user.hover(screen.getByTestId("work-orders-filter-labels"));
    fireEvent.click(await screen.findByTestId("work-orders-filter-labels-review"));

    const board = screen.getByTestId("lines-detail-page");
    expect(within(board).getByText("Label is Review")).toBeInTheDocument();
    expect(within(board).getByText("Fix refund dispatcher timeout loop")).toBeInTheDocument();
    expect(within(board).queryByText("Notify on status change after a reopen")).not.toBeInTheDocument();
  });

  it("narrows the board when the search query changes", async () => {
    const user = userEvent.setup();
    useFactoryWorkOrders.mockReturnValue({
      data: [BOARD_IMPLEMENT_FAILED_ORDER, BOARD_DONE_REJECTED_ORDER],
    });
    renderLinesBoard();

    expect(screen.getByText("Fix refund dispatcher timeout loop")).toBeInTheDocument();
    expect(screen.getByText("Replace the refund batch exporter")).toBeInTheDocument();

    await user.click(screen.getByTestId("work-orders-search-trigger"));
    await user.type(screen.getByTestId("work-orders-search-input"), "timeout");

    expect(screen.getByText("Fix refund dispatcher timeout loop")).toBeInTheDocument();
    expect(screen.queryByText("Replace the refund batch exporter")).not.toBeInTheDocument();
  });

  it("does not show a line overflow Edit menu", () => {
    renderLinesBoard();

    expect(screen.queryByTestId("lines-edit-menu")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Plan and Implement menu" })).not.toBeInTheDocument();
  });

  it("shows automation names and the header view menu by default", () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    renderLinesBoard();

    expect(screen.getByTestId("lines-backlog-automation-rows")).toHaveTextContent("Listens to GitHub issues");
    expect(screen.queryByTestId("lines-backlog-automations")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-board-view-menu")).toBeInTheDocument();
  });

  it("lets the header view menu hide column colors and persist the choice", async () => {
    const user = userEvent.setup();
    const first = renderLinesBoard();

    expect(screen.getByTestId("lines-backlog-column").className).toContain("bg-lime-300");

    await user.click(screen.getByTestId("lines-board-view-menu"));
    await user.hover(screen.getByTestId("lines-board-view-column-color"));
    fireEvent.click(await screen.findByTestId("lines-board-view-no-column-colors"));

    expect(screen.getByTestId("lines-backlog-column").className).not.toContain("bg-lime-300");

    first.unmount();
    renderLinesBoard();

    expect(screen.getByTestId("lines-backlog-column").className).not.toContain("bg-lime-300");
  });

  it("lets the header view menu show column colors as borders", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-board-view-menu"));
    await user.hover(screen.getByTestId("lines-board-view-column-color"));
    fireEvent.click(await screen.findByTestId("lines-board-view-colored-borders"));

    const backlog = screen.getByTestId("lines-backlog-column");
    expect(backlog.className).not.toContain("bg-lime-300");
    expect(backlog.className).toContain("border-lime-400");
  });

  it("lets the header view menu show soft column colors", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    const backlog = screen.getByTestId("lines-backlog-column");
    expect(backlog.className).toContain("bg-lime-300");
    expect(backlog.className).toContain("dark:bg-lime-800");

    await user.click(screen.getByTestId("lines-board-view-menu"));
    await user.hover(screen.getByTestId("lines-board-view-column-color"));
    fireEvent.click(await screen.findByTestId("lines-board-view-dim-column-colors"));

    const softBacklog = screen.getByTestId("lines-backlog-column");
    expect(softBacklog.className).toContain("bg-lime-100");
    expect(softBacklog.className).toContain("dark:bg-lime-950/40");
    expect(softBacklog.className).not.toContain("bg-lime-300");
    expect(softBacklog.className).not.toContain("dark:bg-lime-800");
  });

  it("lets the header view menu switch automation names to icons and persist the choice", async () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    const user = userEvent.setup();
    const first = renderLinesBoard();

    expect(screen.getByTestId("lines-backlog-automation-rows")).toHaveTextContent("Listens to GitHub issues");
    expect(screen.queryByTestId("lines-backlog-automations")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("lines-board-view-menu"));
    await user.hover(screen.getByTestId("lines-board-view-automations"));
    fireEvent.click(await screen.findByTestId("lines-board-view-icons"));

    expect(screen.queryByTestId("lines-backlog-automation-rows")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-automations")).toBeInTheDocument();

    first.unmount();
    renderLinesBoard();

    expect(screen.queryByTestId("lines-backlog-automation-rows")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-automations")).toBeInTheDocument();
  });
});

function stubElementHeights({ scrollHeight, clientHeight }: { scrollHeight: number; clientHeight: number }) {
  Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, value: scrollHeight });
  Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, value: clientHeight });
  return () => {
    delete (HTMLElement.prototype as unknown as { scrollHeight?: number }).scrollHeight;
    delete (HTMLElement.prototype as unknown as { clientHeight?: number }).clientHeight;
  };
}

function kickoffDraft(index: number): FactoriesWorkOrder {
  const createdAt = new Date(Date.now() - (10 - index) * 1000).toISOString();
  return {
    id: `wo-kickoff-${index}`,
    number: `${200 + index}`,
    title: `Kickoff task ${index}`,
    state: "STATE_DRAFT",
    createdAt,
    updatedAt: createdAt,
  };
}

function dispatchDraftToImplement(order: FactoriesWorkOrder, at: string): FactoriesWorkOrder {
  return {
    ...order,
    state: "STATE_OPEN",
    updatedAt: at,
    lineDispatches: [
      planLineActiveDispatch(order.id!, [
        {
          id: `exec-${order.id}`,
          step: "Implement",
          stepIndex: 0,
          state: "STATE_STARTED",
          createdAt: at,
          updatedAt: at,
          run: { id: `run-${order.id}`, appId: "app-refund-implementer", appName: "Implement" },
        },
      ]),
    ],
  };
}

function implementPhaseCards() {
  return within(screen.getByTestId("lines-phase-column-0")).queryAllByTestId(/^lines-phase-run-/);
}

describe("LinesPage Implement phase window", () => {
  let restoreHeights: (() => void) | undefined;

  beforeEach(async () => {
    await resetLinesBoardMocks();
  });

  afterEach(() => {
    restoreHeights?.();
    restoreHeights = undefined;
  });

  it("keeps every Implement card after five consecutive dispatches", async () => {
    restoreHeights = stubElementHeights({ scrollHeight: 80, clientHeight: 400 });
    const drafts = [0, 1, 2, 3, 4].map(kickoffDraft);
    const orders = [...drafts];
    useFactoryWorkOrders.mockReturnValue({ data: orders });
    const view = renderLinesBoard();

    expect(implementPhaseCards()).toHaveLength(0);

    for (let index = 0; index < drafts.length; index++) {
      const at = new Date(Date.now() + index * 1000).toISOString();
      orders[index] = dispatchDraftToImplement(drafts[index], at);
      useFactoryWorkOrders.mockReturnValue({ data: [...orders] });
      view.rerender(
        <LinesBoardSpecHarness path={`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`} />,
      );

      await waitFor(() => {
        expect(implementPhaseCards()).toHaveLength(index + 1);
      });
    }

    expect(implementPhaseCards()).toHaveLength(5);
    expect(screen.getByTestId("lines-phase-run-exec-wo-kickoff-4")).toBeInTheDocument();
  });

  it("starts an overflowing Implement column at the first page and loads more on scroll", async () => {
    restoreHeights = stubElementHeights({ scrollHeight: 2000, clientHeight: 240 });
    const orders = Array.from({ length: 8 }, (_, index) =>
      dispatchDraftToImplement(kickoffDraft(index), new Date(Date.now() + index * 1000).toISOString()),
    );
    const fetchNextOpen = vi.fn();
    useFactoryWorkOrders.mockReturnValue({ data: orders });
    useFactoryBoardWorkOrders.mockReturnValue({
      workOrders: orders,
      isLoading: false,
      isPlaceholderData: false,
      backlog: idleBoardPage(),
      open: { hasNextPage: true, isFetchingNextPage: false, fetchNextPage: fetchNextOpen },
      done: idleBoardPage(),
    });
    renderLinesBoard();

    expect(implementPhaseCards()).toHaveLength(8);

    const scroller = screen.getByTestId("lines-phase-column-scroll-0");
    scroller.scrollTop = 1760;
    fireEvent.scroll(scroller);

    await waitFor(() => {
      expect(fetchNextOpen).toHaveBeenCalled();
    });
  });

  it("keeps Implement column scroll after a card opens and closes", async () => {
    restoreHeights = stubElementHeights({ scrollHeight: 2000, clientHeight: 240 });
    const orders = Array.from({ length: 8 }, (_, index) =>
      dispatchDraftToImplement(kickoffDraft(index), new Date(Date.now() + index * 1000).toISOString()),
    );
    useFactoryWorkOrders.mockReturnValue({ data: orders });
    useWorkOrder.mockReturnValue({ data: orders[0] });
    const user = userEvent.setup();
    renderLinesBoard();

    const scroller = screen.getByTestId("lines-phase-column-scroll-0");
    scroller.scrollTop = 1760;
    fireEvent.scroll(scroller);
    expect(scroller.scrollTop).toBe(1760);
    expect(screen.getByTestId("lines-backlog-column-scroll").scrollTop).toBe(0);

    await user.click(screen.getByRole("button", { name: "Open Kickoff task 0" }));

    expect(screen.getByTestId("work-order-split-run")).toBeInTheDocument();
    expect(screen.getByTestId("lines-phase-column-scroll-0").scrollTop).toBe(1760);
    expect(screen.getByTestId("lines-backlog-column-scroll").scrollTop).toBe(0);

    await user.click(within(screen.getByTestId("work-order-split-run")).getByRole("button", { name: "Close" }));
    await waitFor(() => {
      expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
    });

    expect(screen.getByTestId("lines-phase-column-scroll-0").scrollTop).toBe(1760);
    expect(screen.getByTestId("lines-backlog-column-scroll").scrollTop).toBe(0);
  });

  it("keeps Implement column scroll after a filter placeholder stretch", () => {
    restoreHeights = stubElementHeights({ scrollHeight: 2000, clientHeight: 240 });
    const orders = Array.from({ length: 8 }, (_, index) =>
      dispatchDraftToImplement(kickoffDraft(index), new Date(Date.now() + index * 1000).toISOString()),
    );
    useFactoryWorkOrders.mockReturnValue({ data: orders });
    useFactoryBoardWorkOrders.mockReturnValue({
      workOrders: orders,
      isLoading: false,
      isPlaceholderData: false,
      backlog: idleBoardPage(),
      open: idleBoardPage(),
      done: idleBoardPage(),
    });
    const view = renderLinesBoard();

    const scroller = screen.getByTestId("lines-phase-column-scroll-0");
    scroller.scrollTop = 1760;
    fireEvent.scroll(scroller);
    expect(scroller.scrollTop).toBe(1760);

    useFactoryBoardWorkOrders.mockReturnValue({
      workOrders: orders,
      isLoading: false,
      isPlaceholderData: true,
      backlog: idleBoardPage(),
      open: idleBoardPage(),
      done: idleBoardPage(),
    });
    view.rerender(
      <LinesBoardSpecHarness path={`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`} />,
    );

    const pending = screen.getByTestId("lines-phase-column-scroll-0");
    pending.scrollTop = 40;
    fireEvent.scroll(pending);

    useFactoryBoardWorkOrders.mockReturnValue({
      workOrders: orders,
      isLoading: false,
      isPlaceholderData: false,
      backlog: idleBoardPage(),
      open: idleBoardPage(),
      done: idleBoardPage(),
    });
    view.rerender(
      <LinesBoardSpecHarness path={`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`} />,
    );

    expect(screen.getByTestId("lines-phase-column-scroll-0").scrollTop).toBe(1760);
  });

  it("does not keep backlog scroll when the line changes", async () => {
    restoreHeights = stubElementHeights({ scrollHeight: 2000, clientHeight: 240 });
    const orders = Array.from({ length: 8 }, (_, index) => kickoffDraft(index));
    useFactoryWorkOrders.mockReturnValue({ data: orders });
    const user = userEvent.setup();
    render(
      <LinesBoardSpecHarness
        path={`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`}
        navigateTo={`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_HOTFIX_ID}`}
      />,
    );

    const scroller = screen.getByTestId("lines-backlog-column-scroll");
    scroller.scrollTop = 1760;
    fireEvent.scroll(scroller);
    expect(scroller.scrollTop).toBe(1760);

    await user.click(screen.getByTestId("lines-test-navigate"));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_HOTFIX_ID}`,
    );
    expect(screen.getByTestId("lines-backlog-column-scroll").scrollTop).toBe(0);

    await user.click(screen.getByTestId("lines-test-back"));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`,
    );
    expect(screen.getByTestId("lines-backlog-column-scroll").scrollTop).toBe(1760);
  });
});
