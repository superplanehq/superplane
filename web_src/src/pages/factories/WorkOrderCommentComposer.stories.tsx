import type { Meta, StoryObj } from "@storybook/react-vite";

import type { SuperplaneUsersUser } from "@/api-client";
import {
  FACTORIES_ORGANIZATION_ID,
  ORGANIZATION_USERS,
  toStorybookOrganizationUser,
} from "./__fixtures__/factoryPageResponses";
import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { WorkOrderCommentComposer } from "./WorkOrderCommentComposer";

const storyMembers: SuperplaneUsersUser[] = ORGANIZATION_USERS.map(toStorybookOrganizationUser);

/**
 * Inline composer on the work-order detail page. Type @ to mention an
 * organization member. Submit sends the comment body and selected user IDs.
 */
const meta = {
  title: "Factories/Components/WorkOrderCommentComposer",
  component: WorkOrderCommentComposer,
  parameters: { layout: "padded" },
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    members: storyMembers,
  },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[220px] max-w-2xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
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
