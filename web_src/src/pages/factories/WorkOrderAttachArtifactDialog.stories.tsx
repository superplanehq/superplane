import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { Button } from "@/components/ui/button";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { WorkOrderAttachArtifactDialog } from "./WorkOrderAttachArtifactDialog";

/**
 * Modal for attaching a PR or markdown note to a work order. Wrapped in a
 * story-level state harness so the dialog can be reopened after submit.
 */
const meta = {
  title: "Factories/WorkOrderAttachArtifactDialog",
  component: WorkOrderAttachArtifactDialog,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="flex min-h-[420px] items-center justify-center bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderAttachArtifactDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

function InteractiveDialog({ isSubmitting = false }: { isSubmitting?: boolean }) {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button type="button" variant="outline" onClick={() => setOpen(true)}>
        Open attach artifact dialog
      </Button>
      <WorkOrderAttachArtifactDialog
        open={open}
        onOpenChange={setOpen}
        isSubmitting={isSubmitting}
        onSubmit={async (artifact) => {
          console.log("attach", artifact);
          setOpen(false);
        }}
      />
    </>
  );
}

/** Default — dialog open on the PR tab (URL required). */
export const Open: Story = {
  render: () => <InteractiveDialog />,
};

/** Submitting — Attach button locked while the mutation is in flight. */
export const Submitting: Story = {
  render: () => <InteractiveDialog isSubmitting />,
};
