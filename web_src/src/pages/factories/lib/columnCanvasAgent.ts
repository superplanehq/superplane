import type { CanvasesCanvas } from "@/api-client";
import { AGENT_HARNESS_COMPONENTS, isAgentHarnessComponent } from "@/lib/agentRunnerSteps";
import { materializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";

import type { PlanningReviewComponent, PlanningReviewDraft } from "../pages/planningReviewMockup";

export { AGENT_HARNESS_COMPONENTS };

export type CanvasSpecNode = NonNullable<NonNullable<CanvasesCanvas["spec"]>["nodes"]>[number];

export const PR_FEEDBACK_DISCUSSION_AGENT_NODE_IDS = [
  "address-pr-feedback",
  "address-pr-review-feedback",
  "address-pr-review-reply-feedback",
] as const;

const PR_FEEDBACK_VISUAL_EVIDENCE_NODE_IDS = [
  "has-pr-comment-visual-evidence",
  "comment-pr-comment-visual-evidence",
  "has-pr-review-visual-evidence",
  "comment-pr-review-visual-evidence",
  "has-pr-review-reply-visual-evidence",
  "comment-pr-review-reply-visual-evidence",
] as const;

function hasVisualEvidenceOutputStep(node: CanvasSpecNode): boolean {
  const steps = node.configuration?.steps;
  return (
    Array.isArray(steps) &&
    steps.some(
      (step) => typeof step === "object" && step !== null && "name" in step && step.name === "Publish Visual Evidence",
    )
  );
}

/** Whether a PR discussion canvas can publish agent evidence safely. */
export function supportsPRFeedbackVisualEvidence(spec: CanvasesCanvas["spec"] | null | undefined): boolean {
  const nodes = spec?.nodes ?? [];
  const nodesById = new Map(nodes.map((node) => [node.id, node]));
  const hasEvidenceGraph = PR_FEEDBACK_VISUAL_EVIDENCE_NODE_IDS.every((nodeId) => nodesById.has(nodeId));
  const feedbackAgentsCanPublish = PR_FEEDBACK_DISCUSSION_AGENT_NODE_IDS.every((nodeId) => {
    const node = nodesById.get(nodeId);
    return Boolean(node && isAgentHarnessComponent(node.component) && hasVisualEvidenceOutputStep(node));
  });
  return hasEvidenceGraph && feedbackAgentsCanPublish;
}

/** Agent harness nodes on a column automation canvas. */
export function findAgentNodes(spec: CanvasesCanvas["spec"] | null | undefined): CanvasSpecNode[] {
  return (spec?.nodes ?? []).filter((node) => {
    if (!isAgentHarnessComponent(node.component)) {
      return false;
    }
    if (node.type && node.type !== "TYPE_ACTION") {
      return false;
    }
    return true;
  });
}

/** Preferred agent, or the first agent in canvas order when no preference matches. */
export function primaryAgentNode(
  spec: CanvasesCanvas["spec"] | null | undefined,
  preferredAgentNodeId?: string,
): CanvasSpecNode | undefined {
  const agentNodes = findAgentNodes(spec);
  return agentNodes.find((node) => node.id === preferredAgentNodeId) ?? agentNodes[0];
}

export function canvasNodeToPlanningReviewComponent(node: CanvasSpecNode): PlanningReviewComponent {
  return {
    id: node.id ?? "",
    title: node.name?.trim() || "Agent",
    description: "",
    expanded: true,
    component: node.component,
    configuration: { ...(node.configuration ?? {}) },
    concurrency: {
      max: String(node.concurrency?.max ?? 1),
      key: node.concurrency?.key ?? "",
    },
  };
}

export function planningReviewDraftFromCanvas(canvas: CanvasesCanvas, agentNodeId: string): PlanningReviewDraft | null {
  const node = canvas.spec?.nodes?.find((entry) => entry.id === agentNodeId);
  if (!node) {
    return null;
  }
  const component = canvasNodeToPlanningReviewComponent(node);
  return { title: component.title, components: [component] };
}

export function applyPlanningReviewComponentToNode(
  node: CanvasSpecNode,
  component: PlanningReviewComponent,
): CanvasSpecNode {
  const parsedMax = Number.parseInt(component.concurrency.max, 10);
  const max = Number.isFinite(parsedMax) && parsedMax > 0 ? parsedMax : 1;
  const key = component.concurrency.key.trim();
  return {
    ...node,
    name: component.title,
    configuration: component.configuration,
    concurrency: key ? { max, key } : { max },
  };
}

/** Patch one agent node. Other nodes stay unchanged. */
export function applyPlanningReviewDraftToCanvas(
  canvas: CanvasesCanvas,
  agentNodeId: string,
  draft: PlanningReviewDraft,
): CanvasesCanvas {
  const component = draft.components[0];
  if (!component) {
    return canvas;
  }
  const nodes = (canvas.spec?.nodes ?? []).map((node) =>
    node.id === agentNodeId ? applyPlanningReviewComponentToNode(node, component) : node,
  );
  return {
    ...canvas,
    spec: { ...canvas.spec, nodes },
  };
}

/** Patch one logical agent that is represented by multiple canvas nodes. */
export function applyPlanningReviewDraftToCanvasNodes(
  canvas: CanvasesCanvas,
  agentNodeIds: readonly string[],
  draft: PlanningReviewDraft,
): CanvasesCanvas {
  const component = draft.components[0];
  if (!component) {
    return canvas;
  }
  const targetNodeIds = new Set(agentNodeIds);
  const nodes = (canvas.spec?.nodes ?? []).map((node) =>
    node.id && targetNodeIds.has(node.id) ? applyPlanningReviewComponentToNode(node, component) : node,
  );
  return {
    ...canvas,
    spec: { ...canvas.spec, nodes },
  };
}

/** Patch the agent node and serialize canvas.yaml for staging. */
export function serializeColumnAgentCanvas(
  canvas: CanvasesCanvas,
  agentNodeId: string,
  draft: PlanningReviewDraft,
): string {
  return materializeCanvasSpec(applyPlanningReviewDraftToCanvas(canvas, agentNodeId, draft));
}

export function serializeColumnAgentCanvasNodes(
  canvas: CanvasesCanvas,
  agentNodeIds: readonly string[],
  draft: PlanningReviewDraft,
): string {
  return materializeCanvasSpec(applyPlanningReviewDraftToCanvasNodes(canvas, agentNodeIds, draft));
}
