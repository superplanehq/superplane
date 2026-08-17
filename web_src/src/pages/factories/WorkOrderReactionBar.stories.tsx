import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { WorkOrderReactionBar } from "./WorkOrderReactionBar";

/**
 * Reaction bar for a work order (not a comment). Sits under the work order
 * title on the detail page. Backed by local React state via the Storybook
 * `WorkOrderReactionsSlotContext` in `Factories/Pages/Work Order Detail`;
 * these stories drive the presentational component directly.
 */
const meta = {
  title: "Factories/Components/WorkOrderReactionBar",
  component: WorkOrderReactionBar,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[160px] bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderReactionBar>;

export default meta;

type Story = StoryObj<typeof meta>;

const logToggle = (emoji: string) => {
  console.log("toggle reaction", emoji);
};

/** No reactions yet — just the subtle "Add reaction" affordance. */
export const Empty: Story = {
  args: {
    reactions: [],
    myReaction: null,
    onToggle: logToggle,
  },
};

/** One emoji, one reactor — the simplest populated pill. */
export const OneReactor: Story = {
  name: "One reactor",
  args: {
    reactions: [{ emoji: "👍", reactorNames: ["Alex Reviewer"] }],
    myReaction: null,
    onToggle: logToggle,
  },
};

/** Multiple emojis, multiple reactors each — none of them is the current viewer. */
export const Populated: Story = {
  args: {
    reactions: [
      { emoji: "👍", reactorNames: ["Alex Reviewer", "Jamie Operator", "Priya Kapoor"] },
      { emoji: "🎉", reactorNames: ["Alex Reviewer"] },
      { emoji: "👀", reactorNames: ["Jamie Operator", "Priya Kapoor"] },
    ],
    myReaction: null,
    onToggle: logToggle,
  },
};

/** Mine — the pill matching the current viewer's reaction is highlighted and toggleable. */
export const Mine: Story = {
  args: {
    reactions: [
      { emoji: "👍", reactorNames: ["Alex Reviewer", "Jamie Operator", "Storybook User"] },
      { emoji: "🎉", reactorNames: ["Alex Reviewer"] },
    ],
    myReaction: "👍",
    onToggle: logToggle,
  },
};

/** Picker open — clicking "Add reaction" reveals the curated emoji set. */
export const PickerOpen: Story = {
  name: "Picker open",
  args: {
    reactions: [{ emoji: "👍", reactorNames: ["Alex Reviewer"] }],
    myReaction: null,
    onToggle: logToggle,
    defaultPickerOpen: true,
  },
};
