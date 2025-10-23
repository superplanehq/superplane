import * as React from "react"
import type { Meta, StoryObj } from "@storybook/react"

import { cn } from "@/lib/utils"

import {
  NavigationMenu,
  NavigationMenuIndicator,
  NavigationMenuItem,
  NavigationMenuLink,
  NavigationMenuList,
  navigationMenuTriggerStyle,
} from "./index"

type NavigationLink = {
  label: string
  href: string
  active?: boolean
}

type NavigationMenuStoryArgs = React.ComponentProps<typeof NavigationMenu> & {
  items: NavigationLink[]
}

const meta = {
  title: "shadcn Primitives/NavigationMenu",
  component: NavigationMenu,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
  argTypes: {
    viewport: {
      control: { type: "boolean" },
      table: { category: "Layout" },
    },
    className: {
      control: { type: "text" },
      table: { category: "Layout" },
    },
  },
} satisfies Meta<typeof NavigationMenu>

export default meta

type Story = StoryObj<NavigationMenuStoryArgs>

const NavigationMenuDemo = ({ items, className, viewport, ...props }: NavigationMenuStoryArgs) => (
  <div className="w-full border-b border-border bg-background">
    <div className="mx-auto flex w-full max-w-4xl justify-center px-4 py-8">
      <NavigationMenu
        className={cn("max-w-full justify-center gap-1", className)}
        viewport={viewport}
        {...props}
      >
        <NavigationMenuList>
          {items.map((item) => (
            <NavigationMenuItem key={item.href}>
              <NavigationMenuLink
                href={item.href}
                active={item.active}
                className={navigationMenuTriggerStyle()}
              >
                {item.label}
              </NavigationMenuLink>
            </NavigationMenuItem>
          ))}
        </NavigationMenuList>
        <NavigationMenuIndicator />
      </NavigationMenu>
    </div>
  </div>
)

const baseArgs: NavigationMenuStoryArgs = {
  viewport: false,
  className: "",
  items: [
    { label: "Overview", href: "#", active: true },
    { label: "Features", href: "#features" },
    { label: "Pricing", href: "#pricing" },
    { label: "About", href: "#about" },
    { label: "Contact", href: "#contact" },
  ],
}

export const Default: Story = {
  args: baseArgs,
  render: (args) => <NavigationMenuDemo {...args} />,
}
