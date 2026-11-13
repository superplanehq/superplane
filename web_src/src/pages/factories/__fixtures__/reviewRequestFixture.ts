import type { CanvasesCanvas } from "@/api-client";

export function reviewRequestCanvas(): CanvasesCanvas {
  return {
    metadata: { id: "review-request-app", name: "Review request", liveVersionId: "version-1", factoryId: "factory-1" },
    spec: {
      nodes: [
        {
          id: "review-request-trigger",
          name: "On Merge confidence report",
          component: "onBroadcast",
          type: "TYPE_TRIGGER",
          configuration: {
            enabled: true,
            app: "merge-confidence-app",
            filter: 'message.payload.type == "merge-confidence.completed"',
          },
        },
        {
          id: "select-reviewers",
          name: "Select reviewers",
          component: "runnerSuperPlane",
          type: "TYPE_ACTION",
          concurrency: { max: 1 },
          configuration: {
            fallbackReviewer: "saved-reviewer",
            model: "hosted::openai::gpt-5",
            thinkingLevel: "high",
            steps: [{ name: "Select reviewers", type: "prompt", prompt: "Managed reviewer policy" }],
          },
        },
        {
          id: "request-reviewers",
          name: "Ensure reviewers",
          component: "github.ensurePullRequestReviewers",
          type: "TYPE_ACTION",
          integration: { id: "github-1", name: "GitHub" },
          configuration: {},
        },
        {
          id: "pr-check-pending",
          name: "Register PR check",
          type: "TYPE_ACTION",
          component: "github.setPullRequestReviewCheck",
          integration: { id: "github-1", name: "GitHub" },
          configuration: { enabled: false, activationGeneration: 1, activationNodeId: "pr-check-pending" },
        },
      ],
      edges: [
        { sourceId: "review-request-trigger", targetId: "select-reviewers", channel: "default" },
        { sourceId: "select-reviewers", targetId: "request-reviewers", channel: "passed" },
      ],
    },
  };
}
