import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";
import { ACHIEVEMENTS } from "@/pages/factories/verification/__fixtures__/verificationFixtures";

import { AchievementsGrid } from "./AchievementsGrid";

/**
 * Earned and not-yet-earned achievements with descriptive milestone names.
 * Not-yet-earned entries show what remains to earn them.
 */
const meta = {
  title: "Factories/Health/AchievementsGrid",
  component: AchievementsGrid,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[360px] max-w-5xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof AchievementsGrid>;

export default meta;

type Story = StoryObj<typeof meta>;

/** A mix of earned and not-yet-earned achievements. */
export const Default: Story = {
  args: { achievements: ACHIEVEMENTS },
};
