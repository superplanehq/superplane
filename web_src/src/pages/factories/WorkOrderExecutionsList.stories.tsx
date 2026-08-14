import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import {
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_KEY,
  RUNNING_WORK_ORDER,
  CLOSED_WORK_ORDER,
} from "./__fixtures__/factoryPageResponses";
import { WorkOrderExecutionsList } from "./WorkOrderExecutionsList";

/**
 * List of line-run steps with status icons and run links. Three visual
 * variants: `default` (grouped card), `compact` (inline group), and
 * `inline` (single flat list).
 */
const meta = {
  title: "Factories/Components/WorkOrderExecutionsList",
  component: WorkOrderExecutionsList,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[240px] max-w-2xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderExecutionsList>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Default variant — grouped card with running step in progress. */
export const Default: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    factoryKey: PRIMARY_FACTORY_KEY,
    executions: RUNNING_WORK_ORDER.executions,
    variant: "default",
  },
};

/** Compact variant used in the work order card row. */
export const Compact: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    factoryKey: PRIMARY_FACTORY_KEY,
    executions: CLOSED_WORK_ORDER.executions,
    variant: "compact",
  },
};

/** Inline variant — a single flat list without line grouping headers. */
export const Inline: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    factoryKey: PRIMARY_FACTORY_KEY,
    executions: CLOSED_WORK_ORDER.executions,
    variant: "inline",
  },
};

/** Empty state — default variant renders the helpful message; compact/inline collapse. */
export const Empty: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    factoryKey: PRIMARY_FACTORY_KEY,
    executions: [],
    variant: "default",
  },
};
