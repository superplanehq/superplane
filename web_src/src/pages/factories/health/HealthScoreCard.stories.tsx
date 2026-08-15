import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";
import { HEALTH_SNAPSHOT } from "@/pages/factories/verification/__fixtures__/verificationFixtures";

import { HealthScoreCard } from "./HealthScoreCard";

/**
 * Health score in the console scorecard visual language: status dot, large
 * value, change chip, sparkline, and progress toward the target.
 */
const meta = {
  title: "Factories/Health/HealthScoreCard",
  component: HealthScoreCard,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[220px] max-w-sm bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof HealthScoreCard>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Score improving toward the target. */
export const Improving: Story = {
  args: { label: "Health score", snapshot: HEALTH_SNAPSHOT },
};

/** Score declining after new blocking findings. */
export const Declining: Story = {
  args: {
    label: "Health score",
    snapshot: { score: 64, change: -9, target: 90, series: [82, 80, 79, 76, 73, 70, 68, 64] },
  },
};

/** No target set: the progress bar is hidden. */
export const WithoutTarget: Story = {
  args: {
    label: "Health score",
    snapshot: { score: 77, change: 0, series: [74, 75, 77, 76, 77, 77] },
  },
};
