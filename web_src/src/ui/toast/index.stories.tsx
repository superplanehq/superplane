import type { Meta, StoryObj } from "@storybook/react"
import { useCallback } from "react"

import { Button } from "../button"

import { Toaster } from "@/ui/toaster"
import { toast } from "@/hooks/use-toast"

const meta = {
  title: "shadcn Primitives/Toast",
  component: Button,
  parameters: {
    layout: "centered",
  },
  decorators: [
    (Story) => (
      <>
        <Story />
        <Toaster />
      </>
    ),
  ],
} satisfies Meta<typeof Button>

export default meta

type Story = StoryObj<typeof Button>

const Demo = () => {
  const handleSuccess = useCallback(() => {
    toast({
      title: "Scheduled: Catch up",
      description: "Wednesday, 9:00 AM",
    })
  }, [])

  const handleDestructive = useCallback(() => {
    toast({
      variant: "destructive",
      title: "Heads up!",
      description: "There was a problem with your request.",
    })
  }, [])

  return (
    <div className="flex flex-col items-center gap-3">
      <Button onClick={handleSuccess}>Show toast</Button>
      <Button variant="outline" onClick={handleDestructive}>
        Show destructive toast
      </Button>
    </div>
  )
}

export const Default: Story = {
  render: () => <Demo />,
}
