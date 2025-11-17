import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";

import { ViewToggle } from "./index";

const meta = {
  title: "Components/ViewToggle",
  component: ViewToggle,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    isCollapsed: {
      control: "boolean",
    },
    onToggle: {
      action: "toggled",
    },
  },
} satisfies Meta<typeof ViewToggle>;

export default meta;

type Story = StoryObj<typeof ViewToggle>;

export const Default: Story = {
  args: {
    isCollapsed: false,
  },
  render: (args) => {
    const [isCollapsed, setIsCollapsed] = useState(args.isCollapsed);

    const handleToggle = () => {
      setIsCollapsed((prev) => !prev);
      args.onToggle?.();
    };

    return <ViewToggle {...args} isCollapsed={isCollapsed} onToggle={handleToggle} />;
  },
};

export const Collapsed: Story = {
  args: {
    isCollapsed: true,
  },
  render: (args) => {
    const [isCollapsed, setIsCollapsed] = useState(args.isCollapsed);

    const handleToggle = () => {
      setIsCollapsed((prev) => !prev);
      args.onToggle?.();
    };

    return <ViewToggle {...args} isCollapsed={isCollapsed} onToggle={handleToggle} />;
  },
};

export const WithCustomClassName: Story = {
  args: {
    isCollapsed: false,
    className: "border-2 border-blue-300",
  },
  render: (args) => {
    const [isCollapsed, setIsCollapsed] = useState(args.isCollapsed);

    const handleToggle = () => {
      setIsCollapsed((prev) => !prev);
      args.onToggle?.();
    };

    return <ViewToggle {...args} isCollapsed={isCollapsed} onToggle={handleToggle} />;
  },
};
