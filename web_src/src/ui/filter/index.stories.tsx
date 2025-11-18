import type { Meta, StoryObj } from "@storybook/react";
import { Filter } from "./";

const meta: Meta<typeof Filter> = {
  title: "ui/Filter",
  component: Filter,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: "Filter events based on branch",
    expression:
      '$.monarch_app.branch == "main" and $.monarch_app.branch contains "dev" or $.monarch_app.branch endswith "superplane"',
    lastEvent: {
      receivedAt: new Date(),
      eventState: "success",
      eventTitle: "Build completed successfully",
    },
  },
};
