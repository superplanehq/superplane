import type { Meta, StoryObj } from "@storybook/react-vite";
import { RichMessage } from "./RichMessage";

const meta: Meta<typeof RichMessage> = {
  title: "AgentSidebar/RichMessage",
  component: RichMessage,
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
type Story = StoryObj<typeof RichMessage>;

export const PureMarkdown: Story = {
  args: {
    content: `## Canvas Deployed!

Here's what was built:

- **5 nodes** configured
- **4 edges** connecting them
- Webhook trigger → API call → branch → notifications

\`\`\`yaml
apiVersion: v1
kind: Canvas
metadata:
  name: api-health-check
\`\`\`

> Note: The canvas is now live and accepting webhook events.`,
  },
};

export const Buttons: Story = {
  args: {
    content: `I can set up a few different workflow patterns for you.

:::buttons
Which pattern would you like?
- Webhook → API → Notify
- Schedule → Health Check → Alert
- GitHub Push → Deploy → Verify
- Custom (describe your workflow)
:::`,
  },
};

export const YAMLCodeBlock: Story = {
  args: {
    content: `Here's the canvas configuration:

\`\`\`yaml
apiVersion: v1
kind: Canvas
metadata:
  name: api-health-check
  id: 05bb8e74-6f11-4d1c-bbfd-75d4a28303d6
spec:
  nodes:
    - id: webhook-trigger
      name: Receive Request
      type: TYPE_TRIGGER
      component: webhook
      configuration:
        authentication: "none"
    - id: call-api
      name: Call Target API
      type: TYPE_ACTION
      component: http
      configuration:
        method: GET
        url: "https://api.example.com/health"
        json: ""
        successCodes: "200"
        timeoutSeconds: 30
\`\`\``,
  },
};

export const BashCodeBlock: Story = {
  args: {
    content: `Run these commands to set up:

\`\`\`bash
curl -fsSL https://install.superplane.com/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
superplane connect https://app.superplane.com your-token-here
superplane apps list
\`\`\`

The CLI should now be ready to use.`,
  },
};

export const JSONCodeBlock: Story = {
  args: {
    content: `The webhook payload looks like this:

\`\`\`json
{
  "event": "push",
  "repository": {
    "full_name": "superplanehq/superplane",
    "default_branch": "main"
  },
  "ref": "refs/heads/main",
  "commits": [
    {
      "id": "abc123",
      "message": "fix: resolve timeout issue",
      "author": { "name": "Alex" }
    }
  ]
}
\`\`\``,
  },
};

export const DeployButtons: Story = {
  args: {
    content: `I've prepared the canvas. Ready to deploy.

:::buttons
Where should this be deployed?
- DigitalOcean
- Google Cloud
- Hetzner
- AWS
:::`,
  },
};

export const Confirmation: Story = {
  args: {
    content: `I'm about to overwrite the existing canvas configuration.

:::confirm
message: This will replace all 5 existing nodes with the new pipeline. This action cannot be undone.
yes: Overwrite Canvas
no: Cancel
:::`,
  },
};

export const LineChart: Story = {
  args: {
    content: `Here's your run success rate for the past week:

:::chart
type: line
title: Run Success Rate (Last 7 Days)
x: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
series:
  - name: Successful
    data: [12, 15, 11, 14, 13, 8, 10]
    color: "#22c55e"
  - name: Failed
    data: [2, 1, 3, 0, 2, 1, 0]
    color: "#ef4444"
:::

Overall success rate: **90.4%** — 3 failures on Wednesday were due to a timeout in the API call node.`,
  },
};

export const BarChart: Story = {
  args: {
    content: `Execution count by node:

:::chart
type: bar
title: Executions Per Node (Last 24h)
x: ["Webhook", "API Call", "Check Status", "Notify OK", "Notify Fail"]
series:
  - name: Executions
    data: [48, 48, 48, 41, 7]
    color: "#8b5cf6"
:::`,
  },
};

export const PieChart: Story = {
  args: {
    content: `Run outcome breakdown:

:::chart
type: pie
title: Run Outcomes (Last 30 Days)
data:
  - name: Success
    value: 312
    color: "#22c55e"
  - name: Failed
    value: 28
    color: "#ef4444"
  - name: Timed Out
    value: 8
    color: "#f59e0b"
  - name: Cancelled
    value: 3
    color: "#94a3b8"
:::`,
  },
};

export const Steps: Story = {
  args: {
    content: `Setting up your canvas:

:::steps
- [x] Install CLI
- [x] Connect to SuperPlane API
- [x] Write canvas YAML
- [ ] Deploy canvas
- [ ] Verify node configuration
:::`,
  },
};

export const SuccessBanner: Story = {
  args: {
    content: `:::success
Canvas "api-health-check-with-alerts" deployed successfully! 5 nodes, 4 edges, all validations passed.
:::

The webhook URL is: \`https://app.superplane.com/hooks/05bb8e74-6f11-4d1c-bbfd-75d4a28303d6\``,
  },
};

export const ErrorBanner: Story = {
  args: {
    content: `:::error
Deployment failed: Node "call-api" has validation error — json field is required for HTTP actions.
:::

I'll fix this and retry. The issue is that GET requests still need an empty \`json: ""\` field.`,
  },
};

export const CollapsibleSection: Story = {
  args: {
    content: `Canvas deployed. Here's the full YAML if you want to review:

:::collapse title="Full Canvas YAML"
apiVersion: v1
kind: Canvas
metadata:
  name: api-health-check-with-alerts
  id: 05bb8e74-6f11-4d1c-bbfd-75d4a28303d6
spec:
  nodes:
    - id: webhook-trigger
      name: Receive Request
      type: TYPE_TRIGGER
      component: webhook
      configuration:
        authentication: "none"
    - id: call-api
      name: Call Target API
      type: TYPE_ACTION
      component: http
      configuration:
        method: GET
        url: "https://api.example.com/health"
        json: ""
        successCodes: "200"
        timeoutSeconds: 30
:::`,
  },
};

export const SimpleTable: Story = {
  args: {
    content: `Here are the nodes in your canvas:

| Node | Type | Component | Status |
|------|------|-----------|--------|
| Webhook Trigger | Trigger | webhook | Active |
| Call API | Action | http | Active |
| Check Status | Action | if | Active |
| Notify Success | Action | http | Active |
| Notify Failure | Action | http | Active |`,
  },
};

export const RunHistoryTable: Story = {
  args: {
    content: `## Recent Runs

Last 5 runs for this canvas:

| Run ID | Started | Duration | Status | Trigger |
|--------|---------|----------|--------|---------|
| #1247 | 2 min ago | 3.2s | ✅ Success | Webhook |
| #1246 | 17 min ago | 2.8s | ✅ Success | Webhook |
| #1245 | 32 min ago | 12.1s | ❌ Failed | Webhook |
| #1244 | 1h ago | 3.0s | ✅ Success | Schedule |
| #1243 | 1h 15m ago | 2.9s | ✅ Success | Schedule |

Run #1245 failed at the **Call API** node with a timeout error.`,
  },
};

export const NodeComparisonTable: Story = {
  args: {
    content: `### Node Performance Comparison

| Node | Avg Duration | Success Rate | Executions |
|------|-------------|-------------|------------|
| Webhook Trigger | 12ms | 100% | 1,247 |
| Call API | 2.4s | 95.8% | 1,247 |
| Check Status | 8ms | 100% | 1,195 |
| Notify Success | 340ms | 99.2% | 1,142 |
| Notify Failure | 380ms | 98.1% | 53 |

The **Call API** node has the lowest success rate — consider adding retry logic.`,
  },
};

export const MermaidDiagram: Story = {
  args: {
    content: `Here's the flow for your canvas:

\`\`\`mermaid
flowchart LR
    A[Webhook Trigger] --> B[Call API]
    B --> C{Check Status}
    C -->|200 OK| D[Notify Success]
    C -->|Error| E[Notify Failure]
\`\`\`

The webhook will accept incoming requests and route them through the API health check before notifying via the appropriate channel.`,
  },
};

export const MixedContent: Story = {
  args: {
    content: `## Analysis Complete

I analyzed 351 runs from the past 30 days. Here's what I found:

:::chart
type: line
title: Daily Run Volume
x: ["W1", "W2", "W3", "W4"]
series:
  - name: Runs
    data: [78, 92, 88, 93]
    color: "#8b5cf6"
:::

:::success
Overall health is excellent — 96% success rate.
:::

### Recommendations

1. The "Check Status" node has the highest failure rate (4.2%)
2. Consider adding retry logic to the API call
3. Timeout could be increased from 10s to 15s

:::buttons
What would you like to do?
- Add retry to API call
- Increase timeout
- Show me the failing runs
- Do nothing
:::`,
  },
};
