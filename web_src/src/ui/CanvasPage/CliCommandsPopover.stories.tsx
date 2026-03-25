import type { Meta, StoryObj } from "@storybook/react";
import { CliCommandsPopover } from "./CliCommandsPopover";

const meta = {
  title: "Pages/CanvasPage/CliCommandsPopover",
  component: CliCommandsPopover,
  parameters: {
    layout: "centered",
  },
} satisfies Meta<typeof CliCommandsPopover>;

export default meta;
type Story = StoryObj<typeof CliCommandsPopover>;

export const WithCanvasId: Story = {
  args: {
    canvasId: "abc-123-def-456",
    organizationId: "org-789",
  },
};

export const NoCanvasId: Story = {
  args: {
    organizationId: "org-789",
  },
};
