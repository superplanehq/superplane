import type { Meta, StoryObj } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactFlow, ReactFlowProvider } from "@xyflow/react";
import { RichMessage } from "./RichMessage";
import { canvasKeys } from "@/hooks/useCanvasData";
import type { CanvasesCanvas } from "@/api-client";

const mockCanvas: CanvasesCanvas = {
  metadata: {
    id: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    name: "Test Canvas",
  },
  spec: {
    nodes: [
      {
        id: "github-trigger",
        type: "TYPE_TRIGGER",
        label: "GitHub PR Event",
        trigger: {
          name: "github.pullRequestEvent",
          title: "GitHub PR Event",
          iconSlug: "github",
          metadata: [
            { key: "Repository", value: "superplane/platform" },
            { key: "Events", value: "opened, synchronize" },
          ],
        },
        position: { x: 100, y: 100 },
      },
      {
        id: "slack-notify",
        type: "TYPE_ACTION",
        label: "Notify Team",
        action: {
          name: "slack.sendMessage",
          title: "Send Message",
          iconSlug: "slack",
          metadata: [
            { key: "Channel", value: "#deployments" },
            { key: "Template", value: "PR opened" },
          ],
        },
        position: { x: 100, y: 300 },
      },
      {
        id: "aws-deploy",
        type: "TYPE_ACTION",
        label: "Deploy to AWS",
        action: {
          name: "aws.lambda.deployFunction",
          title: "Deploy Lambda",
          iconSlug: "aws",
          metadata: [
            { key: "Function", value: "api-handler" },
            { key: "Region", value: "us-east-1" },
          ],
        },
        position: { x: 100, y: 500 },
      },
    ],
    edges: [],
  },
};

// Create a query client with mock data
const createMockQueryClient = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  // Pre-seed the canvas query cache
  queryClient.setQueryData(
    canvasKeys.detail("1e880270-cb0b-4310-9479-3e01c14938aa", "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6"),
    mockCanvas,
  );

  return queryClient;
};

const meta: Meta<typeof RichMessage> = {
  title: "AgentSidebar/NodeChips",
  component: RichMessage,
  parameters: {
    layout: "padded",
  },
  decorators: [
    (Story) => {
      const queryClient = createMockQueryClient();
      return (
        <QueryClientProvider client={queryClient}>
          <ReactFlowProvider>
            <MemoryRouter>
              <div className="max-w-md bg-slate-100 rounded-lg p-4">
                <div className="bg-slate-100 rounded-lg px-3 py-2 text-sm text-slate-900">
                  <Story />
                </div>
              </div>
              {/* Hidden ReactFlow for context */}
              <div style={{ display: "none" }}>
                <ReactFlow nodes={[]} edges={[]} />
              </div>
            </MemoryRouter>
          </ReactFlowProvider>
        </QueryClientProvider>
      );
    },
  ],
};

export default meta;
type Story = StoryObj<typeof RichMessage>;

export const NodeReferences: Story = {
  args: {
    content: `Canvas nodes:

- [GitHub PR Event](node:github-trigger) triggers the workflow
- [Notify Team](node:slack-notify) sends a Slack message
- [Deploy to AWS](node:aws-deploy) deploys the Lambda function`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};

export const NodesInTable: Story = {
  args: {
    content: `| Step | Node | Description |
|------|------|-------------|
| 1 | [GitHub PR Event](node:github-trigger) | Listens for PR events |
| 2 | [Notify Team](node:slack-notify) | Posts to Slack |
| 3 | [Deploy to AWS](node:aws-deploy) | Deploys function |`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};

export const InlineText: Story = {
  args: {
    content: `The workflow starts with [GitHub PR Event](node:github-trigger) which triggers [Notify Team](node:slack-notify) to send a notification, then [Deploy to AWS](node:aws-deploy) deploys the changes.`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};

export const MixedWithRuns: Story = {
  args: {
    content: `Recent activity:

- [GitHub PR Event](node:github-trigger) received event
- [run #123](run:78848cb6-0c52-4c69-8e47-b6631bd703ec~passed) completed successfully
- [Notify Team](node:slack-notify) posted to Slack
- [Deploy to AWS](node:aws-deploy) deployment in progress`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};

export const UnknownNode: Story = {
  args: {
    content: `This [missing node](node:nonexistent-node-id) doesn't exist in the canvas.`,
    canvasId: "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6",
    organizationId: "1e880270-cb0b-4310-9479-3e01c14938aa",
  },
};
