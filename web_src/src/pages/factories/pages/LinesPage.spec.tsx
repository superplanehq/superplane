import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import type {
  FactoriesFactory,
  FactoriesFactoryIntake,
  FactoriesWorkOrder,
  FactoriesWorkOrderSummary,
  FactoryAutomation,
  FactoryLineStage,
  CanvasesListRunsResponse,
} from "@/api-client";
import type * as canvasData from "@/hooks/useCanvasData";
import * as apiClient from "@/api-client";
import type * as integrationsData from "@/hooks/useIntegrations";
import { resetFactoryBoardLaneScrollPositions } from "@/hooks/useFactoryBoardLaneScroll";
import {
  FEATURE_FACTORY_CUSTOM_AUTOMATIONS,
  FEATURE_AUTOMATIONS_AS_CODE,
  FEATURE_FACTORY_DATADOG_INTAKE,
  FEATURE_FACTORY_PRODUCTIVE_INTAKE,
  FEATURE_FACTORY_SCHEDULED_INTAKE,
  FEATURE_WORKSPACE_SKILLS,
} from "@/lib/experimentalFeatures";
import { unmockedSrc } from "@/test/unmockedModule";
import { dematerializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";

vi.mock("@monaco-editor/react", () => {
  function MockMonacoEditor({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) {
    return <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />;
  }
  return { default: MockMonacoEditor, Editor: MockMonacoEditor };
});
import {
  factoryColumnAutomationViewPath,
  factoryDatadogIntakeSetupPath,
  factoryDependabotIntakeSetupPath,
  factoryHomePath,
  factoryGitHubIntakeSetupPath,
  factoryJiraIntakeSetupPath,
  factoryLinearIntakeSetupPath,
  factoryPlanningPath,
  factoryProductiveIntakeSetupPath,
  factoryPRFeedbackPath,
  factoryPRFeedbackSetupPath,
  factoryRiskScoreSetupPath,
  factorySentryIntakeSetupPath,
  firstFactoryLineId,
} from "../lib/factoryPagePaths";
import { factoryCustomAutomationSetupPath } from "../lib/customAutomationPaths";
import {
  ACME_ONBOARDING_FACTORY,
  ACME_ONBOARDING_FACTORY_ROUTE_SEGMENT,
  ACME_ONBOARDING_LINE_ID,
  DEFAULT_FACTORY_PLANNING,
  DRAFT_WORK_ORDER,
  factoryWithPlanning,
  GITHUB_ISSUES_INTAKE,
  GITHUB_ISSUES_INTAKE_ID,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
  REFUND_LINE_HOTFIX_ID,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../__fixtures__/lineMetricsBoardOrders";
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
  path = `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`,
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
const permissions = vi.hoisted(() => ({ canUpdateFactories: true }));
const idleBoardPage = () => ({ hasNextPage: false, isFetchingNextPage: false, fetchNextPage: vi.fn() });
const useFactoryWorkOrders = vi.fn(() => ({ data: [] as FactoriesWorkOrderSummary[] }));
const useFactoryBoardWorkOrders = vi.fn((_organizationId?: string, _factoryId?: string, _options?: unknown) => ({
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
  useFactoryBoardWorkOrders: (organizationId: string, factoryId: string, options?: unknown) =>
    useFactoryBoardWorkOrders(organizationId, factoryId, options),
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
  useFactorySchedulePreview: () => ({
    data: new Date(Date.now() + 86_400_000).toISOString(),
    isFuture: true,
    isSuccess: true,
    isFetching: false,
    isPending: false,
    error: null,
    refetch: vi.fn(),
  }),
  useDeleteFactoryAutomation: () => ({ mutateAsync: deleteFactoryAutomationMutateAsync, isPending: false }),
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
  usePermissions: () => ({
    canAct: (resource: string, action: string) =>
      resource === "factories" && action === "update" ? permissions.canUpdateFactories : true,
    currentUserId: "storybook-user",
    isLoading: false,
  }),
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

vi.mock("./useIntakeWebhookRetry", () => ({
  useIntakeWebhookRetry: () => ({ retryIntake: vi.fn(), retrying: false }),
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
    useTriggers: () => ({ data: [], isLoading: false, isPending: false }),
    useWidgets: () => ({ data: [], isLoading: false, isPending: false }),
  };
});

vi.mock("@/hooks/useComponentData", () => ({
  useComponents: () => ({
    data: [{ name: "runnerClaudeCode", label: "Run Claude Code", configuration: [] }],
    isLoading: false,
    isPending: false,
  }),
  useComponent: () => ({
    data: { name: "runnerClaudeCode", configuration: [] },
    isLoading: false,
    isPending: false,
  }),
}));

vi.mock("@/hooks/useIntegrations", () => {
  const actual = unmockedSrc<typeof integrationsData>("hooks/useIntegrations");
  return {
    ...actual,
    useAvailableIntegrations: () => ({ data: [], isLoading: false, isPending: false }),
    useConnectedIntegrations: () => ({ data: [], isLoading: false, isPending: false }),
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
  permissions.canUpdateFactories = true;
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

describe("LinesPage custom Backlog runs", () => {
  beforeEach(async () => {
    await resetLinesBoardMocks();
  });

  it.each(["STATE_STARTED", "STATE_CANCELLING"] as const)(
    "refreshes older %s task-entry runs without reloading cached pages",
    async (state) => {
      const [order] = REVIEW_CANDIDATE_WORK_ORDERS;
      const createdAt = new Date(Date.now() - 60_000).toISOString();
      const customRun = {
        id: "inspect-run",
        state,
        createdAt,
        rootEvent: { data: { workOrder: { id: order.id } } },
      };
      const response = (data: CanvasesListRunsResponse) => ({
        data,
        request: new Request("http://localhost/runs"),
        response: new Response(),
      });
      const listRuns = vi.spyOn(apiClient, "canvasesListRuns").mockImplementation(async (options) => {
        if (options?.path.canvasId !== "inspect-task") return response({ runs: [] });
        if (!options.query?.before) {
          return response({
            runs: [{ ...customRun, id: "other-task-run", rootEvent: { data: { workOrder: { id: "other-task" } } } }],
            hasNextPage: true,
            lastTimestamp: createdAt,
          });
        }
        return response({ runs: [customRun], hasNextPage: false });
      });
      const describeRun = vi.spyOn(apiClient, "canvasesDescribeRun").mockResolvedValue({
        ...response({}),
        data: { run: { ...customRun, state: "STATE_FINISHED", result: "RESULT_PASSED" } },
      });
      try {
        useFactoryWorkOrders.mockReturnValue({ data: [order] });
        useWorkOrder.mockReturnValue({ data: { ...order, checks: [] } });
        useFactoryAutomations.mockReturnValue({
          data: [{ id: "inspect-task", name: "Inspect task", columnKey: "backlog", templateId: "custom-verification" }],
        });
        renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/842`);
        await waitFor(() =>
          expect(
            listRuns.mock.calls.some(([options]) => options.path.canvasId === "inspect-task" && options.query?.before),
          ).toBe(true),
        );
        const stage = await screen.findByTestId("redesign-console-automation-inspect-task-inspect-task");
        expect(stage).toHaveTextContent("Inspect task");
        if (state !== "STATE_CANCELLING") {
          expect(within(stage).getByRole("button", { name: "Stop" })).toBeInTheDocument();
        }
        expect(stage).toHaveTextContent("1 agent run");
        expect(screen.getByTestId("split-run-intent-document")).toBeInTheDocument();

        const user = userEvent.setup();
        await user.click(screen.getByRole("button", { name: "Close" }));
        await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));
        await waitFor(() =>
          expect(screen.getByRole("button", { name: "Toggle Inspect task details" })).toHaveAttribute(
            "aria-expanded",
            "false",
          ),
        );
        expect(listRuns.mock.calls.filter(([options]) => options.query?.before)).toHaveLength(1);
        expect(describeRun.mock.calls.some(([options]) => options.path.runId === customRun.id)).toBe(true);
        listRuns.mockImplementation(async () =>
          response({
            runs: [
              { ...customRun, id: "new-inspect-run", state: "STATE_FINISHED", result: "RESULT_PASSED" },
              { ...customRun, id: "other-task-run", rootEvent: { data: { workOrder: { id: "other-task" } } } },
            ],
            hasNextPage: true,
            lastTimestamp: createdAt,
          }),
        );
        await user.click(screen.getByRole("button", { name: "Close" }));
        await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));
        await waitFor(() =>
          expect(screen.getByTestId("redesign-console-automation-inspect-task-inspect-task")).toHaveTextContent(
            "2 agent runs",
          ),
        );
        expect(listRuns.mock.calls.filter(([options]) => options.query?.before)).toHaveLength(1);
        expect(screen.queryByRole("button", { name: "Stop" })).not.toBeInTheDocument();
      } finally {
        listRuns.mockRestore();
        describeRun.mockRestore();
      }
    },
  );

  it("explains unavailable custom history and retries without hiding cached runs", async () => {
    const [order] = REVIEW_CANDIDATE_WORK_ORDERS;
    const listRuns = vi.spyOn(apiClient, "canvasesListRuns").mockRejectedValue(new Error("History unavailable"));
    try {
      useFactoryWorkOrders.mockReturnValue({ data: [order] });
      useWorkOrder.mockReturnValue({ data: { ...order, checks: [] } });
      useFactoryAutomations.mockReturnValue({
        data: [{ id: "inspect-task", name: "Inspect task", columnKey: "backlog", templateId: "custom-verification" }],
      });
      renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/842`);
      expect(await screen.findByRole("alert")).toHaveTextContent("Could not load Backlog automation runs.");
      listRuns.mockResolvedValue({
        data: {
          runs: [
            {
              id: "inspect-run",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              rootEvent: { data: { workOrder: { id: order.id } } },
            },
          ],
        },
        request: new Request("http://localhost/runs"),
        response: new Response(),
      });
      const user = userEvent.setup();
      await user.click(screen.getByRole("button", { name: "Retry" }));
      expect(await screen.findByRole("button", { name: "Toggle Inspect task details" })).toBeInTheDocument();
      expect(screen.queryByRole("alert")).not.toBeInTheDocument();
      listRuns.mockRejectedValue(new Error("Refresh unavailable"));
      await user.click(screen.getByRole("button", { name: "Close" }));
      await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));
      expect(await screen.findByRole("alert")).toHaveTextContent("Could not load Backlog automation runs.");
      expect(screen.getByRole("button", { name: "Toggle Inspect task details" })).toBeInTheDocument();
    } finally {
      listRuns.mockRestore();
    }
  });
});

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
      <LinesBoardSpecHarness
        path={`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`}
      />,
    );

    expect(screen.queryByRole("status", { name: "Loading tasks" })).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-column-scroll")).toHaveAttribute("data-reveal");
    expect(screen.getByTestId("work-order-card-wo-draft-refunds")).toBeInTheDocument();
  });

  it("sets a pastel colour on the backlog from circular swatches", async () => {
    updateFactoryLineMutateAsync.mockResolvedValueOnce({
      id: REFUND_LINE_PLAN_ID,
      columnColors: { backlog: "sky" },
    });
    const user = userEvent.setup();
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`);

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-color-sky"));

    await waitFor(() =>
      expect(updateFactoryLineMutateAsync).toHaveBeenCalledWith({
        lineId: REFUND_LINE_PLAN_ID,
        columnColors: { backlog: "sky" },
      }),
    );
    expect(screen.getByTestId("lines-backlog-column")).toHaveClass("line-board-color-sky", "line-board-column-vivid");
  });

  it("shows a score on a draft card and hides it on other columns", () => {
    const draft = withBoardChecks(REVIEW_CANDIDATE_WORK_ORDERS)[0];
    useFactoryWorkOrders.mockReturnValue({
      data: [draft, { ...BOARD_IMPLEMENT_NOTIFY_ORDER, checkScores: draft.checkScores }],
    });
    renderLinesBoard();

    expect(screen.getByTestId("work-order-card-score-wo-review-pay-842")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-card-score-wo-board-implement-notify")).not.toBeInTheDocument();
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
    expect(cardScore).toHaveTextContent("Clarity5/5Confidence");
    expect(within(card).getByTestId("work-order-card-score-wo-review-pay-842-confidence")).not.toHaveTextContent(/\d/);
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
    const location = screen.getByTestId("lines-test-location");
    expect(location).toHaveTextContent(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT.toLowerCase()}/task/842`);
    expect(location).not.toHaveTextContent("lineId=");
    expect(screen.getByTestId("lines-detail-page")).not.toHaveClass("animate-in");

    await user.click(within(dialog).getByRole("button", { name: "Close" }));
    await waitFor(() => {
      expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
    });
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT.toLowerCase()}/lines/${REFUND_LINE_PLAN_ID}`,
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
      expect(within(dialog).getByRole("status")).toBeInTheDocument();
      expect(within(dialog).queryByRole("tab", { name: "Task" })).not.toBeInTheDocument();
      expect(within(dialog).queryByRole("tab", { name: "Automations" })).not.toBeInTheDocument();
      expect(within(dialog).queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
      expect(within(dialog).queryByRole("button", { name: "Refine" })).not.toBeInTheDocument();
      expect(within(dialog).getByTestId("popup-work-order-archive-button")).toBeInTheDocument();
    } finally {
      clearBacklogAnalysisPending(analyzingOrderId);
    }
  });

  it("opens the split run from a task permalink", async () => {
    useFactoryWorkOrders.mockReturnValue({ data: REVIEW_CANDIDATE_WORK_ORDERS });
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/842`);

    const popup = await screen.findByTestId("work-order-split-run");
    expect(within(popup).getByTestId("popup-work-order-title")).toHaveTextContent(
      "Add retry handling to webhook delivery",
    );
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/842`,
    );
  });

  it("keeps the line board without a popup when the permalink is unknown", () => {
    useFactoryWorkOrders.mockReturnValue({ data: REVIEW_CANDIDATE_WORK_ORDERS });
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/999`);

    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
  });

  it("keeps the current line when a card opens from a second line", async () => {
    useFactoryWorkOrders.mockReturnValue({ data: REVIEW_CANDIDATE_WORK_ORDERS });
    const user = userEvent.setup();
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_HOTFIX_ID}`);

    await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));

    const location = screen.getByTestId("lines-test-location");
    expect(location).toHaveTextContent(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT.toLowerCase()}/task/842`);
    expect(location).not.toHaveTextContent("lineId=");

    await user.click(within(screen.getByTestId("work-order-split-run")).getByRole("button", { name: "Close" }));
    await waitFor(() => {
      expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
    });
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT.toLowerCase()}/lines/${REFUND_LINE_HOTFIX_ID}`,
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
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`,
    );
  });

  it("redirects to the first board when a missing line id is opened", async () => {
    const user = userEvent.setup();
    const missingLinePath = `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/line-missing`;
    render(
      <LinesBoardSpecHarness
        path={`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`}
        factory={REFUND_FACTORY}
        navigateTo={missingLinePath}
      />,
    );

    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();

    await user.click(screen.getByTestId("lines-test-navigate"));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryHomePath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, firstFactoryLineId(REFUND_FACTORY)),
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
});

