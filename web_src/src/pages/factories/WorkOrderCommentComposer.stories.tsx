import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { WorkOrderCommentComposer } from "./WorkOrderCommentComposer";

/**
 * Inline composer on the work-order detail page. Users type into the textarea
 * and submit; the same component powers automation-facing usage where the
 * backend attributes the comment to a system/LLM author.
 */
const meta = {
  title: "Factories/Components/WorkOrderCommentComposer",
  component: WorkOrderCommentComposer,
  parameters: { layout: "padded" },
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

const logSubmit = async (body: string) => {
  console.log("comment", body);
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
