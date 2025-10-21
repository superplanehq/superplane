import type { Meta, StoryObj } from "@storybook/react"

import { Skeleton } from "./index"

const meta = {
  title: "shadcn Primitives/Skeleton",
  component: Skeleton,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    className: {
      control: { type: "text" },
    },
  },
} satisfies Meta<typeof Skeleton>

export default meta

type Story = StoryObj<typeof Skeleton>

export const Default: Story = {
  render: (args) => (
    <div className="flex items-center space-x-4">
      <Skeleton {...args} className="h-12 w-12 rounded-full" />
      <div className="space-y-2">
        <Skeleton {...args} className="h-4 w-[240px]" />
        <Skeleton {...args} className="h-4 w-[200px]" />
      </div>
    </div>
  ),
}
