import { Position, getBezierPath, getSmoothStepPath } from "@xyflow/react";

const BACKWARD_ROUTE_OFFSET = 80;
const BACKWARD_ROUTE_TARGET_BIAS = 0.75;
const CANVAS_NODE_HEIGHT = 280;
const MIN_DISTANCE_FROM_TARGET_TOP = 50;
const MIN_DISTANCE_FROM_SOURCE_BOTTOM = 50;
const SAME_ROW_TOLERANCE = 40;
const SAME_COLUMN_TOLERANCE = 40;
const TARGET_HANDLE_TOP_OFFSET = 18;
const SMOOTH_STEP_BORDER_RADIUS = 16;
const RECT_BORDER_RADIUS = 0;
const HANDLE_OFFSET = 24;

type Point = { x: number; y: number };

export type CanvasEdgePathParams = {
  sourceX: number;
  sourceY: number;
  sourcePosition: Position;
  targetX: number;
  targetY: number;
  targetPosition: Position;
};

export function isVerticalFlowEdge({
  sourcePosition,
  targetPosition,
}: Pick<CanvasEdgePathParams, "sourcePosition" | "targetPosition">): boolean {
  return (
    (sourcePosition === Position.Bottom && targetPosition === Position.Top) ||
    (sourcePosition === Position.Top && targetPosition === Position.Bottom)
  );
}

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

function distance(a: Point, b: Point): number {
  return Math.hypot(b.x - a.x, b.y - a.y);
}

function getBend(a: Point, b: Point, c: Point, size: number): string {
  const bendSize = Math.min(distance(a, b) / 2, distance(b, c) / 2, size);
  const { x, y } = b;

  if ((a.x === x && x === c.x) || (a.y === y && y === c.y)) {
    return `L ${x} ${y}`;
  }

  if (a.y === y) {
    const xDir = a.x < c.x ? -1 : 1;
    const yDir = a.y < c.y ? 1 : -1;
    return `L ${x + bendSize * xDir},${y}Q ${x},${y} ${x},${y + bendSize * yDir}`;
  }

  const xDir = a.x < c.x ? 1 : -1;
  const yDir = a.y < c.y ? -1 : 1;
  return `L ${x},${y + bendSize * yDir}Q ${x},${y} ${x + bendSize * xDir},${y}`;
}

function buildOrthogonalPath(points: Point[], borderRadius: number): string {
  let path = `M ${points[0].x} ${points[0].y}`;

  for (let i = 1; i < points.length - 1; i++) {
    if (borderRadius <= 0) {
      path += ` L ${points[i].x} ${points[i].y}`;
      continue;
    }

    path += ` ${getBend(points[i - 1], points[i], points[i + 1], borderRadius)}`;
  }

  const last = points[points.length - 1];
  path += ` L ${last.x} ${last.y}`;

  return path;
}

export function getUpwardBackwardGutterY(sourceY: number, targetY: number): number {
  const targetTop = targetY - TARGET_HANDLE_TOP_OFFSET;
  const targetBottom = targetTop + CANVAS_NODE_HEIGHT;
  const sourceTop = sourceY - TARGET_HANDLE_TOP_OFFSET;
  const gap = sourceTop - targetBottom;

  if (gap > 0) {
    return Math.min(
      targetBottom + Math.min(MIN_DISTANCE_FROM_SOURCE_BOTTOM, gap * (1 - BACKWARD_ROUTE_TARGET_BIAS)),
      sourceTop - Math.min(MIN_DISTANCE_FROM_TARGET_TOP, gap * BACKWARD_ROUTE_TARGET_BIAS),
    );
  }

  return targetBottom + Math.min(MIN_DISTANCE_FROM_SOURCE_BOTTOM, 8);
}

export function getBackwardRouteCenterY(sourceY: number, targetY: number): number {
  const verticalDelta = targetY - sourceY;

  if (Math.abs(verticalDelta) <= SAME_ROW_TOLERANCE) {
    return Math.max(sourceY, targetY) + BACKWARD_ROUTE_OFFSET;
  }

  if (verticalDelta < 0) {
    return getUpwardBackwardGutterY(sourceY, targetY);
  }

  const biasedY = sourceY + verticalDelta * BACKWARD_ROUTE_TARGET_BIAS;
  const targetTop = targetY - TARGET_HANDLE_TOP_OFFSET;
  const maxCenterY = targetTop - MIN_DISTANCE_FROM_TARGET_TOP;

  return Math.min(biasedY, maxCenterY);
}

