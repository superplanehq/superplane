import type { Meta, StoryObj } from "@storybook/react";
import { BadgeCheck, ChevronRight, ShieldAlert } from "lucide-react";

import { Button } from "../button";

import { Item, ItemActions, ItemContent, ItemDescription, ItemMedia, ItemTitle } from "./index";

type ItemArgs = React.ComponentProps<typeof Item>;

const meta = {
  title: "shadcn Primitives/Item",
  component: Item,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    variant: {
      control: { type: "radio" },
      options: ["default", "outline", "muted"],
    },
    size: {
      control: { type: "radio" },
      options: ["default", "sm"],
    },
    asChild: {
      control: { type: "boolean" },
    },
    className: {
      control: { type: "text" },
    },
  },
  args: {
    variant: "outline",
    size: "sm",
  },
  render: (args) => (
    <div className="w-[360px]">
      <Item {...args}>
        <ItemMedia variant="icon">
          <ShieldAlert className="text-amber-600" />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>Security Alert</ItemTitle>
          <ItemDescription>New login detected from unknown device.</ItemDescription>
        </ItemContent>
        <ItemActions>
          <Button size="sm" variant="outline">
            Review
          </Button>
        </ItemActions>
      </Item>
    </div>
  ),
} satisfies Meta<ItemArgs>;

export default meta;

type Story = StoryObj<ItemArgs>;

export const Default: Story = {};

export const Muted: Story = {
  args: {
    variant: "muted",
  },
};

export const Compact: Story = {
  render: () => (
    <div className="w-[360px]">
      <Item variant="outline" size="sm" asChild>
        <a href="#">
          <ItemMedia>
            <BadgeCheck className="size-5 text-green-700" />
          </ItemMedia>
          <ItemContent>
            <ItemTitle>Your profile has been verified.</ItemTitle>
          </ItemContent>
          <ItemActions>
            <ChevronRight className="size-4 text-muted-foreground" />
          </ItemActions>
        </a>
      </Item>
    </div>
  ),
};
