import type { Meta, StoryObj } from "@storybook/react-vite";

import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesFactoryPullRequest,
  FactoriesWorkOrder,
} from "@/api-client";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard } from "./WorkOrderCard";

const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: "RF" };
const factoryLines: FactoriesFactoryLine[] = [{ id: "line-a", name: "hotfix" }];

function waitingOrder(overrides: Partial<FactoriesWorkOrder> = {}): FactoriesWorkOrder {
  return {
    id: "wo-waiting",
    number: "12",
    title: "Ship idempotent refund retries",
    state: "STATE_OPEN",
    createdAt: "2026-08-30T10:00:00Z",
    updatedAt: "2026-09-01T10:00:00Z",
    statusNotes: [{ key: "pr-closure", headline: "Waiting for user review", body: "Tag the agent." }],
    lineDispatches: [],
    assignees: [{ id: "user-1", name: "Ada Lovelace" }],
    ...overrides,
  };
}

function attachedPullRequest(overrides: Partial<FactoriesFactoryPullRequest> = {}): FactoriesFactoryPullRequest {
  return {
    id: "pr-2323",
    workOrderId: "wo-waiting",
    number: "2323",
    url: "https://github.com/acme/payments/pull/2323",
    title: "Ship idempotent refund retries",
    state: "STATE_OPEN",
    ...overrides,
  };
}

/**
 * The canonical task card. These stories focus on the middle status
 * row: an attached pull request pill, plus the compact "Status checks
 * passed" mark. The card is width-constrained to a board column (`min-w-72`).
 */
const meta = {
  title: "Factories/Components/WorkOrderCard",
  component: WorkOrderCard,
  parameters: { layout: "centered" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="bg-background p-6">
        <div className="w-72">
          <Story />
        </div>
      </ComponentStoryShell>
    ),
  ],
  args: {
    entry: buildWorkOrderListEntry(waitingOrder(), factory),
    organizationId: "org-1",
    factoryKey: "RF",
    factoryLines,
    canDispatch: true,
    canAssign: true,
    dispatchingOrderIds: new Set<string>(),
    isAssigneesSaving: false,
    onDispatch: async () => {},
    onAssigneesSave: async () => {},
    pullRequests: [attachedPullRequest()],
  },
} satisfies Meta<typeof WorkOrderCard>;

export default meta;

type Story = StoryObj<typeof meta>;

/**
 * Open pull request attached to the task. The pill replaces Waiting
 * for user review and links to the pull request.
 */
export const ReviewPullRequest: Story = {
  name: "Review #2323",
};

/**
 * Review pill plus the completed check-wait title.
 */
export const ChecksPassedWithReview: Story = {
  name: "Review #2323 + checks passed",
  args: {
    checksPassedOrderIds: new Set(["wo-waiting"]),
    checksPassedLabels: new Map([
      ["wo-waiting", "Checks passed on [2e46445](https://github.com/acme/app/commit/2e46445)"],
    ]),
  },
};

/**
 * Draft pull request. The pill uses Draft plus the number.
 */
export const DraftPullRequest: Story = {
  name: "Draft #2323",
  args: {
    pullRequests: [attachedPullRequest({ state: "STATE_DRAFT" })],
  },
};

/**
 * Merged pull request on a finished task.
 */
export const MergedPullRequest: Story = {
  name: "Merged #2323",
  args: {
    entry: buildWorkOrderListEntry(
      waitingOrder({
        state: "STATE_CLOSED",
        result: "RESULT_COMPLETED",
        statusNotes: [],
      }),
      factory,
    ),
    pullRequests: [attachedPullRequest({ state: "STATE_MERGED" })],
  },
};

/**
 * Closed pull request that did not merge.
 */
export const ClosedPullRequest: Story = {
  name: "Closed #2323",
  args: {
    entry: buildWorkOrderListEntry(waitingOrder({ statusNotes: [] }), factory),
    pullRequests: [attachedPullRequest({ state: "STATE_CLOSED" })],
  },
};

/**
 * Two attached pull requests. The pill names the open request and
 * shows +1 for the other.
 */
export const ReviewPlusOne: Story = {
  name: "Review #2323 +1",
  args: {
    pullRequests: [
      attachedPullRequest(),
      attachedPullRequest({
        id: "pr-1801",
        number: "1801",
        url: "https://github.com/acme/payments/pull/1801",
        title: "Earlier refund attempt",
        state: "STATE_CLOSED",
      }),
    ],
  },
};

