import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { WorkOrderDetailHeader } from "./WorkOrderDetailHeader";

/**
 * Header for the work order detail page: title on the left, Copy link + a
 * kebab menu of lifecycle actions on the right. Status and dispatch live in
 * the sidebar, so the header stays minimal.
 */
const meta = {
  title: "Factories/Components/WorkOrderDetailHeader",
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

const commonHandlers = {
  onClose: (result: "RESULT_COMPLETED" | "RESULT_REJECTED" | "RESULT_FAILED") => {
    console.log("close", result);
  },
  onStatusChange: async (state: string, result?: string) => {
    console.log("status change", state, result);
  },
};

const commonFlags = {
  isCompleting: false,
  isRejecting: false,
  isClosing: false,
  isUpdatingStatus: false,
};

/** Open — Back to draft, Complete, Reject all available in the kebab. */
export const Open: Story = {
  args: {
    orderTitle: "Reconcile duplicate refunds in ledger",
    displayStatus: "waiting",
    isOpen: true,
    isDispatchable: true,
    isClosed: false,
    canClose: true,
    canManage: true,
    ...commonFlags,
    ...commonHandlers,
  },
};

/**
 * Running — Complete/Reject remain available so an operator can close
 * mid-run. Back-to-draft is hidden because the FSM rejects `open → draft`
 * while a step is still executing.
 */
export const Running: Story = {
  args: {
    orderTitle: "Add refund reconciliation test",
    displayStatus: "running",
    isOpen: true,
    isDispatchable: true,
    isClosed: false,
    canClose: true,
    canManage: true,
    ...commonFlags,
    ...commonHandlers,
  },
};

/** Closed — single "Reopen" action surfaces. */
export const Closed: Story = {
  args: {
    orderTitle: "Backfill refund audit trail",
    displayStatus: "completed",
    isOpen: false,
    isDispatchable: false,
    isClosed: true,
    canClose: false,
    canManage: true,
    ...commonFlags,
    ...commonHandlers,
  },
};

/** Draft — Reject abandons the order before any work runs. */
export const Draft: Story = {
  args: {
    orderTitle: "Draft: rework refund telemetry",
    displayStatus: "draft",
    isOpen: false,
    isDispatchable: true,
    isClosed: false,
    canClose: true,
    canManage: true,
    ...commonFlags,
    ...commonHandlers,
  },
};

/** Closed as failed — reopen action surfaces. */
export const ClosedFailed: Story = {
  name: "Closed (failed)",
  args: {
    orderTitle: "Failed: reconcile refund ledger for Q1 audit",
    displayStatus: "failed",
    isOpen: false,
    isDispatchable: false,
    isClosed: true,
    canClose: false,
    canManage: true,
    ...commonFlags,
    ...commonHandlers,
  },
};

/** Viewer without update permission — every action item is disabled. */
export const ReadOnly: Story = {
  name: "Read Only",
  args: {
    orderTitle: "Reconcile duplicate refunds in ledger",
    displayStatus: "waiting",
    isOpen: true,
    isDispatchable: true,
    isClosed: false,
    canClose: false,
    canManage: false,
    ...commonFlags,
    ...commonHandlers,
  },
};
