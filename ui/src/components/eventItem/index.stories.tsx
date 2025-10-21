import type { Meta, StoryObj } from "@storybook/react"

import { EventItem } from "./index"

type EventItemArgs = React.ComponentProps<typeof EventItem>

const meta = {
  title: "components/Event Item",
  component: EventItem,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    status: {
      control: { type: "select" },
      options: ["success", "warning", "error", "info"],
    },
    statusIcon: {
      control: { type: "text" },
    },
    title: {
      control: { type: "text" },
    },
    href: {
      control: { type: "text" },
    },
    timestamp: {
      control: { type: "text" },
    },
    className: {
      control: { type: "text" },
    },
    badges: {
      control: { type: "object" },
    },
  },
  args: {
    status: "success",
    title: "refactor: Add alert origin to alert details",
    timestamp: "2m ago",
    badges: [{ label: "push" }],
    href: "#",
  },
} satisfies Meta<EventItemArgs>

export default meta

type Story = StoryObj<EventItemArgs>

export const Default: Story = {}

export const Warning: Story = {
  args: {
    status: "warning",
    title: "Slow response detected",
    timestamp: "Just now",
    badges: [{ label: "alert", variant: "destructive" }],
  },
}

export const Error: Story = {
  args: {
    status: "error",
    title: "Payment gateway failure",
    timestamp: "5m ago",
    badges: [{ label: "payments" }, { label: "priority" }],
  },
}
