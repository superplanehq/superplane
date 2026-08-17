import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { OPERATOR_USER, REVIEWER_USER, STORYBOOK_ME_USER_NAME } from "./__fixtures__/factoryPageResponses";
import { WorkOrderReactions, type WorkOrderReactionSummary } from "./WorkOrderReactions";

/**
 * Reaction "pills" strip for the work order detail page — a lightweight
 * alternative to leaving a comment just to acknowledge or +1 an order.
 * Purely presentational: reactions, counts, and "did I react" all come
 * from props, and every story below manages its own local state so
 * reviewers can click through add/remove/hover without a backend.
 */
const meta = {
  title: "Factories/Components/WorkOrderReactions",
  component: WorkOrderReactions,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[160px] max-w-2xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderReactions>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Interactive wrapper: owns the reaction list so pills actually toggle in Storybook. */
function InteractiveReactions({
  initialReactions,
  canReact = true,
  permissionsLoading = false,
  defaultPickerOpen = false,
}: {
  initialReactions: WorkOrderReactionSummary[];
  canReact?: boolean;
  permissionsLoading?: boolean;
  defaultPickerOpen?: boolean;
}) {
  const [reactions, setReactions] = useState(initialReactions);

  const handleToggle = (emoji: string) => {
    setReactions((current) =>
      current
        .map((reaction) =>
          reaction.emoji === emoji
            ? {
                ...reaction,
                count: reaction.reactedByMe ? reaction.count - 1 : reaction.count + 1,
                reactedByMe: !reaction.reactedByMe,
                reactorNames: reaction.reactedByMe
                  ? reaction.reactorNames.filter((name) => name !== "You")
                  : ["You", ...reaction.reactorNames],
              }
            : reaction,
        )
        .filter((reaction) => reaction.count > 0),
    );
  };

  const handlePickNew = (emoji: string) => {
    setReactions((current) => {
      const existing = current.find((reaction) => reaction.emoji === emoji);
      if (existing) {
        return existing.reactedByMe ? current : current.map((r) => (r.emoji === emoji ? bump(r) : r));
      }
      return [...current, { emoji, count: 1, reactedByMe: true, reactorNames: ["You"] }];
    });
  };

  return (
    <WorkOrderReactions
      reactions={reactions}
      canReact={canReact}
      permissionsLoading={permissionsLoading}
      defaultPickerOpen={defaultPickerOpen}
      onToggle={handleToggle}
      onPickNew={handlePickNew}
    />
  );
}

function bump(reaction: WorkOrderReactionSummary): WorkOrderReactionSummary {
  return { ...reaction, count: reaction.count + 1, reactedByMe: true, reactorNames: ["You", ...reaction.reactorNames] };
}

/** No reactions yet — only the subtle "add reaction" button shows. */
export const Empty: Story = {
  render: () => <InteractiveReactions initialReactions={[]} />,
};

/** A few reactions, mixed — some are mine (highlighted), some aren't. Click a pill to toggle, or "+" to add a new emoji. */
export const Mixed: Story = {
  render: () => (
    <InteractiveReactions
      initialReactions={[
        { emoji: "👍", count: 3, reactedByMe: true, reactorNames: ["You", REVIEWER_USER.name, OPERATOR_USER.name] },
        { emoji: "✅", count: 1, reactedByMe: false, reactorNames: [REVIEWER_USER.name] },
        { emoji: "👀", count: 2, reactedByMe: false, reactorNames: [OPERATOR_USER.name, REVIEWER_USER.name] },
      ]}
    />
  ),
};

/** Single reaction, not mine — clicking it adds my reaction and marks the pill as "yours." */
export const SingleReaction: Story = {
  name: "Single Reaction",
  render: () => (
    <InteractiveReactions
      initialReactions={[{ emoji: "🎉", count: 1, reactedByMe: false, reactorNames: [REVIEWER_USER.name] }]}
    />
  ),
};

/** Many reactions — the strip wraps onto a second row instead of overflowing. */
export const ManyReactions: Story = {
  name: "Many Reactions (overflow)",
  render: () => (
    <InteractiveReactions
      initialReactions={[
        { emoji: "👍", count: 5, reactedByMe: true, reactorNames: ["You", REVIEWER_USER.name, OPERATOR_USER.name] },
        { emoji: "✅", count: 3, reactedByMe: false, reactorNames: [REVIEWER_USER.name, OPERATOR_USER.name] },
        { emoji: "🎉", count: 2, reactedByMe: false, reactorNames: [REVIEWER_USER.name] },
        { emoji: "👀", count: 4, reactedByMe: true, reactorNames: ["You", OPERATOR_USER.name] },
        { emoji: "❤️", count: 1, reactedByMe: false, reactorNames: [REVIEWER_USER.name] },
        { emoji: "🚀", count: 6, reactedByMe: false, reactorNames: [OPERATOR_USER.name, REVIEWER_USER.name] },
        { emoji: "🔥", count: 2, reactedByMe: false, reactorNames: [OPERATOR_USER.name] },
        { emoji: "🙌", count: 1, reactedByMe: false, reactorNames: [REVIEWER_USER.name] },
      ]}
    />
  ),
};

/**
 * Permission check still loading — pills and the "+" button render but stay
 * disabled, matching the composer's `canComment`/`isSubmitting` pattern.
 */
export const Loading: Story = {
  render: () => (
    <InteractiveReactions
      permissionsLoading
      canReact={false}
      initialReactions={[
        { emoji: "👍", count: 3, reactedByMe: true, reactorNames: ["You", REVIEWER_USER.name, OPERATOR_USER.name] },
        { emoji: "✅", count: 1, reactedByMe: false, reactorNames: [REVIEWER_USER.name] },
      ]}
    />
  ),
};

/**
 * Read-only viewer — no permission to react. Pills render as static counts
 * (no hover-to-toggle) and the "+" button is hidden entirely.
 */
export const ReadOnly: Story = {
  name: "Read Only",
  render: () => (
    <InteractiveReactions
      canReact={false}
      initialReactions={[
        {
          emoji: "👍",
          count: 3,
          reactedByMe: false,
          reactorNames: [STORYBOOK_ME_USER_NAME, REVIEWER_USER.name, OPERATOR_USER.name],
        },
        { emoji: "✅", count: 1, reactedByMe: false, reactorNames: [REVIEWER_USER.name] },
      ]}
    />
  ),
};

/** Read-only + nothing to show — the strip renders nothing at all. */
export const ReadOnlyEmpty: Story = {
  name: "Read Only, No Reactions",
  render: () => <InteractiveReactions canReact={false} initialReactions={[]} />,
};

/**
 * Picker open — clicking "+" opens a curated quick-pick row plus a search
 * box backed by a wider emoji set (standing in for a full picker).
 */
export const PickerOpen: Story = {
  name: "Picker Open",
  render: () => (
    <InteractiveReactions
      defaultPickerOpen
      initialReactions={[
        { emoji: "👍", count: 2, reactedByMe: false, reactorNames: [REVIEWER_USER.name, OPERATOR_USER.name] },
      ]}
    />
  ),
};
