import type { Meta, StoryObj } from "@storybook/react-vite";
import type { FactoriesWorkOrder } from "@/api-client";

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
