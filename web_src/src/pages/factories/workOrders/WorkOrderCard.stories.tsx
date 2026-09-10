import type { Meta, StoryObj } from "@storybook/react-vite";

import type { FactoriesFactory, FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard } from "./WorkOrderCard";

const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: "RF" };
const factoryLines: FactoriesFactoryLine[] = [{ id: "line-a", name: "hotfix" }];

/**
 * Builds a waiting task that the card renders with attention chips. A
 * status note produces the "Waiting for user review" chip; the enclosing
 * story then adds the compact "Status checks passed" mark by listing the
 * task in `checksPassedOrderIds`.
 */
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

/**
 * The canonical task card. These stories focus on the middle status
 * row, and in particular on the compact, icon-only "Status checks passed"
 * mark that sits next to the full "Waiting for user review" pill. The card
 * is width-constrained to a board column (`min-w-72`).
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
  },
} satisfies Meta<typeof WorkOrderCard>;

export default meta;

type Story = StoryObj<typeof meta>;

/**
 * Waiting for user review, with checks still green: the full review chip
 * plus the compact checks-passed mark. The mark keeps the meaning through
 * its color, icon, tooltip, and accessible name without a visible label.
 */
export const ChecksPassedWithReview: Story = {
  name: "Checks passed + waiting for review",
  args: {
    checksPassedOrderIds: new Set(["wo-waiting"]),
  },
};

/**
 * Baseline for comparison: the same waiting task without a finished check
 * wait shows only the "Waiting for user review" chip and no mark.
 */
export const ReviewOnly: Story = {
  name: "Waiting for review (no mark)",
};

/**
 * Open task with no attention chip. The footer keeps created time on the
 * left and the owner given name plus avatar on the right.
 */
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
  },
};

/**
 * A long title stresses the card. Status pills stay on the middle row,
 * so the footer keeps created time and the owner on one line.
 */
export const ChecksPassedLongTitle: Story = {
  name: "Checks passed + long title",
  args: {
    entry: buildWorkOrderListEntry(
      waitingOrder({
        id: "wo-waiting",
        title: "Reconcile duplicate refunds across the ledger before the Q1 audit closes",
      }),
      factory,
    ),
    checksPassedOrderIds: new Set(["wo-waiting"]),
  },
};
