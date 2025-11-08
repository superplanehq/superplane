import type { Meta, StoryObj } from "@storybook/react";
import { Pencil, Plus, Share2 } from "lucide-react";

import { Button } from "../button";

import { ButtonGroup, ButtonGroupSeparator, ButtonGroupText } from "./index";

type ButtonGroupArgs = React.ComponentProps<typeof ButtonGroup>;

const meta = {
  title: "shadcn Primitives/ButtonGroup",
  component: ButtonGroup,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    orientation: {
      control: { type: "radio" },
      options: ["horizontal", "vertical"],
    },
    className: {
      control: { type: "text" },
    },
  },
  args: {
    orientation: "horizontal",
  },
} satisfies Meta<ButtonGroupArgs>;

export default meta;

type Story = StoryObj<ButtonGroupArgs>;

export const Default: Story = {
  render: (args) => (
    <ButtonGroup {...args}>
      <Button variant="outline">Copy</Button>
      <Button variant="outline">Paste</Button>
      <Button variant="outline">Cut</Button>
    </ButtonGroup>
  ),
};

export const Vertical: Story = {
  args: {
    orientation: "vertical",
  },
  render: (args) => (
    <ButtonGroup {...args}>
      <Button size="icon" variant="outline" aria-label="Add">
        <Plus className="h-4 w-4" />
      </Button>
      <Button size="icon" variant="outline" aria-label="Edit">
        <Pencil className="h-4 w-4" />
      </Button>
      <Button size="icon" variant="outline" aria-label="Share">
        <Share2 className="h-4 w-4" />
      </Button>
    </ButtonGroup>
  ),
};

export const WithSeparator: Story = {
  render: (args) => (
    <ButtonGroup {...args}>
      <Button variant="secondary">Publish</Button>
      <ButtonGroupSeparator />
      <Button variant="secondary" size="icon" aria-label="Share">
        <Share2 className="h-4 w-4" />
      </Button>
    </ButtonGroup>
  ),
};

export const WithLabel: Story = {
  render: (args) => (
    <ButtonGroup {...args}>
      <ButtonGroupText>Filters</ButtonGroupText>
      <Button variant="outline">All</Button>
      <Button variant="outline">Active</Button>
      <Button variant="outline">Archived</Button>
    </ButtonGroup>
  ),
};
