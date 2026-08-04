import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import {
  CLOSED_WORK_ORDER,
  FACTORIES_ORGANIZATION_ID,
  FAILED_WORK_ORDER,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  REFUND_FACTORY_LINES,
  RUNNING_WORK_ORDER,
} from "./__fixtures__/factoryPageResponses";
import { WorkOrderCard } from "./WorkOrderCard";

/**
 * Single work order row: status badge, title, creator/assignee avatars, updated meta,
 * and a compact execution list beneath. Open orders also expose a dispatch popover.
 */
const meta = {
  title: "Factories/WorkOrderCard",
  component: WorkOrderCard,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[220px] max-w-4xl bg-white p-6 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderCard>;

export default meta;

type Story = StoryObj<typeof meta>;

const factoryHref = `/${FACTORIES_ORGANIZATION_ID}/factories/${PRIMARY_FACTORY_ID}`;

/** Open — dispatch button visible; no line executions yet. */
export const Open: Story = {
  args: {
    order: OPEN_WORK_ORDER,
    factoryHref,
    organizationId: FACTORIES_ORGANIZATION_ID,
    lines: REFUND_FACTORY_LINES,
    canDispatch: true,
    isDispatching: false,
    onDispatch: async (lineName) => {
      console.log("dispatch", lineName);
    },
  },
};

/** Running — execution list shows one finished step and one in-progress. */
export const Running: Story = {
  args: {
    order: RUNNING_WORK_ORDER,
    factoryHref,
    organizationId: FACTORIES_ORGANIZATION_ID,
    lines: REFUND_FACTORY_LINES,
    canDispatch: true,
    onDispatch: async () => undefined,
  },
};

/** Failed — status badge in red, failing step surfaced in the execution list. */
export const Failed: Story = {
  args: {
    order: FAILED_WORK_ORDER,
    factoryHref,
    organizationId: FACTORIES_ORGANIZATION_ID,
    lines: REFUND_FACTORY_LINES,
    canDispatch: true,
    onDispatch: async () => undefined,
  },
};

/** Closed — dispatch affordance hidden, badge shows the closed result. */
export const Closed: Story = {
  args: {
    order: CLOSED_WORK_ORDER,
    factoryHref,
    organizationId: FACTORIES_ORGANIZATION_ID,
    lines: REFUND_FACTORY_LINES,
    canDispatch: false,
  },
};
