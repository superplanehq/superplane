import { Position, getBezierPath, getSmoothStepPath } from "@xyflow/react";

const BACKWARD_ROUTE_OFFSET = 80;
const SMOOTH_STEP_BORDER_RADIUS = 16;
const HANDLE_OFFSET = 24;

export type CanvasEdgePathParams = {
  sourceX: number;
  sourceY: number;
  sourcePosition: Position;
  targetX: number;
  targetY: number;
  targetPosition: Position;
};

export function isBackwardEdge({
  sourceX,
  targetX,
  targetY,
  sourceY,
  sourcePosition,
  targetPosition,
}: CanvasEdgePathParams): boolean {
  if (sourcePosition === Position.Right && targetPosition === Position.Left) {
    return targetX < sourceX;
  }

  if (sourcePosition === Position.Left && targetPosition === Position.Right) {
    return targetX > sourceX;
  }

  if (sourcePosition === Position.Bottom && targetPosition === Position.Top) {
    return targetY < sourceY;
  }

  if (sourcePosition === Position.Top && targetPosition === Position.Bottom) {
    return targetY > sourceY;
  }

  return false;
}

export function getBackwardRouteCenterY(sourceY: number, targetY: number): number {
  if (targetY > sourceY) {
    // Downward branch: route through the gap between the two nodes.
    return (sourceY + targetY) / 2;
  }

  // Loop-back: route below both nodes.
  return Math.max(sourceY, targetY) + BACKWARD_ROUTE_OFFSET;
}

function getBackwardEdgePath(params: CanvasEdgePathParams): [path: string, labelX: number, labelY: number] {
  const { sourceX, sourceY, sourcePosition, targetX, targetY, targetPosition } = params;
  const centerY = getBackwardRouteCenterY(sourceY, targetY);

  const [path, labelX, labelY] = getSmoothStepPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
    borderRadius: SMOOTH_STEP_BORDER_RADIUS,
    offset: HANDLE_OFFSET,
    centerY,
  });

  return [path, labelX, labelY];
}

export function getCanvasEdgePath(params: CanvasEdgePathParams): [path: string, labelX: number, labelY: number] {
  if (isBackwardEdge(params)) {
    return getBackwardEdgePath(params);
  }

  const [path, labelX, labelY] = getBezierPath(params);

  return [path, labelX, labelY];
}
