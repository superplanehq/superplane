import type { Meta, StoryObj } from "@storybook/react";

import { Separator } from "./index";

type SeparatorArgs = React.ComponentProps<typeof Separator>;

const meta = {
  title: "shadcn Primitives/Separator",
  component: Separator,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    orientation: {
      control: { type: "radio" },
      options: ["horizontal", "vertical"],
    },
    decorative: {
      control: { type: "boolean" },
    },
    className: {
      control: { type: "text" },
    },
  },
  args: {
    orientation: "horizontal",
    decorative: true,
  },
} satisfies Meta<SeparatorArgs>;

export default meta;

type Story = StoryObj<SeparatorArgs>;

export const Horizontal: Story = {
  render: (args) => (
    <div className="w-64 space-y-4">
      <p className="text-sm text-muted-foreground">Use separators to divide related sections of content.</p>
      <Separator {...args} />
      <p className="text-sm text-muted-foreground">This illustrates the default horizontal orientation.</p>
    </div>
  ),
};

export const Vertical: Story = {
  args: {
    orientation: "vertical",
  },
  render: (args) => (
    <div className="flex h-24 items-center gap-4">
      <span className="text-sm">First</span>
      <Separator {...args} />
      <span className="text-sm">Second</span>
      <Separator {...args} />
      <span className="text-sm">Third</span>
    </div>
  ),
};
