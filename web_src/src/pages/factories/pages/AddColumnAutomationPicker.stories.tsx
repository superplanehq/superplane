import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { Button } from "@/components/ui/button";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { catalogForColumn } from "../lib/columnAutomations";
import { AddColumnAutomationPicker } from "./AddColumnAutomationPicker";

const meta = {
  title: "Factories/Components/AddColumnAutomationPicker",
  component: AddColumnAutomationPicker,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="flex min-h-[420px] items-center justify-center bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof AddColumnAutomationPicker>;

export default meta;

type Story = StoryObj<typeof meta>;

function OpenPicker({ catalog, takenIds = [] }: { catalog: ReturnType<typeof catalogForColumn>; takenIds?: string[] }) {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button type="button" onClick={() => setOpen(true)}>
        Add automation
      </Button>
      <AddColumnAutomationPicker
        open={open}
        onClose={() => setOpen(false)}
        onSelect={() => setOpen(false)}
        catalog={catalog}
        takenIds={takenIds}
      />
    </>
  );
}

export const BacklogCatalog: Story = {
  name: "Backlog catalog",
  render: () => <OpenPicker catalog={catalogForColumn("backlog")} takenIds={["github-issues"]} />,
};

export const PhaseCatalog: Story = {
  name: "Phase catalog",
  render: () => <OpenPicker catalog={catalogForColumn("phase-0")} />,
};

export const VerifyCatalog: Story = {
  name: "Verify catalog",
  render: () => <OpenPicker catalog={catalogForColumn("verify")} takenIds={["discussion"]} />,
};

export const DoneCatalog: Story = {
  name: "Done catalog",
  render: () => <OpenPicker catalog={catalogForColumn("done")} takenIds={["pr-closure"]} />,
};
