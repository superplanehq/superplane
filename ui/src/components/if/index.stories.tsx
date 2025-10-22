import type { Meta, StoryObj } from "@storybook/react"

import { If } from "./index"

const meta = {
  title: "components/If",
  component: If,
  tags: ["autodocs"],
  parameters: {
    layout: "centered",
  },
  decorators: [
    (Story) => (
      <div style={{ width: "800px", height: "400px" }}>
        <Story />
      </div>
    ),
  ],
  argTypes: {
    data: {
      control: { type: "object" },
    },
    selected: {
      control: { type: "boolean" },
    },
    collapsed: {
      control: { type: "boolean" },
    },
    showHandles: {
      control: { type: "boolean" },
    },
    className: {
      control: { type: "text" },
      table: { disable: true },
    },
  },
} satisfies Meta<typeof If>

export default meta

type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    data: {
      label: "Payment Check",
      configuration: {
        expression: "user.balance > order.total",
      },
      channels: ["proceed", "reject"],
    },
    selected: false,
    showHandles: false,
  },
}

export const Selected: Story = {
  args: {
    data: {
      label: "Data Validation",
      configuration: {
        expression: "data.isValid && data.schema.matches",
      },
      channels: ["valid", "invalid"],
    },
    selected: true,
    showHandles: false,
  },
}

export const Collapsed: Story = {
  args: {
    data: {
      label: "Feature Check",
      channels: ["enabled", "disabled"],
    },
    collapsed: true,
    selected: false,
    showHandles: false,
  },
}

export const CollapsedSelected: Story = {
  args: {
    data: {
      label: "Auth Check",
      channels: ["allowed", "denied"],
    },
    collapsed: true,
    selected: true,
    showHandles: false,
  },
}