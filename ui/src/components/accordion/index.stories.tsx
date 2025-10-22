import type { Meta, StoryObj } from "@storybook/react"

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "./index"

type AccordionArgs = React.ComponentProps<typeof Accordion>

const meta = {
  title: "shadcn Primitives/Accordion",
  component: Accordion,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    type: {
      control: { type: "radio" },
      options: ["single", "multiple"],
      table: { category: "Behavior" },
    },
    collapsible: {
      control: { type: "boolean" },
      table: { category: "Behavior" },
      if: { arg: "type", eq: "single" },
    },
    disabled: {
      control: { type: "boolean" },
      table: { category: "State" },
    },
  },
  args: {
    type: "single",
    collapsible: true,
  },
} satisfies Meta<AccordionArgs>

export default meta

type Story = StoryObj<AccordionArgs>

const items = [
  {
    id: "item-1",
    title: "Is it accessible?",
    content: "Yes. It adheres to the WAI-ARIA design pattern.",
  },
  {
    id: "item-2",
    title: "Is it styled?",
    content:
      "Yes. It comes with default styles that match the other components' aesthetic.",
  },
  {
    id: "item-3",
    title: "Is it animated?",
    content: "Yes. It's animated by default, but you can disable it if needed.",
  },
]

const renderAccordion = (args: AccordionArgs) => (
  <Accordion {...args}>
    {items.map((item) => (
      <AccordionItem key={item.id} value={item.id} disabled={args.disabled}>
        <AccordionTrigger>{item.title}</AccordionTrigger>
        <AccordionContent>{item.content}</AccordionContent>
      </AccordionItem>
    ))}
  </Accordion>
)

export const Default: Story = {
  render: renderAccordion,
}

export const Multiple: Story = {
  args: {
    type: "multiple",
    collapsible: false,
  },
  render: renderAccordion,
  argTypes: {
    collapsible: { control: { disable: true } },
  },
}
