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
const HANDLE_OFFSET = 24;

type Point = { x: number; y: number };

export type CanvasEdgePathParams = {
  sourceX: number;
  sourceY: number;
  sourcePosition: Position;
  targetX: number;
  targetY: number;
  targetPosition: Position;
  /** When set, route via this X gutter (factory run long / cross-column edges). */
  routeGutterX?: number;
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

/**
 * Factory / vertical loop-back (source below target).
 * - Side gutter clears the target node (not through its body).
 * - Approaches the Top handle at handle Y — no U-hook above the node.
 */
function getVerticalUpwardLoopPath(params: CanvasEdgePathParams): [path: string, labelX: number, labelY: number] {
  const { sourceX, sourceY, targetX, targetY } = params;
  // Clear ~half default factory node (140) + pad so the riser sits outside the card.
  const sideClearance = Math.max(BACKWARD_ROUTE_OFFSET * 2, 160);
  const gutterX =
    targetX <= sourceX + SAME_COLUMN_TOLERANCE
      ? Math.min(sourceX, targetX) - sideClearance
      : Math.max(sourceX, targetX) + sideClearance;

  const exitY = sourceY + HANDLE_OFFSET;
  // Horizontal into the Top handle — do not overshoot above targetY.
  const points: Point[] = [
    { x: sourceX, y: sourceY },
    { x: sourceX, y: exitY },
    { x: gutterX, y: exitY },
    { x: gutterX, y: targetY },
    { x: targetX, y: targetY },
  ];

  const path = buildOrthogonalPath(points, SMOOTH_STEP_BORDER_RADIUS);
  return [path, gutterX, (exitY + targetY) / 2];
}

function getVerticalBackwardEdgePath(params: CanvasEdgePathParams): [path: string, labelX: number, labelY: number] {
  const { sourceX, sourceY, targetX, targetY } = params;

  if (targetY < sourceY - SAME_ROW_TOLERANCE) {
    return getVerticalUpwardLoopPath(params);
  }

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

  // Soft corners so factory loops read as curves, not sharp rectangles.
  const path = buildOrthogonalPath(points, SMOOTH_STEP_BORDER_RADIUS);

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

/**
 * Exit down/right → travel vertical gutter → enter target.
 * Keeps long factory edges clear of node cards between columns.
 */
export function getRightGutterEdgePath(params: CanvasEdgePathParams): [path: string, labelX: number, labelY: number] {
  const { sourceX, sourceY, targetX, targetY, routeGutterX } = params;
  const gutterX = routeGutterX ?? Math.max(sourceX, targetX) + BACKWARD_ROUTE_OFFSET;
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

  const path = buildOrthogonalPath(points, SMOOTH_STEP_BORDER_RADIUS);
  return [path, gutterX, (exitY + entryY) / 2];
}

/** Rough check: does the polyline path enter the axis-aligned rect? */
export function doesEdgePathIntersectRect(
  path: string,
  rect: { x: number; y: number; width: number; height: number },
  sampleCount = 48,
): boolean {
  const points = samplePathPoints(path, sampleCount);
  const left = rect.x;
  const right = rect.x + rect.width;
  const top = rect.y;
  const bottom = rect.y + rect.height;

  for (const point of points) {
    if (point.x >= left && point.x <= right && point.y >= top && point.y <= bottom) {
      return true;
    }
  }
  return false;
}

export type PathSamplePoint = Point & { angleDeg: number };

/** Edges longer than this get flow chevrons. */
export const LONG_EDGE_PATH_LENGTH = 320;
const FLOW_ARROW_END_MARGIN = 72;
const FLOW_ARROW_DEFAULT_SPACING = 220;
const FLOW_ARROW_MAX_COUNT = 3;

function cubicBezierPoint(p0: Point, p1: Point, p2: Point, p3: Point, t: number): Point {
  const u = 1 - t;
  const tt = t * t;
  const uu = u * u;
  const uuu = uu * u;
  const ttt = tt * t;
  return {
    x: uuu * p0.x + 3 * uu * t * p1.x + 3 * u * tt * p2.x + ttt * p3.x,
    y: uuu * p0.y + 3 * uu * t * p1.y + 3 * u * tt * p2.y + ttt * p3.y,
  };
}

function quadBezierPoint(p0: Point, p1: Point, p2: Point, t: number): Point {
  const u = 1 - t;
  return {
    x: u * u * p0.x + 2 * u * t * p1.x + t * t * p2.x,
    y: u * u * p0.y + 2 * u * t * p1.y + t * t * p2.y,
  };
}

/**
 * Sample points on the real stroke (cubic/quad curves evaluated, not control-point chords).
 * Naive coord chords pull labels/chevrons off bezier edges.
 */
function samplePathPoints(path: string, sampleCount: number): Point[] {
  const tokens = [...path.matchAll(/([MLQCHlVZmlqchlvz])|(-?\d+(?:\.\d+)?)/g)];
  const sampled: Point[] = [];
  let current: Point = { x: 0, y: 0 };
  let i = 0;

  const pushLine = (to: Point, steps: number) => {
    for (let s = 0; s <= steps; s++) {
      const t = s / steps;
      sampled.push({
        x: current.x + (to.x - current.x) * t,
        y: current.y + (to.y - current.y) * t,
      });
    }
    current = to;
  };

  const pushCubic = (c1: Point, c2: Point, to: Point, steps: number) => {
    const from = current;
    for (let s = 0; s <= steps; s++) {
      sampled.push(cubicBezierPoint(from, c1, c2, to, s / steps));
    }
    current = to;
  };

  const pushQuad = (c1: Point, to: Point, steps: number) => {
    const from = current;
    for (let s = 0; s <= steps; s++) {
      sampled.push(quadBezierPoint(from, c1, to, s / steps));
    }
    current = to;
  };

  while (i < tokens.length) {
    const token = tokens[i][0];
    if (!/^[MLQCHlVZmlqchlvz]$/.test(token)) {
      i += 1;
      continue;
    }
    const cmd = token;
    i += 1;
    const args: number[] = [];
    while (i < tokens.length && tokens[i][2] != null) {
      args.push(Number(tokens[i][0]));
      i += 1;
    }

    const steps = Math.max(2, Math.floor(sampleCount / 4));

    if (cmd === "M" || cmd === "m") {
      for (let a = 0; a + 1 < args.length; a += 2) {
        const next =
          cmd === "M"
            ? { x: args[a], y: args[a + 1] }
            : { x: current.x + args[a], y: current.y + args[a + 1] };
        if (a === 0 && sampled.length === 0) {
          current = next;
          sampled.push({ ...current });
        } else {
          pushLine(next, steps);
        }
      }
      continue;
    }

    if (cmd === "L" || cmd === "l") {
      for (let a = 0; a + 1 < args.length; a += 2) {
        const next =
          cmd === "L"
            ? { x: args[a], y: args[a + 1] }
            : { x: current.x + args[a], y: current.y + args[a + 1] };
        pushLine(next, steps);
      }
      continue;
    }

    if (cmd === "C" || cmd === "c") {
      for (let a = 0; a + 5 < args.length; a += 6) {
        const c1 =
          cmd === "C"
            ? { x: args[a], y: args[a + 1] }
            : { x: current.x + args[a], y: current.y + args[a + 1] };
        const c2 =
          cmd === "C"
            ? { x: args[a + 2], y: args[a + 3] }
            : { x: current.x + args[a + 2], y: current.y + args[a + 3] };
        const to =
          cmd === "C"
            ? { x: args[a + 4], y: args[a + 5] }
            : { x: current.x + args[a + 4], y: current.y + args[a + 5] };
        pushCubic(c1, c2, to, Math.max(8, steps * 2));
      }
      continue;
    }

    if (cmd === "Q" || cmd === "q") {
      for (let a = 0; a + 3 < args.length; a += 4) {
        const c1 =
          cmd === "Q"
            ? { x: args[a], y: args[a + 1] }
            : { x: current.x + args[a], y: current.y + args[a + 1] };
        const to =
          cmd === "Q"
            ? { x: args[a + 2], y: args[a + 3] }
            : { x: current.x + args[a + 2], y: current.y + args[a + 3] };
        pushQuad(c1, to, Math.max(6, steps));
      }
      continue;
    }

    // Fallback: treat remaining number pairs as line targets (H/V/etc. rare here).
    if (args.length >= 2) {
      const nums = args;
      for (let a = 0; a + 1 < nums.length; a += 2) {
        pushLine({ x: nums[a], y: nums[a + 1] }, steps);
      }
    }
  }

  if (sampled.length >= 2) {
    return sampled;
  }

  // Last-resort legacy parse for unexpected path shapes.
  const coords = [...path.matchAll(/(-?\d+(?:\.\d+)?)\s*[,\s]\s*(-?\d+(?:\.\d+)?)/g)].map((match) => ({
    x: Number(match[1]),
    y: Number(match[2]),
  }));
  return coords;
}

type PathPolyline = {
  points: Point[];
  segmentLengths: number[];
  totalLength: number;
};

function buildPathPolyline(path: string, sampleCount = 64): PathPolyline {
  const points = samplePathPoints(path, sampleCount);
  const segmentLengths: number[] = [];
  let totalLength = 0;
  for (let i = 0; i < points.length - 1; i++) {
    const len = distance(points[i], points[i + 1]);
    segmentLengths.push(len);
    totalLength += len;
  }
  return { points, segmentLengths, totalLength };
}

export function estimatePathLength(path: string): number {
  return buildPathPolyline(path).totalLength;
}

function pointAtDistance(polyline: PathPolyline, distanceAlong: number): PathSamplePoint | null {
  const { points, segmentLengths, totalLength } = polyline;
  if (points.length === 0) return null;
  if (points.length === 1 || totalLength <= 0) {
    return { ...points[0], angleDeg: 0 };
  }

  const clamped = Math.max(0, Math.min(distanceAlong, totalLength));
  let remaining = clamped;
  for (let i = 0; i < segmentLengths.length; i++) {
    const segLen = segmentLengths[i];
    if (remaining > segLen && i < segmentLengths.length - 1) {
      remaining -= segLen;
      continue;
    }
    const t = segLen <= 0 ? 0 : remaining / segLen;
    const a = points[i];
    const b = points[i + 1];
    const angleDeg = (Math.atan2(b.y - a.y, b.x - a.x) * 180) / Math.PI;
    return {
      x: a.x + (b.x - a.x) * t,
      y: a.y + (b.y - a.y) * t,
      angleDeg,
    };
  }
  const last = points[points.length - 1];
  const prev = points[points.length - 2];
  return {
    ...last,
    angleDeg: (Math.atan2(last.y - prev.y, last.x - prev.x) * 180) / Math.PI,
  };
}

/** Point + tangent at a fraction along the SVG path (0 = source, 1 = target). */
export function getPointAlongPath(path: string, fraction: number): PathSamplePoint | null {
  const polyline = buildPathPolyline(path);
  if (polyline.points.length === 0) return null;
  return pointAtDistance(polyline, polyline.totalLength * Math.max(0, Math.min(1, fraction)));
}

/**
 * Sparse flow chevrons along long edges (——>——>).
 * Skips short paths; keeps clear of source/target ends.
 */
export function getFlowArrowPoints(path: string, spacing = FLOW_ARROW_DEFAULT_SPACING): PathSamplePoint[] {
  const polyline = buildPathPolyline(path);
  if (polyline.totalLength < LONG_EDGE_PATH_LENGTH) {
    return [];
  }

  const usable = polyline.totalLength - FLOW_ARROW_END_MARGIN * 2;
  if (usable <= 0) return [];

  const count = Math.max(1, Math.min(FLOW_ARROW_MAX_COUNT, Math.floor(usable / spacing)));
  const step = usable / (count + 1);
  const arrows: PathSamplePoint[] = [];
  for (let i = 1; i <= count; i++) {
    const point = pointAtDistance(polyline, FLOW_ARROW_END_MARGIN + step * i);
    if (point) arrows.push(point);
  }
  return arrows;
}

/**
 * Flat stroke for the longer edge in a touch/cross pair (CSS + label border).
 * Short fork stays pale so the two wires read apart at the junction.
 */
export const TOUCHING_EDGE_STROKE = "#64748b";
export const TOUCHING_EDGE_STROKE_DARK = "#94a3b8";
/** Default factory edge stroke — chevrons match this when not contrast. */
export const DEFAULT_FACTORY_EDGE_STROKE = "#cbd5e1";
/** Class stamped on contrast edges — must beat `--xy-edge-stroke !important`. */
export const FACTORY_TOUCHING_EDGE_CLASS = "sp-edge-factory-touching";

const PATH_TOUCH_THRESHOLD_PX = 14;
/** Ignore proximity near shared source/target ends so normal joins are not "tangles". */
const PATH_TOUCH_ENDPOINT_IGNORE_PX = 28;

function clamp01(value: number): number {
  return Math.max(0, Math.min(1, value));
}

/** Minimum distance between two finite segments. */
function segmentDistance(a1: Point, a2: Point, b1: Point, b2: Point): number {
  const ax = a2.x - a1.x;
  const ay = a2.y - a1.y;
  const bx = b2.x - b1.x;
  const by = b2.y - b1.y;
  const cx = a1.x - b1.x;
  const cy = a1.y - b1.y;

  const aLenSq = ax * ax + ay * ay;
  const bLenSq = bx * bx + by * by;
  const aDotB = ax * bx + ay * by;
  const aDotC = ax * cx + ay * cy;
  const bDotC = bx * cx + by * cy;

  let s: number;
  let t: number;
  const denom = aLenSq * bLenSq - aDotB * aDotB;

  if (aLenSq <= 0 && bLenSq <= 0) {
    return distance(a1, b1);
  }
  if (aLenSq <= 0) {
    t = clamp01(bDotC / bLenSq);
    return distance(a1, { x: b1.x + bx * t, y: b1.y + by * t });
  }
  if (bLenSq <= 0) {
    s = clamp01(-aDotC / aLenSq);
    return distance({ x: a1.x + ax * s, y: a1.y + ay * s }, b1);
  }

  if (Math.abs(denom) < 1e-8) {
    // Parallel — project a1 onto B and clamp.
    t = clamp01(bDotC / bLenSq);
    s = clamp01((-aDotC + aDotB * t) / aLenSq);
  } else {
    s = clamp01((aDotB * bDotC - bLenSq * aDotC) / denom);
    t = clamp01((aLenSq * bDotC - aDotB * aDotC) / denom);
  }

  const pa = { x: a1.x + ax * s, y: a1.y + ay * s };
  const pb = { x: b1.x + bx * t, y: b1.y + by * t };
  return distance(pa, pb);
}

function isNearPolylineEndpoint(point: Point, polyline: Point[], ignorePx: number): boolean {
  if (polyline.length === 0) return false;
  const start = polyline[0];
  const end = polyline[polyline.length - 1];
  return distance(point, start) <= ignorePx || distance(point, end) <= ignorePx;
}

/**
 * True when two SVG edge paths cross or run within threshold,
 * ignoring proximity only at shared endpoints (normal joins).
 */
export function pathsTouch(
  pathA: string,
  pathB: string,
  thresholdPx = PATH_TOUCH_THRESHOLD_PX,
  endpointIgnorePx = PATH_TOUCH_ENDPOINT_IGNORE_PX,
): boolean {
  const pointsA = samplePathPoints(pathA, 48);
  const pointsB = samplePathPoints(pathB, 48);
  if (pointsA.length < 2 || pointsB.length < 2) return false;

  for (let i = 0; i < pointsA.length - 1; i++) {
    const a1 = pointsA[i];
    const a2 = pointsA[i + 1];
    for (let j = 0; j < pointsB.length - 1; j++) {
      const b1 = pointsB[j];
      const b2 = pointsB[j + 1];
      const dist = segmentDistance(a1, a2, b1, b2);
      if (dist > thresholdPx) continue;

      // Midpoint of the closest approach — reject if both sides are at an endpoint join.
      const mid = {
        x: (a1.x + a2.x + b1.x + b2.x) / 4,
        y: (a1.y + a2.y + b1.y + b2.y) / 4,
      };
      const nearEndA = isNearPolylineEndpoint(mid, pointsA, endpointIgnorePx);
      const nearEndB = isNearPolylineEndpoint(mid, pointsB, endpointIgnorePx);
      if (nearEndA && nearEndB) continue;

      return true;
    }
  }
  return false;
}

export type TouchEdgePathEntry = {
  id: string;
  path: string;
  source?: string;
  target?: string;
};

/** Edge ids whose paths geometrically touch/cross at least one other edge. */
export function findTouchingEdgeIds(edges: TouchEdgePathEntry[]): Set<string> {
  return findTouchContrastEdgeIds(edges).involvedIds;
}

export type TouchContrastEdgeIds = {
  /** Longer edge in each touch pair — gets the darker stroke. */
  contrastIds: Set<string>;
  /** Both edges in every touch pair — labels may need a source-ward nudge. */
  involvedIds: Set<string>;
};

/**
 * For each geometric touch/cross, darken only the longer path so the pair
 * keeps contrast (short true/false forks stay pale).
 * Skips pairs that only meet at a shared source/target node (normal fork/merge).
 */
export function findTouchContrastEdgeIds(edges: TouchEdgePathEntry[]): TouchContrastEdgeIds {
  const contrastIds = new Set<string>();
  const involvedIds = new Set<string>();
  const lengths = edges.map((edge) => estimatePathLength(edge.path));

  for (let i = 0; i < edges.length; i++) {
    for (let j = i + 1; j < edges.length; j++) {
      const a = edges[i];
      const b = edges[j];
      // Fan-out / fan-in at the same node is not a mid-path tangle.
      if (a.source && a.source === b.source) continue;
      if (a.target && a.target === b.target) continue;
      if (!pathsTouch(a.path, b.path)) continue;
      involvedIds.add(a.id);
      involvedIds.add(b.id);

      const lenA = lengths[i];
      const lenB = lengths[j];
      if (lenA > lenB) {
        contrastIds.add(a.id);
      } else if (lenB > lenA) {
        contrastIds.add(b.id);
      } else if (a.id.localeCompare(b.id) <= 0) {
        contrastIds.add(a.id);
      } else {
        contrastIds.add(b.id);
      }
    }
  }
  return { contrastIds, involvedIds };
}

export function getCanvasEdgePath(params: CanvasEdgePathParams): [path: string, labelX: number, labelY: number] {
  if (typeof params.routeGutterX === "number" && Number.isFinite(params.routeGutterX)) {
    return getRightGutterEdgePath(params);
  }

  // Factory (vertical) canvases use curvy bezier edges like WorkOrderCanvas Storybook.
  if (isVerticalFlowEdge(params)) {
    if (isBackwardEdge(params)) {
      return getVerticalBackwardEdgePath(params);
    }
    const [path, labelX, labelY] = getBezierPath(params);
    return [path, labelX, labelY];
  }

  if (isBackwardEdge(params)) {
    return getHorizontalBackwardEdgePath(params);
  }

  const [path, labelX, labelY] = getBezierPath(params);

  return [path, labelX, labelY];
}
