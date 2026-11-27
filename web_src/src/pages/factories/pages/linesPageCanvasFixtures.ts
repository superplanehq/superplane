import type { CanvasesCanvas } from "@/api-client";

export const IMPLEMENTER_AGENT_STEPS = [
  { name: "Clone Repo", type: "bash" as const, command: "git clone $REPO repo" },
  { name: "Implement", type: "prompt" as const, prompt: "Implement the plan.", workingDirectory: "repo" },
];

export const implementerCanvas = {
  metadata: { id: "app-refund-implementer", liveVersionId: "version-live" },
  spec: {
    nodes: [
      { id: "on-run", name: "On run", type: "TYPE_TRIGGER", component: "onRun", configuration: {} },
      {
        id: "implementation-agent-no-issue",
        name: "Implement From Task Description",
        type: "TYPE_ACTION",
        component: "runnerClaudeCode",
        concurrency: { max: 5 },
        configuration: {
          machineType: "e1-large-amd64",
          model: "sonnet",
          credentials: { source: "integration", integration: { name: "claude" } },
          steps: IMPLEMENTER_AGENT_STEPS,
        },
      },
    ],
    edges: [],
  },
};

export const canvasWithoutAgent = {
  metadata: { id: "app-no-agent", liveVersionId: "version-live" },
  spec: {
    nodes: [{ id: "on-run", name: "On run", type: "TYPE_TRIGGER", component: "onRun", configuration: {} }],
    edges: [],
  },
};

export const closureCanvas: CanvasesCanvas = {
  metadata: { id: "app-pr-closure", name: "PR Closure", liveVersionId: "version-live" },
  spec: {
    nodes: [
      {
        id: "on-pr-closed",
        name: "On Pull Request Closed",
        type: "TYPE_TRIGGER",
        component: "github.onPullRequest",
        configuration: { actions: ["closed"], onlyFactoryPullRequests: true },
      },
      {
        id: "complete-work-order",
        name: "Complete Task",
        type: "TYPE_ACTION",
        component: "updateWorkOrderStatus",
        configuration: { orderId: "{{ root().data.workOrder.id }}", status: "closed" },
      },
    ],
    edges: [{ channel: "default", sourceId: "on-pr-closed", targetId: "complete-work-order" }],
  },
};

export const mergeConfidenceCanvas: CanvasesCanvas = {
  metadata: { id: "app-risk-score", name: "Merge confidence", liveVersionId: "version-live" },
  spec: {
    nodes: [
      {
        id: "on-pr-risk",
        name: "On Pull Request",
        type: "TYPE_TRIGGER",
        component: "github.onPullRequest",
        configuration: { actions: ["opened", "synchronize"], onlyFactoryPullRequests: true },
      },
      {
        id: "assess-risk",
        name: "Assess Merge Confidence",
        type: "TYPE_ACTION",
        component: "runnerClaudeCode",
        configuration: { model: "sonnet", steps: [] },
      },
    ],
    edges: [{ channel: "default", sourceId: "on-pr-risk", targetId: "assess-risk" }],
  },
};

export function canvasQuery(data: typeof implementerCanvas | typeof canvasWithoutAgent | CanvasesCanvas) {
  return { data, isPending: false, isError: false };
}
