import type { Meta, StoryObj } from "@storybook/react-vite";
import { Toaster } from "sonner";
import { useState } from "react";

import { showErrorToast } from "@/lib/toast";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { STORYBOOK_ME_USER_ID, STORYBOOK_ME_USER_NAME } from "./__fixtures__/factoryPageResponses";
import { WorkOrderReactionBar, type WorkOrderReactionGroup } from "./WorkOrderReactionBar";

/**
 * Reaction bar for a work order — a row of emoji pill counters plus an
 * "add reaction" trigger, mirroring GitHub issue reactions. Purely
 * controlled: reaction data and permissions come in as props, toggles go
 * out via `onToggleReaction`. See `WorkOrderDetailLoadedView.stories.tsx`
 * ("With Reactions") for it composed into the full detail page.
 */
const meta = {
  title: "Factories/Components/WorkOrderReactionBar",
  component: WorkOrderReactionBar,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="flex min-h-[160px] min-w-[420px] items-center justify-center bg-gray-50 p-8 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderReactionBar>;

export default meta;

type Story = StoryObj<typeof meta>;

const ALEX = { id: "user-alex", name: "Alex Chen" };
const PRIYA = { id: "user-priya", name: "Priya Patel" };
const JORDAN = { id: "user-jordan", name: "Jordan Lee" };
const SAM = { id: "user-sam", name: "Sam Rivera" };
const TAYLOR = { id: "user-taylor", name: "Taylor Kim" };
const MORGAN = { id: "user-morgan", name: "Morgan Diaz" };
const ME = { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME };

const MIXED_REACTIONS: WorkOrderReactionGroup[] = [
  { emoji: "👍", users: [ME, ALEX, PRIYA] },
  { emoji: "🎉", users: [JORDAN] },
  { emoji: "👀", users: [ALEX, SAM] },
];

const MANY_REACTORS: WorkOrderReactionGroup[] = [
  { emoji: "🚀", users: [ALEX, PRIYA, JORDAN, SAM, TAYLOR, MORGAN] },
  { emoji: "👍", users: [ME] },
];

/** Empty state — nothing reacted yet, just the "Add reaction" trigger. */
export const Empty: Story = {
  args: {
    reactions: [],
    currentUserId: STORYBOOK_ME_USER_ID,
    canReact: true,
    onToggleReaction: (emoji) => console.log("toggle reaction", emoji),
  },
};

/** A few reactions with mixed ownership — the 👍 pill is mine (tinted). */
export const MixedReactions: Story = {
  name: "Mixed Reactions",
  args: {
    reactions: MIXED_REACTIONS,
    currentUserId: STORYBOOK_ME_USER_ID,
    canReact: true,
    onToggleReaction: (emoji) => console.log("toggle reaction", emoji),
  },
};

/** Hover the 🚀 pill — tooltip truncates to "Alex Chen, Priya Patel, and 4 others". */
export const ManyReactorsTooltip: Story = {
  name: "Many Reactors (tooltip truncation)",
  args: {
    reactions: MANY_REACTORS,
    currentUserId: STORYBOOK_ME_USER_ID,
    canReact: true,
    onToggleReaction: (emoji) => console.log("toggle reaction", emoji),
  },
};

/** The "+" trigger opened, showing the curated emoji set and stubbed search. */
export const PickerOpen: Story = {
  name: "Picker Open",
  args: {
    reactions: MIXED_REACTIONS,
    currentUserId: STORYBOOK_ME_USER_ID,
    canReact: true,
    defaultPickerOpen: true,
    onToggleReaction: (emoji) => console.log("toggle reaction", emoji),
  },
};

/** Closed/read-only work order — trigger disabled, same tooltip pattern as the comment composer. */
export const NoPermission: Story = {
  name: "No Permission",
  args: {
    reactions: MIXED_REACTIONS,
    currentUserId: STORYBOOK_ME_USER_ID,
    canReact: false,
    permissionMessage: "You don't have permission to react to this work order.",
    onToggleReaction: (emoji) => console.log("toggle reaction", emoji),
  },
};

/**
 * Interactive demo — click a pill or pick a new emoji to see the
 * optimistic pill update. The 😕 emoji is wired to always fail, reverting
 * the optimistic change and surfacing an error toast, matching the
 * comment composer's failure handling.
 */
export const OptimisticToggleWithFailure: Story = {
  name: "Optimistic Toggle (with simulated failure)",
  render: () => <OptimisticDemo />,
  args: {
    reactions: [],
    currentUserId: STORYBOOK_ME_USER_ID,
    canReact: true,
    onToggleReaction: () => undefined,
  },
};

function OptimisticDemo() {
  const [reactions, setReactions] = useState<WorkOrderReactionGroup[]>(MIXED_REACTIONS);
  const [pendingEmoji, setPendingEmoji] = useState<string | null>(null);

  const toggle = (emoji: string) => {
    const before = reactions;
    const hasMine = before
      .find((group) => group.emoji === emoji)
      ?.users.some((user) => user.id === STORYBOOK_ME_USER_ID);

    const after = hasMine
      ? before
          .map((group) =>
            group.emoji === emoji
              ? { ...group, users: group.users.filter((user) => user.id !== STORYBOOK_ME_USER_ID) }
              : group,
          )
          .filter((group) => group.users.length > 0)
      : upsertMine(before, emoji);

    setReactions(after);
    setPendingEmoji(emoji);

    const willFail = emoji === "😕";
    window.setTimeout(() => {
      setPendingEmoji(null);
      if (willFail) {
        setReactions(before);
        showErrorToast("Couldn't update your reaction. Try again.");
      }
    }, 500);
  };

  return (
    <>
      <Toaster position="bottom-center" closeButton />
      <div className="space-y-2">
        <WorkOrderReactionBar
          reactions={reactions}
          currentUserId={STORYBOOK_ME_USER_ID}
          canReact
          pendingEmoji={pendingEmoji}
          onToggleReaction={toggle}
        />
        <p className="text-xs text-muted-foreground">
          Picking 😕 always fails — watch it revert and toast an error after ~500ms.
        </p>
      </div>
    </>
  );
}

function upsertMine(groups: WorkOrderReactionGroup[], emoji: string): WorkOrderReactionGroup[] {
  const me = { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME };
  const existing = groups.find((group) => group.emoji === emoji);
  if (existing) {
    return groups.map((group) => (group.emoji === emoji ? { ...group, users: [...group.users, me] } : group));
  }
  return [...groups, { emoji, users: [me] }];
}
