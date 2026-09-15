import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";

import { SuggestionCard } from "./SuggestionCard";
import { SUGGESTIONS } from "./__fixtures__/verificationFixtures";

/**
 * One open finding as an actionable suggestion, with the dispatch-fix action
 * that routes the fix as a draft work order or a direct agent run.
 */
const meta = {
  title: "Factories/Verification/SuggestionCard",
  component: SuggestionCard,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[280px] max-w-2xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
  args: {
    onDispatchFix: (suggestionId, target) => console.log("dispatch fix", suggestionId, target),
    onDismiss: (suggestionId) => console.log("dismiss", suggestionId),
    onAccept: (suggestionId) => console.log("accept", suggestionId),
  },
} satisfies Meta<typeof SuggestionCard>;

export default meta;

type Story = StoryObj<typeof meta>;

/** A blocking suggestion with several occurrences. */
export const Blocking: Story = {
  args: { suggestion: SUGGESTIONS[0] },
};

/** A suggestion whose fix was already dispatched. */
export const FixInProgress: Story = {
  args: { suggestion: SUGGESTIONS[1] },
};

/** An advisory suggestion; it never stops the line. */
export const Advisory: Story = {
  args: { suggestion: SUGGESTIONS[2] },
};
