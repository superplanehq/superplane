import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RichMessage } from "./RichMessage";
import { canvasKeys } from "@/hooks/useCanvasData";
import type { CanvasesCanvas } from "@/api-client";

const ORG_ID = "1e880270-cb0b-4310-9479-3e01c14938aa";
const CANVAS_ID = "05bb8e74-6f11-4d1c-bbfd-75d4a28303d6";

const mockCanvas: CanvasesCanvas = {
  spec: {
    nodes: [
      {
        id: "webhook-trigger",
        name: "Webhook Trigger",
        type: "TYPE_TRIGGER",
        component: "webhook",
        configuration: { authentication: "none" },
      },
      {
        id: "call-api",
        name: "Call Target API",
        type: "TYPE_ACTION",
        component: "http",
        configuration: { method: "GET", url: "https://httpbin.org/status/200" },
      },
      {
        id: "check-result",
        name: "Check API Result",
        type: "TYPE_ACTION",
        component: "if",
        configuration: { expression: "{{ previous().data.statusCode >= 200 && previous().data.statusCode < 300 }}" },
      },
      {
        id: "random-wait",
        name: "Random Wait",
        type: "TYPE_ACTION",
        component: "wait",
        configuration: { duration: "30" },
      },
      {
        id: "notify-success",
        name: "Notify Success",
        type: "TYPE_ACTION",
        component: "http",
        configuration: { method: "POST", url: "https://httpbin.org/post" },
      },
      {
        id: "ssh-deploy",
        name: "SSH Deploy",
        type: "TYPE_ACTION",
        component: "ssh",
        configuration: { host: "192.168.1.1", username: "ubuntu" },
        errorMessage: "Integration SSH is disconnected",
      },
      {
        id: "create-issue",
        name: "Create GitHub Issue",
        type: "TYPE_ACTION",
        component: "github.createIssue",
        configuration: { title: "Bug report" },
        metadata: { repository: { name: "acme/widgets" } },
      },
      {
        id: "hub-node",
        name: "Fan Out Hub",
        type: "TYPE_ACTION",
        component: "http",
        configuration: { method: "POST", url: "https://example.com/fanout" },
      },
      {
        id: "branch-a",
        name: "Branch A",
        type: "TYPE_ACTION",
        component: "http",
        configuration: { method: "GET", url: "https://example.com/a" },
      },
      {
        id: "branch-b",
        name: "Branch B",
        type: "TYPE_ACTION",
        component: "http",
        configuration: { method: "GET", url: "https://example.com/b" },
      },
      {
        id: "branch-c",
        name: "Branch C",
        type: "TYPE_ACTION",
        component: "http",
        configuration: { method: "GET", url: "https://example.com/c" },
      },
      {
        id: "branch-d",
        name: "Branch D",
        type: "TYPE_ACTION",
        component: "http",
        configuration: { method: "GET", url: "https://example.com/d" },
      },
    ],
    edges: [
      { sourceId: "webhook-trigger", targetId: "call-api", channel: "default" },
      { sourceId: "call-api", targetId: "check-result", channel: "success" },
      { sourceId: "check-result", targetId: "notify-success", channel: "true" },
      { sourceId: "webhook-trigger", targetId: "hub-node", channel: "default" },
      { sourceId: "hub-node", targetId: "branch-a", channel: "default" },
      { sourceId: "hub-node", targetId: "branch-b", channel: "default" },
      { sourceId: "hub-node", targetId: "branch-c", channel: "default" },
      { sourceId: "hub-node", targetId: "branch-d", channel: "default" },
    ],
  },
};

function createSeededClient() {
  const qc = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity } } });
  qc.setQueryData(canvasKeys.detail(ORG_ID, CANVAS_ID), mockCanvas);
  return qc;
}

const meta: Meta<typeof RichMessage> = {
  title: "AgentSidebar/NodeChips",
  component: RichMessage,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => {
      const qc = createSeededClient();
      return (
        <QueryClientProvider client={qc}>
          <MemoryRouter>
            <div className="max-w-md bg-slate-100 rounded-lg p-4">
              <div className="bg-slate-100 rounded-lg px-3 py-2 text-sm text-slate-900">
                <Story />
              </div>
            </div>
          </MemoryRouter>
        </QueryClientProvider>
      );
    },
  ],
};

export default meta;
type Story = StoryObj<typeof RichMessage>;

export const NodeReferences: Story = {
  args: {
    content: `Hover [Call Target API](node:call-api) for HTTP details, [Create GitHub Issue](node:create-issue) for repo metadata, or [SSH Deploy](node:ssh-deploy) for an error state.`,
    canvasId: CANVAS_ID,
    organizationId: ORG_ID,
  },
};

export const AllComponentTypes: Story = {
  args: {
    content: `Node types:

- Trigger: [Webhook Trigger](node:webhook-trigger)
- HTTP: [Call Target API](node:call-api)
- If: [Check API Result](node:check-result)
- Wait: [Random Wait](node:random-wait)
- SSH: [SSH Deploy](node:ssh-deploy)
- Notify: [Notify Success](node:notify-success)`,
    canvasId: CANVAS_ID,
    organizationId: ORG_ID,
  },
};

export const OverflowNeighbors: Story = {
  args: {
    content: `Hub with many edges: [Fan Out Hub](node:hub-node) should show capped neighbors and +N more.`,
    canvasId: CANVAS_ID,
    organizationId: ORG_ID,
  },
};

export const NodesInTable: Story = {
  args: {
    content: `| Node | Component | Notes |
|------|-----------|-------|
| [Webhook Trigger](node:webhook-trigger) | webhook | Entry point |
| [Call Target API](node:call-api) | http | GET request |
| [Check API Result](node:check-result) | if | Status check |
| [Random Wait](node:random-wait) | wait | 30s delay |`,
    canvasId: CANVAS_ID,
    organizationId: ORG_ID,
  },
};

export const UnknownNode: Story = {
  args: {
    content: `This references a [Missing Node](node:does-not-exist) that doesn't exist on the canvas.`,
    canvasId: CANVAS_ID,
    organizationId: ORG_ID,
  },
};
