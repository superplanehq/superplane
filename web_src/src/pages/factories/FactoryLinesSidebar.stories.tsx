import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import {
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_ID,
  REFUND_FACTORY_LINES,
} from "./__fixtures__/factoryPageResponses";
import { FactoryLinesSidebar } from "./FactoryLinesSidebar";

/**
 * Sidebar section listing configured factory lines and their steps. Names
 * link to the line editor when the viewer has `factories:update`.
 */
const meta = {
  title: "Factories/FactoryLinesSidebar",
  component: FactoryLinesSidebar,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[320px] w-[320px] bg-white p-6 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof FactoryLinesSidebar>;

export default meta;

type Story = StoryObj<typeof meta>;

const factoryHref = `/${FACTORIES_ORGANIZATION_ID}/factories/${PRIMARY_FACTORY_ID}`;

/** Populated: two lines (multi-step `plan-and-implement` + `hotfix`). */
export const Populated: Story = {
  args: {
    factoryHref,
    lines: REFUND_FACTORY_LINES,
    canUpdate: true,
  },
};

/** Read-only viewer: line names render as plain text, no "+ New" button. */
export const ReadOnly: Story = {
  name: "Read Only",
  args: {
    factoryHref,
    lines: REFUND_FACTORY_LINES,
    canUpdate: false,
  },
};

/** Empty: no lines configured yet — "+ New" prompt only. */
export const Empty: Story = {
  args: {
    factoryHref,
    lines: [],
    canUpdate: true,
  },
};
