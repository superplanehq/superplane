import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";
import { STREAKS } from "@/pages/factories/verification/__fixtures__/verificationFixtures";

import { StreakIndicator } from "./StreakIndicator";

/** Current and best streaks without new blocking findings. */
const meta = {
  title: "Factories/Health/StreakIndicator",
  component: StreakIndicator,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[200px] max-w-2xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof StreakIndicator>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Active streaks. */
export const Active: Story = {
  args: { streaks: STREAKS },
};

/** After a failing run reset the counters. */
export const AfterReset: Story = {
  args: {
    streaks: [
      { label: "Work orders without blocking findings", current: 0, best: 21, unit: "work orders" },
      { label: "Days without blocking findings", current: 0, best: 15, unit: "days" },
    ],
  },
};
