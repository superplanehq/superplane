import type { CanvasesCanvas } from "@/api-client";
import { vi } from "bun:test";

import type { CanvasSpecNode } from "../lib/columnCanvasAgent";
import type { PlanningReviewDraft } from "./planningReviewMockup";

const implementerNode: CanvasSpecNode = {
  id: "implementation-agent",
  name: "Implement From Task Description",
  type: "TYPE_ACTION",
  component: "runnerClaudeCode",
  concurrency: { max: 5 },
  configuration: { model: "sonnet", steps: [{ name: "Clone Repo", type: "bash", command: "git clone" }] },
};

export const agentCanvas: CanvasesCanvas = {
  metadata: { id: "app-refund-implementer", liveVersionId: "version-live" },
  spec: {
    nodes: [{ id: "onrun-implement", name: "On run", type: "TYPE_TRIGGER", component: "onWorkOrder" }, implementerNode],
    edges: [],
  },
};

export const backlogCanvas: CanvasesCanvas = {
  metadata: { id: "backlog", liveVersionId: "version-live" },
  spec: {
    nodes: [
      { ...implementerNode, id: "analyze", name: "Analyze intake" },
      { ...implementerNode, id: "refine-task", name: "Refine Task" },
    ],
    edges: [],
  },
};

export const prFeedbackCanvas: CanvasesCanvas = {
  metadata: { id: "pr-feedback", liveVersionId: "version-live" },
  spec: {
    nodes: [
      { ...implementerNode, id: "address-pr-feedback", name: "Address PR feedback" },
      { ...implementerNode, id: "address-pr-review-feedback", name: "Address PR feedback" },
      { ...implementerNode, id: "address-pr-review-reply-feedback", name: "Address PR feedback" },
      { ...implementerNode, id: "custom-runner", name: "Custom agent" },
    ],
    edges: [],
  },
};

export const agentDraft: PlanningReviewDraft = {
  title: "Implement From Task Description",
  components: [
    {
      id: "implementation-agent",
      title: "Implement From Task Description",
      description: "",
      expanded: true,
      configuration: {
        model: "opus",
        includeVisualEvidence: true,
        steps: [{ name: "Clone Repo", type: "bash", command: "git clone --depth 1" }],
      },
      concurrency: { max: "5", key: "" },
    },
  ],
};

export function stagingSaveDeps(summary?: { hasStaging?: boolean; stale?: boolean }) {
  return {
    readStagingSummary: vi.fn().mockResolvedValue(summary),
    refreshCanvas: vi.fn(async () => {
      throw new Error("live canvas refresh should not run");
    }),
    readStagedCanvas: vi.fn(async () => {
      throw new Error("staged canvas read should not run");
    }),
  };
}

export function canvasWithPublishedNode(versionId: string): CanvasesCanvas {
  return {
    metadata: { id: "app-refund-implementer", liveVersionId: versionId },
    spec: {
      nodes: [
        ...(agentCanvas.spec?.nodes ?? []),
        { id: "published-elsewhere", name: "Published elsewhere", type: "TYPE_ACTION", component: "http" },
      ],
      edges: [{ sourceId: "onrun-implement", targetId: "published-elsewhere" }],
    },
  };
}