function getUpwardBackwardEdgePath(
  params: CanvasEdgePathParams,
  borderRadius: number,
): [path: string, labelX: number, labelY: number] {
  const { sourceX, sourceY, targetX, targetY } = params;
  const gutterY = getUpwardBackwardGutterY(sourceY, targetY);
  const exitX = sourceX + HANDLE_OFFSET;
  const entryX = targetX - HANDLE_OFFSET;

  const points: Point[] = [
    { x: sourceX, y: sourceY },
    { x: exitX, y: sourceY },
    { x: exitX, y: gutterY },
    { x: entryX, y: gutterY },
    { x: entryX, y: targetY },
    { x: targetX, y: targetY },
  ];

  const path = buildOrthogonalPath(points, borderRadius);

  return [path, (exitX + entryX) / 2, gutterY];
}

function getLeftwardBackwardGutterX(sourceX: number, targetX: number): number {
  const targetLeft = targetX;
  const sourceRight = sourceX;
  const gap = sourceRight - targetLeft;

  if (gap > 0) {
    return Math.min(
      targetLeft + Math.min(MIN_DISTANCE_FROM_SOURCE_BOTTOM, gap * (1 - BACKWARD_ROUTE_TARGET_BIAS)),
      sourceRight - Math.min(MIN_DISTANCE_FROM_TARGET_TOP, gap * BACKWARD_ROUTE_TARGET_BIAS),
    );
  }

  return targetLeft - BACKWARD_ROUTE_OFFSET;
}

function getVerticalBackwardEdgePath(params: CanvasEdgePathParams): [path: string, labelX: number, labelY: number] {
  const { sourceX, sourceY, targetX, targetY } = params;
  const horizontalDelta = targetX - sourceX;
  const gutterX =
    Math.abs(horizontalDelta) <= SAME_COLUMN_TOLERANCE
      ? Math.max(sourceX, targetX) + BACKWARD_ROUTE_OFFSET
      : horizontalDelta < 0
        ? getLeftwardBackwardGutterX(sourceX, targetX)
        : sourceX + horizontalDelta * BACKWARD_ROUTE_TARGET_BIAS;

  const exitY = sourceY + HANDLE_OFFSET;
  const entryY = targetY - HANDLE_OFFSET;

  const points: Point[] = [
    { x: sourceX, y: sourceY },
    { x: sourceX, y: exitY },
    { x: gutterX, y: exitY },
    { x: gutterX, y: entryY },
    { x: targetX, y: entryY },
    { x: targetX, y: targetY },
  ];

  const path = buildOrthogonalPath(points, RECT_BORDER_RADIUS);

  return [path, gutterX, (exitY + entryY) / 2];
}

function getHorizontalBackwardEdgePath(params: CanvasEdgePathParams): [path: string, labelX: number, labelY: number] {
  const { sourceX, sourceY, sourcePosition, targetX, targetY, targetPosition } = params;
  const verticalDelta = targetY - sourceY;

  if (verticalDelta < -SAME_ROW_TOLERANCE) {
    return getUpwardBackwardEdgePath(params, SMOOTH_STEP_BORDER_RADIUS);
  }

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

function getRectEdgePath(params: CanvasEdgePathParams): [path: string, labelX: number, labelY: number] {
  if (isBackwardEdge(params)) {
    if (isVerticalFlowEdge(params)) {
      return getVerticalBackwardEdgePath(params);
    }

    return getHorizontalBackwardEdgePath(params);
  }

  const [path, labelX, labelY] = getSmoothStepPath({
    ...params,
    borderRadius: RECT_BORDER_RADIUS,
  });

  return [path, labelX, labelY];
}

export function getCanvasEdgePath(params: CanvasEdgePathParams): [path: string, labelX: number, labelY: number] {
  if (isVerticalFlowEdge(params)) {
    return getRectEdgePath(params);
  }

  if (isBackwardEdge(params)) {
    return getHorizontalBackwardEdgePath(params);
  }

  const [path, labelX, labelY] = getBezierPath(params);

  return [path, labelX, labelY];
}
