import type { CanvasesCanvas } from "@/api-client";
import { AGENT_HARNESS_COMPONENTS, isAgentHarnessComponent } from "@/lib/agentRunnerSteps";
import { materializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";

import type { PlanningReviewComponent, PlanningReviewDraft } from "../pages/planningReviewMockup";

export { AGENT_HARNESS_COMPONENTS };

export type CanvasSpecNode = NonNullable<NonNullable<CanvasesCanvas["spec"]>["nodes"]>[number];

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
