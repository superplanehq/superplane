import type { Meta, StoryObj } from "@storybook/react"

import { Button } from "../button"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "./index"

type AlertDialogArgs = React.ComponentProps<typeof AlertDialog>

const meta = {
  title: "shadcn Primitives/AlertDialog",
  component: AlertDialog,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    defaultOpen: {
      control: { type: "boolean" },
      table: { category: "State" },
    },
  },
  args: {
    defaultOpen: false,
  },
} satisfies Meta<AlertDialogArgs>

export default meta

type Story = StoryObj<AlertDialogArgs>

const renderAlertDialog = (args: AlertDialogArgs) => (
  <AlertDialog {...args}>
    <AlertDialogTrigger asChild>
      <Button variant="outline">Open</Button>
    </AlertDialogTrigger>
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
        <AlertDialogDescription>
          This action cannot be undone. This will permanently delete your
          account and remove your data from our servers.
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <AlertDialogAction>Continue</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
)

export const Default: Story = {
  render: renderAlertDialog,
}
