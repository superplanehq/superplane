export type Point = { x: number; y: number };

export type PathSamplePoint = Point & { angleDeg: number };

/** Edges longer than this get flow chevrons. */
export const LONG_EDGE_PATH_LENGTH = 320;
const FLOW_ARROW_END_MARGIN = 72;
const FLOW_ARROW_DEFAULT_SPACING = 220;
const FLOW_ARROW_MAX_COUNT = 3;

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

export function distance(a: Point, b: Point): number {
  return Math.hypot(b.x - a.x, b.y - a.y);
}

function clamp01(value: number): number {
  return Math.max(0, Math.min(1, value));
}

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

function resolvePoint(absolute: boolean, current: Point, x: number, y: number): Point {
  if (absolute) return { x, y };
  return { x: current.x + x, y: current.y + y };
}

function legacyCoordParse(path: string): Point[] {
  return [...path.matchAll(/(-?\d+(?:\.\d+)?)\s*[,\s]\s*(-?\d+(?:\.\d+)?)/g)].map((match) => ({
    x: Number(match[1]),
    y: Number(match[2]),
  }));
}

type PathSampleState = {
  current: Point;
  sampled: Point[];
};

function tokenizePath(path: string): RegExpMatchArray[] {
  return [...path.matchAll(/([MLQCHlVZmlqchlvz])|(-?\d+(?:\.\d+)?)/g)];
}

function takeCommandArgs(tokens: RegExpMatchArray[], startIndex: number): { args: number[]; nextIndex: number } {
  const args: number[] = [];
  let i = startIndex;
  while (i < tokens.length && tokens[i][2] != null) {
    args.push(Number(tokens[i][0]));
    i += 1;
  }
  return { args, nextIndex: i };
}

function pushLineSample(state: PathSampleState, to: Point, steps: number): void {
  for (let s = 0; s <= steps; s++) {
    const t = s / steps;
    state.sampled.push({
      x: state.current.x + (to.x - state.current.x) * t,
      y: state.current.y + (to.y - state.current.y) * t,
    });
  }
  state.current = to;
}

function pushCubicSample(state: PathSampleState, c1: Point, c2: Point, to: Point, steps: number): void {
  const from = state.current;
  for (let s = 0; s <= steps; s++) {
    state.sampled.push(cubicBezierPoint(from, c1, c2, to, s / steps));
  }
  state.current = to;
}

function pushQuadSample(state: PathSampleState, c1: Point, to: Point, steps: number): void {
  const from = state.current;
  for (let s = 0; s <= steps; s++) {
    state.sampled.push(quadBezierPoint(from, c1, to, s / steps));
  }
  state.current = to;
}

function applyMoveCommand(state: PathSampleState, absolute: boolean, args: number[], steps: number): void {
  for (let a = 0; a + 1 < args.length; a += 2) {
    const next = resolvePoint(absolute, state.current, args[a], args[a + 1]);
    if (a === 0 && state.sampled.length === 0) {
      state.current = next;
      state.sampled.push({ ...state.current });
      continue;
    }
    pushLineSample(state, next, steps);
  }
}

function applyLineCommand(state: PathSampleState, absolute: boolean, args: number[], steps: number): void {
  for (let a = 0; a + 1 < args.length; a += 2) {
    pushLineSample(state, resolvePoint(absolute, state.current, args[a], args[a + 1]), steps);
  }
}

function applyCubicCommand(state: PathSampleState, absolute: boolean, args: number[], steps: number): void {
  for (let a = 0; a + 5 < args.length; a += 6) {
    const c1 = resolvePoint(absolute, state.current, args[a], args[a + 1]);
    const c2 = resolvePoint(absolute, state.current, args[a + 2], args[a + 3]);
    const to = resolvePoint(absolute, state.current, args[a + 4], args[a + 5]);
    pushCubicSample(state, c1, c2, to, Math.max(8, steps * 2));
  }
}

