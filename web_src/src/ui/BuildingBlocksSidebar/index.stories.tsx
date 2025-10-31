import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";

import { BuildingBlocksSidebar, BuildingBlock, BuildingBlockCategory } from "./index";
import React from "react";

const sampleTriggers: BuildingBlock[] = [
  {
    name: "start",
    label: "Manual Start",
    description: "Manually trigger the workflow",
    type: "trigger",
    icon: "play",
    color: "green",
    outputChannels: [{ name: "default" }],
    configuration: [],
  },
  {
    name: "schedule",
    label: "Schedule",
    description: "Run workflow on a schedule",
    type: "trigger",
    icon: "clock",
    color: "blue",
    outputChannels: [{ name: "default" }],
    configuration: [
      { name: "cron", type: "string" },
    ],
  },
  {
    name: "webhook",
    label: "Webhook",
    description: "Trigger via HTTP webhook",
    type: "trigger",
    icon: "webhook",
    color: "purple",
    outputChannels: [{ name: "default" }],
    configuration: [],
  },
];

const sampleComponents: BuildingBlock[] = [
  {
    name: "http",
    label: "HTTP Request",
    description: "Make an HTTP request",
    type: "component",
    icon: "globe",
    color: "blue",
    outputChannels: [{ name: "default" }],
    configuration: [
      { name: "url", type: "string" },
      { name: "method", type: "string" },
    ],
  },
  {
    name: "approval",
    label: "Approval",
    description: "Wait for manual approval",
    type: "component",
    icon: "hand",
    color: "orange",
    outputChannels: [{ name: "approved" }, { name: "rejected" }],
    configuration: [
      { name: "approvers", type: "array" },
    ],
  },
  {
    name: "if",
    label: "If Condition",
    description: "Branch based on a condition",
    type: "component",
    icon: "split",
    color: "yellow",
    outputChannels: [{ name: "true" }, { name: "false" }],
    configuration: [
      { name: "condition", type: "string" },
    ],
  },
];

const sampleBlueprints: BuildingBlock[] = [
  {
    id: "bp-1",
    name: "deploy-to-k8s",
    label: "Deploy to Kubernetes",
    description: "Deploy an application to a Kubernetes cluster",
    type: "blueprint",
    icon: "box",
    color: "indigo",
    outputChannels: [{ name: "success" }, { name: "failure" }],
    configuration: [
      { name: "namespace", type: "string" },
      { name: "deployment", type: "string" },
    ],
  },
  {
    id: "bp-2",
    name: "send-notification",
    label: "Send Notification",
    description: "Send a notification to multiple channels",
    type: "blueprint",
    icon: "bell",
    color: "pink",
    outputChannels: [{ name: "default" }],
    configuration: [
      { name: "message", type: "string" },
      { name: "channels", type: "array" },
    ],
  },
];

const sampleBlocks: BuildingBlockCategory[] = [
  {
    name: "Primitives",
    blocks: [...sampleTriggers, ...sampleComponents],
  },
  {
    name: "Custom Components",
    blocks: sampleBlueprints,
  },
];

const meta = {
  title: "ui/BuildingBlocksSidebar",
  component: BuildingBlocksSidebar,
  parameters: {
    layout: "fullscreen",
  },
  argTypes: {
    isOpen: {
      control: "boolean",
    },
    onToggle: {
      action: "toggled",
    },
  },
} satisfies Meta<typeof BuildingBlocksSidebar>;

export default meta;

type Story = StoryObj<typeof BuildingBlocksSidebar>;

export const Default: Story = {
  args: {
    isOpen: true,
    blocks: sampleBlocks,
  },
  render: (args) => {
    const [isOpen, setIsOpen] = useState(args.isOpen);

    const handleToggle = (open: boolean) => {
      setIsOpen(open);
      args.onToggle?.(open);
    };

    return (
      <div className="h-screen w-screen flex bg-gray-100">
        <BuildingBlocksSidebar
          {...args}
          isOpen={isOpen}
          onToggle={handleToggle}
        />
      </div>
    );
  },
};

export const Closed: Story = {
  args: {
    isOpen: false,
    blocks: sampleBlocks,
  },
  render: (args) => {
    const [isOpen, setIsOpen] = useState(args.isOpen);

    const handleToggle = (open: boolean) => {
      setIsOpen(open);
      args.onToggle?.(open);
    };

    return (
      <div className="h-screen w-screen flex bg-gray-100 relative">
        <BuildingBlocksSidebar
          {...args}
          isOpen={isOpen}
          onToggle={handleToggle}
        />
      </div>
    );
  },
};

export const EmptyLists: Story = {
  args: {
    isOpen: true,
    blocks: [],
  },
  render: (args) => {
    const [isOpen, setIsOpen] = useState(args.isOpen);

    const handleToggle = (open: boolean) => {
      setIsOpen(open);
      args.onToggle?.(open);
    };

    return (
      <div className="h-screen w-screen flex bg-gray-100">
        <BuildingBlocksSidebar
          {...args}
          isOpen={isOpen}
          onToggle={handleToggle}
        />
      </div>
    );
  },
};

export const OnlyTriggers: Story = {
  args: {
    isOpen: true,
    blocks: [
      {
        name: "Primitives",
        blocks: sampleTriggers,
      },
    ],
  },
  render: (args) => {
    const [isOpen, setIsOpen] = useState(args.isOpen);

    const handleToggle = (open: boolean) => {
      setIsOpen(open);
      args.onToggle?.(open);
    };

    return (
      <div className="h-screen w-screen flex bg-gray-100">
        <BuildingBlocksSidebar
          {...args}
          isOpen={isOpen}
          onToggle={handleToggle}
        />
      </div>
    );
  },
};
