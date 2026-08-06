import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { WorkOrderMarkdownArtifactDialog } from "./WorkOrderMarkdownArtifactDialog";

/**
 * Dialog that renders a markdown work-order artifact's body in full,
 * using the app's shared MarkdownContent pipeline (GFM + sanitized raw HTML).
 * Opened from the artifacts sidebar and the activity timeline when the user
 * clicks the artifact label (the user-provided title, or "Note" if none).
 */
const meta = {
  title: "Factories/WorkOrderMarkdownArtifactDialog",
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

/** User provided a title — the dialog shows it verbatim. */
export const WithTitle: Story = {
  args: {
    open: true,
    onClose: () => {},
    title: "Investigation notes",
    body: SAMPLE_BODY,
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
