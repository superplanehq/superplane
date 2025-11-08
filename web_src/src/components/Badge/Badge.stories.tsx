import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { Badge } from "./badge";

const meta: Meta<typeof Badge> = {
  title: "Components/Badge",
  component: Badge,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    color: {
      control: "select",
      options: ["indigo", "gray", "blue", "red", "green", "yellow", "zinc"],
    },
    icon: {
      control: "text",
    },
    truncate: {
      control: "boolean",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    children: "Badge",
  },
};

export const Colors: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <Badge color="indigo">Indigo</Badge>
      <Badge color="gray">Gray</Badge>
      <Badge color="blue">Blue</Badge>
      <Badge color="red">Red</Badge>
      <Badge color="green">Green</Badge>
      <Badge color="yellow">Yellow</Badge>
      <Badge color="zinc">Zinc</Badge>
    </div>
  ),
};

export const WithIcon: Story = {
  args: {
    children: "With Icon",
    icon: "check_circle",
    color: "green",
  },
};

export const Truncated: Story = {
  args: {
    children: "This is a very long badge text that should be truncated",
    truncate: true,
    color: "blue",
    className: "max-w-32",
  },
};
