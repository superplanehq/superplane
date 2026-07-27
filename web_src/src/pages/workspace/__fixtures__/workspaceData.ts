import type { WorkspacePageData } from "../types";

export const projectXWorkspaceData = {
  project: {
    name: "Project X",
    factoryName: "Software Factory",
    repository: "superplane/project-x",
    defaultBranch: "main",
  },
  metrics: [
    { label: "Factory health", value: "99.2%", detail: "All systems operational", tone: "positive" },
    { label: "Active work", value: "3", detail: "8 agents in flight", tone: "neutral" },
    { label: "Merged this week", value: "18", detail: "22% above last week", tone: "positive" },
    { label: "Median cycle time", value: "42m", detail: "Down from 51m", tone: "positive" },
  ],
  stages: [
    { id: "intake", label: "Intake", detail: "Issue accepted", state: "complete" },
    { id: "plan", label: "Plan", detail: "Scope prepared", state: "complete" },
    { id: "build", label: "Build", detail: "3 agents active", state: "active" },
    { id: "verify", label: "Verify", detail: "Tests and review", state: "active" },
    { id: "deliver", label: "Deliver", detail: "Merge to main", state: "queued" },
  ],
  workItems: [
    {
      id: "PX-128",
      title: "Add SSO session recovery",
      stage: "Verify",
      branch: "feat/sso-recovery",
      agentCount: 3,
      elapsed: "12m",
      status: "running",
    },
    {
      id: "PX-127",
      title: "Improve retry diagnostics",
      stage: "Build",
      branch: "feat/retry-diagnostics",
      agentCount: 2,
      elapsed: "28m",
      status: "running",
    },
    {
      id: "PX-126",
      title: "Fix stale deployment status",
      stage: "Review",
      branch: "fix/deployment-status",
      agentCount: 3,
      elapsed: "46m",
      status: "waiting",
    },
  ],
  throughput: [
    { label: "Mon", value: 2 },
    { label: "Tue", value: 4 },
    { label: "Wed", value: 3 },
    { label: "Thu", value: 5 },
    { label: "Fri", value: 4 },
    { label: "Sat", value: 0 },
    { label: "Sun", value: 0 },
  ],
  recentDeliveries: [
    {
      id: "delivery-1",
      title: "Rate limit metrics",
      reference: "PR #342",
      timestamp: "18 min ago",
    },
    {
      id: "delivery-2",
      title: "Webhook replay controls",
      reference: "PR #339",
      timestamp: "2h ago",
    },
    {
      id: "delivery-3",
      title: "Audit log retention",
      reference: "PR #335",
      timestamp: "Yesterday",
    },
  ],
} satisfies WorkspacePageData;
