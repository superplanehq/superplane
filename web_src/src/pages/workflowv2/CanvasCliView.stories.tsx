import type { Meta, StoryObj } from "@storybook/react";
import { CanvasCliView } from "./CanvasCliView";

const meta = {
  title: "Pages/Workflow/CanvasCliView",
  component: CanvasCliView,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof CanvasCliView>;

export default meta;
type Story = StoryObj<typeof CanvasCliView>;

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
