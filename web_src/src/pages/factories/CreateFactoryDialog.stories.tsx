import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { Button } from "@/components/ui/button";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { CreateFactoryDialog } from "./CreateFactoryDialog";

/**
 * Create factory modal: name (required) + description (optional). Wraps in
 * a story-level state harness so the dialog can be opened and re-opened.
 */
const meta = {
  title: "Factories/Components/CreateFactoryDialog",
  component: CreateFactoryDialog,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="flex min-h-[400px] items-center justify-center bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof CreateFactoryDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

function InteractiveDialog({ isSaving = false }: { isSaving?: boolean }) {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button type="button" variant="outline" onClick={() => setOpen(true)}>
        Open create factory dialog
      </Button>
      <CreateFactoryDialog
        open={open}
        isSaving={isSaving}
        onClose={() => setOpen(false)}
        onCreate={async (input) => {
          console.log("create factory", input);
          setOpen(false);
        }}
      />
    </>
  );
}

/** Default: dialog open with empty form. */
export const Open: Story = {
  render: () => <InteractiveDialog />,
};

/** Saving state: submit button in loading state, cancel disabled. */
export const Saving: Story = {
  render: () => <InteractiveDialog isSaving />,
};
