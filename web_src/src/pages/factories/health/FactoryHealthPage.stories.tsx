import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";
import {
  ACHIEVEMENTS,
  HEALTH_SNAPSHOT,
  RECURRING_PATTERNS,
  RUN_SUMMARY_FAILED,
  STREAKS,
} from "@/pages/factories/verification/__fixtures__/verificationFixtures";

import { FactoryHealthPage } from "./FactoryHealthPage";

/**
 * The full Health tab for a factory: score, streaks, recurring patterns,
 * achievements, and the latest run summary.
 */
const meta = {
  title: "Factories/Health/FactoryHealthPage",
  component: FactoryHealthPage,
  parameters: { layout: "fullscreen" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-screen w-full">
        <Story />
      </ComponentStoryShell>
    ),
  ],
  args: {
    onViewSuggestions: (patternId) => console.log("view suggestions", patternId),
  },
} satisfies Meta<typeof FactoryHealthPage>;

export default meta;

type Story = StoryObj<typeof meta>;

/** A factory with an improving score and active streaks. */
export const Default: Story = {
  args: {
    factoryName: "Shipping",
    snapshot: HEALTH_SNAPSHOT,
    streaks: STREAKS,
    patterns: RECURRING_PATTERNS,
    achievements: ACHIEVEMENTS,
    latestReport: RUN_SUMMARY_FAILED,
  },
};
