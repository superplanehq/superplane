import type { Meta, StoryObj } from "@storybook/react";
import { MermaidWidget } from "./MermaidWidget";

const meta: Meta<typeof MermaidWidget> = {
  title: "AgentSidebar/Mermaid",
  component: MermaidWidget,
  parameters: {
    layout: "padded",
  },
  decorators: [
    (Story) => (
      <div className="max-w-md bg-white border border-slate-200 rounded-lg p-4">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof MermaidWidget>;

export const CanvasFlowchart: Story = {
  args: {
    content: `flowchart LR
    A[Webhook Trigger] --> B[Call API]
    B --> C{Check Status}
    C -->|200 OK| D[Notify Success]
    C -->|Error| E[Notify Failure]`,
  },
};

export const DeployPipeline: Story = {
  args: {
    content: `flowchart TD
    A[GitHub Push] --> B[Build Image]
    B --> C[Run Tests]
    C -->|Pass| D[Deploy to Staging]
    C -->|Fail| E[Notify Team]
    D --> F[Run Smoke Tests]
    F -->|Pass| G[Deploy to Production]
    F -->|Fail| H[Rollback]`,
  },
};

export const SequenceDiagram: Story = {
  args: {
    content: `sequenceDiagram
    participant User
    participant SuperPlane
    participant API
    participant Slack

    User->>SuperPlane: Trigger Webhook
    SuperPlane->>API: GET /health
    API-->>SuperPlane: 200 OK
    SuperPlane->>Slack: Post Success Message
    Slack-->>User: Notification`,
  },
};

export const NodeStateDiagram: Story = {
  args: {
    content: `stateDiagram-v2
    [*] --> Pending
    Pending --> Running: Trigger Received
    Running --> Success: Execution Complete
    Running --> Failed: Error Occurred
    Failed --> Running: Retry
    Success --> [*]
    Failed --> [*]: Max Retries`,
  },
};

export const CanvasTopology: Story = {
  args: {
    content: `flowchart LR
    subgraph Triggers
      T1[Schedule: Every 5min]
      T2[Webhook: /deploy]
    end
    subgraph Actions
      A1[HTTP: Health Check]
      A2[SSH: Deploy Script]
      A3[If: Status == 200]
    end
    subgraph Notifications
      N1[Slack: #ops]
      N2[Email: on-call@team.com]
    end

    T1 --> A1
    T2 --> A2
    A1 --> A3
    A3 -->|true| N1
    A3 -->|false| N2
    A2 --> N1`,
  },
};
