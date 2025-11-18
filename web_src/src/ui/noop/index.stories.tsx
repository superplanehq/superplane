import type { Meta, StoryObj } from "@storybook/react";
import { Noop } from "./";

const meta: Meta<typeof Noop> = {
  title: "ui/Noop",
  component: Noop,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: "Don't do anything",
    lastEvent: {
      receivedAt: new Date(),
      eventState: "success",
      eventTitle: "Build completed successfully",
    },
  },
};