describe("LinesPage board stages", () => {
  beforeEach(async () => {
    await resetLinesBoardMocks();
  });

  it("moves a task that passed a custom last stage to Verify", () => {
    const factory = {
      ...REFUND_FACTORY,
      lines: (REFUND_FACTORY.lines ?? []).map((line) =>
        line.id === REFUND_LINE_PLAN_ID
          ? {
              ...line,
              stages: [
                { id: "backlog", key: "backlog", name: "Backlog", kind: "STAGE_KIND_BACKLOG" as const },
                { id: "phase-0", key: "phase-0", name: "Implement", kind: "STAGE_KIND_PHASE" as const, stepIndex: 0 },
                { id: "docs", key: "custom-docs", name: "Update docs", kind: "STAGE_KIND_CUSTOM" as const },
                { id: "verify", key: "verify", name: "Verify", kind: "STAGE_KIND_VERIFY" as const },
                { id: "done", key: "done", name: "Done", kind: "STAGE_KIND_DONE" as const },
              ],
            }
          : line,
      ),
    };
    const lineRef = { id: REFUND_LINE_PLAN_ID, name: "plan-and-implement" };
    useFactoryWorkOrders.mockReturnValue({
      data: [
        {
          id: "wo-docs-passed",
          title: "Docs stage passed",
          state: "STATE_OPEN",
          lineDispatches: [
            {
              id: "dispatch-docs",
              line: lineRef,
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              stepExecutions: [
                {
                  id: "exec-implement",
                  step: "Implement",
                  stepIndex: 0,
                  stageKey: "phase-0",
                  state: "STATE_FINISHED",
                  result: "RESULT_PASSED",
                  createdAt: "2026-08-11T12:00:00.000Z",
                },
                {
                  id: "exec-docs",
                  step: "Update Docs",
                  stepIndex: 1,
                  stageKey: "custom-docs",
                  state: "STATE_FINISHED",
                  result: "RESULT_PASSED",
                  createdAt: "2026-08-11T12:10:00.000Z",
                },
              ],
            },
          ],
        },
      ],
    });

    renderLinesBoard(undefined, vi.fn(), factory);

    expect(screen.getByTestId("lines-verify-column")).toHaveTextContent("Docs stage passed");
    expect(screen.getByTestId("lines-custom-stage-custom-docs")).not.toHaveTextContent("Docs stage passed");
  });

  const customStageFactory = (name = "QA"): FactoriesFactory => ({
    ...REFUND_FACTORY,
    lines: (REFUND_FACTORY.lines ?? []).map((line) =>
      line.id === REFUND_LINE_PLAN_ID
        ? {
            ...line,
            columnColors: { "custom-qa": "blue" },
            stages: [
              { id: "backlog", key: "backlog", name: "Backlog", kind: "STAGE_KIND_BACKLOG" },
              { id: "phase-0", key: "phase-0", name: "Implement", kind: "STAGE_KIND_PHASE", stepIndex: 0 },
              { id: "custom", key: "custom-qa", name, kind: "STAGE_KIND_CUSTOM" },
              { id: "verify", key: "verify", name: "Verify", kind: "STAGE_KIND_VERIFY" },
              { id: "done", key: "done", name: "Done", kind: "STAGE_KIND_DONE" },
            ],
          }
        : line,
    ),
  });
  const renameCustomStage = (name: string, titleId = "lines-column-title-custom-qa") => {
    fireEvent.click(screen.getByTestId(titleId));
    const input = screen.getByTestId(`${titleId}-input`);
    fireEvent.change(input, { target: { value: name } });
    fireEvent.keyDown(input, { key: "Enter" });
  };
  it.each([false, true])(
    "opens custom-stage automation configuration from a saved board row (YAML: %s)",
    async (yamlEnabled) => {
      enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
      const fetch = yamlEnabled
        ? vi.spyOn(globalThis, "fetch").mockImplementation(async () =>
            Response.json({
              id: "app-qa",
              factory_id: PRIMARY_FACTORY_ID,
              stage: "custom-qa",
              name: "Check QA",
              revision: 1,
              yaml: "apiVersion: v2\nkind: Automation\nname: Check QA\ntriggers:\n  - use: task_enters_stage\nsteps:\n  - name: Inspect\n    bash: make lint\n",
            }),
          )
        : undefined;
      if (yamlEnabled) enabledExperimentalFeatures.add(FEATURE_AUTOMATIONS_AS_CODE);
      useFactoryAutomations.mockReturnValue({ data: [{ id: "app-qa", name: "Check QA", columnKey: "custom-qa" }] });
      renderLinesBoard(undefined, vi.fn(), customStageFactory());
      await userEvent
        .setup()
        .click(screen.getByTestId("lines-custom-stage-custom-qa-automation-rows-row-custom-app-qa"));
      expect(await screen.findByTestId("custom-automation-setup-modal")).toBeInTheDocument();
      expect(screen.getByTestId("lines-test-location")).toHaveTextContent("customAutomation=app-qa");
      expect(screen.getByTestId("lines-test-location")).toHaveTextContent("automationColumn=custom-qa");
      expect(updateCanvasVersionMutateAsync).not.toHaveBeenCalled();
      if (yamlEnabled) {
        await userEvent.setup().click(await screen.findByRole("button", { name: "Edit YAML" }));
        expect((screen.getByLabelText("Automation YAML") as HTMLTextAreaElement).value).toContain("task_enters_stage");
        await userEvent.setup().click(screen.getByRole("button", { name: "Cancel" }));
        expect(screen.queryByTestId("custom-automation-setup-modal")).not.toBeInTheDocument();
        expect(fetch?.mock.calls.every(([, init]) => !init?.method)).toBe(true);
        fetch?.mockRestore();
      }
    },
  );
  it("opens custom-stage automation configuration before the feature list loads", async () => {
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-qa", name: "Check QA", columnKey: "custom-qa" }] });
    renderLinesBoard(undefined, vi.fn(), customStageFactory());
    await userEvent.setup().click(screen.getByTestId("lines-custom-stage-custom-qa-automation-rows-row-custom-app-qa"));
    expect(await screen.findByTestId("custom-automation-setup-modal")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent("customAutomation=app-qa");
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent("automationColumn=custom-qa");
  });
  it("opens shared setup from the stage menu before the feature list loads", async () => {
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), customStageFactory());
    await user.click(screen.getByTestId("lines-custom-stage-menu-custom-qa"));
    await user.click(screen.getByTestId("lines-custom-stage-menu-custom-qa-add-automation"));
    expect(await screen.findByTestId("custom-automation-setup-modal")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent("customAutomation=new");
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent("automationColumn=custom-qa");
    expect(screen.queryByTestId("add-column-automation-picker")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factory-app-name-input")).not.toBeInTheDocument();
  });
  const editStages = async () => {
    const user = userEvent.setup();
    await user.click(screen.getByTestId("lines-board-view-menu"));
    await user.click(screen.getByRole("menuitem", { name: "Edit stages" }));
  };

  it.each(["drag", "keyboard"])("moves a custom card to Verify with %s and discards placement", async (interaction) => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-qa", name: "Check QA", columnKey: "custom-qa" }] });
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), customStageFactory());
    await editStages();
    const handle = within(screen.getByTestId("lines-stage-shell-custom-qa")).getByRole("button", {
      name: "Move Check QA",
    });
    if (interaction === "drag") {
      fireEvent.dragStart(handle, { dataTransfer: new DataTransfer() });
      fireEvent.drop(screen.getByTestId("lines-stage-shell-verify"));
      fireEvent.dragEnd(handle);
    } else {
      fireEvent.keyDown(handle, { key: "ArrowRight" });
    }
    const moved = within(screen.getByTestId("lines-stage-shell-verify")).getByRole("button", { name: "Move Check QA" });
    if (interaction === "keyboard") expect(moved).toHaveFocus();
    await user.click(screen.getByRole("button", { name: "Discard" }));
    await user.click(
      within(screen.getByTestId("discard-line-stages-dialog")).getByRole("button", { name: "Discard changes" }),
    );
    expect(within(screen.getByTestId("lines-stage-shell-custom-qa")).getByText("Check QA")).toBeInTheDocument();
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
  });

  it("rejects agent drops and keyboard moves into Backlog", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    renderLinesBoard(undefined, vi.fn(), customStageFactory());
    await editStages();
    const source = within(screen.getByTestId("lines-stage-shell-phase-0"));
    const handle = source.getByRole("button", { name: /^Move / });
    fireEvent.keyDown(handle, { key: "ArrowLeft" });
    fireEvent.dragStart(handle, { dataTransfer: new DataTransfer() });
    fireEvent.drop(screen.getByTestId("lines-stage-shell-backlog"));
    fireEvent.dragEnd(handle);
    expect(source.getByRole("button", { name: /^Move / })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Save changes" })).not.toBeInTheDocument();
  });

  it("edits and saves stage auto-advance settings", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const user = userEvent.setup();
    const factory = customStageFactory();
    const selectedLine = factory.lines?.find((line) => line.id === REFUND_LINE_PLAN_ID);
    updateFactoryLineMutateAsync.mockResolvedValueOnce({
      ...selectedLine,
      stages: selectedLine?.stages?.map((stage) =>
        stage.key === "custom-qa" ? { ...stage, autoAdvance: false } : stage,
      ),
    });
    renderLinesBoard(undefined, vi.fn(), factory);

    await editStages();
    expect(screen.getByRole("switch", { name: "Auto-advance Implement" })).toBeChecked();
    expect(screen.getByRole("switch", { name: "Auto-advance QA" })).toBeChecked();
    expect(screen.getByRole("switch", { name: "Auto-advance Verify" })).not.toBeChecked();
    expect(screen.queryByRole("switch", { name: "Auto-advance Backlog" })).not.toBeInTheDocument();
    expect(screen.queryByRole("switch", { name: "Auto-advance Done" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("switch", { name: "Auto-advance QA" }));
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    const dialog = screen.getByTestId("save-line-stages-dialog");
    expect(within(dialog).getByTestId("save-line-stages-summary")).toHaveTextContent("Disable auto-advance: QA");
    await user.click(within(dialog).getByRole("button", { name: "Save changes" }));

    const savedStages = updateFactoryLineMutateAsync.mock.calls[0][0].stageConfiguration.stages;
    expect(savedStages.find((stage: FactoryLineStage) => stage.key === "custom-qa")?.autoAdvance).toBe(false);
  });

  it("saves only the custom name and uses the refreshed line after remount", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const factory = customStageFactory();
    const savedFactory = customStageFactory("Quality review");
    updateFactoryLineMutateAsync.mockResolvedValueOnce(
      savedFactory.lines?.find((line) => line.id === REFUND_LINE_PLAN_ID),
    );
    const view = renderLinesBoard(undefined, vi.fn(), factory);
    renameCustomStage("  Quality review  ");
    await waitFor(() => expect(updateFactoryLineMutateAsync).toHaveBeenCalledTimes(1));
    expect(updateFactoryLineMutateAsync.mock.calls[0][0]).toEqual({
      lineId: REFUND_LINE_PLAN_ID,
      stageConfiguration: { stages: savedFactory.lines?.find((line) => line.id === REFUND_LINE_PLAN_ID)?.stages },
    });
    view.rerender(<LinesBoardSpecHarness factory={savedFactory} />);
    expect(screen.getByTestId("lines-column-title-custom-qa")).toHaveTextContent("Quality review");
    view.unmount();
    renderLinesBoard(undefined, vi.fn(), savedFactory);
    expect(screen.getByTestId("lines-column-title-custom-qa")).toHaveTextContent("Quality review");
  });

  it("explains automation deletion before saving and cancels or discards without saving", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const user = userEvent.setup();
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-qa", name: "Check QA", columnKey: "custom-qa" }] });
    renderLinesBoard(undefined, vi.fn(), customStageFactory());
    await editStages();
    await user.click(screen.getByRole("button", { name: "Delete QA" }));
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    const dialog = screen.getByTestId("save-line-stages-dialog");
    expect(dialog).toHaveTextContent(
      "Saving a custom-stage deletion also deletes automations that execute in that stage.",
    );
    expect(dialog).toHaveTextContent("Automations that only appear there stay in the workspace.");
    expect(dialog).toHaveTextContent("Removing a system stage keeps its automations.");
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Discard" }));
    await user.click(
      within(screen.getByTestId("discard-line-stages-dialog")).getByRole("button", { name: "Discard changes" }),
    );
    expect(screen.getByTestId("lines-custom-stage-custom-qa-automation-rows-row-custom-app-qa")).toHaveTextContent(
      "Check QA",
    );
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
  });

  it("keeps renames in the draft, clears reverted changes, and discards without saving", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), customStageFactory());
    await editStages();
    renameCustomStage("Quality review");
    expect(screen.getByRole("button", { name: "Save changes" })).toBeEnabled();
    renameCustomStage("QA");
    expect(screen.queryByRole("button", { name: "Save changes" })).not.toBeInTheDocument();
    renameCustomStage("Quality review");
    fireEvent.click(screen.getByTestId("lines-column-title-custom-qa"));
    fireEvent.keyDown(screen.getByTestId("lines-column-title-custom-qa-input"), { key: "Escape" });
    expect(screen.queryByTestId("discard-line-stages-dialog")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Discard" }));
    await user.click(
      within(screen.getByTestId("discard-line-stages-dialog")).getByRole("button", { name: "Discard changes" }),
    );
    expect(screen.getByTestId("lines-column-title-custom-qa")).toHaveTextContent("QA");
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
  });

  it.each([false, true])("saves a custom stage rename with pending card move: %s", async (moveCard) => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const user = userEvent.setup();
    const savedFactory = customStageFactory("Quality review");
    updateFactoryLineMutateAsync.mockResolvedValueOnce(
      savedFactory.lines?.find((line) => line.id === REFUND_LINE_PLAN_ID),
    );
    const view = renderLinesBoard(undefined, vi.fn(), customStageFactory());
    await editStages();
    if (moveCard) {
      const source = within(screen.getByTestId("lines-stage-shell-phase-0"));
      fireEvent.keyDown(source.getByRole("button", { name: /^Move / }), { key: "ArrowRight" });
    }
    renameCustomStage("Quality review");
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    const dialog = screen.getByTestId("save-line-stages-dialog");
    expect(within(dialog).getByTestId("save-line-stages-summary")).toHaveTextContent("QA → Quality review");
    await user.click(within(dialog).getByRole("button", { name: "Save changes" }));
    await waitFor(() => expect(screen.queryByTestId("line-stage-edit-toolbar")).not.toBeInTheDocument());
    expect(updateFactoryLineMutateAsync.mock.calls[0][0].stageConfiguration.stages).toEqual(
      savedFactory.lines?.find((line) => line.id === REFUND_LINE_PLAN_ID)?.stages,
    );
    if (moveCard) {
      expect(updateFactoryLineMutateAsync.mock.calls[0][0].stageConfiguration.automationMoves).toEqual([
        expect.objectContaining({
          type: "agent-step",
          executionKey: "phase-0",
          sourceKey: "phase-0",
          destinationKey: "custom-qa",
        }),
      ]);
    }
    view.rerender(<LinesBoardSpecHarness factory={savedFactory} />);
    expect(screen.getByTestId("lines-column-title-custom-qa")).toHaveTextContent("Quality review");
  });

  it.each(["   ", "x".repeat(65), "vErIfY"])("rejects an invalid custom stage name: %s", (name) => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    renderLinesBoard(undefined, vi.fn(), customStageFactory());
    renameCustomStage(name);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
    fireEvent.keyDown(screen.getByTestId("lines-column-title-custom-qa-input"), { key: "Escape" });
    expect(screen.getByTestId("lines-column-title-custom-qa")).toHaveTextContent("QA");
  });

  it.each(["feature", "permission"])("disables custom title editing without %s", (gate) => {
    if (gate === "permission") {
      enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
      permissions.canUpdateFactories = false;
    }
    renderLinesBoard(undefined, vi.fn(), customStageFactory());
    const title = screen.getByTestId("lines-column-title-custom-qa");
    fireEvent.click(title);
    fireEvent.keyDown(title, { key: "Enter" });
    expect(screen.queryByTestId("lines-column-title-custom-qa-input")).not.toBeInTheDocument();
    expect(title).not.toHaveAttribute("tabindex");
  });

  it("prevents overlapping renames and retains the saved name on failure", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const { showErrorToast } = await import("@/lib/toast");
    vi.mocked(showErrorToast).mockClear();
    let rejectSave!: (error: Error) => void;
    updateFactoryLineMutateAsync.mockImplementationOnce(
      () =>
        new Promise((_, reject) => {
          rejectSave = reject;
        }),
    );
    renderLinesBoard(undefined, vi.fn(), customStageFactory());
    const title = screen.getByTestId("lines-column-title-custom-qa");
    fireEvent.keyDown(title, { key: "Enter" });
    const input = screen.getByTestId("lines-column-title-custom-qa-input");
    fireEvent.change(input, { target: { value: "QA" } });
    fireEvent.keyDown(input, { key: "Enter" });
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
    renameCustomStage("Quality review");
    fireEvent.click(title);
    expect(screen.queryByTestId("lines-column-title-custom-qa-input")).not.toBeInTheDocument();
    await act(async () => rejectSave(new Error("Save failed")));
    expect(title).toHaveTextContent("QA");
    expect(showErrorToast).toHaveBeenCalledWith("Save failed Try again.");
    expect(updateFactoryLineMutateAsync).toHaveBeenCalledTimes(1);
  });

  it("gates stage editing and renders saved custom stages", async () => {
    const user = userEvent.setup();
    const factory = {
      ...REFUND_FACTORY,
      lines: (REFUND_FACTORY.lines ?? []).map((line) =>
        line.id === REFUND_LINE_PLAN_ID
          ? {
              ...line,
              stages: [
                { id: "backlog", key: "backlog", name: "Incoming", kind: "STAGE_KIND_BACKLOG" as const },
                {
                  id: "phase-0",
                  key: "phase-0",
                  name: "Implement",
                  kind: "STAGE_KIND_PHASE" as const,
                  stepIndex: 0,
                },
                { id: "custom", key: "custom-qa", name: "QA", kind: "STAGE_KIND_CUSTOM" as const },
                {
                  id: "phase-1",
                  key: "phase-1",
                  name: "Verify changes",
                  kind: "STAGE_KIND_PHASE" as const,
                  stepIndex: 1,
                },
                {
                  id: "phase-2",
                  key: "phase-2",
                  name: "Removed step",
                  kind: "STAGE_KIND_PHASE" as const,
                  stepIndex: 2,
                },
                { id: "verify", key: "verify", name: "Check", kind: "STAGE_KIND_VERIFY" as const },
                { id: "done", key: "done", name: "Shipped", kind: "STAGE_KIND_DONE" as const },
              ],
            }
          : line,
      ),
    };

    const disabled = renderLinesBoard(undefined, vi.fn(), factory);
    expect(screen.getByTestId("lines-custom-stage-custom-qa")).toHaveTextContent("No tasks in this stage.");
    expect(screen.getByTestId("lines-phase-board").children).toHaveLength(6);
    await user.click(screen.getByTestId("lines-board-view-menu"));
    expect(screen.queryByRole("menuitem", { name: "Edit stages" })).not.toBeInTheDocument();
    disabled.unmount();

    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const selectedLine = factory.lines?.find((line) => line.id === REFUND_LINE_PLAN_ID);
    updateFactoryLineMutateAsync.mockResolvedValueOnce({
      ...selectedLine,
      stages: [
        ...(selectedLine?.stages?.slice(0, -1) ?? []),
        {
          id: "custom-deploy",
          key: "custom-deploy",
          name: "Deploy to prod",
          kind: "STAGE_KIND_CUSTOM" as const,
        },
        ...(selectedLine?.stages?.slice(-1) ?? []),
      ],
    });
    renderLinesBoard(undefined, vi.fn(), factory);
    await user.click(screen.getByTestId("lines-board-view-menu"));
    await user.click(screen.getByRole("menuitem", { name: "Edit stages" }));
    await user.click(screen.getByRole("button", { name: "Add stage" }));
    const addDialog = screen.getByTestId("add-line-stage-dialog");
    await user.type(within(addDialog).getByRole("textbox", { name: "Name" }), "Deploy");
    await user.click(within(addDialog).getByRole("button", { name: "Add stage" }));

    const draftLane = screen.getByRole("region", { name: "Deploy" });
    expect(screen.getByRole("switch", { name: "Auto-advance Deploy" })).toBeChecked();
    const draftTitle = within(draftLane).getByTestId(/^lines-column-title-new-/);
    const draftTitleId = draftTitle.getAttribute("data-testid")!;
    renameCustomStage("Deploy to prod", draftTitleId);
    expect(draftLane).toHaveAccessibleName("Deploy to prod");
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    const saveDialog = screen.getByTestId("save-line-stages-dialog");
    expect(within(saveDialog).getByTestId("save-line-stages-summary")).toHaveTextContent("Deploy to prod");
    await user.click(within(saveDialog).getByRole("button", { name: "Save changes" }));

    expect(await screen.findByTestId("custom-automation-setup-modal")).toBeInTheDocument();
    expect(screen.queryByTestId("factory-app-name-input")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent("customAutomation=new");
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent("automationColumn=custom-deploy");
    const savedStages = updateFactoryLineMutateAsync.mock.calls[0][0].stageConfiguration.stages;
    expect(savedStages.at(-2)).toMatchObject({
      id: "",
      name: "Deploy to prod",
      kind: "STAGE_KIND_CUSTOM",
      autoAdvance: true,
    });
    expect(screen.queryByTestId("line-stage-edit-toolbar")).not.toBeInTheDocument();
  });

  it("hides stage mutations without factory update permission", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    permissions.canUpdateFactories = false;
    const user = userEvent.setup();
    renderLinesBoard();

    expect(screen.queryByTestId("lines-stage-reorder-phase-0")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("lines-board-view-menu"));
    expect(screen.queryByRole("menuitem", { name: "Edit stages" })).not.toBeInTheDocument();
    await user.keyboard("{Escape}");
    await user.click(screen.getByTestId("lines-phase-menu-0"));
    expect(screen.queryByTestId("lines-phase-menu-0-delete-stage")).not.toBeInTheDocument();
  });

  it("uses a persisted stage name after the line refreshes", async () => {
    const user = userEvent.setup();
    const factoryWithStageName = (name: string): FactoriesFactory => ({
      ...REFUND_FACTORY,
      lines: (REFUND_FACTORY.lines ?? []).map((line) =>
        line.id === REFUND_LINE_PLAN_ID
          ? {
              ...line,
              stages: [
                { id: "backlog", key: "backlog", name: "Backlog", kind: "STAGE_KIND_BACKLOG" as const },
                {
                  id: "phase-0",
                  key: "phase-0",
                  name,
                  kind: "STAGE_KIND_PHASE" as const,
                  stepIndex: 0,
                },
                {
                  id: "phase-1",
                  key: "phase-1",
                  name: "Verify changes",
                  kind: "STAGE_KIND_PHASE" as const,
                  stepIndex: 1,
                },
                { id: "verify", key: "verify", name: "Verify", kind: "STAGE_KIND_VERIFY" as const },
                { id: "done", key: "done", name: "Done", kind: "STAGE_KIND_DONE" as const },
              ],
            }
          : line,
      ),
    });
    const path = `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`;
    const view = renderLinesBoard(path, vi.fn(), factoryWithStageName("Implement"));

    await user.click(screen.getByTestId("lines-column-title-phase-0"));
    const input = await screen.findByTestId("lines-column-title-phase-0-input");
    await user.clear(input);
    await user.type(input, "Temporary name");
    await user.keyboard("{Enter}");
    expect(screen.getByTestId("lines-column-title-phase-0")).toHaveTextContent("Temporary name");

    view.rerender(<LinesBoardSpecHarness path={path} factory={factoryWithStageName("Saved name")} />);

    expect(screen.getByTestId("lines-column-title-phase-0")).toHaveTextContent("Saved name");
  });
});

