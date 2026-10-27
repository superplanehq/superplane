import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { REFUND_IMPLEMENTER_APP } from "../../__fixtures__/factoryOwnedCanvasFixture";
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
