import type { Edge, Node } from "@xyflow/react";

import type { ComponentsEdge, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { agentRunnerStepTitles } from "@/lib/agentRunnerSteps";
import { factoryEdgePalette, factoryNodeCardSize } from "@/lib/factoryCanvasChrome";
import { layoutFactoryRunLeafGraph } from "@/lib/layout/factoryRunLeafLayout";
import { buildStyledCanvasEdges } from "@/ui/CanvasPage/factoryCanvasEdgeStyle";
import type { FactoryNodeStatus } from "@/ui/factoryNodeChrome/types";

import { componentPresentation, type SplitRunCanvasModel } from "./splitRunCanvases";

export type LineNodeData = {
  title: string;
  subtitle: string;
  iconSlug: string;
  iconSrc?: string;
  status: FactoryNodeStatus;
  metrics: string;
  nodeId: string;
  isSelected: boolean;
  onSelect?: (id: string) => void;
  editHref?: string;
  isSideSource: boolean;
  isSpineSource: boolean;
  isSideTarget: boolean;
  steps: string[];
} & Record<string, unknown>;

function origin(nodes: ComponentsNode[]): { x: number; y: number } {
  const xs = nodes.map((node) => node.position?.x ?? 0);
  const ys = nodes.map((node) => node.position?.y ?? 0);
  return { x: Math.min(...xs, 0), y: Math.min(...ys, 0) };
}

function toFlowEdge(edge: ComponentsEdge, index: number): Edge {
  const channel = edge.channel ?? "default";
  return {
    id: `e-${edge.sourceId}-${edge.targetId}-${index}`,
    source: edge.sourceId ?? "",
    target: edge.targetId ?? "",
    sourceHandle: channel,
    type: "custom",
  };
}

export function compactLineCanvasGraph(
  canvas: SplitRunCanvasModel,
  selectedId: string | null,
  onSelect: ((id: string) => void) | undefined,
  resolvedThemeIsDark: boolean,
  nodeEditHref?: (nodeId: string) => string,
): { nodes: Node<LineNodeData>[]; edges: Edge[] } {
  const zero = origin(canvas.nodes);
  const rawNodes: Node<LineNodeData>[] = canvas.nodes
    .filter((node): node is ComponentsNode & { id: string } => Boolean(node.id))
    .map((node) => {
      const presentation = componentPresentation(node.component);
      const steps = node.component === "runnerClaudeCode" ? agentRunnerStepTitles(node.configuration) : [];
      const size = factoryNodeCardSize(steps.length);
      return {
        id: node.id,
        type: "lineCanvas",
        position: {
          x: (node.position?.x ?? 0) - zero.x,
          y: (node.position?.y ?? 0) - zero.y,
        },
        data: {
          title: presentation.title,
          subtitle: node.name ?? presentation.title,
          iconSlug: presentation.iconSlug,
          iconSrc: presentation.iconSrc,
          status: canvas.statuses[node.id] ?? "pending",
          metrics: canvas.metrics[node.id] ?? "—",
          nodeId: node.id,
          isSelected: node.id === selectedId,
          onSelect,
          editHref: node.id === selectedId ? nodeEditHref?.(node.id) : undefined,
          isSideSource: false,
          isSpineSource: false,
          isSideTarget: false,
          steps,
        },
        selected: node.id === selectedId,
        draggable: false,
        width: size.width,
        height: size.height,
      };
    });

  const rawEdges = canvas.edges.map((edge, index) => toFlowEdge(edge, index));
  const layout = layoutFactoryRunLeafGraph(
    rawNodes.map((node) => ({ id: node.id, position: node.position, width: node.width, height: node.height })),
    rawEdges.map((edge) => ({
      id: edge.id,
      source: edge.source,
      target: edge.target,
      sourceHandle: edge.sourceHandle,
    })),
  );

  const nodes = rawNodes.map((node) => ({
    ...node,
    position: layout.positions.get(node.id) ?? node.position,
    data: {
      ...node.data,
      isSideSource: layout.sideHandleNodeIds.has(node.id),
      isSpineSource: layout.spineSourceNodeIds.has(node.id),
      isSideTarget: layout.sideTargetNodeIds.has(node.id),
    },
  }));

  const edges =
    buildStyledCanvasEdges({
      edges: rawEdges,
      nodes,
      isVerticalFlow: true,
      resolvedThemeIsDark,
      edgeDefaults: { type: "custom", style: factoryEdgePalette(resolvedThemeIsDark).default },
      hoveredEdgeId: null,
      isEditMode: false,
      isReadOnly: true,
      stableEdgeDelete: () => undefined,
      factoryRunLeafLayout: layout,
    }) ?? rawEdges;

  return { nodes, edges };
}
