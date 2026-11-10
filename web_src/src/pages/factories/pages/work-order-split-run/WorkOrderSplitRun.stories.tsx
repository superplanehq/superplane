import type { Meta, StoryObj } from "@storybook/react-vite";
import type { FactoriesWorkOrder } from "@/api-client";
import { FEATURE_FACTORY_STEP_ACTIONS } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { REFUND_IMPLEMENTER_APP, factoryOwnedCanvasFixture } from "../../__fixtures__/factoryOwnedCanvasFixture";
import { CUSTOM_AUTOMATION_TEMPLATE_ID } from "../../lib/customAutomation";
import { failedImplementationCanvasFixture } from "../../__fixtures__/failedImplementationFixture";
import {
  DRAFT_WORK_ORDER,
  LINE_RUN_IMPLEMENT_FAILED_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
} from "../../__fixtures__/factoryPageResponses";
import { lineMetricsFactoriesFixture } from "../../__fixtures__/lineMetricsFactoriesFixture";
import { prReadyFactoriesFixture } from "../../__fixtures__/prReadyFactoriesFixture";
import { LINE_BOARD_VERIFY_PR_REVIEW_ORDER } from "../../__fixtures__/lineMetricsFactoriesFixture";
import { refineChatBoardFixture } from "../../__fixtures__/refineChatBoardFixture";
import { ARNOLD_USER, TWO_HOURS_AGO } from "../../__fixtures__/factoryPageIds";
import { BOARD_IMPLEMENT_FAILED_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";

/**
 * Line board with the work-order popup. Open a card: finished steps stay
 * collapsed and the current step expands into a log.
 */
const meta = {
  title: "Factories/Pages/Task Split Run",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const BacklogCustomRuns: Story = {
  name: "Backlog custom run without checks",
  render: () => {
    const app = {
      id: "app-inspect-task",
      name: "Inspect task",
      columnKey: "backlog",
      templateId: CUSTOM_AUTOMATION_TEMPLATE_ID,
    };
    const run = {
      id: "inspect-task-run",
      canvasId: app.id,
      state: "STATE_STARTED",
      createdAt: new Date(Date.now() - 60_000).toISOString(),
      rootEvent: { id: "inspect-task-event", data: { workOrder: { id: DRAFT_WORK_ORDER.id } } },
    };
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/task/${DRAFT_WORK_ORDER.number}`}
        factoriesFixture={{
          ...lineMetricsFactoriesFixture,
          appsByFactoryId: { ...lineMetricsFactoriesFixture.appsByFactoryId, [PRIMARY_FACTORY_ID]: [app] },
          workOrdersByFactoryId: {
            ...lineMetricsFactoriesFixture.workOrdersByFactoryId,
            [PRIMARY_FACTORY_ID]: [DRAFT_WORK_ORDER],
          },
          checksByOrderId: { [DRAFT_WORK_ORDER.id!]: [] },
        }}
        appFixture={factoryOwnedCanvasFixture(app, {
          runs: { runs: [run], totalCount: 1, hasNextPage: false },
          runDetail: { run },
          executions: { executions: [] },
        })}
      />
    );
  },
};

export const Running: Story = {
  name: "Terminal log",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />
    );
  },
};

export const AutomationRun: Story = {
  name: "Automation run",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    const appId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/automations/${appId}/runs/${LINE_RUN_IMPLEMENT_FAILED_ID}?from=task&lineId=${line.id}&orderNumber=106`}
        factoriesFixture={lineMetricsFactoriesFixture}
        appFixture={failedImplementationCanvasFixture()}
      />
    );
  },
};

function startedTask(autoStarted: boolean) {
  return (
    <FactoriesHarness
      mode="app"
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/task/106?lineId=${REFUND_FACTORY_LINES[0].id}`}
      factoriesFixture={{
        ...lineMetricsFactoriesFixture,
        workOrdersByFactoryId: {
          ...lineMetricsFactoriesFixture.workOrdersByFactoryId,
          [PRIMARY_FACTORY_ID]: [
            {
              ...BOARD_IMPLEMENT_FAILED_ORDER,
              state: "STATE_OPEN",
              result: undefined,
              origin: { url: "https://github.com/acme/payments/issues/12", label: "acme/payments#12" },
              createdBy: { user: { id: "importer", name: "Alex Importer" } },
              startedBy: autoStarted ? undefined : { id: ARNOLD_USER.id, name: ARNOLD_USER.name },
              autoStarted,
              startedAt: TWO_HOURS_AGO,
            },
          ],
        },
      }}
      appFixture={failedImplementationCanvasFixture()}
    />
  );
}

export const StartedByReviewer: Story = {
  name: "Started by a different person",
  render: () => startedTask(false),
};

export const AutoStarted: Story = {
  name: "Auto-started without a person",
  render: () => startedTask(true),
};

export const FailedImplementation: Story = {
  name: "Failed implementation",
  render: () => (
    <FactoriesHarness
      mode="app"
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/task/106?lineId=${REFUND_FACTORY_LINES[0].id}`}
      factoriesFixture={{
        ...lineMetricsFactoriesFixture,
        workOrdersByFactoryId: {
          ...lineMetricsFactoriesFixture.workOrdersByFactoryId,
          [PRIMARY_FACTORY_ID]: [{ ...BOARD_IMPLEMENT_FAILED_ORDER, state: "STATE_OPEN", result: undefined }],
        },
      }}
      appFixture={failedImplementationCanvasFixture()}
    />
  ),
};