/**
 * A long title stresses the card. The pull request pill stays on the
 * middle row, so the footer keeps created time and the owner on one line.
 */
export const ChecksPassedLongTitle: Story = {
  name: "Checks passed + long title",
  args: {
    entry: buildWorkOrderListEntry(
      waitingOrder({
        title: "Reconcile duplicate refunds across the ledger before the Q1 audit closes",
      }),
      factory,
    ),
    checksPassedOrderIds: new Set(["wo-waiting"]),
    checksPassedLabels: new Map([
      ["wo-waiting", "Checks passed on [2e46445](https://github.com/acme/app/commit/2e46445)"],
    ]),
  },
};

/**
 * Open task with no pull request and no attention chip. The footer
 * keeps created time on the left and the owner given name plus avatar
 * on the right.
 */
/**
 * Draft task while the agent still works. The meter slot shows
 * thinking states and a matrix loader.
 */
export const DraftAnalyzing: Story = {
  name: "Draft analyzing",
  args: {
    entry: buildWorkOrderListEntry(
      waitingOrder({
        id: "wo-draft",
        number: "1",
        title: "The site feels weird lately",
        state: "STATE_DRAFT",
        statusNotes: [],
        assignees: [],
      }),
      factory,
    ),
    pullRequests: [],
    isAnalyzing: true,
  },
};

/**
 * Follow-up agent work after a score already exists. The meter slot
 * stays on thinking states until the agent waits for the user.
 */
export const DraftAnalyzingWithScore: Story = {
  name: "Draft analyzing after score",
  args: {
    entry: buildWorkOrderListEntry(
      waitingOrder({
        id: "wo-draft-scored",
        number: "2",
        title: "The site feels weird lately",
        state: "STATE_DRAFT",
        statusNotes: [],
        assignees: [],
      }),
      factory,
    ),
    pullRequests: [],
    isAnalyzing: true,
    clarityScore: 4,
    confidenceScore: 3,
  },
};

/** Refine finished: Clarity on top, Confidence below, in one meter footprint. */
export const DraftScoredPair: Story = {
  name: "Draft with Clarity and Confidence",
  args: {
    entry: buildWorkOrderListEntry(
      waitingOrder({
        id: "wo-draft-pair",
        number: "3",
        title: "Retry webhook delivery after provider timeouts",
        state: "STATE_DRAFT",
        statusNotes: [],
        assignees: [],
      }),
      factory,
    ),
    pullRequests: [],
    clarityScore: 5,
    confidenceScore: 3,
  },
};

/** Intake only: one Confidence meter until a refine session scores Clarity. */
export const DraftScoredIntakeOnly: Story = {
  name: "Draft with Confidence only",
  args: {
    entry: buildWorkOrderListEntry(
      waitingOrder({
        id: "wo-draft-intake",
        number: "4",
        title: "Reconcile settled ledger entries",
        state: "STATE_DRAFT",
        statusNotes: [],
        assignees: [],
      }),
      factory,
    ),
    pullRequests: [],
    confidenceScore: 4,
  },
};

export const OpenOwned: Story = {
  name: "Open with owner",
  args: {
    entry: buildWorkOrderListEntry(
      waitingOrder({
        id: "wo-open",
        title: "Add refund reconciliation test",
        statusNotes: [],
        createdAt: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
      }),
      factory,
    ),
    pullRequests: [],
  },
};

/**
 * GitHub origin on the card. The source icon is first in the pill
 * row and opens the issue in a new tab.
 */
export const GitHubOrigin: Story = {
  name: "GitHub origin",
  args: {
    entry: buildWorkOrderListEntry(
      waitingOrder({
        origin: {
          url: "https://github.com/acme/payments/issues/12",
          label: "acme/payments#12",
        },
      }),
      factory,
    ),
  },
};

/**
 * Manual origin on the card. The product logo sits first in the pill
 * row and is not a link.
 */
export const ManualOrigin: Story = {
  name: "Manual origin",
  args: {
    entry: buildWorkOrderListEntry(
      waitingOrder({
        createdBy: { user: { id: "user-1", name: "Ada Lovelace" } },
      }),
      factory,
    ),
    pullRequests: [],
  },
};
