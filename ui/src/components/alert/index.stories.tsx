import type { Meta, StoryObj } from "@storybook/react"
import { AlertCircle } from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "./index"

type AlertArgs = React.ComponentProps<typeof Alert>

const meta = {
  title: "shadcn Primitives/Alert",
  component: Alert,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    variant: {
      control: { type: "radio" },
      options: ["default", "destructive"],
    },
    className: {
      control: { type: "text" },
    },
  },
  args: {
    variant: "default",
  },
} satisfies Meta<AlertArgs>

export default meta

type Story = StoryObj<AlertArgs>

const renderAlert = (args: AlertArgs) => (
  <Alert {...args}>
    <AlertTitle>Heads up!</AlertTitle>
    <AlertDescription>
      You can add components to your app using the CLI.
    </AlertDescription>
  </Alert>
)

export const Default: Story = {
  render: renderAlert,
}

export const Destructive: Story = {
  args: {
    variant: "destructive",
  },
  render: (args) => (
    <Alert {...args}>
      <AlertCircle className="h-4 w-4" />
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>
        Your session has expired. Please log in again.
      </AlertDescription>
    </Alert>
  ),
}
