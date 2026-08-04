import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import {
  CLOSED_WORK_ORDER,
  CLOSED_WORK_ORDER_EVENTS,
  FACTORIES_ORGANIZATION_ID,
  FAILED_WORK_ORDER,
  FAILED_WORK_ORDER_EVENTS,
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_EVENTS,
  RUNNING_WORK_ORDER,
  RUNNING_WORK_ORDER_EVENTS,
} from "./__fixtures__/factoryPageResponses";
import { WorkOrderActivityTimeline } from "./WorkOrderActivityTimeline";

/**
 * Vertical timeline of the work order lifecycle: `created` marker, dispatch
 * batches per line, and (when closed) a "Closed as …" footer.
 */
const meta = {
  title: "Factories/WorkOrderActivityTimeline",
  component: WorkOrderActivityTimeline,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[380px] max-w-2xl bg-white p-6 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderActivityTimeline>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Freshly opened order — only a `created` marker. */
export const JustCreated: Story = {
  name: "Just Created",
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: OPEN_WORK_ORDER,
    events: OPEN_WORK_ORDER_EVENTS,
  },
};

/** Running — dispatch batch with a step in progress. */
export const Running: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: RUNNING_WORK_ORDER,
    events: RUNNING_WORK_ORDER_EVENTS,
  },
};

/** Failed — dispatch batch with a failed step surfaced. */
export const Failed: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: FAILED_WORK_ORDER,
    events: FAILED_WORK_ORDER_EVENTS,
  },
};

/** Closed — full history + "Closed as completed" footer. */
export const Closed: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    order: CLOSED_WORK_ORDER,
    events: CLOSED_WORK_ORDER_EVENTS,
  },
};
