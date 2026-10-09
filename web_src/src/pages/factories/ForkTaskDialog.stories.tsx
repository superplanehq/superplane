import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";

import { Button } from "@/components/ui/button";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "./__fixtures__/factoriesStoryTheme";
import { ForkTaskDialog, type ForkTaskTarget } from "./ForkTaskDialog";

const queryClient = new QueryClient();

const baseTarget: ForkTaskTarget = {
  organizationId: "org-1",
  factoryId: "factory-1",
  factoryKey: "SP",
  orderId: "order-1",
  hasPlan: true,
  canFork: true,
};

const meta = {
  title: "Factories/Components/ForkTaskDialog",
  component: ForkTaskDialog,
  parameters: { layout: "centered" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <QueryClientProvider client={queryClient}>
        <ComponentStoryShell className="flex min-h-[520px] items-center justify-center bg-background p-6">
          <Story />
        </ComponentStoryShell>
      </QueryClientProvider>
    ),
  ],
} satisfies Meta<typeof ForkTaskDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

function OpenDialog({ target, isPending = false }: { target: ForkTaskTarget; isPending?: boolean }) {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button type="button" variant="outline" onClick={() => setOpen(true)}>
        Open fork dialog
      </Button>
      <ForkTaskDialog open={open} onOpenChange={setOpen} target={target} isPending={isPending} />
    </>
  );
}

/** Both choices are available. Copy request is selected. */
export const Default: Story = {
  render: () => <OpenDialog target={baseTarget} />,
};

/** Copy plan stays disabled when the source task has no plan. */
export const MissingPlan: Story = {
  render: () => <OpenDialog target={{ ...baseTarget, hasPlan: false }} />,
};

/** Both choices and the submit action stay disabled without permission. */
export const DeniedPermission: Story = {
  render: () => <OpenDialog target={{ ...baseTarget, canFork: false }} />,
};

/** The submit action shows that the fork request is in progress. */
export const Pending: Story = {
  render: () => <OpenDialog target={baseTarget} isPending />,
};
