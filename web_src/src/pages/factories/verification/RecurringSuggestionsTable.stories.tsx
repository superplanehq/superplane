import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";

import { RecurringSuggestionsTable } from "./RecurringSuggestionsTable";
import { RECURRING_SUGGESTION_ROWS } from "./__fixtures__/verificationFixtures";

/**
 * Factory-level aggregation of repeated suggestions. Rows link to the
 * matching pattern card on the Health tab.
 */
const meta = {
  title: "Factories/Verification/RecurringSuggestionsTable",
  component: RecurringSuggestionsTable,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[320px] max-w-4xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
  args: {
    onOpenPattern: (rowId) => console.log("open pattern", rowId),
  },
} satisfies Meta<typeof RecurringSuggestionsTable>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Patterns ordered by open count, with trend and last-seen time. */
export const Populated: Story = {
  args: { rows: RECURRING_SUGGESTION_ROWS },
};

/** No repeated findings yet. */
export const Empty: Story = {
  args: { rows: [] },
};
