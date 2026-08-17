import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { organizationKeys } from "@/hooks/useOrganizationData";
import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { FACTORIES_ORGANIZATION_ID, ORGANIZATION_USERS } from "./__fixtures__/factoryPageResponses";
import { WorkOrderCommentComposer } from "./WorkOrderCommentComposer";

/**
 * Inline composer on the work-order detail page. Users type into the textarea
 * and submit; the same component powers automation-facing usage where the
 * backend attributes the comment to a system/LLM author. Typing `@` opens a
 * keyboard-navigable member picker (`WorkOrderMentionMenu`) that inserts a
 * `@[Name](user:id)` token — rendered as a chip once posted (see
 * `WorkOrderActivityTimeline`'s "With Comments & Artifacts" story).
 */

// The picker reads organization members from the `useOrganizationUsers`
// cache. Component stories don't run through the app's real API, so we seed
// a private `QueryClient` with the same fixture users used elsewhere in the
// Factories stories, mirroring `NodeChip.stories.tsx`'s pattern for chips
// that resolve names from a query cache.
function createSeededClient() {
  const client = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity, retry: false } } });
  client.setQueryData(
    organizationKeys.users(FACTORIES_ORGANIZATION_ID),
    ORGANIZATION_USERS.map((user) => ({
      metadata: { id: user.id, email: user.email },
      spec: { displayName: user.name },
    })),
  );
  return client;
}

const meta = {
  title: "Factories/Components/WorkOrderCommentComposer",
  component: WorkOrderCommentComposer,
  parameters: { layout: "padded" },
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
  },
  decorators: [
    (Story) => (
      <QueryClientProvider client={createSeededClient()}>
        <ComponentStoryShell className="min-h-[220px] max-w-2xl bg-gray-50 p-6 dark:bg-gray-950">
          <Story />
        </ComponentStoryShell>
      </QueryClientProvider>
    ),
  ],
} satisfies Meta<typeof WorkOrderCommentComposer>;

export default meta;

type Story = StoryObj<typeof meta>;

const logSubmit = async (body: string, mentionedUserIds: string[]) => {
  console.log("comment", body, mentionedUserIds);
};

/** Default — empty textarea, submit disabled until the user types. */
export const Empty: Story = {
  args: {
    canComment: true,
    isSubmitting: false,
    onSubmit: logSubmit,
  },
};

/** Posting — submit locked while the mutation is in flight. */
export const Submitting: Story = {
  args: {
    canComment: true,
    isSubmitting: true,
    onSubmit: logSubmit,
  },
};

/** Viewer without update permission — textarea and buttons disabled. */
export const ReadOnly: Story = {
  name: "Read Only",
  args: {
    canComment: false,
    isSubmitting: false,
    onSubmit: logSubmit,
  },
};

/**
 * Type `@` in the textarea to open the member picker (seeded with the same
 * fixture members `useOrganizationUsers` would return); arrow keys move the
 * highlight, Enter/Tab inserts `@[Name](user:id)`, Escape dismisses.
 */
export const MentioningAMember: Story = {
  name: "Mentioning a Member",
  args: {
    canComment: true,
    isSubmitting: false,
    onSubmit: logSubmit,
  },
};
