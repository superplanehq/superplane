import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as FactoryData from "@/hooks/useFactoryData";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

import type { PlanningSessionPayload } from "../planningSessionView";

const factoryPlanning = { current: { enabled: true, clarity: true, confidence: true } };
const taskConsoleEnabled = { current: true };
const { closeMutateAsync } = vi.hoisted(() => ({
  closeMutateAsync: vi.fn(),
}));
const mergeability = {
  current: {
    canMerge: true,
    allowedMethods: ["MERGE_METHOD_SQUASH", "MERGE_METHOD_REBASE"],
    headSha: "abc123",
  } as {
    canMerge: boolean;
    blockedReason?: string;
    message?: string;
    allowedMethods: string[];
    headSha: string;
  },
};
const mergeMutate = vi.fn();

vi.mock("@/hooks/useFactoryData", () => {
  const actual = unmockedSrc<typeof FactoryData>("hooks/useFactoryData");
  return {
    ...actual,
    useFactory: () => ({
      data: { id: "factory-1", planning: factoryPlanning.current },
      isPending: false,
    }),
    useCloseWorkOrder: () => ({ mutateAsync: closeMutateAsync, isPending: false }),
  };
});

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

vi.mock("@/hooks/useFactoryPullRequestMerge", () => ({
  useFactoryPullRequestMergeability: () => ({ data: mergeability.current }),
  useMergeFactoryPullRequest: () => ({ mutate: mergeMutate, isPending: false }),
}));

const useLiveLogStreamMock = vi.fn();
const findPlanningSessionMock = vi.fn<(...args: unknown[]) => Promise<PlanningSessionPayload | null>>(async () => null);

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: () => taskConsoleEnabled.current,
    enabledExperimentalFeatures: [],
    isLoading: false,
  }),
}));

vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
}));

vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({
  useLiveLogStream: (...args: unknown[]) => useLiveLogStreamMock(...args),
}));

vi.mock("../planningSessionClient", () => ({
  findPlanningSessionByWorkOrder: (...args: unknown[]) => findPlanningSessionMock(...args),
  sendPlanningSessionMessage: vi.fn(),
  answerPlanningSessionSurvey: vi.fn(),
}));

import { showSuccessToast } from "@/lib/toast";

import {
  DRAFT_WORK_ORDER,
  FACTORIES_ORGANIZATION_ID,
  FAILED_WORK_ORDER,
  INGEST_DRAFT_WORK_ORDER,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  RUNNING_WORK_ORDER,
} from "../../__fixtures__/factoryPageResponses";
import {
  BOARD_DONE_REJECTED_ORDER,
  BOARD_IMPLEMENT_FAILED_ORDER,
  BOARD_IMPLEMENT_NOTIFY_ORDER,
} from "../../__fixtures__/lineMetricsBoardOrders";
import {
  LINE_BOARD_DONE_RECEIPTS_ORDER,
  LINE_BOARD_VERIFY_ENUM_ORDER,
} from "../../__fixtures__/lineMetricsFactoriesFixture";
import { OPEN_WORK_ORDER_CHECKS, VERIFY_STEP_CHECKS } from "../../__fixtures__/workOrderCheckFixtures";
import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { SPEC_ARTIFACT_NAME } from "../../lib/intentDocument";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import { idleLiveLogStream } from "./PhaseLogCard.testHelpers";
import { LiveHeaderSpendProvider } from "./liveHeaderSpendContext";
import { AutomationsConsoleVariant } from "./redesign/AutomationsConsoleVariant";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { buildSplitRunFooter } from "./splitRunFooter";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder, type SplitRunFixture } from "./splitRunMocks";
import { WORK_ORDER_FULL_PAGE_STORAGE_KEY } from "./workOrderFullPagePreference";

function withPublishedSpec(fixture: SplitRunFixture): SplitRunFixture {
  const spec = {
    id: "art-spec-test",
    type: "TYPE_MARKDOWN" as const,
    data: {
      name: SPEC_ARTIFACT_NAME,
      title: SPEC_ARTIFACT_NAME,
      body: "# Spec\n\n## Executive summary\n\nA published plan.\n",
    },
  };
  const [first, ...rest] = fixture.phases;
  if (!first) {
    return fixture;
  }
  return { ...fixture, phases: [{ ...first, artifacts: [...first.artifacts, spec] }, ...rest] };
}

function fixtureWithReviewPullRequest(state: "STATE_OPEN" | "STATE_MERGED"): SplitRunFixture {
  const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER);
  const [first, ...rest] = fixture.phases;
  if (!first) {
    return fixture;
  }
  return {
    ...fixture,
    phases: [
      {
        ...first,
        stream: [
          ...first.stream,
          {
            id: "pr-review-line",
            at: "now",
            componentName: "Pull request",
            status: "passed",
            pullRequest: {
              id: "pr-open",
              provider: "PROVIDER_GITHUB",
              url: "https://github.com/superplanehq/superplane/pull/6812",
              number: "6812",
              state,
            },
          },
        ],
      },
      ...rest,
    ],
  };
}

function renderPopup(props: ComponentProps<typeof WorkOrderSplitRunPopup>) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const ui = (next: ComponentProps<typeof WorkOrderSplitRunPopup>) => (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup {...next} />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(ui(props));
  return {
    ...view,
    rerenderPopup: (next: ComponentProps<typeof WorkOrderSplitRunPopup> = props) => view.rerender(ui(next)),
  };
}

function renderSplitRun() {
  return renderPopup({ fixture: SPLIT_RUN_RUNNING });
}

function liveUsageTelemetry(inputTokens: number, totalCostUsd: number) {
  return {
    num_turns: 1,
    usage: {
      input_tokens: inputTokens,
      output_tokens: 0,
      cache_read_input_tokens: 0,
      cache_creation_input_tokens: 0,
      reasoning_tokens: 0,
      total_cost_usd: totalCostUsd,
    },
    tool_counts: { bash: 1 },
    turns: [
      {
        turn: 1,
        usage: {
          input_tokens: inputTokens,
          output_tokens: 0,
          cache_read_input_tokens: 0,
          cache_creation_input_tokens: 0,
          reasoning_tokens: 0,
        },
        tools: [{ kind: "bash", text: "git status" }],
      },
    ],
  };
}

function runningImplementWithZeroSavedSpend(): SplitRunFixture {
  const fixture = splitRunFixtureForWorkOrder({
    ...RUNNING_WORK_ORDER,
    totalTokens: "0",
    totalCostCents: "0",
    usageByModel: [],
    usageByMachineType: [],
  });
  return {
    ...fixture,
    costUsd: "$0.00",
    tokensLabel: "0 tokens",
    usageByModel: [],
    usageByMachineType: [],
    phases: fixture.phases.map((phase) =>
      phase.id.startsWith("implement")
        ? {
            ...phase,
            canvasKey: null,
            costCents: "0",
            totalTokens: "0",
            appId: "canvas-1",
            stream: [
              {
                id: "impl-agent",
                nodeId: "impl-agent",
                at: "12:00",
                componentName: "Agent",
                component: "runnerClaudeCode",
                executionId: "exec-1",
                status: "running" as const,
              },
            ],
          }
        : phase,
    ),
  };
}