describe("LinesPage board automations", () => {
  beforeEach(async () => {
    await resetLinesBoardMocks();
  });

  it.each([
    { columnKey: "verify", templateId: "risk-score", cardId: "risk-score-moved-app", sourceKey: "verify" },
    { columnKey: "implement", cardId: "custom-moved-app", sourceKey: "phase-0" },
    { columnKey: "custom-source", cardId: "custom-moved-app", sourceKey: "custom-source" },
  ])(
    "shows a moved $columnKey workspace card on a line without its source stage",
    ({ columnKey, templateId, cardId, sourceKey }) => {
      const destinationKey = columnKey === "custom-source" ? "custom-target" : "backlog";
      useFactoryAutomations.mockReturnValue({
        data: [
          { id: "moved-app", name: "Moved automation", columnKey, templateId, displayColumnKey: destinationKey },
          { id: "unmoved-app", name: "Unmoved automation", columnKey, templateId },
        ],
      });
      const factory: FactoriesFactory = {
        ...REFUND_FACTORY,
        lines: (REFUND_FACTORY.lines ?? []).map((line) => ({
          ...line,
          stages: [
            { id: "backlog", key: "backlog", name: "Backlog", kind: "STAGE_KIND_BACKLOG" },
            { id: "phase-1", key: "phase-1", name: "Check", kind: "STAGE_KIND_PHASE", stepIndex: 1 },
            { id: "custom-target", key: "custom-target", name: "QA", kind: "STAGE_KIND_CUSTOM" },
            { id: "done", key: "done", name: "Done", kind: "STAGE_KIND_DONE" },
          ],
        })),
      };
      renderLinesBoard(undefined, vi.fn(), factory);

      expect(screen.queryByTestId(`lines-stage-shell-${sourceKey}`)).not.toBeInTheDocument();
      const destinationTestId = destinationKey === "backlog" ? "lines-backlog" : `lines-custom-stage-${destinationKey}`;
      expect(
        within(screen.getByTestId(`lines-stage-shell-${destinationKey}`)).getByTestId(
          `${destinationTestId}-automation-rows-row-${cardId}`,
        ),
      ).toBeInTheDocument();
      expect(screen.queryByText("Unmoved automation")).not.toBeInTheDocument();
    },
  );

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
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`);

    await user.click(screen.getByTestId(`lines-backlog-automation-rows-row-${GITHUB_ISSUES_INTAKE_ID}`));

    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(`intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`);
    expect(screen.getByTestId("lines-test-location")).not.toHaveTextContent("settings=automation");
    expect(screen.getByTestId("intake-source-settings")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "General" })).toHaveAttribute("data-state", "active");
  });

  it("opens supported Implement settings from the phase automation row", async () => {
    enabledExperimentalFeatures.add(FEATURE_WORKSPACE_SKILLS);
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-refund-implementer", name: "Implement" }] });
    const user = userEvent.setup();
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`);

    await user.click(screen.getByTestId("lines-phase-0-automation-rows-row-step-0-app-refund-implementer"));

    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    const location = screen.getByTestId("lines-test-location");
    expect(location).toHaveTextContent(
      factoryColumnAutomationViewPath(
        "org-1",
        PRIMARY_FACTORY_ROUTE_SEGMENT,
        REFUND_LINE_PLAN_ID,
        "app-refund-implementer",
      ),
    );
    expect(location).not.toHaveTextContent("/apps/");
    expect(location).not.toHaveTextContent("configure=1");
    expect(screen.queryByTestId("column-automation-view")).not.toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-view-tab-agent")).not.toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-view-tab-general")).not.toBeInTheDocument();
    const dialog = screen.getByTestId("merge-confidence-config");
    expect(within(dialog).getByRole("heading", { name: "Implement" })).toBeInTheDocument();
    expect(within(dialog).queryByTestId("merge-confidence-config-canvas")).not.toBeInTheDocument();
    expect(within(dialog).getByTestId("implement-settings-save")).toBeInTheDocument();
    expect(within(dialog).getByTestId("merge-confidence-config-tab-automation")).toHaveAttribute(
      "data-state",
      "active",
    );
    expect(within(dialog).getByTestId("merge-confidence-config-tab-runs")).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Agent" })).not.toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "General" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Save Agent" })).not.toBeInTheDocument();
    expect(within(dialog).getByRole("spinbutton", { name: "Maximum simultaneous runs" })).toHaveValue(5);
    expect(within(dialog).queryByText("MCP servers")).not.toBeInTheDocument();
    expect(within(dialog).queryByText("Skills")).not.toBeInTheDocument();
  });

  it("opens Planning settings from the Task analysis row", async () => {
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-refund-backlog", name: "Ingest" }] });
    const user = userEvent.setup();
    renderLinesBoard(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`);

    await user.click(screen.getByTestId("lines-backlog-automation-rows-row-analysis-app-refund-backlog"));

    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryPlanningPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID),
    );
    expect(screen.getByTestId("planning-settings")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).not.toHaveTextContent("configure=1");
  });

  it("opens Planning settings from the Task analysis row before setup is confirmed", async () => {
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-refund-backlog", name: "Ingest" }] });
    const user = userEvent.setup();
    renderLinesBoard(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`,
      vi.fn(),
      PLANNING_OPEN_FACTORY,
    );

    await user.click(screen.getByTestId("lines-backlog-automation-rows-row-analysis-app-refund-backlog"));

    expect(screen.queryByTestId("planning-setup")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryPlanningPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID),
    );
    expect(screen.getByTestId("planning-settings")).toBeInTheDocument();
  });

  it("opens the phase automation view from the header icon", async () => {
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-refund-implementer", name: "Implement" }] });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-0-automation-rows-row-step-0-app-refund-implementer"));

    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    expect(screen.getByTestId("merge-confidence-config")).toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-view")).not.toBeInTheDocument();
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
      factoryPRFeedbackPath(
        "org-1",
        PRIMARY_FACTORY_ROUTE_SEGMENT,
        REFUND_LINE_PLAN_ID,
        undefined,
        "handler-discussion",
      ),
    );

    unmount();
    renderLinesBoard();
    await user.click(screen.getByTestId("lines-done-automation-rows-row-closure-app-pr-closure"));
    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryColumnAutomationViewPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID, "app-pr-closure"),
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
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT.toLowerCase()}/lines/${REFUND_LINE_PLAN_ID}?prFeedback=1&prFeedbackHandler=handler-checks`,
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
      factoryPRFeedbackSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID, "checks"),
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
      factoryPRFeedbackSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID, "comments"),
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
    expect(screen.queryByTestId("workspace-next-steps-progress")).not.toBeInTheDocument();
    expect(screen.getByTestId("workspace-next-steps")).not.toHaveTextContent("2/4");
    expect(screen.getByTestId("workspace-next-steps")).not.toHaveTextContent("3/4");
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
    expect(screen.queryByTestId("workspace-next-steps-progress")).not.toBeInTheDocument();
    expect(screen.getByTestId("workspace-next-steps")).not.toHaveTextContent("2/4");
    expect(screen.getByTestId("workspace-next-steps")).not.toHaveTextContent("3/4");
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
    expect(screen.queryByTestId("workspace-next-steps-progress")).not.toBeInTheDocument();
    expect(restore).not.toHaveTextContent("2/4");
    expect(restore).not.toHaveTextContent("3/4");
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
    expect(screen.queryByTestId("workspace-next-steps-progress")).not.toBeInTheDocument();
    expect(screen.getByTestId("workspace-next-steps-restore")).not.toHaveTextContent("2/4");
    expect(screen.getByTestId("workspace-next-steps-restore")).not.toHaveTextContent("3/4");
    expect(screen.getByTestId("workspace-next-steps-restore")).toHaveTextContent("Configure status checks");
    unmount();

    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    expect(screen.queryByTestId("workspace-next-steps")).not.toBeInTheDocument();
    expect(screen.queryByTestId("workspace-next-steps-progress")).not.toBeInTheDocument();
    expect(screen.getByTestId("workspace-next-steps-restore")).not.toHaveTextContent("2/4");
    expect(screen.getByTestId("workspace-next-steps-restore")).not.toHaveTextContent("3/4");
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
      factoryPRFeedbackSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID, "comments"),
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
      factoryPRFeedbackSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID, "checks"),
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

  it("shows Add automation on Verify when custom automations are off", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-verify-menu"));
    expect(screen.getByRole("menuitem", { name: "Add automation" })).toBeInTheDocument();
    await user.keyboard("{Escape}");

    await user.click(screen.getByTestId("lines-done-menu"));
    expect(screen.queryByRole("menuitem", { name: "Add automation" })).not.toBeInTheDocument();
  });

  it("opens the Verify catalog when custom automations are off", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    expect(screen.queryByTestId("lines-verify-add-pr-feedback")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("lines-verify-menu"));
    await user.click(screen.getByTestId("lines-verify-menu-add-automation"));

    expect(screen.getByTestId("add-column-automation-picker")).toBeInTheDocument();
    expect(screen.getByTestId("add-column-automation-template-discussion")).toBeInTheDocument();
    expect(screen.getByTestId("add-column-automation-template-checks")).toBeInTheDocument();
    expect(screen.getByTestId("add-column-automation-template-risk-score")).toBeInTheDocument();
    expect(screen.getByTestId("add-column-automation-template-visual-evidence")).toBeInTheDocument();
    expect(screen.queryByTestId("add-column-automation-template-custom")).not.toBeInTheDocument();
  });

  it("opens the risk score setup page from the Verify catalog", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-verify-menu"));
    await user.click(screen.getByTestId("lines-verify-menu-add-automation"));
    await user.click(screen.getByTestId("add-column-automation-template-risk-score"));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryRiskScoreSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID),
    );
  });

  it("opens custom Verify setup when only custom automation remains", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    useFactoryPRFeedbackHandlers.mockReturnValue({
      data: [
        { id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION", healthy: true },
        { id: "handler-checks", source: "SOURCE_PULL_REQUEST_CHECKS", healthy: true },
      ],
      isPending: false,
    });
    useFactoryAutomations.mockReturnValue({
      data: [
        { id: "app-risk", name: "Risk score", columnKey: "verify" },
        { id: "app-evidence", name: "Visual Evidence", columnKey: "verify" },
      ],
    });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-verify-menu"));
    await user.click(screen.getByTestId("lines-verify-menu-add-automation"));

    expect(screen.queryByTestId("add-column-automation-picker")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryCustomAutomationSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID),
    );
  });

  it("opens custom Verify setup from the catalog", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-verify-menu"));
    await user.click(screen.getByTestId("lines-verify-menu-add-automation"));
    await user.click(screen.getByTestId("add-column-automation-template-custom-verification"));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryCustomAutomationSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID),
    );
  });

  it("opens reused custom setup with Backlog triggers from Add intake", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const user = userEvent.setup();
    renderLinesBoard();
    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-add-intake"));
    await user.click(screen.getByTestId("add-intake-template-custom-verification"));
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent("automationColumn=backlog");
    await user.click(screen.getByRole("button", { name: "Add trigger" }));
    expect(screen.getByRole("menuitem", { name: /When a task enters Backlog/ })).toBeEnabled();
    expect(screen.queryByRole("menuitem", { name: /Every pull request commit/ })).not.toBeInTheDocument();
  });

  it("opens the created scheduled intake in the board automation view", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_SCHEDULED_INTAKE);
    createFactoryAutomationMutateAsync.mockResolvedValueOnce({ id: "canvas-scheduled", name: "Scheduled intake" });
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-add-intake"));
    await user.click(screen.getByTestId("add-intake-template-scheduled-intake"));
    await user.click(screen.getByRole("button", { name: "Create intake" }));

    await waitFor(() => {
      expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
        factoryColumnAutomationViewPath(
          "org-1",
          PRIMARY_FACTORY_ROUTE_SEGMENT,
          REFUND_LINE_PLAN_ID,
          "canvas-scheduled",
        ),
      );
    });
    expect(screen.getByTestId("column-automation-view")).toBeInTheDocument();
    expect(screen.queryByLabelText("What should the agent look for?")).not.toBeInTheDocument();
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

  it("opens GitHub intake setup from the picker", async () => {
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, { addIntakeControl: true, columnAutomations: false });

    await user.click(screen.getByTestId("line-intake-add"));
    await user.click(screen.getByTestId("add-intake-template-github-issues"));

    expect(screen.getByTestId("github-intake-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryGitHubIntakeSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID),
    );
    expect(createFactoryIntakeMutateAsync).not.toHaveBeenCalled();
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
    expect(screen.getByTestId("add-intake-template-dependabot-alerts")).toBeEnabled();
    expect(screen.getByTestId("add-intake-template-dependabot-alerts")).not.toHaveTextContent(
      ADD_INTAKE_COPY.comingSoon,
    );
    expect(screen.getByTestId("add-intake-template-jira-issues")).toBeEnabled();
    expect(screen.getByTestId("add-intake-template-jira-issues")).not.toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("add-intake-template-sentry-exceptions")).toBeEnabled();
    expect(screen.getByTestId("add-intake-template-sentry-exceptions")).not.toHaveTextContent(
      ADD_INTAKE_COPY.comingSoon,
    );
    expect(screen.getByTestId("add-intake-template-productive-tasks")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("add-intake-template-datadog")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("add-intake-template-linear-issues")).toBeEnabled();
    expect(screen.getByTestId("add-intake-template-linear-issues")).not.toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("add-intake-template-notion")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);

    await user.click(screen.getByTestId("add-intake-template-productive-tasks"));
    await user.click(screen.getByTestId("add-intake-template-datadog"));

    expect(screen.queryByTestId("jira-intake-setup")).not.toBeInTheDocument();
    expect(screen.queryByTestId("productive-intake-setup")).not.toBeInTheDocument();
    expect(createFactoryIntakeMutateAsync).not.toHaveBeenCalled();
  });

  it("opens guided Datadog setup from the overflow menu", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_DATADOG_INTAKE);
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-add-intake"));

    const datadog = screen.getByTestId("add-intake-template-datadog");
    expect(datadog).toBeEnabled();
    expect(datadog).not.toHaveTextContent(ADD_INTAKE_COPY.comingSoon);

    await user.click(datadog);

    expect(screen.getByTestId("datadog-intake-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryDatadogIntakeSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID),
    );
    expect(createFactoryIntakeMutateAsync).not.toHaveBeenCalled();
  });

  it("opens guided Linear setup from the overflow menu", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-add-intake"));

    const linear = screen.getByTestId("add-intake-template-linear-issues");
    expect(linear).toBeEnabled();
    expect(linear).not.toHaveTextContent(ADD_INTAKE_COPY.comingSoon);

    await user.click(linear);

    expect(screen.getByTestId("linear-intake-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryLinearIntakeSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID),
    );
    expect(createFactoryIntakeMutateAsync).not.toHaveBeenCalled();
  });

  it("opens guided Sentry setup from the overflow menu", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-add-intake"));

    expect(screen.getByTestId("add-intake-template-github-issues")).toBeInTheDocument();
    expect(screen.getByTestId("add-intake-template-sentry-exceptions")).toBeInTheDocument();
    expect(screen.getByTestId("add-intake-template-jira-issues")).toBeEnabled();
    expect(screen.getByTestId("add-intake-template-jira-issues")).not.toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("add-intake-template-datadog")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("add-intake-template-linear-issues")).toBeEnabled();
    expect(screen.getByTestId("add-intake-template-notion")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.queryByTestId("add-intake-template-pagerduty-incidents")).not.toBeInTheDocument();
    expect(screen.getByTestId("add-intake-template-productive-tasks")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);

    await user.click(screen.getByTestId("add-intake-template-sentry-exceptions"));

    expect(screen.getByTestId("sentry-intake-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factorySentryIntakeSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID),
    );
    expect(createFactoryIntakeMutateAsync).not.toHaveBeenCalled();
  });

  it("opens guided Dependabot setup from the overflow menu", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-add-intake"));
    await user.click(screen.getByTestId("add-intake-template-dependabot-alerts"));

    expect(screen.getByTestId("dependabot-intake-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryDependabotIntakeSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID),
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
      factoryJiraIntakeSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID),
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
    expect(screen.getByTestId("add-intake-template-jira-issues")).toBeEnabled();
    expect(screen.getByTestId("add-intake-template-jira-issues")).not.toHaveTextContent(ADD_INTAKE_COPY.comingSoon);

    await user.click(productive);

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryProductiveIntakeSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID),
    );
    expect(createFactoryIntakeMutateAsync).not.toHaveBeenCalled();
  });

  it("sends a legacy Jira OAuth return to the setup page", () => {
    renderLinesBoard(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}?jiraIntake=1&jiraIntegrationId=int-new`,
      vi.fn(),
      REFUND_FACTORY,
      LANE_BANNERS,
    );

    expect(screen.getByTestId("jira-intake-setup")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      factoryJiraIntakeSetupPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT, REFUND_LINE_PLAN_ID, {
        integrationId: "int-new",
      }),
    );
  });

  it("shows only declared intakes", () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    renderLinesBoard(
      `/org-1/workspaces/${ACME_ONBOARDING_FACTORY_ROUTE_SEGMENT}/lines/${ACME_ONBOARDING_LINE_ID}`,
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

  it("opens intake run history from the row and shares the selected tab", async () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    await user.click(screen.getByRole("button", { name: `Open ${GITHUB_ISSUES_INTAKE.name} settings` }));
    await user.click(screen.getByRole("tab", { name: "Runs" }));

    expect(screen.getByRole("table", { name: "Runs" })).toBeInTheDocument();
    const location = screen.getByTestId("lines-test-location");
    expect(location).toHaveTextContent(`intakeId=${GITHUB_ISSUES_INTAKE_ID}`);
    expect(location).toHaveTextContent("settings=runs");
    expect(screen.queryByTestId("intake-source-automation")).not.toBeInTheDocument();
  });

  it("opens the settings of the intake whose row was used", async () => {
    useFactoryIntakes.mockReturnValue({
      data: [
        GITHUB_ISSUES_INTAKE,
        {
          id: "intake-triage",
          canvasId: "app-triage",
          name: "Triage issues",
          source: "SOURCE_GITHUB_ISSUES",
          settings: { labels: ["triage"] },
        },
      ],
    });
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, LANE_BANNERS);

    await user.click(screen.getByRole("button", { name: "Open Triage issues settings" }));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent("intakeId=intake-triage");
    expect(within(screen.getByTestId("intake-source-settings")).getByText("triage")).toBeInTheDocument();
  });

  it("does not list intake runs under the GitHub issues row", () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    renderLinesBoard(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`,
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
          ...BOARD_IMPLEMENT_NOTIFY_ORDER,
          pullRequests: [
            {
              id: "pr-114",
              workOrderId: BOARD_IMPLEMENT_NOTIFY_ORDER.id,
              number: "114",
              url: "https://github.com/acme/payments/pull/114",
              title: "Notify on status change after a reopen",
              state: "STATE_CLOSED",
            },
          ],
        },
      ],
    });
    renderLinesBoard();

    const card = screen.getByTestId("work-order-card-wo-board-implement-notify");
    const pill = within(card).getByRole("link", { name: "Closed pull request #114." });
    expect(pill).toHaveTextContent("Closed #114");
    expect(pill).toHaveAttribute("href", "https://github.com/acme/payments/pull/114");
  });
});

describe("LinesPage stage editing", () => {
  beforeEach(async () => {
    await resetLinesBoardMocks();
  });

  const stageShellOrder = () =>
    [...screen.getByTestId("lines-phase-board").children].map((element) => element.getAttribute("data-testid"));

  const enterStageEditing = async (user: ReturnType<typeof userEvent.setup>) => {
    await user.click(screen.getByTestId("lines-board-view-menu"));
    await user.click(screen.getByRole("menuitem", { name: "Edit stages" }));
  };

  it("shows reorder handles only while editing stages", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const user = userEvent.setup();
    renderLinesBoard();

    expect(screen.queryByTestId("lines-stage-reorder-phase-0")).not.toBeInTheDocument();
    await enterStageEditing(user);

    expect(screen.getByTestId("lines-stage-reorder-phase-0")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reorder Backlog" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reorder Done" })).not.toBeInTheDocument();
  });

  it("moves an agent without reordering stages and saves only the confirmed placement", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const user = userEvent.setup();
    renderLinesBoard(undefined, vi.fn(), REFUND_FACTORY, ICON_VIEW);
    await enterStageEditing(user);
    const order = stageShellOrder();
    const source = within(screen.getByTestId("lines-stage-shell-phase-0"));
    const target = screen.getByTestId("lines-stage-shell-phase-1");
    const handle = source.getByRole("button", { name: /^Move / });
    const label = handle.getAttribute("aria-label")!;
    const content = source.getByRole("button", { name: /^Open automation:/ });
    const dataTransfer = { effectAllowed: "none", setData: vi.fn() };
    fireEvent.dragStart(content, { dataTransfer });
    fireEvent.dragEnter(target);
    fireEvent.dragEnd(content);
    expect(screen.queryByRole("button", { name: "Save changes" })).not.toBeInTheDocument();
    fireEvent.dragStart(content, { dataTransfer });
    fireEvent.dragOver(target);
    fireEvent.drop(target);
    expect(stageShellOrder()).toEqual(order);
    expect(source.queryByRole("button", { name: label })).not.toBeInTheDocument();
    expect(within(target).getByRole("button", { name: label })).toBeInTheDocument();
    fireEvent.keyDown(within(target).getByRole("button", { name: label }), { key: "ArrowLeft" });
    expect(source.getByRole("button", { name: label })).toHaveFocus();
    expect(screen.queryByRole("button", { name: "Save changes" })).not.toBeInTheDocument();
    fireEvent.keyDown(source.getByRole("button", { name: label }), { key: "ArrowRight" });
    await user.click(screen.getByRole("button", { name: "Discard" }));
    await user.click(screen.getByRole("button", { name: "Discard changes" }));
    await enterStageEditing(user);
    fireEvent.keyDown(source.getByRole("button", { name: label }), { key: "ArrowRight" });
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
    updateFactoryLineMutateAsync.mockResolvedValueOnce({});
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    await user.click(within(screen.getByRole("alertdialog")).getByRole("button", { name: "Save changes" }));
    expect(updateFactoryLineMutateAsync.mock.calls[0][0].stageConfiguration.automationMoves).toEqual([
      expect.objectContaining({
        type: "agent-step",
        executionKey: "phase-0",
        sourceKey: "phase-0",
        destinationKey: "phase-1",
      }),
    ]);
  });

  it("offers Save only after a change and exits with Done when nothing changed", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const user = userEvent.setup();
    renderLinesBoard();
    await enterStageEditing(user);

    expect(screen.queryByRole("button", { name: "Save changes" })).not.toBeInTheDocument();
    await user.type(screen.getByTestId("lines-stage-reorder-phase-0"), "{ArrowRight}");
    expect(screen.getByRole("button", { name: "Save changes" })).toBeInTheDocument();
    await user.type(screen.getByTestId("lines-stage-reorder-phase-0"), "{ArrowLeft}");

    await user.click(screen.getByRole("button", { name: "Done" }));
    expect(screen.queryByTestId("line-stage-edit-toolbar")).not.toBeInTheDocument();
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
  });

  it("saves the reordered draft once after confirmation", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    updateFactoryLineMutateAsync.mockResolvedValueOnce({});
    const user = userEvent.setup();
    renderLinesBoard();
    await enterStageEditing(user);

    await user.type(screen.getByTestId("lines-stage-reorder-phase-0"), "{ArrowRight}");
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Save changes" }));
    const saveDialog = screen.getByTestId("save-line-stages-dialog");
    expect(within(saveDialog).getByTestId("save-line-stages-summary")).toHaveTextContent("Stage order changes");
    await user.click(within(saveDialog).getByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(updateFactoryLineMutateAsync).toHaveBeenCalledTimes(1));
    const stages = updateFactoryLineMutateAsync.mock.calls[0][0].stageConfiguration.stages;
    expect(stages.map((stage: { key: string }) => stage.key)).toEqual([
      "backlog",
      "phase-1",
      "phase-0",
      "verify",
      "done",
    ]);
    await waitFor(() => expect(screen.queryByTestId("line-stage-edit-toolbar")).not.toBeInTheDocument());
  });

  it("restores the saved order when changes are discarded", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const user = userEvent.setup();
    renderLinesBoard();
    await enterStageEditing(user);

    await user.type(screen.getByTestId("lines-stage-reorder-phase-0"), "{ArrowRight}");
    await user.click(screen.getByRole("button", { name: "Discard" }));
    await user.click(screen.getByRole("button", { name: "Discard changes" }));

    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
    expect(screen.queryByTestId("line-stage-edit-toolbar")).not.toBeInTheDocument();
    expect(stageShellOrder()).toEqual([
      "lines-stage-shell-backlog",
      "lines-stage-shell-phase-0",
      "lines-stage-shell-phase-1",
      "lines-stage-shell-verify",
      "lines-stage-shell-done",
    ]);
  });

  it("leaves a deleted draft stage out of the save", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    updateFactoryLineMutateAsync.mockResolvedValueOnce({});
    const user = userEvent.setup();
    renderLinesBoard();
    await enterStageEditing(user);

    await user.click(screen.getByTestId("lines-stage-delete-phase-1"));
    expect(screen.queryByTestId("lines-stage-shell-phase-1")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    await user.click(
      within(screen.getByTestId("save-line-stages-dialog")).getByRole("button", { name: "Save changes" }),
    );

    await waitFor(() => expect(updateFactoryLineMutateAsync).toHaveBeenCalledTimes(1));
    const stages = updateFactoryLineMutateAsync.mock.calls[0][0].stageConfiguration.stages;
    expect(stages.map((stage: { key: string }) => stage.key)).toEqual(["backlog", "phase-0", "verify", "done"]);
  });

  it("deletes a stage from the column menu only in the draft", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-phase-menu-1"));
    expect(screen.queryByTestId("lines-phase-menu-1-delete-stage")).not.toBeInTheDocument();
    await user.keyboard("{Escape}");

    await enterStageEditing(user);
    await user.click(screen.getByTestId("lines-phase-menu-1"));
    await user.click(screen.getByTestId("lines-phase-menu-1-delete-stage"));

    expect(screen.queryByTestId("lines-stage-shell-phase-1")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save changes" })).toBeInTheDocument();
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
  });

  it("keeps a drag move in the draft when the target emits repeated drag events", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
    const user = userEvent.setup();
    renderLinesBoard();
    await enterStageEditing(user);
    const handle = screen.getByTestId("lines-stage-reorder-phase-0");
    const target = screen.getByTestId("lines-stage-shell-phase-1");
    const dataTransfer = { effectAllowed: "none", setData: vi.fn() };

    fireEvent.dragStart(handle, { dataTransfer });
    fireEvent.dragEnter(target);
    fireEvent.dragEnter(target);
    fireEvent.dragEnd(handle);

    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(stageShellOrder()).toEqual([
      "lines-stage-shell-backlog",
      "lines-stage-shell-phase-1",
      "lines-stage-shell-phase-0",
      "lines-stage-shell-verify",
      "lines-stage-shell-done",
    ]);
  });
});

describe("LinesPage board editing", () => {
  beforeEach(async () => {
    await resetLinesBoardMocks();
  });

  it("shows the workspace name and does not rename the line", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    const title = screen.getByTestId("workspace-page-header-title");
    expect(title).toHaveTextContent("Semaphore");
    expect(title).not.toHaveTextContent("Plan and Implement");
    await user.click(title);
    expect(screen.queryByTestId("lines-board-title-input")).not.toBeInTheDocument();
    expect(updateFactoryLineMutateAsync).not.toHaveBeenCalled();
  });

  it("shows an emoji workspace name as stored", () => {
    renderLinesBoard(undefined, vi.fn(), { ...REFUND_FACTORY, name: "SuperPlane Prod 🚀" });

    expect(screen.getByTestId("workspace-page-header-title")).toHaveTextContent("SuperPlane Prod 🚀");
  });

  it("shows Workspace when the workspace name is empty", () => {
    renderLinesBoard(undefined, vi.fn(), { ...REFUND_FACTORY, name: "   " });

    expect(screen.getByTestId("workspace-page-header-title")).toHaveTextContent(/^Workspace$/);
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
    const linePath = `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`;
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
    expect(screen.getByTestId("merge-confidence-config")).toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-view")).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Edit Agent" })).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Edit Automation" })).not.toBeInTheDocument();
  });

  it("saves supported Implement settings after the user confirms", async () => {
    useFactoryAutomations.mockReturnValue({ data: [{ id: "app-refund-implementer", name: "Implement" }] });
    renderLinesBoard();

    fireEvent.click(screen.getByTestId("lines-phase-0-automation-rows-row-step-0-app-refund-implementer"));
    const dialog = await screen.findByTestId("merge-confidence-config");
    const concurrency = within(dialog).getByRole("spinbutton", { name: "Maximum simultaneous runs" });
    fireEvent.change(concurrency, { target: { value: "3" } });
    expect(updateCanvasVersionMutateAsync).not.toHaveBeenCalled();
    expect(commitCanvasStagingMutateAsync).not.toHaveBeenCalled();

    await act(async () => {
      fireEvent.click(within(dialog).getByTestId("implement-settings-save"));
    });

    expect(commitCanvasStagingMutateAsync).toHaveBeenCalledWith("Update step");
    const saved = updateCanvasVersionMutateAsync.mock.calls.at(-1)?.[0] as { canvasYaml?: string };
    const savedAgent = dematerializeCanvasSpec(saved.canvasYaml ?? "")?.nodes?.find(
      (node) => node.id === "implementation-agent-no-issue",
    );
    expect(savedAgent?.concurrency?.max).toBe(3);
    expect(savedAgent?.configuration?.steps).toEqual(implementerCanvas.spec.nodes[1].configuration.steps);
    expect(screen.queryByTestId("merge-confidence-config")).not.toBeInTheDocument();
  }, 15000);

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
          path={`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`}
          factory={REFUND_FACTORY}
        />,
      );
      await Promise.resolve();
      updateLineIsPending.value = false;
      view.rerender(
        <LinesBoardSpecHarness
          path={`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`}
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
    const scopeMy = within(actions).getByTestId("work-orders-scope-my");
    const scopeUnassigned = within(actions).getByTestId("work-orders-scope-unassigned");
    expect(scopeMy).toHaveTextContent("My");
    expect(scopeUnassigned).toHaveTextContent("Unassigned");
    expect(scopeAll.compareDocumentPosition(scopeMy) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(scopeMy.compareDocumentPosition(scopeUnassigned) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
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

  it("filters Rejected on the board without opening a dialog", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    await user.hover(screen.getByTestId("work-orders-filter-statuses"));
    fireEvent.click(await screen.findByTestId("work-orders-filter-statuses-rejected"));

    expect(screen.queryByTestId("work-order-closed-status-dialog")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-orders-filter-chips")).toHaveTextContent("Rejected");
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

  it("keeps Unassigned pressed and requests only tasks with no owner", async () => {
    const user = userEvent.setup();
    const closedUnassigned: FactoriesWorkOrder = {
      id: "wo-closed-unassigned",
      title: "Close stale refund draft",
      state: "STATE_CLOSED",
      result: "RESULT_COMPLETED",
      assignees: [],
      lineDispatches: [{ id: "dispatch-closed", line: { id: REFUND_LINE_PLAN_ID } }],
    };
    useFactoryWorkOrders.mockReturnValue({
      data: [DRAFT_WORK_ORDER, BOARD_IMPLEMENT_NOTIFY_ORDER, closedUnassigned],
    });
    renderLinesBoard();

    await user.click(screen.getByTestId("work-orders-scope-unassigned"));

    const actions = within(screen.getByTestId("lines-detail-header")).getByTestId("workspace-page-header-actions");
    expect(within(actions).getByTestId("work-orders-scope-unassigned")).toHaveAttribute("aria-pressed", "true");
    expect(within(actions).getByTestId("work-orders-scope-all")).toHaveAttribute("aria-pressed", "false");
    expect(within(actions).getByTestId("work-orders-scope-my")).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByText("Draft: rework refund telemetry")).toBeInTheDocument();
    expect(screen.getByText("Close stale refund draft")).toBeInTheDocument();
    expect(screen.queryByText("Notify on status change after a reopen")).not.toBeInTheDocument();

    const options = useFactoryBoardWorkOrders.mock.calls.at(-1)?.[2] as { userId?: string; unassigned?: boolean };
    expect(options.unassigned).toBe(true);
    expect(options.userId).toBeUndefined();
  });

  it("restores a saved Unassigned choice and does not add a selected owner to the request", () => {
    window.localStorage.setItem(`sp:work-orders:scope:${PRIMARY_FACTORY_ID}`, "unassigned");
    window.localStorage.setItem(
      `sp:work-orders:filters:${PRIMARY_FACTORY_ID}`,
      JSON.stringify({ assigneeIds: ["storybook-user"] }),
    );
    renderLinesBoard();

    const actions = within(screen.getByTestId("lines-detail-header")).getByTestId("workspace-page-header-actions");
    expect(within(actions).getByTestId("work-orders-scope-unassigned")).toHaveAttribute("aria-pressed", "true");
    expect(within(actions).getByTestId("work-orders-scope-all")).toHaveAttribute("aria-pressed", "false");

    const options = useFactoryBoardWorkOrders.mock.calls.at(-1)?.[2] as { userId?: string; unassigned?: boolean };
    expect(options.unassigned).toBe(true);
    expect(options.userId).toBeUndefined();
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
      data: [dispatchDraftToImplement(OPEN_WORK_ORDER, OPEN_WORK_ORDER.updatedAt!), BOARD_IMPLEMENT_NOTIFY_ORDER],
    });
    renderLinesBoard();

    expect(screen.getByText("Reconcile duplicate refunds in ledger")).toBeInTheDocument();
    expect(screen.getByText("Notify on status change after a reopen")).toBeInTheDocument();

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    await user.hover(screen.getByTestId("work-orders-filter-sourceIds"));
    fireEvent.click(await screen.findByTestId("work-orders-filter-sourceIds-github-issues"));

    const board = screen.getByTestId("lines-detail-page");
    expect(within(board).getByText("Source is GitHub issues")).toBeInTheDocument();
    expect(within(board).getByText("Reconcile duplicate refunds in ledger")).toBeInTheDocument();
    expect(within(board).queryByText("Notify on status change after a reopen")).not.toBeInTheDocument();
  });

  it("narrows the board when a Review label filter is selected", async () => {
    const user = userEvent.setup();
    useFactoryWorkOrders.mockReturnValue({
      data: [
        {
          ...dispatchDraftToImplement(OPEN_WORK_ORDER, OPEN_WORK_ORDER.updatedAt!),
          pullRequests: [
            {
              id: "pr-review",
              workOrderId: OPEN_WORK_ORDER.id,
              number: "101",
              url: "https://github.com/acme/payments/pull/101",
              title: "Reconcile duplicate refunds in ledger",
              state: "STATE_OPEN",
            },
          ],
        },
        BOARD_IMPLEMENT_NOTIFY_ORDER,
      ],
    });
    renderLinesBoard();

    expect(screen.getByText("Reconcile duplicate refunds in ledger")).toBeInTheDocument();
    expect(screen.getByText("Notify on status change after a reopen")).toBeInTheDocument();

    await user.click(screen.getByTestId("work-orders-filter-trigger"));
    await user.hover(screen.getByTestId("work-orders-filter-labels"));
    fireEvent.click(await screen.findByTestId("work-orders-filter-labels-review"));

    const board = screen.getByTestId("lines-detail-page");
    expect(within(board).getByText("Label is Review")).toBeInTheDocument();
    expect(within(board).getByText("Reconcile duplicate refunds in ledger")).toBeInTheDocument();
    expect(within(board).queryByText("Notify on status change after a reopen")).not.toBeInTheDocument();
  });

  it("narrows the board when the search query changes", async () => {
    const user = userEvent.setup();
    useFactoryWorkOrders.mockReturnValue({
      data: [dispatchDraftToImplement(OPEN_WORK_ORDER, OPEN_WORK_ORDER.updatedAt!), BOARD_IMPLEMENT_NOTIFY_ORDER],
    });
    renderLinesBoard();

    expect(screen.getByText("Reconcile duplicate refunds in ledger")).toBeInTheDocument();
    expect(screen.getByText("Notify on status change after a reopen")).toBeInTheDocument();

    await user.click(screen.getByTestId("work-orders-search-trigger"));
    await user.type(screen.getByTestId("work-orders-search-input"), "reconcile");

    expect(screen.getByText("Reconcile duplicate refunds in ledger")).toBeInTheDocument();
    expect(screen.queryByText("Notify on status change after a reopen")).not.toBeInTheDocument();
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

    expect(screen.getByTestId("lines-backlog-column")).toHaveClass("line-board-color-lime", "line-board-column-vivid");

    await user.click(screen.getByTestId("lines-board-view-menu"));
    await user.hover(screen.getByTestId("lines-board-view-column-color"));
    fireEvent.click(await screen.findByTestId("lines-board-view-no-column-colors"));

    expect(screen.getByTestId("lines-backlog-column")).not.toHaveClass("line-board-color-lime");
    expect(screen.getByTestId("lines-backlog-column")).toHaveClass("bg-muted");

    first.unmount();
    renderLinesBoard();

    expect(screen.getByTestId("lines-backlog-column")).not.toHaveClass("line-board-color-lime");
    expect(screen.getByTestId("lines-backlog-column")).toHaveClass("bg-muted");
  });

  it("lets the header view menu show column colors as borders", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    await user.click(screen.getByTestId("lines-board-view-menu"));
    await user.hover(screen.getByTestId("lines-board-view-column-color"));
    fireEvent.click(await screen.findByTestId("lines-board-view-colored-borders"));

    const backlog = screen.getByTestId("lines-backlog-column");
    expect(backlog).toHaveClass("line-board-color-lime", "line-board-column-bordered", "bg-card");
    expect(backlog).not.toHaveClass("line-board-column-vivid");
  });

  it("lets the header view menu show soft column colors", async () => {
    const user = userEvent.setup();
    renderLinesBoard();

    const backlog = screen.getByTestId("lines-backlog-column");
    expect(backlog).toHaveClass("line-board-color-lime", "line-board-column-vivid");

    await user.click(screen.getByTestId("lines-board-view-menu"));
    await user.hover(screen.getByTestId("lines-board-view-column-color"));
    fireEvent.click(await screen.findByTestId("lines-board-view-dim-column-colors"));

    const softBacklog = screen.getByTestId("lines-backlog-column");
    expect(softBacklog).toHaveClass("line-board-color-lime", "line-board-column-soft");
    expect(softBacklog).not.toHaveClass("line-board-column-vivid");
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
        <LinesBoardSpecHarness
          path={`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`}
        />,
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
      <LinesBoardSpecHarness
        path={`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`}
      />,
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
      <LinesBoardSpecHarness
        path={`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`}
      />,
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
        path={`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`}
        navigateTo={`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_HOTFIX_ID}`}
      />,
    );

    const scroller = screen.getByTestId("lines-backlog-column-scroll");
    scroller.scrollTop = 1760;
    fireEvent.scroll(scroller);
    expect(scroller.scrollTop).toBe(1760);

    await user.click(screen.getByTestId("lines-test-navigate"));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_HOTFIX_ID}`,
    );
    expect(screen.getByTestId("lines-backlog-column-scroll").scrollTop).toBe(0);

    await user.click(screen.getByTestId("lines-test-back"));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`,
    );
    expect(screen.getByTestId("lines-backlog-column-scroll").scrollTop).toBe(1760);
  });
});
