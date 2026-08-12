import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import type { CanvasFlowDirection } from "@/lib/canvasFlowDirection";

export type LayoutPosition = {
  x: number;
  y: number;
};

export type LayoutBounds = {
  minX: number;
  minY: number;
  maxX: number;
  maxY: number;
  width: number;
  height: number;
};

export function resolveLayoutBounds(
  layoutNodes: ComponentsNode[],
  layoutedPositions: Map<string, LayoutPosition>,
  estimateNodeSize: (node: ComponentsNode) => { width: number; height: number },
): LayoutBounds {
  let minX = Number.POSITIVE_INFINITY;
  let minY = Number.POSITIVE_INFINITY;
  let maxX = Number.NEGATIVE_INFINITY;
  let maxY = Number.NEGATIVE_INFINITY;

  for (const node of layoutNodes) {
    const nodeID = node.id;
    if (!nodeID) {
      continue;
    }

    const position = layoutedPositions.get(nodeID);
    if (!position) {
      continue;
    }

    const { width, height } = estimateNodeSize(node);
    minX = Math.min(minX, position.x);
    minY = Math.min(minY, position.y);
    maxX = Math.max(maxX, position.x + width);
    maxY = Math.max(maxY, position.y + height);
  }

  if (!Number.isFinite(minX) || !Number.isFinite(minY) || !Number.isFinite(maxX) || !Number.isFinite(maxY)) {
    return {
      minX: 0,
      minY: 0,
      maxX: 0,
      maxY: 0,
      width: 0,
      height: 0,
    };
  }

  return {
    minX,
    minY,
    maxX,
    maxY,
    width: maxX - minX,
    height: maxY - minY,
  };
}

export function resolveMinPositionFromNodes(nodes: ComponentsNode[]): LayoutPosition {
  let minX = Number.POSITIVE_INFINITY;
  let minY = Number.POSITIVE_INFINITY;

  for (const node of nodes) {
    minX = Math.min(minX, node.position?.x || 0);
    minY = Math.min(minY, node.position?.y || 0);
  }

  if (!Number.isFinite(minX)) {
    minX = 0;
  }

  if (!Number.isFinite(minY)) {
    minY = 0;
  }

  return { x: minX, y: minY };
}

export function resolveMinPositionFromLayout(layoutedPositions: Map<string, LayoutPosition>): LayoutPosition {
  let minX = Number.POSITIVE_INFINITY;
  let minY = Number.POSITIVE_INFINITY;

  layoutedPositions.forEach((position) => {
    minX = Math.min(minX, position.x);
    minY = Math.min(minY, position.y);
  });

  if (!Number.isFinite(minX)) {
    minX = 0;
  }

  if (!Number.isFinite(minY)) {
    minY = 0;
  }

  return { x: minX, y: minY };
}

export function sortComponentsByCurrentPosition(
  components: ComponentsNode[][],
  direction: CanvasFlowDirection,
): ComponentsNode[][] {
  return [...components].sort((componentA, componentB) => {
    const a = resolveMinPositionFromNodes(componentA);
    const b = resolveMinPositionFromNodes(componentB);

    if (direction === "vertical") {
      if (a.x !== b.x) {
        return a.x - b.x;
      }

      return a.y - b.y;
    }

    if (a.y !== b.y) {
      return a.y - b.y;
    }

    return a.x - b.x;
  });
}

export function applyLayoutedPositions(
  nodes: ComponentsNode[],
  layoutedPositions: Map<string, LayoutPosition>,
  offset: LayoutPosition,
): ComponentsNode[] {
  return nodes.map((node) => {
    const nodeID = node.id;
    if (!nodeID) {
      return node;
    }

    const position = layoutedPositions.get(nodeID);
    if (!position) {
      return node;
    }

    return {
      ...node,
      position: {
        x: Math.round(position.x + offset.x),
        y: Math.round(position.y + offset.y),
      },
    };
  });
}

export function packComponentPositions(
  componentPositions: Map<string, LayoutPosition>,
  bounds: LayoutBounds,
  currentPackOffset: number,
  packAlongCrossAxis: boolean,
): Map<string, LayoutPosition> {
  const packed = new Map<string, LayoutPosition>();
  for (const [nodeID, position] of componentPositions.entries()) {
    packed.set(nodeID, {
      x: position.x - bounds.minX + (packAlongCrossAxis ? currentPackOffset : 0),
      y: position.y - bounds.minY + (packAlongCrossAxis ? 0 : currentPackOffset),
    });
  }
  return packed;
}
