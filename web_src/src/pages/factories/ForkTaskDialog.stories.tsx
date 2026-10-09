import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { useState } from "react";
import { Route, Routes, useParams } from "react-router";

import { Button } from "@/components/ui/button";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "./__fixtures__/factoriesStoryTheme";
import { ForkTaskDialog, type ForkTaskTarget } from "./ForkTaskDialog";

const queryClient = new QueryClient();

const forkedOrder = {
  id: "order-forked",
  number: "42",
  title: "Forked task",
  state: "STATE_DRAFT",
};

const forkHandler = http.post("*/api/v1/factories/:factoryId/orders/:orderId/fork", () =>
  HttpResponse.json({ order: forkedOrder }),
);

const storyPath = "/org-1/workspaces/sp";

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
  parameters: {
    layout: "centered",
    msw: { handlers: [forkHandler] },
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <QueryClientProvider client={queryClient}>
        <ComponentStoryShell
          initialPath={storyPath}
          className="flex min-h-[520px] items-center justify-center bg-background p-6"
        >
          <Story />
        </ComponentStoryShell>
      </QueryClientProvider>
    ),
  ],
} satisfies Meta<typeof ForkTaskDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

function OpenDialog({ target, isPending = false }: { target: ForkTaskTarget; isPending?: boolean }) {
  return (
    <Routes>
      <Route path={storyPath} element={<DialogControls target={target} isPending={isPending} />} />
      <Route path={`${storyPath}/task/:number`} element={<ForkedTaskNote />} />
    </Routes>
  );
}

function DialogControls({ target, isPending = false }: { target: ForkTaskTarget; isPending?: boolean }) {
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

function ForkedTaskNote() {
  const { number } = useParams();
  return <p data-testid="forked-task">Forked task {number}</p>;
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
