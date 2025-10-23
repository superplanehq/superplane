import type { Meta, StoryObj } from "@storybook/react"

import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "./index"

type SelectStoryArgs = React.ComponentProps<typeof Select> & {
  placeholder: string
}

const meta = {
  title: "shadcn Primitives/Select",
  component: Select,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    defaultValue: {
      control: { type: "select" },
      options: ["", "apple", "banana", "blueberry", "grapes", "pineapple"],
      table: { category: "State" },
    },
    disabled: {
      control: { type: "boolean" },
      table: { category: "State" },
    },
    placeholder: {
      control: { type: "text" },
      table: { category: "Content" },
    },
  },
  args: {
    defaultValue: "",
    disabled: false,
    placeholder: "Select a fruit",
  },
} satisfies Meta<SelectStoryArgs>

export default meta

type Story = StoryObj<SelectStoryArgs>

const renderSelect = ({ placeholder, defaultValue, ...props }: SelectStoryArgs) => (
  <Select
    {...props}
    defaultValue={defaultValue ? defaultValue : undefined}
  >
    <SelectTrigger title="Select" className="w-64">
      <SelectValue placeholder={placeholder} />
    </SelectTrigger>
    <SelectContent>
      <SelectGroup>
        <SelectLabel>Fruits</SelectLabel>
        <SelectItem value="apple">Apple</SelectItem>
        <SelectItem value="banana">Banana</SelectItem>
        <SelectItem value="blueberry">Blueberry</SelectItem>
        <SelectItem value="grapes">Grapes</SelectItem>
        <SelectItem value="pineapple">Pineapple</SelectItem>
      </SelectGroup>
      <SelectSeparator />
      <SelectGroup>
        <SelectLabel>Vegetables</SelectLabel>
        <SelectItem value="aubergine">Aubergine</SelectItem>
        <SelectItem value="broccoli">Broccoli</SelectItem>
        <SelectItem value="carrot" disabled>
          Carrot
        </SelectItem>
        <SelectItem value="courgette">Courgette</SelectItem>
        <SelectItem value="leek">Leek</SelectItem>
      </SelectGroup>
      <SelectSeparator />
      <SelectGroup>
        <SelectLabel>Meat</SelectLabel>
        <SelectItem value="beef">Beef</SelectItem>
        <SelectItem value="chicken">Chicken</SelectItem>
        <SelectItem value="lamb">Lamb</SelectItem>
        <SelectItem value="pork">Pork</SelectItem>
      </SelectGroup>
    </SelectContent>
  </Select>
)

export const Default: Story = {
  render: renderSelect,
}

export const Disabled: Story = {
  args: {
    disabled: true,
  },
  render: renderSelect,
  argTypes: {
    defaultValue: { control: { disable: true } },
  },
}
