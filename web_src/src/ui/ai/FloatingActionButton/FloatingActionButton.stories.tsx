import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { action } from "@storybook/addon-actions";

import { FloatingActionButton } from "./index";
import { IconRobotFace, IconMessage } from "../icons";

const meta: Meta<typeof FloatingActionButton> = {
  title: "Components/FloatingActionButton",
  component: FloatingActionButton,
  parameters: {
    layout: "fullscreen",
  },
  tags: ["autodocs"],
  argTypes: {
    variant: {
      control: "select",
      options: ["primary", "secondary"],
    },
    size: {
      control: "select", 
      options: ["normal", "large"],
    },
    position: {
      control: "select",
      options: ["bottom-right", "bottom-left", "top-right", "top-left"],
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Basic examples
export const IconOnly: Story = {
  args: {
    icon: <IconRobotFace size={20} />,
    onClick: action("ai-assistant-clicked"),
    label: "Ask Alfred for help",
    variant: "primary",
    size: "normal",
    position: "bottom-right",
  },
};

export const WithText: Story = {
  args: {
    icon: <IconRobotFace size={20} />,
    text: "Ask Alfred",
    onClick: action("ai-assistant-with-text-clicked"),
    label: "Ask Alfred for help",
    variant: "primary",
    size: "normal",
    position: "bottom-right",
  },
};

export const Secondary: Story = {
  args: {
    icon: <IconMessage size={20} />,
    text: "Get Help",
    onClick: action("secondary-clicked"),
    label: "Get help with this page",
    variant: "secondary",
    size: "normal",
    position: "bottom-right",
  },
};

