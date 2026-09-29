import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "@/components/ui/button";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { REFUND_FACTORY_LINES } from "./__fixtures__/factoryPageResponses";
import { DispatchWorkOrderPopover } from "./DispatchWorkOrderPopover";

/**
 * Popover for dispatching a task to a specific factory line. Wraps
 * any trigger element via `children`; opens a line picker + Dispatch button.
 */
const meta = {
  title: "Factories/Components/DispatchWorkOrderPopover",
  component: DispatchWorkOrderPopover,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="flex min-h-[280px] items-center justify-center bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof DispatchWorkOrderPopover>;

export default meta;

type Story = StoryObj<typeof meta>;

/** With lines: select from `plan-and-implement` / `hotfix` and dispatch. */
export const WithLines: Story = {
  args: {
    lines: REFUND_FACTORY_LINES,
    isSaving: false,
    canDispatch: true,
    align: "center",
    onDispatch: async (input) => {
      console.log("dispatch to", input);
    },
    children: <Button type="button">Dispatch</Button>,
  },
};

/** Saving state — button shows the loading label. */
export const Saving: Story = {
  args: {
    lines: REFUND_FACTORY_LINES,
    isSaving: true,
    canDispatch: true,
    align: "center",
    onDispatch: async () => undefined,
    children: <Button type="button">Dispatch</Button>,
  },
};

/** No lines configured — popover shows guidance to define a line first. */
export const NoLines: Story = {
  name: "No Lines",
  args: {
    lines: [],
    isSaving: false,
    canDispatch: true,
    align: "center",
    onDispatch: async () => undefined,
    children: <Button type="button">Dispatch</Button>,
  },
};
