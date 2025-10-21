import type { Meta, StoryObj } from "@storybook/react"

import { cn } from "@/lib/utils"

import { Tabs, TabsContent, TabsList, TabsTrigger } from "./index"

type TabsStoryArgs = React.ComponentProps<typeof Tabs>

const meta = {
  title: "shadcn Primitives/Tabs",
  component: Tabs,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    defaultValue: {
      control: { type: "select" },
      options: ["account", "password"],
      table: { category: "State" },
    },
    orientation: {
      control: { type: "radio" },
      options: ["horizontal", "vertical"],
      table: { category: "Layout" },
    },
    dir: {
      control: { type: "radio" },
      options: ["ltr", "rtl"],
      table: { category: "Layout" },
    },
    className: {
      control: { type: "text" },
      table: { category: "Style" },
    },
  },
  args: {
    defaultValue: "account",
    orientation: "horizontal",
    dir: "ltr",
    className: "w-96",
  },
} satisfies Meta<TabsStoryArgs>

export default meta

type Story = StoryObj<TabsStoryArgs>

const renderTabs = ({ className, orientation, ...props }: TabsStoryArgs) => {
  const orientationValue = orientation ?? "horizontal"
  const isVertical = orientationValue === "vertical"

  return (
    <Tabs
      {...props}
      orientation={orientationValue}
      className={cn(
        isVertical ? "flex max-w-xl gap-6" : "w-96",
        className,
      )}
    >
      <TabsList
        className={cn(isVertical ? "flex h-full w-44 flex-col" : "grid grid-cols-2")}
      >
        <TabsTrigger value="account">Account</TabsTrigger>
        <TabsTrigger value="password">Password</TabsTrigger>
      </TabsList>
      <TabsContent
        value="account"
        className={cn("mt-2", isVertical && "mt-0 flex-1")}
      >
        Make changes to your account here.
      </TabsContent>
      <TabsContent
        value="password"
        className={cn("mt-2", isVertical && "mt-0 flex-1")}
      >
        Change your password here.
      </TabsContent>
    </Tabs>
  )
}

export const Default: Story = {
  render: renderTabs,
}

export const Vertical: Story = {
  args: {
    orientation: "vertical",
    className: "flex max-w-xl gap-6",
  },
  render: renderTabs,
  argTypes: {
    className: { control: { disable: true } },
  },
}
