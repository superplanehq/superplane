import type { Meta, StoryObj } from "@storybook/react-vite";
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

export const GitGraph: Story = {
  args: {
    content: `gitGraph
    commit id: "init"
    commit id: "add trigger"
    branch feature/api-node
    checkout feature/api-node
    commit id: "add HTTP node"
    commit id: "add error handling"
    checkout main
    merge feature/api-node
    commit id: "add notifications"
    branch fix/timeout
    checkout fix/timeout
    commit id: "bump timeout to 30s"
    checkout main
    merge fix/timeout
    commit id: "v1.0 release" tag: "v1.0"`,
  },
};

export const GanttChart: Story = {
  args: {
    content: `gantt
    title Canvas Build Timeline
    dateFormat YYYY-MM-DD
    axisFormat %b %d

    section Setup
    Install CLI           :done, setup1, 2026-05-01, 1d
    Connect to API        :done, setup2, after setup1, 1d

    section Build
    Create trigger node   :done, build1, after setup2, 2d
    Add API health check  :done, build2, after build1, 2d
    Add branching logic   :active, build3, after build2, 2d
    Add notifications     :build4, after build3, 3d

    section Deploy
    Staging deploy        :deploy1, after build4, 1d
    Production deploy     :deploy2, after deploy1, 1d`,
  },
};

export const XYChart: Story = {
  args: {
    content: `xychart-beta
    title "Run Duration by Node (ms)"
    x-axis ["Webhook", "API Call", "If Check", "SSH Deploy", "Notify"]
    y-axis "Duration (ms)" 0 --> 5000
    bar [120, 2400, 50, 4200, 350]
    line [120, 2400, 50, 4200, 350]`,
  },
};

export const Timeline: Story = {
  args: {
    content: `timeline
    title Canvas Evolution
    section v0.1 - MVP
      Webhook trigger : Basic HTTP listener
      Single API call : GET health endpoint
    section v0.2 - Branching
      If node added : Status code routing
      Success path : Slack notification
      Failure path : Email alert
    section v0.3 - Reliability
      Retry logic : 3 attempts with backoff
      Timeout config : 30s per node
      SSH deploy : Remote script execution
    section v1.0 - Production
      Monitoring : Run analytics dashboard
      Approvals : Manual gate before deploy
      Scheduling : Cron-based triggers`,
  },
};

export const PieChart: Story = {
  args: {
    content: `pie title Run Outcomes (Last 7 Days)
    "Success" : 312
    "Failed" : 28
    "Timed Out" : 8
    "Cancelled" : 3`,
  },
};
