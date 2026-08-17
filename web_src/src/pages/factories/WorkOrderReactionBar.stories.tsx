import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "./__fixtures__/factoriesStoryTheme";
import { OPEN_WORK_ORDER } from "./__fixtures__/factoryPageResponses";
import { WorkOrderDescription } from "./WorkOrderDescription";
import { WorkOrderDetailHeader } from "./WorkOrderDetailHeader";
import { WorkOrderReactionBar } from "./WorkOrderReactionBar";
import { toggleWorkOrderReaction, type WorkOrderReaction } from "./lib/workOrderReactions";

/**
 * Reaction bar for the work order detail page — a row of small emoji chips
 * (with counts) sitting under the title, plus a compact add-reaction
 * control. Clicking a chip you've picked removes your reaction; clicking
 * one you haven't picked adds you to it. Presentational only: the parent
 * owns the reaction list and passes an `onToggleReaction(emoji)` callback.
 *
 * Prototype note: this is not wired into the real work order page, store,
 * or API — every handler below just `console.log`s.
 */
const meta = {
  title: "Factories/Components/WorkOrderReactionBar",
  component: WorkOrderReactionBar,
  parameters: { layout: "fullscreen" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-[220px] bg-background p-8">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderReactionBar>;

export default meta;

type Story = StoryObj<typeof meta>;

const onToggleReaction = (emoji: string) => {
  console.log("toggle reaction", emoji);
};

/** No reactions yet — only the subtle "+" add-reaction control is shown. */
export const Empty: Story = {
  args: {
    reactions: [],
    canReact: true,
    onToggleReaction,
  },
};

/** A handful of reactions with mixed counts, none of them the viewer's own. */
export const MixedReactions: Story = {
  args: {
    reactions: [
      { emoji: "👍", count: 3, mine: false, reactorNames: ["Alice Anderson", "Bob Brown", "Priya Shah"] },
      { emoji: "👀", count: 1, mine: false, reactorNames: ["Priya Shah"] },
      { emoji: "🎉", count: 2, mine: false, reactorNames: ["Bob Brown", "Alice Anderson"] },
    ],
    canReact: true,
    onToggleReaction,
  },
};

/**
 * One of the chips is the viewer's own reaction — it's highlighted with a
 * tinted border. Clicking it removes the reaction.
 */
export const MyReactionHighlighted: Story = {
  name: "My Reaction (Highlighted)",
  args: {
    reactions: [
      { emoji: "👍", count: 4, mine: true, reactorNames: ["You", "Alice Anderson", "Bob Brown", "Priya Shah"] },
      { emoji: "😕", count: 1, mine: false, reactorNames: ["Bob Brown"] },
    ],
    canReact: true,
    onToggleReaction,
  },
};

/** Clicking the add-reaction control opens the curated emoji picker. */
export const PickerOpen: Story = {
  name: "Emoji Picker Open",
  args: {
    reactions: [{ emoji: "👍", count: 1, mine: false, reactorNames: ["Alice Anderson"] }],
    canReact: true,
    onToggleReaction,
  },
  play: async ({ canvasElement }) => {
    const button = canvasElement.querySelector<HTMLButtonElement>('[data-testid="work-order-reaction-add-button"]');
    button?.click();
  },
};

/**
 * Viewer without permission to react: existing chips are still visible, but
 * neither the chips nor the add control respond to clicks (mirrors the
 * `PermissionTooltip` pattern used for comments).
 */
export const ReadOnly: Story = {
  args: {
    reactions: [
      { emoji: "👍", count: 2, mine: false, reactorNames: ["Alice Anderson", "Bob Brown"] },
      { emoji: "🚀", count: 1, mine: false, reactorNames: ["Priya Shah"] },
    ],
    canReact: false,
    onToggleReaction,
  },
};

/**
 * Fully interactive demo: local state simulates the add/remove toggle and a
 * brief "pending" look right after clicking, since there's no real network
 * call in this prototype.
 */
export const Interactive: Story = {
  render: () => <InteractiveReactionBar />,
};

function InteractiveReactionBar() {
  const [reactions, setReactions] = useState<WorkOrderReaction[]>([
    { emoji: "👍", count: 2, mine: false, reactorNames: ["Alice Anderson", "Bob Brown"] },
    { emoji: "🎉", count: 1, mine: true, reactorNames: ["You"] },
  ]);
  const [pendingEmoji, setPendingEmoji] = useState<string | null>(null);

  const handleToggle = (emoji: string) => {
    console.log("toggle reaction", emoji);
    setPendingEmoji(emoji);

    // No real network call in this prototype — just a brief pending look,
    // then the simulated local toggle settles.
    window.setTimeout(() => {
      setReactions((current) => toggleWorkOrderReaction(current, emoji));
      setPendingEmoji(null);
    }, 400);
  };

  return (
    <WorkOrderReactionBar reactions={reactions} canReact onToggleReaction={handleToggle} pendingEmoji={pendingEmoji} />
  );
}

/**
 * In-page placement: the reaction bar sits directly under the work order
 * header, above the description — a natural extension of the existing
 * layout rather than a bolted-on widget. Purely illustrative; the real
 * `WorkOrderDetailLoadedView` is not modified.
 */
export const InPageContext: Story = {
  render: () => <ReactionBarInContext />,
  parameters: { layout: "fullscreen" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-screen w-full bg-gray-50 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
};

function ReactionBarInContext() {
  const [reactions, setReactions] = useState<WorkOrderReaction[]>([
    { emoji: "👍", count: 3, mine: true, reactorNames: ["You", "Alice Anderson", "Bob Brown"] },
    { emoji: "👀", count: 1, mine: false, reactorNames: ["Priya Shah"] },
  ]);

  const handleToggle = (emoji: string) => {
    console.log("toggle reaction", emoji);
    setReactions((current) => toggleWorkOrderReaction(current, emoji));
  };

  return (
    <div className="mx-auto max-w-3xl px-6 py-6">
      <WorkOrderDetailHeader
        orderTitle={OPEN_WORK_ORDER.title ?? "Work Order"}
        orderIdentifier={OPEN_WORK_ORDER.key}
        backHref="/org-1/workspaces/RF/work-orders"
        displayStatus="waiting"
        isOpen
        isDispatchable
        isClosed={false}
        canClose
        canManage
        isCompleting={false}
        isRejecting={false}
        isClosing={false}
        isUpdatingStatus={false}
        onClose={(result) => console.log("close", result)}
        onStatusChange={async (state, result) => console.log("status change", state, result)}
      />
      <div className="mt-3">
        <WorkOrderReactionBar reactions={reactions} canReact onToggleReaction={handleToggle} />
      </div>
      <div className="mt-6">
        <WorkOrderDescription description={OPEN_WORK_ORDER.description ?? ""} />
      </div>
    </div>
  );
}
