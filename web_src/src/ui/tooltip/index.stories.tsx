import type { Meta, StoryObj } from "@storybook/react";
import { Plus } from "lucide-react";

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./index";

const meta = {
  title: "shadcn Primitives/Tooltip",
  component: TooltipContent,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    side: {
      control: { type: "radio" },
      options: ["top", "bottom", "left", "right"],
      table: { category: "Layout" },
    },
    align: {
      control: { type: "radio" },
      options: ["start", "center", "end"],
      table: { category: "Layout" },
    },
    children: {
      control: { type: "text" },
      table: { category: "Content" },
    },
  },
  args: {
    side: "top",
    align: "center",
    children: "Add to library",
  },
  render: (args) => (
    <TooltipProvider delayDuration={150}>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-input bg-background text-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50"
          >
            <Plus className="h-4 w-4" />
            <span className="sr-only">Add</span>
          </button>
        </TooltipTrigger>
        <TooltipContent {...args} />
      </Tooltip>
    </TooltipProvider>
  ),
} satisfies Meta<typeof TooltipContent>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Bottom: Story = {
  args: {
    side: "bottom",
  },
};

export const Left: Story = {
  args: {
    side: "left",
  },
};

export const Right: Story = {
  args: {
    side: "right",
  },
};
