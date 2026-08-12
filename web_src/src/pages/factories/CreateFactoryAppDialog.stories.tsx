import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { Button } from "@/components/ui/button";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { CreateFactoryAppDialog } from "./CreateFactoryAppDialog";

/**
 * Create factory app modal — same shape as CreateFactoryDialog, but scoped
 * to an app owned by a factory. Independent story so both dialogs surface
 * in the sidebar and can be reviewed in parallel.
 */
const meta = {
  title: "Factories/Components/CreateFactoryAppDialog",
  component: CreateFactoryAppDialog,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="flex min-h-[400px] items-center justify-center bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof CreateFactoryAppDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

function InteractiveDialog({ isSaving = false }: { isSaving?: boolean }) {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button type="button" variant="outline" onClick={() => setOpen(true)}>
        Open create app dialog
      </Button>
      <CreateFactoryAppDialog
        open={open}
        isSaving={isSaving}
        onClose={() => setOpen(false)}
        onCreate={async (input) => {
          console.log("create factory app", input);
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
