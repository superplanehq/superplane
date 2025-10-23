import type { Meta, StoryObj } from "@storybook/react"
import { useState } from "react"

import { Button } from "../button"

import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "./index"

type DrawerArgs = React.ComponentProps<typeof Drawer>

const meta = {
  title: "shadcn Primitives/Drawer",
  component: Drawer,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    shouldScaleBackground: {
      control: { type: "boolean" },
    },
  },
  args: {
    shouldScaleBackground: true,
  },
} satisfies Meta<DrawerArgs>

export default meta

type Story = StoryObj<DrawerArgs>

const DrawerPreview = (args: DrawerArgs) => {
  const [open, setOpen] = useState(false)

  return (
    <Drawer {...args} open={open} onOpenChange={setOpen}>
      <DrawerTrigger asChild>
        <Button>Open drawer</Button>
      </DrawerTrigger>
      <DrawerContent>
        <DrawerHeader>
          <DrawerTitle>Plan upgrade</DrawerTitle>
          <DrawerDescription>
            Upgrade to the Pro plan to unlock advanced analytics, SSO, and SLA
            support.
          </DrawerDescription>
        </DrawerHeader>
        <div className="space-y-3 px-4 text-sm text-muted-foreground">
          <p>What you get:</p>
          <ul className="list-disc pl-4">
            <li>Unlimited workspaces</li>
            <li>Priority support</li>
            <li>Advanced security controls</li>
          </ul>
        </div>
        <DrawerFooter>
          <Button onClick={() => setOpen(false)}>Upgrade now</Button>
          <DrawerClose asChild>
            <Button variant="outline">Cancel</Button>
          </DrawerClose>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  )
}

export const Default: Story = {
  render: (args) => <DrawerPreview {...args} />,
}
