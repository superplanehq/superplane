import type { Meta, StoryObj } from "@storybook/react";
import { EmitEventModal } from "./index";
import { useState } from "react";
import { Button } from "@/components/ui/button";

const meta = {
  title: "UI/EmitEventModal",
  component: EmitEventModal,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof EmitEventModal>;

export default meta;
type Story = StoryObj<typeof meta>;

// Wrapper component to manage modal state in stories
function EmitEventModalWrapper({
  channels,
  nodeName,
}: {
  channels: string[];
  nodeName: string;
}) {
  const [isOpen, setIsOpen] = useState(false);

  const handleEmit = async (channel: string, data: any) => {
    console.log("Emitting event:", { channel, data });
    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 1000));
  };

  return (
    <div>
      <Button onClick={() => setIsOpen(true)}>Open Emit Event Modal</Button>
      <EmitEventModal
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        nodeId="test-node-123"
        nodeName={nodeName}
        workflowId="workflow-123"
        organizationId="org-123"
        channels={channels}
        onEmit={handleEmit}
      />
    </div>
  );
}

export const SingleChannel: Story = {
  render: () => (
    <EmitEventModalWrapper channels={["default"]} nodeName="My Test Node" />
  ),
};

export const MultipleChannels: Story = {
  render: () => (
    <EmitEventModalWrapper
      channels={["default", "success", "error", "retry"]}
      nodeName="HTTP Request Node"
    />
  ),
};

export const LongNodeName: Story = {
  render: () => (
    <EmitEventModalWrapper
      channels={["default", "approved", "rejected"]}
      nodeName="Very Long Node Name That Might Need Truncation In The UI Display"
    />
  ),
};

export const ManyChannels: Story = {
  render: () => (
    <EmitEventModalWrapper
      channels={[
        "default",
        "success",
        "error",
        "warning",
        "info",
        "retry",
        "timeout",
        "validation_failed",
        "approved",
        "rejected",
      ]}
      nodeName="Complex Approval Node"
    />
  ),
};