function fallbackIntakeTask(mcpClient?: FactoriesWorkOrder["mcpClient"]) {
  return (
    <FactoriesHarness
      mode="app"
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/task/106?lineId=${REFUND_FACTORY_LINES[0].id}`}
      factoriesFixture={{
        ...lineMetricsFactoriesFixture,
        workOrdersByFactoryId: {
          ...lineMetricsFactoriesFixture.workOrdersByFactoryId,
          [PRIMARY_FACTORY_ID]: [
            {
              ...BOARD_IMPLEMENT_FAILED_ORDER,
              state: "STATE_OPEN",
              result: undefined,
              origin: undefined,
              createdBy: DRAFT_WORK_ORDER.createdBy,
              mcpClient,
            },
          ],
        },
      }}
      appFixture={failedImplementationCanvasFixture()}
    />
  );
}

export const ManualIntake: Story = {
  name: "Manual Intake",
  render: () => fallbackIntakeTask(),
};

export const UnknownMcpIntake: Story = {
  name: "Unknown MCP Intake",
  render: () => fallbackIntakeTask({ id: "superplane-local", name: "Codex" }),
};

function refineChatOnBoard() {
  const line = REFUND_FACTORY_LINES[0];
  return (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/task/${DRAFT_WORK_ORDER.number}?lineId=${line.id}`}
      factoriesFixture={refineChatBoardFixture()}
    />
  );
}

/** Line board with the refine popup open: agent and user chat, a prior survey answer, and a pending survey. */
export const RefineChatOnBoard: Story = {
  name: "Refine chat on the board",
  render: () => refineChatOnBoard(),
};

function prReadyTask(assessment: "ready" | "review" | "partial") {
  return (
    <FactoriesHarness
      mode="app"
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/task/${LINE_BOARD_VERIFY_PR_REVIEW_ORDER.number}?lineId=${REFUND_FACTORY_LINES[0].id}`}
      factoriesFixture={prReadyFactoriesFixture(assessment)}
    />
  );
}

export const PullRequestReady: Story = {
  name: "PR ready",
  render: () => prReadyTask("ready"),
};

export const PullRequestNeedsReview: Story = {
  name: "PR ready — checks need review",
  render: () => prReadyTask("review"),
};

export const PullRequestPartialChecks: Story = {
  name: "PR ready — partial checks",
  render: () => prReadyTask("partial"),
};

function humanReviewTask(experimentalFeatures: string[], reviewStatus: "decided" | "failed" = "decided") {
  const fixture = prReadyFactoriesFixture("review");
  const orderId = LINE_BOARD_VERIFY_PR_REVIEW_ORDER.id!;
  const app = { id: "human-review-app", name: "Request review", columnKey: "verify", templateId: "human-review" };
  const pr = fixture.pullRequestsByOrderId?.[orderId]?.[0];
  const run = {
    id: "human-review-run",
    canvasId: app.id,
    state: "STATE_FINISHED" as const,
    result: "RESULT_PASSED" as const,
    createdAt: new Date(Date.now() - 60_000).toISOString(),
    finishedAt: new Date().toISOString(),
  };
  return (
    <FactoriesHarness
      mode="app"
      experimentalFeatures={experimentalFeatures}
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/task/${LINE_BOARD_VERIFY_PR_REVIEW_ORDER.number}?lineId=${REFUND_FACTORY_LINES[0].id}`}
      factoriesFixture={{
        ...fixture,
        appsByFactoryId: {
          ...fixture.appsByFactoryId,
          [PRIMARY_FACTORY_ID]: [...(fixture.appsByFactoryId?.[PRIMARY_FACTORY_ID] ?? []), app],
        },
        pullRequestsByOrderId: {
          ...fixture.pullRequestsByOrderId,
          [orderId]: [
            {
              ...pr,
              currentRevision: { id: "review-revision", sha: "review-head" },
              humanReview:
                reviewStatus === "failed"
                  ? {
                      status: "failed",
                      assessedCommit: "review-head",
                      blockingReason: "Request review could not start. Retry Request review.",
                    }
                  : {
                      runId: run.id,
                      status: "decided",
                      assessedCommit: "review-head",
                      reviewRequired: true,
                      rationale: "The pull request changes files outside the task scope.",
                      blockingReason: "A selected reviewer must approve the current head commit.",
                      selectedReviewers: [
                        { id: "7", login: "alex", reason: "Owns the affected application files." },
                        { id: "8", login: "sam", reason: "Owns the regression tests." },
                      ],
                      checkUrl: "https://github.com/acme/payments/runs/7",
                    },
              activities: [
                ...(pr?.activities ?? []),
                ...(reviewStatus === "failed" ? [] : [{ title: "Request review", run }]),
              ],
            },
          ],
        },
      }}
      appFixture={factoryOwnedCanvasFixture(app, {
        runs: { runs: [run], totalCount: 1, hasNextPage: false },
        runDetail: { run },
      })}
    />
  );
}

export const HumanReviewActions: Story = {
  name: "Request review — decision in Actions",
  render: () => humanReviewTask([FEATURE_FACTORY_STEP_ACTIONS]),
};

export const HumanReviewActionsDisabled: Story = {
  name: "Request review — Actions disabled",
  render: () => humanReviewTask([]),
};

export const HumanReviewStartupFailed: Story = {
  name: "Request review — agent startup failed",
  render: () => humanReviewTask([FEATURE_FACTORY_STEP_ACTIONS], "failed"),
};
