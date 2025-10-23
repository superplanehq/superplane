import type { Meta, StoryObj } from "@storybook/react"
import * as React from "react"

import { Calendar } from "./index"

type CalendarArgs = React.ComponentProps<typeof Calendar>

const meta = {
  title: "shadcn Primitives/Calendar",
  component: Calendar,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    mode: {
      control: { type: "radio" },
      options: ["single", "multiple", "range"],
    },
    captionLayout: {
      control: { type: "radio" },
      options: ["label", "dropdown"],
    },
    showWeekNumber: {
      control: { type: "boolean" },
    },
    buttonVariant: {
      control: { type: "select" },
      options: ["ghost", "outline", "secondary"],
    },
  },
  args: {
    mode: "single",
    buttonVariant: "ghost",
    showOutsideDays: true,
  },
} satisfies Meta<CalendarArgs>

export default meta

type Story = StoryObj<CalendarArgs>

const CalendarPreview = (args: CalendarArgs) => {
  const [selection, setSelection] = React.useState<unknown>()

  React.useEffect(() => {
    setSelection(undefined)
  }, [args.mode])

  return (
    <Calendar
      {...args}
      selected={selection as CalendarArgs["selected"]}
      onSelect={setSelection as CalendarArgs["onSelect"]}
    />
  )
}

export const Default: Story = {
  render: (args) => <CalendarPreview {...args} />,
}

export const WithWeekNumbers: Story = {
  args: {
    showWeekNumber: true,
  },
  render: (args) => <CalendarPreview {...args} />,
}

export const RangeSelection: Story = {
  args: {
    mode: "range",
    captionLayout: "dropdown",
  },
  render: (args) => <CalendarPreview {...args} />,
}
