import type { Meta, StoryObj } from "@storybook/react";
import { GroupNode, type GroupColor, type GroupNodeProps } from "./";

const colorOptions: GroupColor[] = ["purple", "blue", "green", "cyan", "orange", "rose", "amber"];

const meta: Meta<typeof GroupNode> = {
  title: "Canvas/GroupNode",
  component: GroupNode,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    groupColor: {
      control: "select",
      options: colorOptions,
    },
  },
};

export default meta;
type Story = StoryObj<typeof GroupNode>;

const defaultProps: GroupNodeProps = {
  groupLabel: "My Group",
  groupDescription: "",
  groupColor: "purple",
  onGroupUpdate: (updates) => console.log("Group update:", updates),
  onUngroup: () => console.log("Ungroup clicked"),
  onDelete: () => console.log("Delete clicked"),
};

export const Default: Story = {
  args: defaultProps,
};

export const Selected: Story = {
  args: {
    ...defaultProps,
    selected: true,
  },
};

export const WithDescription: Story = {
  args: {
    ...defaultProps,
    groupLabel: "Ingest pipeline",
    groupDescription: "Pull events from webhooks, normalize, and enqueue for workers.",
  },
};

export const Blue: Story = {
  args: {
    ...defaultProps,
    groupColor: "blue",
    groupLabel: "API Services",
    groupDescription: "REST endpoints and auth for the checkout flow.",
  },
};

export const Green: Story = {
  args: {
    ...defaultProps,
    groupColor: "green",
    groupLabel: "Monitoring Pipeline",
  },
};

export const Cyan: Story = {
  args: {
    ...defaultProps,
    groupColor: "cyan",
    groupLabel: "Staging",
  },
};

export const Orange: Story = {
  args: {
    ...defaultProps,
    groupColor: "orange",
    groupLabel: "Alerts",
  },
};

export const Rose: Story = {
  args: {
    ...defaultProps,
    groupColor: "rose",
    groupLabel: "Review",
  },
};

export const Amber: Story = {
  args: {
    ...defaultProps,
    groupColor: "amber",
    groupLabel: "Deploy",
  },
};

export const ReadOnly: Story = {
  args: {
    ...defaultProps,
    hideActionsButton: true,
  },
};
