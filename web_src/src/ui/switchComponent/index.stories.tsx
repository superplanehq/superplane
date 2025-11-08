import type { Meta, StoryObj } from "@storybook/react";
import { SwitchComponent } from "./";

const meta: Meta<typeof SwitchComponent> = {
  title: "ui/SwitchComponent",
  component: SwitchComponent,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: "Branch processed events",
    hideHandle: true,
    stages: [
      {
        pathName: "MAIN",
        field: "$.title",
        operator: "contains",
        value: '"superplane"',
        receivedAt: new Date(),
        eventState: "success",
        eventTitle: "fix: Branch name contains 'superplane'",
      },
      {
        pathName: "STAGE",
        field: "$.author",
        operator: "contains",
        value: '"pedro"',
        receivedAt: new Date(),
        eventState: "success",
        eventTitle: "feature: Branch name contains 'dev'",
      },
      {
        pathName: "DEV",
        field: "$.branch",
        operator: "is",
        value: '"dev"',
        receivedAt: new Date(),
        eventState: "failed",
        eventTitle: "Build failed",
      },
    ],
  },
};
