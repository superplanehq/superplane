import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { WorkOrderMarkdownArtifactDialog } from "./WorkOrderMarkdownArtifactDialog";

/**
 * Dialog that renders a markdown work-order artifact's body in full,
 * using the app's shared MarkdownContent pipeline (GFM + sanitized raw HTML).
 * Opened from the artifacts sidebar and the activity timeline when the user
 * clicks the artifact label (the user-provided title, or "Note" if none).
 */
const meta = {
  title: "Factories/Components/WorkOrderMarkdownArtifactDialog",
  component: WorkOrderMarkdownArtifactDialog,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-96 bg-white p-4 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderMarkdownArtifactDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

const SAMPLE_BODY = `## Investigation notes

Retry policy exceeded the idempotency window when the ledger writer was under load.

- **Repro**: replay batch \`2026-08-05-14\` against staging.
- **Fix**: widen the idempotency window to 24h and dedupe on \`request_id\`.

See the [design doc](https://example.com/design) for the write-path diagram.
`;

/**
 * Controlled story that lets the user actually dismiss the dialog (Escape,
 * backdrop click, or the top-right close button), then reopen it.
 */
export const WithTitle: Story = {
  args: {
    open: true,
    onClose: () => {},
    title: "Investigation notes",
    body: SAMPLE_BODY,
  },
  render: (args) => {
    const [open, setOpen] = useState(args.open);

    return (
      <>
        <button
          type="button"
          onClick={() => setOpen(true)}
          className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 shadow-sm hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100 dark:hover:bg-gray-700"
        >
          Open dialog
        </button>
        <WorkOrderMarkdownArtifactDialog {...args} open={open} onClose={() => setOpen(false)} />
      </>
    );
  },
};

/** No title — the calling components fall back to "Note". */
export const FallbackTitle: Story = {
  args: {
    open: true,
    onClose: () => {},
    title: "Note",
    body: SAMPLE_BODY,
  },
};

/** Empty body — placeholder replaces the markdown viewport. */
export const EmptyBody: Story = {
  args: {
    open: true,
    onClose: () => {},
    title: "Note",
    body: "",
  },
};