describe("WorkOrderSplitRunPopup", () => {
  beforeEach(() => {
    window.localStorage.clear();
    factoryPlanning.current = { enabled: true, clarity: true, confidence: true };
    taskConsoleEnabled.current = true;
    useLiveLogStreamMock.mockReset();
    useLiveLogStreamMock.mockReturnValue(idleLiveLogStream(vi.fn()));
    findPlanningSessionMock.mockReset();
    findPlanningSessionMock.mockResolvedValue(null);
    closeMutateAsync.mockReset().mockResolvedValue({});
    vi.mocked(showSuccessToast).mockReset();
    mergeMutate.mockReset();
    mergeability.current = {
      canMerge: true,
      allowedMethods: ["MERGE_METHOD_SQUASH", "MERGE_METHOD_REBASE"],
      headSha: "abc123",
    };
  });

  it("does not put an expand control on the Log heading", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
      orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number,
      fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER),
    });

    expect(screen.queryByTestId("split-run-log-expand")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Open automation run" })).not.toBeInTheDocument();
  });

  it("shows the console without tabs when pull request activity exists", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
        prFeedbackRuns: [
          {
            canvasId: "canvas-fb",
            handlerName: "Address PR feedback",
            pullRequestNumber: "12",
            pullRequest: {
              id: "pr-12",
              number: "12",
              title: "feat: add endpoint to re-shuffle an existing deck",
              url: "https://github.com/example/repo/pull/12",
              state: "STATE_OPEN",
            },
            run: {
              id: "run-fb",
              canvasId: "canvas-fb",
              state: "STATE_STARTED",
              result: "RESULT_UNKNOWN",
              createdAt: "2026-08-26T11:00:00Z",
            },
          },
        ],
      }),
    });

    expect(screen.queryByRole("tablist", { name: "Task views" })).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-task-automations")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-pull-request-activity")).not.toBeInTheDocument();
  });

  it("keeps a queued activity title and uses the waiting clock", async () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
        prFeedbackRuns: [
          {
            canvasId: "canvas-fb",
            title:
              "[@lucaspin](https://github.com/lucaspin) left a [review](https://github.com/acme/app/pull/12#pullrequestreview-1)",
            description: "Read the requested changes.",
            waitingForAccess: true,
            pullRequest: {
              id: "pr-12",
              number: "12",
              title: "feat: add endpoint to re-shuffle an existing deck",
              url: "https://github.com/example/repo/pull/12",
              state: "STATE_OPEN",
            },
            run: {
              id: "run-queued",
              canvasId: "canvas-fb",
              state: "STATE_STARTED",
              result: "RESULT_UNKNOWN",
              createdAt: "2026-08-26T11:00:00Z",
            },
          },
        ],
      }),
    });

    expect(screen.queryByRole("tablist", { name: "Task views" })).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-pr-feedback-run-queued")).not.toBeInTheDocument();
  });

  it("shows pull request activity as a flat timestamp-first timeline", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
        prFeedbackRuns: [
          {
            canvasId: "canvas-fb",
            pullRequest: {
              id: "pr-12",
              number: "12",
              title: "feat: add endpoint to re-shuffle an existing deck",
              url: "https://github.com/example/repo/pull/12",
              state: "STATE_OPEN",
            },
            title:
              "[@lucaspin](https://github.com/lucaspin) left a [comment](https://github.com/acme/app/pull/12#issuecomment-1)",
            description: "Read the [requested changes](https://example.com/review).",
            run: {
              id: "run-comment",
              canvasId: "canvas-fb",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T11:00:00Z",
            },
          },
          {
            canvasId: "canvas-checks",
            pullRequest: {
              id: "pr-12",
              number: "12",
              title: "feat: add endpoint to re-shuffle an existing deck",
              url: "https://github.com/example/repo/pull/12",
              state: "STATE_OPEN",
            },
            revision: { id: "revision-a", sha: "a82fd91" },
            title: "Wait for checks",
            run: {
              id: "run-checks",
              canvasId: "canvas-checks",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T10:00:00Z",
            },
          },
          {
            canvasId: "canvas-fb",
            pullRequest: {
              id: "pr-12",
              number: "12",
              title: "feat: add endpoint to re-shuffle an existing deck",
              url: "https://github.com/example/repo/pull/12",
              state: "STATE_OPEN",
            },
            revision: { id: "revision-b", sha: "d8b80c2" },
            title: "Address new review comment",
            run: {
              id: "run-new-revision",
              canvasId: "canvas-fb",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T12:00:00Z",
            },
          },
        ],
      }),
    });

    expect(screen.queryByRole("tablist", { name: "Task views" })).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-pull-request-timeline-0")).not.toBeInTheDocument();
  });

  it("shows one Address PR feedback card for each comment run", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
        prFeedbackRuns: [
          {
            canvasId: "canvas-comment-1",
            handlerName: "Address PR feedback",
            title: "Read the requested changes",
            pullRequestNumber: "12",
            run: {
              id: "run-comment-1",
              canvasId: "canvas-comment-1",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T11:00:00Z",
            },
          },
          {
            canvasId: "canvas-comment-2",
            handlerName: "Address PR feedback",
            title: "Address new review comment",
            pullRequestNumber: "12",
            run: {
              id: "run-comment-2",
              canvasId: "canvas-comment-2",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T12:00:00Z",
            },
          },
        ],
      }),
    });

    const verify = screen.getByTestId("redesign-console-column-verify");
    expect(within(verify).getAllByText("Address PR feedback")).toHaveLength(1);

    if (!within(verify).queryByRole("button", { name: "Toggle Address new review comment" })) {
      await user.click(within(verify).getByRole("button", { name: "Toggle Address PR feedback details" }));
    }
    // The runs list is the card's only page, so it renders without a tab bar.
    expect(within(verify).queryByRole("tab")).not.toBeInTheDocument();
    expect(within(verify).getByRole("button", { name: "Toggle Address new review comment" })).toBeInTheDocument();
    expect(within(verify).getByRole("button", { name: "Toggle Read the requested changes" })).toBeInTheDocument();
    expect(within(verify).queryByRole("button", { name: /View \d+ runs/ })).not.toBeInTheDocument();
    const header = within(verify).getByTestId(/^redesign-console-card-header-/);
    expect(within(header).getByText("2 agent runs")).toBeInTheDocument();
    expect(within(header).queryByRole("button", { name: /agent run/ })).not.toBeInTheDocument();
  });

  it("shows agent run, artifact, and check counts as badges on the card", () => {
    renderSplitRun();

    const implement = screen
      .getByRole("button", { name: "Toggle Implementation details" })
      .closest("[data-testid^='redesign-console-automation-']") as HTMLElement;
    const header = within(implement).getByTestId(/^redesign-console-card-header-/);

    expect(within(header).getByText("1 agent run")).toBeInTheDocument();
    expect(within(header).getByText(/\d+ artifacts?/)).toBeInTheDocument();
    expect(within(header).queryByRole("button", { name: /agent run|artifact|check/ })).not.toBeInTheDocument();
    expect(within(header).queryByRole("button", { name: "Full log" })).not.toBeInTheDocument();
  });

  it("shows the run console on the Automations tab", () => {
    renderSplitRun();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-log-scroll")).not.toBeInTheDocument();
  });

  it("moves owner and spend off the header into the summary panel", () => {
    renderSplitRun();
    expect(screen.queryByTestId("popup-edit-owner")).not.toBeInTheDocument();
    expect(screen.queryByTestId("popup-owner-time-cost")).not.toBeInTheDocument();
    expect(screen.queryByRole("tablist", { name: "Task views" })).not.toBeInTheDocument();
    const panel = screen.getByTestId("redesign-console-summary");
    expect(panel).toHaveTextContent("$0.73");
    expect(panel).toHaveTextContent("2.7k tokens");
    expect(within(panel).getByTestId("split-run-source")).toHaveTextContent("GitHub issues");
    expect(within(panel).getByTestId("split-run-source-ticket")).toHaveTextContent("acme/payments-service#103");
  });

  it("does not list attached files in the console summary", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <LiveHeaderSpendProvider>
                <AutomationsConsoleVariant
                  fixture={SPLIT_RUN_RUNNING}
                  source={SPLIT_RUN_RUNNING.source}
                  files={[{ id: "file-bug", filename: "bug.png", downloadUrl: "https://cdn.example/bug.png" }]}
                />
              </LiveHeaderSpendProvider>
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const panel = screen.getByTestId("redesign-console-summary");
    expect(within(panel).queryByText("Files")).not.toBeInTheDocument();
    expect(within(panel).queryByText("bug.png")).not.toBeInTheDocument();
  });

  it("keeps the model breakdown behind the summary-panel spend hover card", async () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER) });

    expect(screen.queryByTestId("popup-owner-time-cost")).not.toBeInTheDocument();
    const panel = screen.getByTestId("redesign-console-summary");
    expect(panel).toHaveTextContent("$0.73");
    expect(panel).not.toHaveTextContent("claude-sonnet-4-6");

    fireEvent.focus(within(panel).getByTestId("popup-spend-breakdown-trigger"));

    const breakdown = await screen.findByTestId("popup-spend-breakdown");
    expect(breakdown).toHaveTextContent("sonnet 4-6");
    expect(breakdown).toHaveTextContent("Machine time");
    expect(breakdown).toHaveTextContent("$0.28");
  });

  it("shows the run console while a step runs", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      fixture: runningImplementWithZeroSavedSpend(),
    });

    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-summary")).toHaveTextContent("$0.00");
  });

  it("shows live run spend in the console summary while a step runs", async () => {
    const live = liveUsageTelemetry(2100, 0.45);
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      telemetry: live,
      usageSeries: [{ name: "Prompt", telemetry: live }],
    });

    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      fixture: runningImplementWithZeroSavedSpend(),
    });

    await waitFor(() => {
      expect(screen.getByTestId("redesign-console-summary")).toHaveTextContent("$0.45");
    });
    expect(screen.getByTestId("redesign-console-summary")).toHaveTextContent("2.1k tokens");
  });

  it("shows planning tokens on a draft header while the agent runs and waits", async () => {
    const earlier = liveUsageTelemetry(1500, 0.2);
    const current = liveUsageTelemetry(700, 0.05);
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      telemetry: current,
      usageSeries: [
        { name: "First prompt", telemetry: earlier },
        { name: "Second prompt", telemetry: current },
      ],
    });
    findPlanningSessionMock.mockResolvedValue({
      id: "session-plan",
      state: "active",
      canvasId: "canvas-plan",
      executionId: "exec-plan",
    });

    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: DRAFT_WORK_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER),
    });

    await waitFor(() => {
      expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("$0.25 · 2.2k tokens");
    });
    expect(screen.queryByRole("tab", { name: "Automations" })).not.toBeInTheDocument();
    expect(useLiveLogStreamMock).toHaveBeenCalledWith("exec-plan", true, null, null, {
      organizationId: FACTORIES_ORGANIZATION_ID,
      canvasId: "canvas-plan",
    });
  });

  it("keeps planning spend on a draft header while the agent waits", async () => {
    const live = liveUsageTelemetry(2100, 0.45);
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      telemetry: live,
      usageSeries: [{ name: "Prompt", telemetry: live }],
    });
    findPlanningSessionMock.mockResolvedValue({
      id: "session-plan",
      state: "active",
      canvasId: "canvas-plan",
      executionId: "exec-plan",
      waitState: "pending",
    });

    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: DRAFT_WORK_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER),
    });

    await waitFor(() => {
      expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("$0.45 · 2.1k tokens");
    });
    expect(useLiveLogStreamMock).toHaveBeenCalledWith("exec-plan", true, null, null, {
      organizationId: FACTORIES_ORGANIZATION_ID,
      canvasId: "canvas-plan",
    });
  });

  it("leaves a sub-cent planning turn at $0.00 and still shows tokens", async () => {
    const live = liveUsageTelemetry(40, 0.004);
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      telemetry: live,
      usageSeries: [{ name: "Prompt", telemetry: live }],
    });
    findPlanningSessionMock.mockResolvedValue({
      id: "session-plan",
      state: "active",
      canvasId: "canvas-plan",
      executionId: "exec-plan",
      waitState: "pending",
    });

    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: DRAFT_WORK_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER),
    });

    await waitFor(() => {
      expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("$0.00 · 40 tokens");
    });
  });

  it("does not stream planning spend after the machine has stopped", async () => {
    const live = liveUsageTelemetry(2100, 0.45);
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      telemetry: live,
      usageSeries: [{ name: "Prompt", telemetry: live }],
    });
    findPlanningSessionMock.mockResolvedValue({
      id: "session-plan",
      state: "ended",
      canvasId: "canvas-plan",
      executionId: "exec-plan",
    });

    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: DRAFT_WORK_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER),
    });

    await waitFor(() => {
      expect(screen.getByTestId("popup-owner-time-cost")).toBeInTheDocument();
    });
    expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("$0.00");
    expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("0 tokens");
    expect(useLiveLogStreamMock.mock.calls.some((call) => call[0] === "exec-plan")).toBe(false);
  });

  it("does not add saved planning spend on top of the live log", async () => {
    const live = liveUsageTelemetry(2100, 0.45);
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      telemetry: live,
      usageSeries: [{ name: "Prompt", telemetry: live }],
    });
    findPlanningSessionMock.mockResolvedValue({
      id: "session-plan",
      state: "active",
      canvasId: "canvas-plan",
      executionId: "exec-plan",
      waitState: "pending",
    });

    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: DRAFT_WORK_ORDER.id,
      fixture: splitRunFixtureForWorkOrder({
        ...DRAFT_WORK_ORDER,
        totalTokens: "2100",
        totalCostCents: "45",
      }),
    });

    await waitFor(() => {
      expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("$0.45 · 2.1k tokens");
    });
    expect(screen.getByTestId("popup-owner-time-cost")).not.toHaveTextContent("$0.90");
    expect(screen.getByTestId("popup-owner-time-cost")).not.toHaveTextContent("4.2k tokens");
  });

  it("adds a follow-up planning run to saved draft usage", async () => {
    const live = liveUsageTelemetry(1000, 0.1);
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      telemetry: live,
      usageSeries: [{ name: "Prompt", telemetry: live }],
    });
    findPlanningSessionMock.mockResolvedValue({
      id: "session-plan",
      state: "active",
      canvasId: "canvas-plan",
      executionId: "exec-follow-up",
      waitState: "pending",
    });

    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: DRAFT_WORK_ORDER.id,
      fixture: splitRunFixtureForWorkOrder({
        ...DRAFT_WORK_ORDER,
        totalTokens: "2000",
        totalCostCents: "20",
      }),
    });

    await waitFor(() => {
      expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("$0.30 · 3k tokens");
    });
  });

  it("does not round each planning prompt before it adds the cost", async () => {
    const first = liveUsageTelemetry(10, 0.006);
    const second = liveUsageTelemetry(12, 0.006);
    useLiveLogStreamMock.mockReturnValue({
      ...idleLiveLogStream(vi.fn()),
      telemetry: second,
      usageSeries: [
        { name: "First prompt", telemetry: first },
        { name: "Second prompt", telemetry: second },
      ],
    });
    findPlanningSessionMock.mockResolvedValue({
      id: "session-plan",
      state: "active",
      canvasId: "canvas-plan",
      executionId: "exec-plan",
    });

    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: DRAFT_WORK_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER),
    });

    await waitFor(() => {
      expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("$0.01 · 22 tokens");
    });
  });

  it("does not show a model on a draft that has not started", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    const row = screen.getByTestId("popup-owner-time-cost");
    expect(row).toHaveTextContent("$0.00");
    expect(row).toHaveTextContent("0 tokens");
    expect(row).not.toHaveTextContent("claude-sonnet-4-6");
    expect(row).not.toHaveTextContent("grok-4.6");
    expect(screen.queryByTestId("popup-spend-breakdown-trigger")).not.toBeInTheDocument();
    expect(screen.queryByTestId("popup-spend-breakdown")).not.toBeInTheDocument();
  });

  it("shows the console without tabs when no automation is running", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER) });

    expect(screen.queryByRole("tablist", { name: "Task views" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("popup-owner-time-cost")).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-summary")).toBeInTheDocument();
  });

  it("shows the run console for a running line step", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER, { demoArtifacts: false }) });

    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-duration-implement-0")).not.toBeInTheDocument();
  });

  it("does not put an Open task link next to close", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER),
    });

    expect(screen.queryByTestId("split-run-open-work-order")).not.toBeInTheDocument();

    expect(screen.queryByRole("link", { name: "Open task" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
  });

  it("expands next to Close into a full page that leaves the sidebar uncovered", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER),
      fixed: true,
    });

    const expand = screen.getByRole("button", { name: "Open full screen" });
    const close = screen.getByRole("button", { name: "Close" });
    expect(screen.getByRole("heading", { name: "RF-101 Reconcile duplicate refunds in ledger" })).toBeInTheDocument();
    expect(expand.compareDocumentPosition(close) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(expand).toHaveClass("h-6", "w-6", "rounded-full");
    expect(close).toHaveClass("h-6", "w-6", "rounded-full");

    const dialog = screen.getByTestId("work-order-split-run");
    expect(dialog.className).toContain("w-[min(72rem");
    expect(dialog.parentElement).toHaveClass("fixed");
    expect(dialog.parentElement?.className).not.toContain("left-[var(--workspace-navigation-width)]");

    await user.click(expand);

    const fullPage = screen.getByTestId("work-order-split-run");
    expect(fullPage.className).toContain("h-full");
    expect(screen.getByRole("heading", { name: "RF-101 Reconcile duplicate refunds in ledger" })).toBeInTheDocument();
    expect(fullPage.className).toContain("w-full");
    expect(fullPage.className).not.toContain("w-[min(72rem");
    expect(fullPage.parentElement).toHaveClass("fixed");
    expect(fullPage.parentElement?.className).toContain("left-[var(--workspace-navigation-width)]");
    expect(fullPage.parentElement).not.toHaveClass("bg-black/50");
    expect(screen.getByRole("button", { name: "Exit full screen" })).toBeInTheDocument();

    expect(window.localStorage.getItem(WORK_ORDER_FULL_PAGE_STORAGE_KEY)).toBe("1");

    await user.click(screen.getByRole("button", { name: "Exit full screen" }));

    expect(screen.getByTestId("work-order-split-run").className).toContain("w-[min(72rem");
    expect(screen.getByRole("button", { name: "Open full screen" })).toBeInTheDocument();
    expect(window.localStorage.getItem(WORK_ORDER_FULL_PAGE_STORAGE_KEY)).toBe("0");
  });

  it("opens the task modal in full screen from the stored preference", () => {
    window.localStorage.setItem(WORK_ORDER_FULL_PAGE_STORAGE_KEY, "1");
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER),
      fixed: true,
    });

    const dialog = screen.getByTestId("work-order-split-run");
    expect(dialog.className).toContain("h-full");
    expect(dialog.className).toContain("w-full");
    expect(screen.getByRole("button", { name: "Exit full screen" })).toBeInTheDocument();
  });

  it("collapses finished steps and expands the running component stream", () => {
    renderSplitRun();

    const dialog = screen.getByTestId("work-order-split-run");
    expect(dialog.className).toContain("w-[min(72rem");
    expect(within(dialog).getByRole("heading", { name: "Add refund reconciliation test" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("tab", { name: "Task" })).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-log-tab-dot")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("heading", { name: "Automations" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("switch", { name: "Follow" })).not.toBeInTheDocument();
    expect(within(dialog).getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Implement" })).toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-log-scroll")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("region", { name: "Run" })).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();
    expect(screen.queryByText("Factory Lines")).not.toBeInTheDocument();
  });

  it("expands a console card from the whole summary row", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    const backlog = screen.getByTestId("redesign-console-column-backlog");
    const header = within(backlog).getAllByTestId(/^redesign-console-card-header-/)[0];
    const card = header.closest("[data-testid^='redesign-console-automation-']");
    expect(card).not.toBeNull();
    expect(within(card as HTMLElement).queryByTestId("redesign-console-task-description")).not.toBeInTheDocument();

    await user.click(within(header).getByRole("button", { name: "Toggle Ingest details" }));
    // The task text is the description.md document; Artifacts is the only
    // page, so there is no tab bar.
    expect(within(card as HTMLElement).queryByRole("tab")).not.toBeInTheDocument();
    expect(within(card as HTMLElement).getByRole("button", { name: "description.md" })).toBeInTheDocument();
    expect(within(card as HTMLElement).getByRole("button", { name: "Download description.md" })).toBeInTheDocument();
    expect(within(card as HTMLElement).getByText(/^\d+(\.\d+)? (B|KB|MB|GB)$/)).toBeInTheDocument();
    expect(within(card as HTMLElement).getByTestId("redesign-console-task-description")).toBeInTheDocument();
  });

  it("opens produced artifacts on the card's Artifacts page", async () => {
    const user = userEvent.setup();
    renderSplitRun();

    const card = screen
      .getByRole("button", { name: "Toggle Implementation details" })
      .closest("[data-testid^='redesign-console-automation-']") as HTMLElement;
    const logTab = within(card).getByRole("tab", { name: "Agent runs 1" });
    expect(logTab).toHaveAttribute("aria-selected", "true");
    expect(within(card).queryByRole("link", { name: /feature\/refund-retry/ })).not.toBeInTheDocument();

    await user.click(within(card).getByRole("tab", { name: /Artifacts/ }));

    expect(within(card).getByRole("link", { name: "feature/refund-retry" })).toBeInTheDocument();
    expect(within(card).getByRole("link", { name: "Open feature/refund-retry in a new tab" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-implement")).not.toBeInTheDocument();
  });

  it("shows canvas artifacts on the run console", () => {
    renderSplitRun();

    const consoleView = screen.getByTestId("redesign-console-variant");
    expect(within(consoleView).getByRole("heading", { name: "Backlog" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-backlog")).not.toBeInTheDocument();
  });

  it("does not use the old log line highlight on the Automations tab", () => {
    renderSplitRun();

    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-line-onrun-implement")).not.toBeInTheDocument();
  });

  it("puts risk score and code quality on the verify step", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(
        {
          ...OPEN_WORK_ORDER,
          title: "Add refund reason enum to schema",
          lineDispatches: [
            {
              id: "dispatch-verify",
              line: { id: "line-1", name: "plan-and-implement" },
              state: "STATE_ACTIVE",
              stepExecutions: [
                {
                  id: "e-impl",
                  step: "Implement",
                  stepIndex: 0,
                  state: "STATE_FINISHED",
                  result: "RESULT_PASSED",
                },
                {
                  id: "e-verify",
                  step: "Verify",
                  stepIndex: 1,
                  state: "STATE_STARTED",
                  result: "RESULT_UNKNOWN",
                  run: { id: "run-verify", appId: "app-verify" },
                },
              ],
            },
          ],
        },
        { checks: OPEN_WORK_ORDER_CHECKS },
      ),
    });

    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-checks-verify-1")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();

    // The tab shows the check count. The page lists every score.
    await user.click(screen.getByRole("tab", { name: "Checks 2" }));
    expect(screen.getByText(/Moderate risk: retry policy/)).toBeInTheDocument();
  });

  it("pins the pull request review to the waiting implement log", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        ...OPEN_WORK_ORDER,
        title: "Ship idempotent refund retries",
        lineDispatches: [
          {
            id: "dispatch-waiting",
            line: { id: "line-1", name: "plan-and-implement" },
            state: "STATE_FINISHED",
            stepExecutions: [
              {
                id: "e-impl",
                step: "Implement",
                stepIndex: 0,
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
              },
            ],
          },
        ],
      }),
    });

    expect(screen.getByTestId("split-run-review")).toBeInTheDocument();
  });

  it("opens a compact check in the analysis dialog", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(
        {
          ...OPEN_WORK_ORDER,
          title: "Add refund reason enum to schema",
          lineDispatches: [
            {
              id: "dispatch-verify",
              line: { id: "line-1", name: "plan-and-implement" },
              state: "STATE_ACTIVE",
              stepExecutions: [
                {
                  id: "e-verify",
                  step: "Verify",
                  stepIndex: 2,
                  state: "STATE_STARTED",
                  result: "RESULT_UNKNOWN",
                },
              ],
            },
          ],
        },
        { checks: OPEN_WORK_ORDER_CHECKS },
      ),
    });

    await user.click(screen.getByTestId("split-run-check-check-risk-review"));

    expect(screen.getByRole("heading", { name: "Risk score" })).toBeInTheDocument();
    expect(screen.getByText(/Moderate risk: retry policy/)).toBeInTheDocument();
  });

  it("omits the decision footer when logs are complete and the order waits with no note", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        ...OPEN_WORK_ORDER,
        title: "dasdas",
        statusNotes: [],
        assignees: [{ id: "user-1", name: "test test" }],
        lineDispatches: [
          {
            id: "dispatch-wait",
            line: { id: "line-1", name: "plan-and-implement" },
            state: "STATE_FINISHED",
            stepExecutions: [
              {
                id: "e-1",
                step: "dasdasdas",
                stepIndex: 0,
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
              },
            ],
          },
        ],
      }),
    });

    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(screen.queryByText("This task needs attention from test test.")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-attention-note")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
  });

  it("keeps the running log visible when the task has no note or checks", () => {
    renderPopup({ fixture: { ...SPLIT_RUN_RUNNING, waitingNotes: [], checks: [] } });

    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Automations" })).not.toBeInTheDocument();
    expect(screen.queryByRole("switch", { name: "Follow" })).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-log-scroll")).not.toBeInTheDocument();
  });

  it("shows pull request review guidance beside the pull request list", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER) });

    const panel = screen.getByTestId("redesign-console-summary");
    const note = within(panel).getByTestId("split-run-attention-note");
    expect(note).toHaveAttribute("data-variant", "pull-request");
    expect(within(note).getByRole("heading", { name: "The pull request is ready for review" })).toBeInTheDocument();
    expect(within(note).queryByText("Waiting for user review")).not.toBeInTheDocument();
    expect(within(note).queryByRole("list")).not.toBeInTheDocument();
    expect(note).toHaveTextContent("This task closes when the pull request is merged or closed.");
    const heading = within(note).getByRole("heading", { name: "The pull request is ready for review" });
    const reviewLink = within(note).getByRole("link", { name: "Review PR #6812" });
    expect(reviewLink).toHaveAttribute("href", "https://github.com/superplanehq/superplane/pull/6812");
    expect(reviewLink).not.toHaveClass("w-full");
    const closing = within(note).getByText("This task closes when the pull request is merged or closed.");
    expect(closing.parentElement).toBe(heading.parentElement);
    expect(note.querySelector(".lucide-git-pull-request")).toBeNull();
    expect(within(note).queryByText("PR Closure")).not.toBeInTheDocument();
    expect(within(note).queryByText(/ago/)).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: /Update manually/ })).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "More actions" })).not.toBeInTheDocument();

    const moreActions = screen.getByRole("button", { name: "More actions" });
    expect(moreActions.closest("header")).not.toBeNull();
    await user.click(moreActions);
    const menu = await screen.findByRole("menu");
    expect(within(menu).getByRole("menuitem", { name: "Reject" })).toBeInTheDocument();
    expect(within(menu).getByRole("menuitem", { name: "Approve" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Stop and Close" })).not.toBeInTheDocument();
  });

  it("enables merge on the pull request review strip when mergeability is true", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: fixtureWithReviewPullRequest("STATE_OPEN") });

    const note = within(screen.getByTestId("redesign-console-summary")).getByTestId("split-run-attention-note");
    expect(within(note).getByTestId("split-run-merge-button")).toBeEnabled();
    await user.click(within(note).getByTestId("split-run-merge-method"));
    const menu = await screen.findByRole("menu");
    expect(
      within(menu)
        .getAllByRole("menuitem")
        .map((item) => item.textContent),
    ).toEqual(["✓ Squash and merge", "Rebase and merge"]);
    await user.keyboard("{Escape}");
    await user.click(within(note).getByTestId("split-run-merge-button"));
    expect(mergeMutate).toHaveBeenCalledTimes(1);
    expect(mergeMutate.mock.calls[0]?.[0]).toEqual({
      pullRequestId: "pr-open",
      mergeMethod: "MERGE_METHOD_SQUASH",
      expectedHeadSha: "abc123",
    });
  });

  it("disables merge and shows the reason on hover when automation is running", async () => {
    const user = userEvent.setup();
    mergeability.current = {
      canMerge: false,
      blockedReason: "BLOCKED_REASON_ACTIVE_RUN",
      message: "Automation is still running.",
      allowedMethods: ["MERGE_METHOD_SQUASH"],
      headSha: "abc123",
    };
    renderPopup({ fixture: fixtureWithReviewPullRequest("STATE_OPEN") });

    const note = within(screen.getByTestId("redesign-console-summary")).getByTestId("split-run-attention-note");
    expect(within(note).getByTestId("split-run-merge-button")).toBeDisabled();
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();

    await user.hover(within(note).getByTestId("split-run-merge-reason"));
    expect(await screen.findByRole("tooltip")).toHaveTextContent("Automation is still running.");
  });

  it("lists the reviewed pull request once, on the review strip only", () => {
    renderPopup({ fixture: fixtureWithReviewPullRequest("STATE_OPEN") });

    const summary = screen.getByTestId("redesign-console-summary");
    const note = within(summary).getByTestId("split-run-attention-note");
    expect(within(note).getByRole("link", { name: "Review PR #6812" })).toBeInTheDocument();
    expect(within(summary).queryByTestId("redesign-console-pull-requests")).not.toBeInTheDocument();
    // "Waiting" is the status, not a duration.
    expect(within(summary).queryByText("Duration")).not.toBeInTheDocument();
  });

  it("hides merge when the pull request is merged", () => {
    renderPopup({ fixture: fixtureWithReviewPullRequest("STATE_MERGED") });

    const note = within(screen.getByTestId("redesign-console-summary")).getByTestId("split-run-attention-note");
    expect(within(note).queryByTestId("split-run-merge-button")).not.toBeInTheDocument();
    expect(within(note).getByTestId("split-run-pr-merged")).toHaveTextContent("The pull request is merged.");
  });

  it("hides work-order close actions when the user cannot update the task", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER),
      canUpdate: false,
    });

    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stop")).not.toBeInTheDocument();
    expect(screen.queryByTestId("popup-work-order-archive-button")).not.toBeInTheDocument();
  });

  it("offers automation Stop on a live running task", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
      orderId: "wo-running",
      orderNumber: "103",
      fixture: SPLIT_RUN_RUNNING,
    });

    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    // Stop shows on the running card and on the summary strip.
    expect(screen.getAllByRole("button", { name: "Stop" })).toHaveLength(2);
    expect(screen.getByTestId("redesign-console-stop-run")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "View run" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Open run" })).not.toBeInTheDocument();
  });

  it("hides automation Stop when the user cannot update the task", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: "wo-running",
      fixture: SPLIT_RUN_RUNNING,
      canUpdate: false,
    });

    expect(screen.queryByRole("button", { name: "Stop" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
  });

  it("hides automation Rerun when the failed phase has no step index", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: "wo-failed-implement",
      fixture: splitRunFixtureForWorkOrder({
        id: "wo-failed-implement",
        title: "Later step",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              { id: "e-plan", step: "Planning", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
              { id: "e-impl", step: "Implement", state: "STATE_FINISHED", result: "RESULT_FAILED" },
              { id: "e-verify", step: "Verify", stepIndex: 2, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
            ],
          },
        ],
      }),
    });

    expect(screen.queryByTestId(/split-run-phase-rerun-/)).not.toBeInTheDocument();
  });

  it("pins a default failed note and keeps Reopen on the note", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
      orderNumber: BOARD_IMPLEMENT_FAILED_ORDER.number,
      fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER),
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "This task is closed as failed" })).toBeInTheDocument();
    expect(within(note).getByText("Reopen this task to start the line again.")).toBeInTheDocument();
    expect(within(note).queryByRole("link", { name: "Debug" })).not.toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Send to backlog" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Reopen" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Stop and Close" })).not.toBeInTheDocument();
  });

  it("offers Reject and Rerun on a failed open implement", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        id: "wo-2",
        title: "Test task 2",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_FINISHED",
            stepExecutions: [
              { id: "e-plan", step: "Planning", state: "STATE_FINISHED", result: "RESULT_PASSED" },
              {
                id: "e-impl",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
              },
            ],
          },
        ],
      }),
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Rerun" })).toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Rerun step" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Choose how to stop" })).not.toBeInTheDocument();
  });

  it("explains a hosted credit failure and links to billing", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
      fixture: splitRunFixtureForWorkOrder({
        id: "wo-credit",
        title: "Out of credit",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_FINISHED",
            stepExecutions: [
              {
                id: "e-impl",
                step: "Implement",
                stepIndex: 0,
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
                failureReason: "no_hosted_credit",
                run: { id: "run-1", appId: "app-1" },
              },
            ],
          },
        ],
      }),
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "Implement did not pass" })).toBeInTheDocument();
    expect(
      within(note).getByText("This agent run is blocked. The organization has no SuperPlane hosted credit."),
    ).toBeInTheDocument();
    expect(within(note).getByRole("link", { name: "Add credits" })).toHaveAttribute(
      "href",
      factorySettingsSectionPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "organization", "billing"),
    );
    expect(within(note).queryByRole("link", { name: "Debug" })).not.toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Rerun" })).toBeInTheDocument();
  });

  it("explains a hosted credit failure when backlog analysis does not start", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
      fixture: splitRunFixtureForWorkOrder(
        {
          id: "wo-draft-credit",
          title: "test",
          state: "STATE_DRAFT",
          lineDispatches: [],
        },
        {
          analysisRuns: [
            {
              canvasId: "canvas-1",
              workOrderId: "wo-draft-credit",
              run: {
                id: "run-1",
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
                createdAt: "2026-09-30T14:37:29Z",
                executions: [
                  {
                    id: "exec-1",
                    result: "RESULT_FAILED",
                    resultMessage: "This organization has no hosted credit.",
                  },
                ],
              },
            },
          ],
        },
      ),
    });

    const verdict = screen.getByTestId("split-run-intent-verdict");
    expect(verdict).toHaveAttribute("data-tone", "blocked");
    expect(within(verdict).getByText("No credit")).toBeInTheDocument();
    expect(verdict).not.toHaveTextContent("This task is ready to start");
    expect(
      within(verdict).getByText("This agent run is blocked. The organization has no SuperPlane hosted credit.", {
        exact: false,
      }),
    ).toBeInTheDocument();
    expect(within(verdict).getByRole("link", { name: "Add credits" })).toHaveAttribute(
      "href",
      factorySettingsSectionPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "organization", "billing"),
    );
  });

  it("keeps the billing link when a scored draft fails analysis for credit", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
      fixture: splitRunFixtureForWorkOrder(
        {
          id: "wo-draft-credit-scored",
          title: "test",
          state: "STATE_DRAFT",
          lineDispatches: [],
        },
        {
          demoArtifacts: false,
          checks: [
            { id: "clarity", name: "Clarity score", score: 5, maxScore: 5 },
            { id: "confidence", name: "Confidence score", score: 5, maxScore: 5 },
          ],
          analysisRuns: [
            {
              canvasId: "canvas-1",
              workOrderId: "wo-draft-credit-scored",
              run: {
                id: "run-1",
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
                createdAt: "2026-09-30T14:37:29Z",
                executions: [
                  {
                    id: "exec-1",
                    result: "RESULT_FAILED",
                    resultMessage: "This organization has no hosted credit.",
                  },
                ],
              },
            },
          ],
        },
      ),
    });

    const verdict = screen.getByTestId("split-run-intent-verdict");
    expect(verdict).toHaveAttribute("data-tone", "blocked");
    expect(verdict).not.toHaveTextContent("This task is ready to start");
    expect(within(verdict).getByRole("link", { name: "Add credits" })).toHaveAttribute(
      "href",
      factorySettingsSectionPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "organization", "billing"),
    );
  });

  it("offers Reject and Rerun after a person stops the run", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        id: "wo-stopped",
        title: "Stopped job",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_FINISHED",
            stepExecutions: [
              {
                id: "e-impl",
                step: "Implement",
                stepIndex: 0,
                state: "STATE_FINISHED",
                result: "RESULT_CANCELLED",
              },
            ],
          },
        ],
      }),
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "A person stopped this automation" })).toBeInTheDocument();
    expect(
      within(note).getByText("This automation did not finish. This task still needs a decision."),
    ).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Rerun" })).toBeInTheDocument();
    expect(within(note).queryByRole("link", { name: "Debug" })).not.toBeInTheDocument();
    expect(screen.queryByTestId(/split-run-phase-rerun-/)).not.toBeInTheDocument();
  });

  it("names the person who stopped the automation", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(
        {
          id: "wo-stopped-named",
          title: "Stopped job",
          state: "STATE_OPEN",
          lineDispatches: [
            {
              id: "d-1",
              line: { id: "line-1", name: "Software delivery" },
              state: "STATE_FINISHED",
              stepExecutions: [
                {
                  id: "e-impl",
                  step: "Implement",
                  stepIndex: 0,
                  state: "STATE_FINISHED",
                  result: "RESULT_CANCELLED",
                },
              ],
            },
          ],
        },
        { stoppedBy: { id: "user-1", name: "Alex", initials: "A" } },
      ),
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByTestId("work-order-mention")).toHaveTextContent("Alex");
    expect(within(note).getByRole("heading", { name: /stopped this automation/ })).toBeInTheDocument();
  });

  it("names the person who marked the task successful, with avatar", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER, {
        closer: {
          actor: { id: "user-1", name: "Alex", initials: "A", avatarUrl: "https://example.com/alex.png" },
        },
      }),
    });

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).getByTestId("work-order-mention")).toHaveTextContent("Alex");
    expect(within(note).getByTestId("work-order-mention").querySelector("img")).toHaveAttribute(
      "src",
      "https://example.com/alex.png",
    );
    expect(within(note).getByRole("heading", { name: /marked this task as successful/ })).toBeInTheDocument();
  });

  it("keeps Reject and Approve off the header on a running order", () => {
    renderSplitRun();

    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Stop and Close" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stop")).not.toBeInTheDocument();
    expect(screen.queryByTestId("popup-work-order-archive-button")).not.toBeInTheDocument();
  });

  it("shows Archive in the header of a started open popup with Task and Automations", () => {
    taskConsoleEnabled.current = false;
    renderPopup({ fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER) });

    expect(screen.getByRole("tab", { name: "Task" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Automations" })).toBeInTheDocument();
    const archive = screen.getByTestId("popup-work-order-archive-button");
    const share = screen.getByTestId("popup-work-order-copy-link-button");
    expect(archive).toHaveAttribute("aria-label", "Archive");
    expect(archive.compareDocumentPosition(share) & Node.DOCUMENT_POSITION_FOLLOWING).toBeGreaterThan(0);
  });

  it("archives a started open task as rejected and closes the popup", async () => {
    taskConsoleEnabled.current = false;
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: OPEN_WORK_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER),
      onClose,
    });

    await user.click(screen.getByTestId("popup-work-order-archive-button"));

    await waitFor(() => {
      expect(closeMutateAsync).toHaveBeenCalledWith({ orderId: OPEN_WORK_ORDER.id, result: "RESULT_REJECTED" });
    });
    expect(showSuccessToast).toHaveBeenCalledWith("Task archived.");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("hides Archive while an Implement run is pending", () => {
    const fixture = splitRunFixtureForWorkOrder({
      ...OPEN_WORK_ORDER,
      lineDispatches: [
        {
          id: "dispatch-pending-implement",
          line: { id: "line-1", name: "plan-and-implement" },
          state: "STATE_ACTIVE",
          stepExecutions: [
            {
              id: "e-impl",
              step: "Implement",
              stepIndex: 0,
              state: "STATE_PENDING",
              result: "RESULT_UNKNOWN",
            },
          ],
        },
      ],
    });

    expect(fixture.footer.kind).toBe("waiting");
    expect(fixture.footer.status).toBe("running");
    renderPopup({ fixture });

    expect(screen.queryByTestId("popup-work-order-archive-button")).not.toBeInTheDocument();
  });

  it("hides Archive in the header of a closed popup", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER) });

    expect(screen.queryByTestId("popup-work-order-archive-button")).not.toBeInTheDocument();
  });

  it("opens a draft refine view without Automations", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    expect(screen.queryByRole("tab", { name: "Task" })).not.toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Automations" })).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-work-order-tab")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-overview-sidebar")).not.toBeInTheDocument();
    const description = screen.getByTestId("split-run-description");
    expect(description).toHaveClass("justify-end");
    expect(within(description).getByTestId("split-run-source")).toHaveTextContent("Leonardo DiCaprio");
    expect(within(description).queryByText("Created manually")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-log-tab-dot")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    const card = screen.getByTestId("split-run-intent-status-card");
    expect(card).toHaveAttribute("data-slot", "frame");
    expect(screen.queryByTestId("split-run-intent-composer-score-copy")).not.toBeInTheDocument();
    expect(within(card).getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(within(card).getByRole("button", { name: /^Model/ })).toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-plan-updated")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-verdict-analyzing").querySelector(".t-matrix")).not.toBeNull();
    expect(screen.queryByTestId("split-run-intent-plan-chip-analyzing")).not.toBeInTheDocument();
    const archive = screen.getByTestId("popup-work-order-archive-button");
    const share = screen.getByTestId("popup-work-order-copy-link-button");
    expect(archive).toHaveAttribute("aria-label", "Archive");
    expect(archive.compareDocumentPosition(share) & Node.DOCUMENT_POSITION_FOLLOWING).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: "Refine" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-decision-tip")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-result")).toHaveAttribute("data-state", "closed");
    expect(screen.getByTestId("work-order-split-run").className).toContain("w-[min(70rem");
    expect(screen.getByTestId("work-order-split-run").className).toContain(
      "has-[[data-refine-chat-solo]]:w-[min(48rem",
    );
    expect(screen.getByTestId("work-order-split-run").className).toContain(
      "has-[[data-refine-plan-open]]:w-[min(80rem",
    );
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-chat-solo")).toBe(true);
    expect(screen.queryByRole("button", { name: "Plan" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-log-pane")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
  });

  it("hides Refine on a backlog draft", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    expect(screen.queryByRole("button", { name: "Refine" })).toBeNull();
  });

  it("shows the console with a Start note on a Planning-off draft", () => {
    factoryPlanning.current = { enabled: false, clarity: true, confidence: true };
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    expect(screen.queryByRole("tablist", { name: "Task views" })).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-chat")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-session")).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-task-description")).toHaveTextContent("emoji reactions");
    expect(screen.getByTestId("split-run-description-edit")).toBeInTheDocument();
    const note = within(screen.getByTestId("redesign-console-summary")).getByTestId("split-run-attention-note");
    expect(note).toHaveTextContent("This task is ready to start");
    expect(within(note).getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-status-card")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-composer")).not.toBeInTheDocument();
  });

  it("tells a draft is under analysis and keeps Archive in the header", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER, {
        analysisRuns: [
          {
            canvasId: "canvas-backlog",
            workOrderId: DRAFT_WORK_ORDER.id ?? "",
            run: {
              id: "run-analysis",
              canvasId: "canvas-backlog",
              state: "STATE_STARTED",
              createdAt: "2026-08-28T12:00:00Z",
              updatedAt: "2026-08-28T12:00:00Z",
            },
          },
        ],
      }),
    });

    expect(screen.queryByRole("tab", { name: "Task" })).not.toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Automations" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-decision-tip")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();
    const card = screen.getByTestId("split-run-intent-status-card");
    expect(card).toHaveAttribute("data-slot", "frame");
    expect(screen.queryByTestId("split-run-intent-composer-score-copy")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-verdict-analyzing").querySelector(".t-matrix")).not.toBeNull();
    expect(screen.queryByTestId("split-run-intent-plan-chip-analyzing")).not.toBeInTheDocument();
    expect(within(card).getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(within(card).getByRole("button", { name: /^Model/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Refine" })).not.toBeInTheDocument();
    expect(screen.getByTestId("popup-work-order-archive-button")).toHaveAttribute("aria-label", "Archive");
  });

  it("replaces chat with the console after Start", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER) });

    expect(screen.queryByRole("tablist", { name: "Task views" })).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-summary")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-chat")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-request")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();
  });

  it("shows the console after reopen when the started card has a score", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: OPEN_WORK_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, { checks: OPEN_WORK_ORDER_CHECKS }),
    });

    expect(screen.queryByRole("tablist", { name: "Task views" })).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-request")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-overview-checks")).not.toBeInTheDocument();
  });

  it("hides source, artifacts, and pull requests on the description tab", async () => {
    renderPopup({
      fixture: withPublishedSpec(
        splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0], { checks: OPEN_WORK_ORDER_CHECKS }),
      ),
    });

    const tab = screen.getByTestId("split-run-work-order-tab");
    expect(within(tab).queryByTestId("split-run-intent-session")).not.toBeInTheDocument();
    expect(within(tab).getByTestId("split-run-description")).toHaveTextContent(
      "Webhook delivery stops after a transient provider error",
    );
    expect(within(tab).queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    expect(within(tab).queryByTestId("split-run-overview-sidebar")).not.toBeInTheDocument();
    expect(within(tab).queryByRole("heading", { name: "Source" })).not.toBeInTheDocument();
    expect(within(tab).queryByRole("heading", { name: "Artifacts" })).not.toBeInTheDocument();
    expect(within(tab).queryByRole("heading", { name: "Pull requests" })).not.toBeInTheDocument();
    expect(within(tab).getByTestId("split-run-intent-document")).toBeInTheDocument();
    expect(within(tab).queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();
    expect(within(tab).getByTestId("split-run-intent-plan-updated")).toBeInTheDocument();
    expect(within(tab).queryByTestId("split-run-check-comment-wo-review-pay-842-confidence")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-draft-action-group")).getByRole("button", { name: "Start" }),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-attention-note")).not.toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-chat-solo")).toBe(true);
    await userEvent.click(screen.getByRole("button", { name: "Plan" }));
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-chat-solo")).toBe(false);
    expect(screen.getByTestId("split-run-intent-document").hasAttribute("data-refine-plan-open")).toBe(true);
    expect(within(tab).getByTestId("split-run-check-comment-check-risk-review")).not.toHaveAttribute("open");
    expect(within(tab).getByTestId("split-run-check-comment-check-code-coverage")).not.toHaveAttribute("open");
    expect(within(tab).getByText(/Moderate risk: retry policy/)).toBeInTheDocument();
    expect(within(tab).getByText(/The change replaces the retry policy/)).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-checks")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
  });

  it("keeps the verdict and both score slots on the matrix while a review draft is analyzed", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0]) });

    const tab = screen.getByTestId("split-run-work-order-tab");
    expect(within(tab).queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();
    expect(within(tab).getByTestId("split-run-intent-plan-updated")).toBeInTheDocument();
    const verdict = within(tab).getByTestId("split-run-intent-verdict");
    expect(verdict).toHaveAttribute("data-tone", "analyzing");
    expect(within(verdict).getByTestId("split-run-intent-verdict-analyzing").querySelector(".t-matrix")).not.toBeNull();
    const chips = within(tab).getByTestId("split-run-intent-composer-chips");
    expect(within(chips).getByTestId("split-run-intent-composer-score-analyzing")).toBeInTheDocument();
    expect(within(chips).getByTestId("split-run-intent-composer-confidence-analyzing")).toBeInTheDocument();
    expect(within(chips).queryByTestId("split-run-intent-composer-confidence")).not.toBeInTheDocument();
    expect(within(tab).getByTestId("split-run-intent-status-card")).toHaveAttribute("data-slot", "frame");
    expect(within(tab).getByTestId("split-run-intent-document")).toBeInTheDocument();
    expect(within(tab).getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(within(tab).getByRole("button", { name: /^Model/ })).toBeInTheDocument();
  });

  it("opens and closes a check with details and summary", async () => {
    const user = userEvent.setup();
    renderPopup({
      fixture: withPublishedSpec(
        splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0], { checks: OPEN_WORK_ORDER_CHECKS }),
      ),
    });

    await user.click(screen.getByRole("button", { name: "Plan" }));
    const risk = screen.getByTestId("split-run-check-comment-check-risk-review");
    const coverage = screen.getByTestId("split-run-check-comment-check-code-coverage");
    expect(risk).not.toHaveAttribute("open");
    expect(coverage).not.toHaveAttribute("open");

    await user.click(screen.getByTestId("split-run-check-comment-toggle-check-code-coverage"));
    expect(coverage).toHaveAttribute("open");
  });

  it("shows the full check summary without a one-line clamp", async () => {
    renderPopup({
      fixture: withPublishedSpec(
        splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0], { checks: OPEN_WORK_ORDER_CHECKS }),
      ),
    });

    await userEvent.click(screen.getByRole("button", { name: "Plan" }));
    const risk = screen.getByTestId("split-run-check-comment-check-risk-review");
    const summary = within(risk).getByText(/Moderate risk: retry policy changes affect every refund path/);
    expect(summary.tagName).toBe("P");
    expect(summary).not.toHaveClass("truncate");
    expect(summary).toHaveClass("break-words");
  });

  it("keeps description check comments off the console when the task is not a draft", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(
        {
          ...OPEN_WORK_ORDER,
          title: "Add refund reason enum to schema",
          lineDispatches: [
            {
              id: "dispatch-verify",
              line: { id: "line-1", name: "plan-and-implement" },
              state: "STATE_ACTIVE",
              stepExecutions: [
                {
                  id: "e-verify",
                  step: "Verify",
                  stepIndex: 2,
                  state: "STATE_STARTED",
                  result: "RESULT_UNKNOWN",
                },
              ],
            },
          ],
        },
        { checks: OPEN_WORK_ORDER_CHECKS },
      ),
    });

    expect(screen.queryByRole("tablist", { name: "Task views" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-check-comment-check-risk-review")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
  });

  it("hides confidence on a downstream task and keeps other checks", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder(LINE_BOARD_VERIFY_ENUM_ORDER, { checks: VERIFY_STEP_CHECKS }),
    });

    expect(screen.queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();
    expect(screen.queryByText(/fit for an agent on this factory line/)).toBeNull();
    expect(within(screen.getByTestId("redesign-console-summary")).getByText("Risk score")).toBeInTheDocument();
  });

  it("shows the console when a GitHub automation created the draft", () => {
    factoryPlanning.current = { enabled: false, clarity: true, confidence: true };
    renderPopup({
      factoryId: PRIMARY_FACTORY_ID,
      orderId: INGEST_DRAFT_WORK_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(INGEST_DRAFT_WORK_ORDER),
    });

    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-backlog")).not.toBeInTheDocument();
    expect(screen.queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();
  });

  it("shows a successful result in the summary panel", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER) });

    const panel = screen.getByTestId("redesign-console-summary");
    const note = within(panel).getByTestId("split-run-attention-note");
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(within(note).getByRole("heading", { name: "This task succeeded" })).toBeInTheDocument();
    expect(within(note).getByText("The work is done. The result met the goal.")).toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Reopen" })).not.toBeInTheDocument();
    expect(screen.queryByRole("tablist", { name: "Task views" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-request")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-confidence")).not.toBeInTheDocument();

    // The result reads top-down: pull request, then artifacts, then details.
    const pullRequestSection = within(panel).getByTestId("redesign-console-pull-requests");
    expect(within(pullRequestSection).getByRole("link", { name: /#510/ })).toBeInTheDocument();
    const artifactsSection = within(panel).getByTestId("redesign-console-artifacts");
    expect(
      pullRequestSection.compareDocumentPosition(artifactsSection) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    // The pull request supersedes its branch, so the branch is not listed again.
    expect(within(panel).queryByRole("link", { name: /feature\// })).not.toBeInTheDocument();
    expect(within(panel).getByText("Duration")).toBeInTheDocument();
  });

  it("shows a manual source by the owner without repeating the person", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        ...LINE_BOARD_DONE_RECEIPTS_ORDER,
        origin: undefined,
        createdBy: { user: { id: "user-owner-1", name: "Igor Šarčević" } },
        assignees: [],
      }),
    });

    const panel = screen.getByTestId("redesign-console-summary");
    expect(within(panel).getByText("Created manually")).toBeInTheDocument();
    expect(within(panel).queryByTestId("split-run-source")).not.toBeInTheDocument();
    // The avatar title also carries the name; visible text shows it once.
    expect(within(panel).getAllByText("Igor Šarčević", { ignore: "script, style, title" })).toHaveLength(1);
  });

  it("shows an unsuccessful result in the summary panel", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(BOARD_DONE_REJECTED_ORDER) });

    const panel = screen.getByTestId("redesign-console-summary");
    const note = within(panel).getByTestId("split-run-attention-note");
    expect(within(note).getByRole("heading", { name: "This task did not succeed" })).toBeInTheDocument();
    expect(within(note).getByText("The work is done. The result did not meet the goal.")).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Send to backlog" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Reopen" })).toBeInTheDocument();
  });

  it("hides invented files and ledger pull requests on a live task", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryId: PRIMARY_FACTORY_ID,
      orderId: LINE_BOARD_DONE_RECEIPTS_ORDER.id,
      fixture: splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER, { demoArtifacts: false }),
    });

    expect(screen.getByTestId("redesign-console-summary")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /#510/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "closure.md" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /merge-screenshot/ })).not.toBeInTheDocument();
  });

  it("shows a failed implement stream with Reject and Rerun on the note", () => {
    renderPopup({ fixture: splitRunFixtureForWorkOrder(FAILED_WORK_ORDER) });

    const note = screen.getByTestId("split-run-attention-note");
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(within(note).getByRole("button", { name: "Rerun" })).toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-implement-0")).not.toBeInTheDocument();
  });

  it("expands the new implement run after a rerun of the same step", () => {
    const fixture = splitRunFixtureForWorkOrder(
      {
        id: "wo-2",
        title: "Test task 2",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              {
                id: "e-plan",
                step: "Planning",
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
                run: { id: "run-plan" },
              },
              {
                id: "e-impl-old",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
                run: { id: "run-old" },
              },
              {
                id: "e-impl-new",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_STARTED",
                result: "RESULT_UNKNOWN",
                run: { id: "run-new" },
              },
            ],
          },
        ],
      },
      { lineId: "line-1" },
    );
    const implementPhases = fixture.phases.filter((phase) => phase.name === "Implement");

    renderPopup({ fixture });

    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId(`split-run-phase-${implementPhases[1].id}`)).not.toBeInTheDocument();
  });

  it("opens the selected step log when a log row is clicked", () => {
    renderSplitRun();

    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-backlog")).not.toBeInTheDocument();
    expect(screen.queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();
  });

  it("does not show Edit automation on the console", () => {
    renderPopup({
      organizationId: FACTORIES_ORGANIZATION_ID,
      factoryKey: PRIMARY_FACTORY_KEY,
      orderNumber: BOARD_IMPLEMENT_NOTIFY_ORDER.number,
      fixture: splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER),
    });

    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-pr-creation-2")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
  });

  it("opens a mapped implement-running task on the implement log", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({
        ...OPEN_WORK_ORDER,
        title: "Implement job",
        lineDispatches: [
          {
            id: "dispatch-1",
            line: { id: "line-1", name: "plan-and-implement" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
            ],
          },
        ],
      }),
    });

    expect(screen.getByRole("heading", { name: "RF-101 Implement job" })).toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-ingest")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-implement-0")).not.toBeInTheDocument();
    expect(screen.queryByTestId("run-overlay-compact-canvas")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
  });

  it("highlights the PR Closure log for the selected canvas component", () => {
    renderPopup({
      fixture: {
        ...SPLIT_RUN_RUNNING,
        title: "Send refund receipts after provider confirm",
        lineStatus: "passed",
        footer: buildSplitRunFooter({ kind: "done" }),
        footerTone: "done",
        currentPhaseId: "done",
        phases: [
          ...SPLIT_RUN_RUNNING.phases.map((phase) => ({ ...phase, status: "passed" as const })),
          {
            id: "done",
            name: "Done",
            status: "passed",
            duration: "1m 12s",
            componentName: "PR Closure",
            artifacts: [],
            stream: [],
            canvasSteps: [],
          },
        ],
      },
    });

    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-phase-done")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-done")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-canvas-node-find-pull-request")).not.toBeInTheDocument();
  });

  it("lets you rename the title on a draft", async () => {
    const user = userEvent.setup();
    renderPopup({ fixture: splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER) });

    const heading = screen.getByRole("heading", { name: "RF-105 Draft: rework refund telemetry" });
    expect(within(heading).getByTestId("popup-work-order-key")).toHaveTextContent("RF-105");
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    await user.click(screen.getByTestId("popup-work-order-title"));
    const titleInput = await screen.findByTestId("popup-work-order-title-input");
    expect(titleInput).toHaveValue("Draft: rework refund telemetry");
    expect(within(heading).getByTestId("popup-work-order-key")).toHaveTextContent("RF-105");
    await user.clear(titleInput);
    await user.type(titleInput, "Renamed draft");
    await user.keyboard("{Enter}");
    expect(screen.getByTestId("popup-work-order-title")).toHaveTextContent("Renamed draft");
    expect(screen.getByTestId("popup-work-order-title")).not.toHaveTextContent("RF-105");
    expect(screen.getByRole("heading", { name: "RF-105 Renamed draft" })).toBeInTheDocument();
    expect(screen.getByTestId("popup-work-order-key")).toHaveTextContent("RF-105");
  });

  it("shows the title alone when the task has no key and no number", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({ ...DRAFT_WORK_ORDER, key: "", number: "" }),
    });

    expect(screen.getByRole("heading", { name: "Draft: rework refund telemetry" })).toBeInTheDocument();
    expect(screen.queryByTestId("popup-work-order-key")).not.toBeInTheDocument();
  });

  it("builds the heading key from the workspace key and task number", () => {
    renderPopup({
      fixture: splitRunFixtureForWorkOrder({ ...OPEN_WORK_ORDER, key: "" }),
      factoryKey: "RF",
      orderNumber: "101",
    });

    expect(screen.getByTestId("popup-work-order-key")).toHaveTextContent("RF-101");
    expect(screen.getByRole("heading", { name: "RF-101 Reconcile duplicate refunds in ledger" })).toBeInTheDocument();
  });

  it("does not let you edit a completed task", () => {
    renderPopup({
      fixture: {
        ...SPLIT_RUN_RUNNING,
        footer: buildSplitRunFooter({ kind: "done" }),
        footerTone: "done",
      },
    });

    expect(screen.queryByTestId("popup-work-order-title")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("popup-edit-owner")).not.toBeInTheDocument();
  });
});
