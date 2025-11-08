import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";
import { Node } from "@xyflow/react";
import { OutputChannelConfigurationModal } from "./OutputChannelConfigurationModal";
import { SuperplaneBlueprintsOutputChannel } from "@/api-client";
import { Button } from "@/components/ui/button";

const meta = {
  title: "ui/OutputChannelConfigurationModal",
  component: OutputChannelConfigurationModal,
  parameters: {
    layout: "centered",
  },
  argTypes: {},
} satisfies Meta<typeof OutputChannelConfigurationModal>;

export default meta;

type Story = StoryObj<typeof OutputChannelConfigurationModal>;

const mockNodes: Node[] = [
  {
    id: "deploy-node-1",
    type: "default",
    position: { x: 0, y: 0 },
    data: {
      label: "Deploy to Production",
      state: "pending",
      type: "noop",
      outputChannels: ["default", "success", "error"],
    },
  },
  {
    id: "approval-node-1",
    type: "default",
    position: { x: 250, y: 0 },
    data: {
      label: "Manager Approval",
      state: "pending",
      type: "approval",
      outputChannels: ["default", "approved", "rejected"],
    },
  },
  {
    id: "notification-node-1",
    type: "default",
    position: { x: 500, y: 0 },
    data: {
      label: "Send Notification",
      state: "pending",
      type: "noop",
      outputChannels: ["default"],
    },
  },
];

export const AddNewOutputChannel: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>Open Add Output Channel Modal</Button>
        <OutputChannelConfigurationModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          nodes={mockNodes}
          onSave={(outputChannel) => {
            console.log("Saved output channel:", outputChannel);
            setIsOpen(false);
          }}
        />
      </div>
    );
  },
};

export const EditOutputChannel: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    const existingOutputChannel: SuperplaneBlueprintsOutputChannel = {
      name: "success",
      nodeId: "deploy-node-1",
      nodeOutputChannel: "success",
    };

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>Open Edit Output Channel Modal</Button>
        <OutputChannelConfigurationModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          outputChannel={existingOutputChannel}
          nodes={mockNodes}
          onSave={(outputChannel) => {
            console.log("Updated output channel:", outputChannel);
            setIsOpen(false);
          }}
        />
      </div>
    );
  },
};

export const EditWithDefaultChannel: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    const existingOutputChannel: SuperplaneBlueprintsOutputChannel = {
      name: "default",
      nodeId: "notification-node-1",
      nodeOutputChannel: "default",
    };

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>Open Edit Default Channel Modal</Button>
        <OutputChannelConfigurationModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          outputChannel={existingOutputChannel}
          nodes={mockNodes}
          onSave={(outputChannel) => {
            console.log("Updated output channel:", outputChannel);
            setIsOpen(false);
          }}
        />
      </div>
    );
  },
};

export const EditApprovalChannel: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    const existingOutputChannel: SuperplaneBlueprintsOutputChannel = {
      name: "approved",
      nodeId: "approval-node-1",
      nodeOutputChannel: "approved",
    };

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>Open Edit Approval Channel Modal</Button>
        <OutputChannelConfigurationModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          outputChannel={existingOutputChannel}
          nodes={mockNodes}
          onSave={(outputChannel) => {
            console.log("Updated output channel:", outputChannel);
            setIsOpen(false);
          }}
        />
      </div>
    );
  },
};

export const EmptyNodeList: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>Open Modal with No Nodes</Button>
        <OutputChannelConfigurationModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          nodes={[]}
          onSave={(outputChannel) => {
            console.log("Saved output channel:", outputChannel);
            setIsOpen(false);
          }}
        />
      </div>
    );
  },
};