function applyQuadCommand(state: PathSampleState, absolute: boolean, args: number[], steps: number): void {
  for (let a = 0; a + 3 < args.length; a += 4) {
    const c1 = resolvePoint(absolute, state.current, args[a], args[a + 1]);
    const to = resolvePoint(absolute, state.current, args[a + 2], args[a + 3]);
    pushQuadSample(state, c1, to, Math.max(6, steps));
  }
}

function applyPathCommand(state: PathSampleState, cmd: string, args: number[], steps: number): void {
  if (cmd === "M" || cmd === "m") {
    applyMoveCommand(state, cmd === "M", args, steps);
    return;
  }
  if (cmd === "L" || cmd === "l") {
    applyLineCommand(state, cmd === "L", args, steps);
    return;
  }
  if (cmd === "C" || cmd === "c") {
    applyCubicCommand(state, cmd === "C", args, steps);
    return;
  }
  if (cmd === "Q" || cmd === "q") {
    applyQuadCommand(state, cmd === "Q", args, steps);
    return;
  }
  // Fallback: treat remaining number pairs as line targets (H/V/etc. rare here).
  if (args.length < 2) return;
  for (let a = 0; a + 1 < args.length; a += 2) {
    pushLineSample(state, { x: args[a], y: args[a + 1] }, steps);
  }
}

/**
 * Sample points on the real stroke (cubic/quad curves evaluated, not control-point chords).
 * Naive coord chords pull labels/chevrons off bezier edges.
 */
export function samplePathPoints(path: string, sampleCount: number): Point[] {
  const tokens = tokenizePath(path);
  if (tokens.length === 0) return legacyCoordParse(path);

  const state: PathSampleState = { current: { x: 0, y: 0 }, sampled: [] };
  let i = 0;
  while (i < tokens.length) {
    const token = tokens[i][0];
    if (!/^[MLQCHlVZmlqchlvz]$/.test(token)) {
      i += 1;
      continue;
    }
    i += 1;
    const { args, nextIndex } = takeCommandArgs(tokens, i);
    i = nextIndex;
    applyPathCommand(state, token, args, Math.max(2, Math.floor(sampleCount / 4)));
  }

  if (state.sampled.length >= 2) return state.sampled;
  return legacyCoordParse(path);
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
  if (polyline.totalLength < LONG_EDGE_PATH_LENGTH) return [];

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

type SegmentGeom = {
  aLenSq: number;
  bLenSq: number;
  aDotB: number;
  aDotC: number;
  bDotC: number;
};

function closestSegmentParams(geom: SegmentGeom): { s: number; t: number } {
  const { aLenSq, bLenSq, aDotB, aDotC, bDotC } = geom;
  const denom = aLenSq * bLenSq - aDotB * aDotB;
  if (Math.abs(denom) < 1e-8) {
    // Parallel — project a1 onto B and clamp.
    const t = clamp01(bDotC / bLenSq);
    const s = clamp01((-aDotC + aDotB * t) / aLenSq);
    return { s, t };
  }
  return {
    s: clamp01((aDotB * bDotC - bLenSq * aDotC) / denom),
    t: clamp01((aLenSq * bDotC - aDotB * aDotC) / denom),
  };
}

/** Minimum distance between two finite segments. */
export function segmentDistance(a1: Point, a2: Point, b1: Point, b2: Point): number {
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

  if (aLenSq <= 0 && bLenSq <= 0) return distance(a1, b1);
  if (aLenSq <= 0) {
    const t = clamp01(bDotC / bLenSq);
    return distance(a1, { x: b1.x + bx * t, y: b1.y + by * t });
  }
  if (bLenSq <= 0) {
    const s = clamp01(-aDotC / aLenSq);
    return distance({ x: a1.x + ax * s, y: a1.y + ay * s }, b1);
  }

  const { s, t } = closestSegmentParams({ aLenSq, bLenSq, aDotB, aDotC, bDotC });
  return distance({ x: a1.x + ax * s, y: a1.y + ay * s }, { x: b1.x + bx * t, y: b1.y + by * t });
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
