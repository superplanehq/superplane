import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";

import { WorkOrderSuggestionsList } from "./WorkOrderSuggestionsList";
import { SUGGESTIONS } from "./__fixtures__/verificationFixtures";

/**
 * Open suggestions for one work order with severity and domain filters. The
 * empty state confirms a clean verification run.
 */
const meta = {
  title: "Factories/Verification/WorkOrderSuggestionsList",
  component: WorkOrderSuggestionsList,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[480px] max-w-3xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
  args: {
    onDispatchFix: (suggestionId, target) => console.log("dispatch fix", suggestionId, target),
    onDismiss: (suggestionId) => console.log("dismiss", suggestionId),
    onAccept: (suggestionId) => console.log("accept", suggestionId),
  },
} satisfies Meta<typeof WorkOrderSuggestionsList>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Open suggestions across severities and domains. */
export const Populated: Story = {
  args: { suggestions: SUGGESTIONS },
};

/** No open suggestions: the last verification run was clean. */
export const Empty: Story = {
  args: { suggestions: [] },
};
