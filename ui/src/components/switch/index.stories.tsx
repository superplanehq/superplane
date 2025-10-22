import type { Meta, StoryObj } from "@storybook/react"

import { Label } from "../label"

import { Switch } from "./index"

const meta = {
  title: "shadcn Primitives/Switch",
  component: Switch,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    disabled: {
      control: { type: "boolean" },
    },
  },
  args: {
    id: "switch",
    disabled: false,
  },
} satisfies Meta<typeof Switch>

export default meta

type Story = StoryObj<typeof Switch>

const renderSwitch = (args: React.ComponentProps<typeof Switch>) => (
  <div className="flex items-center space-x-3">
    <Switch {...args} />
    <Label htmlFor={args.id}>Airplane Mode</Label>
  </div>
)

export const Default: Story = {
  render: renderSwitch,
}

export const Disabled: Story = {
  args: {
    id: "switch-disabled",
    disabled: true,
  },
  render: renderSwitch,
}
