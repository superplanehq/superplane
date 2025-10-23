import type { Meta, StoryObj } from "@storybook/react"

import { Badge } from "./index"

const meta = {
  title: "shadcn Primitives/Badge",
  component: Badge,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    variant: {
      control: { type: "select" },
      options: ["default", "secondary", "destructive", "outline"],
    },
    children: {
      control: { type: "text" },
    },
    className: {
      control: { type: "text" },
    },
  },
  args: {
    variant: "default",
    children: "Badge",
  },
} satisfies Meta<typeof Badge>

export default meta

type Story = StoryObj<typeof Badge>

export const Default: Story = {}

export const Secondary: Story = {
  args: {
    variant: "secondary",
  },
}

export const Destructive: Story = {
  args: {
    variant: "destructive",
  },
}

export const Outline: Story = {
  args: {
    variant: "outline",
  },
}
