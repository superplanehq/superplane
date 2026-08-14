import type { Edge, Node } from "@xyflow/react";

export const FACTORY_LAYOUT_ANIMATION_DURATION_MS = 140;

function interpolate(start: number, end: number, progress: number): number {
  return start + (end - start) * progress;
}

export function easeOutQuadratic(progress: number): number {
  return 1 - (1 - progress) * (1 - progress);
}

export function createLayoutNodeInterpolator(currentNodes: Node[], edges: Edge[]) {
  const currentById = new Map(currentNodes.map((node) => [node.id, node]));
  const incomingByTarget = new Map(edges.map((edge) => [edge.target, edge.source]));

  return (targetNodes: Node[], progress: number): Node[] =>
    targetNodes.map((targetNode) => {
      const currentNode = currentById.get(targetNode.id);
      const sourceNode = currentById.get(incomingByTarget.get(targetNode.id) || "");
      const startPosition = currentNode?.position ?? sourceNode?.position ?? targetNode.position;
      const isNew = !currentNode;
      const isStationary =
        !isNew && startPosition.x === targetNode.position.x && startPosition.y === targetNode.position.y;

      if (isStationary) {
        return targetNode;
      }

      return {
        ...targetNode,
        position: {
          x: interpolate(startPosition.x, targetNode.position.x, progress),
          y: interpolate(startPosition.y, targetNode.position.y, progress),
        },
        style: isNew ? { ...targetNode.style, opacity: progress } : targetNode.style,
      };
    });
}

export function interpolateLayoutNodes(
  currentNodes: Node[],
  targetNodes: Node[],
  edges: Edge[],
  progress: number,
): Node[] {
  return createLayoutNodeInterpolator(currentNodes, edges)(targetNodes, progress);
}
