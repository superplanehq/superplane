import type { Edge, Node } from "@xyflow/react";
import { factoryCanvasBackground, factoryEdgePalette } from "@/lib/factoryCanvasChrome";
import type { StepNodeData } from "./workOrderCanvasTypes";
import { WORKFLOW_GRAPH_SEED } from "./workOrderCanvasGraphSeed";

export const EDGE_TYPE = "default";

export function edgePalette(isDark: boolean) {
  return factoryEdgePalette(isDark);
}

export function backgroundColors(isDark: boolean, isEditing = false) {
  return factoryCanvasBackground(isDark, isEditing);
}

export function buildWorkflowGraph(editable = false): { nodes: Node<StepNodeData>[]; edges: Edge[] } {
  return {
    nodes: WORKFLOW_GRAPH_SEED.nodes.map((node) => ({ ...node, draggable: editable })),
    edges: WORKFLOW_GRAPH_SEED.edges,
  };
}
