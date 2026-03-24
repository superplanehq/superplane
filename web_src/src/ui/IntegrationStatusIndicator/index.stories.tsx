import type { Meta, StoryObj } from "@storybook/react";
import { IntegrationStatusIndicator, type MissingIntegration } from "./";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useState } from "react";

const meta: Meta<typeof IntegrationStatusIndicator> = {
  title: "Canvas/IntegrationStatusIndicator",
  component: IntegrationStatusIndicator,
  parameters: {
    layout: "centered",
  },
  decorators: [
    (Story) => (
      <TooltipProvider>
        <div style={{ minHeight: 400, display: "flex", alignItems: "flex-end" }}>
          <Story />
        </div>
      </TooltipProvider>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof IntegrationStatusIndicator>;

const singleMissing: MissingIntegration[] = [
  {
    integrationName: "github",
    affectedNodeCount: 3,
  },
];

const multipleMissing: MissingIntegration[] = [
  {
    integrationName: "github",
    affectedNodeCount: 3,
  },
  {
    integrationName: "slack",
    affectedNodeCount: 1,
  },
  {
    integrationName: "aws",
    affectedNodeCount: 2,
  },
];

export const SingleIntegration: Story = {
  args: {
    missingIntegrations: singleMissing,
    onConnect: (name: string) => console.log("Connect:", name),
  },
};

export const MultipleIntegrations: Story = {
  args: {
    missingIntegrations: multipleMissing,
    onConnect: (name: string) => console.log("Connect:", name),
  },
};

export const ReadOnly: Story = {
  args: {
    missingIntegrations: multipleMissing,
    onConnect: (name: string) => console.log("Connect:", name),
    readOnly: true,
  },
};

export const NoPermission: Story = {
  args: {
    missingIntegrations: multipleMissing,
    onConnect: (name: string) => console.log("Connect:", name),
    canCreateIntegrations: false,
  },
};

export const Empty: Story = {
  args: {
    missingIntegrations: [],
    onConnect: (name: string) => console.log("Connect:", name),
  },
};

function InteractiveDemo() {
  const [missing, setMissing] = useState<MissingIntegration[]>([
    { integrationName: "github", affectedNodeCount: 3 },
    { integrationName: "slack", affectedNodeCount: 1 },
    { integrationName: "jira", affectedNodeCount: 2 },
  ]);

  const handleConnect = (name: string) => {
    setMissing((prev) => prev.map((m) => (m.integrationName === name ? { ...m, justConnected: true } : m)));
    setTimeout(() => {
      setMissing((prev) => prev.filter((m) => m.integrationName !== name));
    }, 2000);
  };

  return <IntegrationStatusIndicator missingIntegrations={missing} onConnect={handleConnect} />;
}

export const Interactive: Story = {
  render: () => <InteractiveDemo />,
};
