import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { REFUND_FACTORY_LINES } from "./__fixtures__/factoryPageResponses";
import { WorkOrderDetailHeader } from "./WorkOrderDetailHeader";
import { getWorkOrderDisplayStatusMeta } from "./workOrderProgress";

/**
 * Header for the work order detail page: status badge, title, and (when open)
 * Dispatch / Complete / Reject actions.
 */
const meta = {
  title: "Factories/WorkOrderDetailHeader",
  component: WorkOrderDetailHeader,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[220px] max-w-5xl bg-white p-6 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderDetailHeader>;

export default meta;

type Story = StoryObj<typeof meta>;

const openMeta = getWorkOrderDisplayStatusMeta("open");
const runningMeta = getWorkOrderDisplayStatusMeta("running");
const completedMeta = getWorkOrderDisplayStatusMeta("completed");

const commonHandlers = {
  onDispatch: async (lineName: string) => {
    console.log("dispatch", lineName);
  },
  onClose: (result: "RESULT_COMPLETED" | "RESULT_REJECTED") => {
    console.log("close", result);
  },
};

/** Open — Dispatch, Complete, Reject all available. */
export const Open: Story = {
  args: {
    orderTitle: "Reconcile duplicate refunds in ledger",
    statusMeta: openMeta,
    displayStatus: "open",
    isOpen: true,
    factoryLines: REFUND_FACTORY_LINES,
    canDispatch: true,
    canClose: true,
    permissionsLoading: false,
    isDispatching: false,
    isCompleting: false,
    isRejecting: false,
    isClosing: false,
    ...commonHandlers,
  },
};

/** Running — badge shows the spinner; actions still visible for cancel/complete. */
export const Running: Story = {
  args: {
    orderTitle: "Add refund reconciliation test",
    statusMeta: runningMeta,
    displayStatus: "running",
    isOpen: true,
    factoryLines: REFUND_FACTORY_LINES,
    canDispatch: true,
    canClose: true,
    permissionsLoading: false,
    isDispatching: true,
    isCompleting: false,
    isRejecting: false,
    isClosing: false,
    ...commonHandlers,
  },
};

/** Closed — action row hidden, only status badge + title remain. */
export const Closed: Story = {
  args: {
    orderTitle: "Backfill refund audit trail",
    statusMeta: completedMeta,
    displayStatus: "completed",
    isOpen: false,
    factoryLines: REFUND_FACTORY_LINES,
    canDispatch: false,
    canClose: false,
    permissionsLoading: false,
    isDispatching: false,
    isCompleting: false,
    isRejecting: false,
    isClosing: false,
    ...commonHandlers,
  },
};
