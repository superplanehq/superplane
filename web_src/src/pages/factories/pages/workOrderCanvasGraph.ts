import type { Edge, Node } from "@xyflow/react";
import type { StepNodeData } from "./workOrderCanvasTypes";
import { WORKFLOW_GRAPH_SEED } from "./workOrderCanvasGraphSeed";

export const EDGE_TYPE = "default";

export function edgePalette(isDark: boolean) {
  if (isDark) {
    return {
      default: { stroke: "#4a4740", strokeWidth: 1.5 },
      running: { stroke: "#818cf8", strokeWidth: 1.5 },
      failed: { stroke: "#f87171", strokeWidth: 1.5 },
    };
  }
  return {
    default: { stroke: "#cbd5e1", strokeWidth: 1.5 },
    running: { stroke: "#818cf8", strokeWidth: 1.5 },
    failed: { stroke: "#fca5a5", strokeWidth: 1.5 },
  };
}

export function backgroundColors(isDark: boolean) {
  if (isDark) {
    return { gap: 22, size: 1, color: "#33312b", bgColor: "#14120b" };
  }
  return { gap: 22, size: 1, color: "#e5e7eb", bgColor: "#f9fafb" };
}

export function buildWorkflowGraph(editable = false): { nodes: Node<StepNodeData>[]; edges: Edge[] } {
  return {
    nodes: WORKFLOW_GRAPH_SEED.nodes.map((node) => ({ ...node, draggable: editable })),
    edges: WORKFLOW_GRAPH_SEED.edges,
  };
}
