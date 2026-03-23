import type { Meta, StoryObj } from "@storybook/react";
import { GroupNode, type GroupNodeProps } from "./";

const meta: Meta<typeof GroupNode> = {
  title: "Canvas/GroupNode",
  component: GroupNode,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    groupColor: {
      control: "select",
      options: ["gray", "blue", "green", "purple"],
    },
  },
};

export default meta;
type Story = StoryObj<typeof GroupNode>;

const defaultProps: GroupNodeProps = {
  groupLabel: "My Group",
  groupColor: "gray",
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

export const Blue: Story = {
  args: {
    ...defaultProps,
    groupColor: "blue",
    groupLabel: "API Services",
  },
};

export const Green: Story = {
  args: {
    ...defaultProps,
    groupColor: "green",
    groupLabel: "Monitoring Pipeline",
  },
};

export const Purple: Story = {
  args: {
    ...defaultProps,
    groupColor: "purple",
    groupLabel: "Deploy Stage",
  },
};

export const ReadOnly: Story = {
  args: {
    ...defaultProps,
    hideActionsButton: true,
  },
};
