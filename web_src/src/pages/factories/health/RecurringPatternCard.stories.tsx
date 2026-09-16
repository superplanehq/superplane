import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";
import { RECURRING_PATTERNS } from "@/pages/factories/verification/__fixtures__/verificationFixtures";

import { RecurringPatternCard } from "./RecurringPatternCard";

/**
 * One recurring finding pattern: plain description, top offender files,
 * standard remediation, and the occurrence trend.
 */
const meta = {
  title: "Factories/Health/RecurringPatternCard",
  component: RecurringPatternCard,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[380px] max-w-xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
  args: {
    onViewSuggestions: (patternId) => console.log("view suggestions", patternId),
  },
} satisfies Meta<typeof RecurringPatternCard>;

export default meta;

type Story = StoryObj<typeof meta>;

/** A pattern whose occurrence count rises. */
export const Rising: Story = {
  args: { pattern: RECURRING_PATTERNS[0] },
};

/** A pattern whose occurrence count falls. */
export const Falling: Story = {
  args: { pattern: RECURRING_PATTERNS[1] },
};
