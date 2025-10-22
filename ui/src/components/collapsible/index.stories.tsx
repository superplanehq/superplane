import type { Meta, StoryObj } from "@storybook/react"
import { ChevronsUpDown } from "lucide-react"
import * as React from "react"

import { Button } from "../button"

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "./index"

type CollapsibleArgs = React.ComponentProps<typeof Collapsible>

const meta = {
  title: "shadcn Primitives/Collapsible",
  component: Collapsible,
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
    open: false,
  },
} satisfies Meta<CollapsibleArgs>

export default meta

type Story = StoryObj<CollapsibleArgs>

const CollapsiblePreview = (args: CollapsibleArgs) => {
  const [open, setOpen] = React.useState(args.open ?? false)

  React.useEffect(() => {
    if (args.open !== undefined) {
      setOpen(args.open)
    }
  }, [args.open])

  return (
    <Collapsible {...args} open={open} onOpenChange={setOpen} className="w-72">
      <CollapsibleTrigger asChild>
        <Button variant="outline" className="w-full justify-between">
          Toggle Details
          <ChevronsUpDown className="ml-2 h-4 w-4" />
        </Button>
      </CollapsibleTrigger>
      <CollapsibleContent className="rounded-md border border-dashed px-4 py-3 text-sm text-muted-foreground">
        Share team updates, meeting notes, or onboarding instructions inside a
        collapsible panel to keep the interface tidy.
      </CollapsibleContent>
    </Collapsible>
  )
}

export const Default: Story = {
  render: (args) => <CollapsiblePreview {...args} />,
}

export const InitiallyOpen: Story = {
  args: {
    open: true,
  },
  render: (args) => <CollapsiblePreview {...args} />,
}
