import type { Meta, StoryObj } from "@storybook/react"

import { AspectRatio } from "./index"

type AspectRatioArgs = React.ComponentProps<typeof AspectRatio>

const meta = {
  title: "shadcn Primitives/AspectRatio",
  component: AspectRatio,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    ratio: {
      control: { type: "number" },
      table: { category: "Layout" },
    },
    className: {
      control: { type: "text" },
      table: { category: "Style" },
    },
  },
  args: {
    ratio: 16 / 9,
  },
  decorators: [
    (Story) => (
      <div className="w-[320px] sm:w-[480px]">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<AspectRatioArgs>

export default meta

type Story = StoryObj<AspectRatioArgs>

const placeholder =
  "https://images.unsplash.com/photo-1576075796033-848c2a5f3696?w=800&dpr=2&q=80"

const renderAspectRatio = (args: AspectRatioArgs) => (
  <AspectRatio {...args} className="overflow-hidden rounded-md bg-muted">
    <img
      src={placeholder}
      alt="City skyline at night"
      className="h-full w-full object-cover"
    />
  </AspectRatio>
)

export const Default: Story = {
  render: renderAspectRatio,
}

export const Square: Story = {
  args: {
    ratio: 1,
  },
  render: renderAspectRatio,
}

export const Portrait: Story = {
  args: {
    ratio: 3 / 4,
  },
  render: renderAspectRatio,
}
