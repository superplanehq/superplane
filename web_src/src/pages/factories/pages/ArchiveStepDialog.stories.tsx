import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { Button } from "@/components/ui/button";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { ArchiveStepDialog } from "./ArchiveStepDialog";

/**
 * Phase-column archive popup. Empty columns ask for confirmation. Columns
 * that still have tasks only offer Close.
 */
const meta = {
  title: "Factories/Components/ArchiveStepDialog",
  component: ArchiveStepDialog,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="flex min-h-[360px] items-center justify-center bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof ArchiveStepDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

function InteractiveArchiveDialog({
  hasTasks,
  isLastStep = false,
  stepName = "Implement",
}: {
  hasTasks: boolean;
  isLastStep?: boolean;
  stepName?: string;
}) {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button type="button" variant="outline" onClick={() => setOpen(true)}>
        Open archive dialog
      </Button>
      <ArchiveStepDialog
        open={open}
        stepName={stepName}
        hasTasks={hasTasks}
        isLastStep={isLastStep}
        onOpenChange={setOpen}
        onConfirm={() => {
          console.log("archive step", stepName);
          setOpen(false);
        }}
      />
    </>
  );
}

/** Empty column: Cancel and Archive. */
export const Confirm: Story = {
  args: {
    open: true,
    stepName: "Implement",
    hasTasks: false,
    onOpenChange: () => undefined,
    onConfirm: () => undefined,
  },
  render: () => <InteractiveArchiveDialog hasTasks={false} />,
};

/** Column still has tasks: Close only. */
export const Blocked: Story = {
  args: {
    open: true,
    stepName: "Plan",
    hasTasks: true,
    onOpenChange: () => undefined,
    onConfirm: () => undefined,
  },
  render: () => <InteractiveArchiveDialog hasTasks stepName="Plan" />,
};

/** Only one step remains: Close only. */
export const LastStep: Story = {
  args: {
    open: true,
    stepName: "Implement",
    hasTasks: false,
    isLastStep: true,
    onOpenChange: () => undefined,
    onConfirm: () => undefined,
  },
  render: () => <InteractiveArchiveDialog hasTasks={false} isLastStep />,
};
