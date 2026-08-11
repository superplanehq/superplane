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
  title: "Factories/Components/WorkOrderCard",
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

const commonArgs = {
  organizationId: FACTORIES_ORGANIZATION_ID,
  factoryId: PRIMARY_FACTORY_ID,
  lines: REFUND_FACTORY_LINES,
};

/** Open — dispatch button visible; no line executions yet. */
export const Open: Story = {
  args: {
    ...commonArgs,
    order: OPEN_WORK_ORDER,
    canDispatch: true,
    isDispatching: false,
    onDispatch: async (input) => {
      console.log("dispatch", input);
    },
  },
};

/** Running — execution list shows one finished step and one in-progress. */
export const Running: Story = {
  args: {
    ...commonArgs,
    order: RUNNING_WORK_ORDER,
    canDispatch: true,
    onDispatch: async () => undefined,
  },
};

/** Failed — status badge in red, failing step surfaced in the execution list. */
export const Failed: Story = {
  args: {
    ...commonArgs,
    order: FAILED_WORK_ORDER,
    canDispatch: true,
    onDispatch: async () => undefined,
  },
};

/** Closed — dispatch affordance hidden, badge shows the closed result. */
export const Closed: Story = {
  args: {
    ...commonArgs,
    order: CLOSED_WORK_ORDER,
    canDispatch: false,
  },
};
