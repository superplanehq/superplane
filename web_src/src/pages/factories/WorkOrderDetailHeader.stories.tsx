import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { REFUND_FACTORY_LINES } from "./__fixtures__/factoryPageResponses";
import { WorkOrderDetailHeader } from "./WorkOrderDetailHeader";
import { getWorkOrderDisplayStatusMeta } from "./lib/workOrderProgress";

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
const draftMeta = getWorkOrderDisplayStatusMeta("draft");
const closedFailedMeta = getWorkOrderDisplayStatusMeta("closedFailed");

const commonHandlers = {
  onDispatch: async (lineName: string) => {
    console.log("dispatch", lineName);
  },
  onClose: (result: "RESULT_COMPLETED" | "RESULT_REJECTED" | "RESULT_FAILED") => {
    console.log("close", result);
  },
  onStatusChange: async (state: string, result?: string) => {
    console.log("status change", state, result);
  },
};

const commonFlags = {
  factoryLines: REFUND_FACTORY_LINES,
  permissionsLoading: false,
  isDispatching: false,
  isCompleting: false,
  isRejecting: false,
  isClosing: false,
  isUpdatingStatus: false,
};

/** Open — Dispatch, Back to draft, Complete, Reject all available. */
export const Open: Story = {
  args: {
    orderTitle: "Reconcile duplicate refunds in ledger",
    statusMeta: openMeta,
    displayStatus: "open",
    isOpen: true,
    isDispatchable: true,
    isClosed: false,
    canDispatch: true,
    canClose: true,
    canManage: true,
    ...commonFlags,
    ...commonHandlers,
  },
};

/**
 * Running — badge shows the spinner; Complete/Reject remain available so an
 * operator can close mid-run. Back-to-draft is hidden because the FSM rejects
 * `open → draft` while a step is still executing.
 */
export const Running: Story = {
  args: {
    orderTitle: "Add refund reconciliation test",
    statusMeta: runningMeta,
    displayStatus: "running",
    isOpen: true,
    isDispatchable: true,
    isClosed: false,
    canDispatch: true,
    canClose: true,
    canManage: true,
    ...commonFlags,
    isDispatching: true,
    ...commonHandlers,
  },
};

/** Closed — single "Reopen" action surfaces. */
export const Closed: Story = {
  args: {
    orderTitle: "Backfill refund audit trail",
    statusMeta: completedMeta,
    displayStatus: "completed",
    isOpen: false,
    isDispatchable: false,
    isClosed: true,
    canDispatch: false,
    canClose: false,
    canManage: true,
    ...commonFlags,
    ...commonHandlers,
  },
};

/** Draft — Dispatch is available so scoping can flip straight into a run. */
export const Draft: Story = {
  args: {
    orderTitle: "Draft: rework refund telemetry",
    statusMeta: draftMeta,
    displayStatus: "draft",
    isOpen: false,
    isDispatchable: true,
    isClosed: false,
    canDispatch: true,
    canClose: false,
    canManage: true,
    ...commonFlags,
    ...commonHandlers,
  },
};

/** Closed as failed — reopen action, but the badge uses the failed styling. */
export const ClosedFailed: Story = {
  name: "Closed (failed)",
  args: {
    orderTitle: "Failed: reconcile refund ledger for Q1 audit",
    statusMeta: closedFailedMeta,
    displayStatus: "closedFailed",
    isOpen: false,
    isDispatchable: false,
    isClosed: true,
    canDispatch: false,
    canClose: false,
    canManage: true,
    ...commonFlags,
    ...commonHandlers,
  },
};

/** Viewer without update permission — every action button is disabled. */
export const ReadOnly: Story = {
  name: "Read Only",
  args: {
    orderTitle: "Reconcile duplicate refunds in ledger",
    statusMeta: openMeta,
    displayStatus: "open",
    isOpen: true,
    isDispatchable: true,
    isClosed: false,
    canDispatch: false,
    canClose: false,
    canManage: false,
    ...commonFlags,
    ...commonHandlers,
  },
};
