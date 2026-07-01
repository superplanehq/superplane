import { describe, expect, it } from "vitest";
import { findFreePositionInViewport } from "./find-free-position-in-viewport";

const DEFAULT_VIEWPORT = { x: 0, y: 0, zoom: 1 };
const DEFAULT_CANVAS = { width: 1000, height: 800 };
const NOTE_SIZE = { width: 320, height: 160 };
const PADDING = 16;

interface Rect {
  minX: number;
  minY: number;
  maxX: number;
  maxY: number;
}

function rectFor(position: { x: number; y: number }, size: { width: number; height: number }): Rect {
  return {
    minX: position.x,
    minY: position.y,
    maxX: position.x + size.width,
    maxY: position.y + size.height,
  };
}

function overlaps(a: Rect, b: Rect): boolean {
  return a.minX < b.maxX && a.maxX > b.minX && a.minY < b.maxY && a.maxY > b.minY;
}

function visibleBoundsFor(
  viewport: { x: number; y: number; zoom: number },
  canvas: { width: number; height: number },
): Rect {
  return {
    minX: -viewport.x / viewport.zoom,
    minY: -viewport.y / viewport.zoom,
    maxX: (canvas.width - viewport.x) / viewport.zoom,
    maxY: (canvas.height - viewport.y) / viewport.zoom,
  };
}

describe("findFreePositionInViewport", () => {
  it("places the node inside the viewport when the canvas is empty", () => {
    const result = findFreePositionInViewport({
      viewport: DEFAULT_VIEWPORT,
      canvasRect: DEFAULT_CANVAS,
      nodes: [],
      nodeSize: NOTE_SIZE,
    });

    const visible = visibleBoundsFor(DEFAULT_VIEWPORT, DEFAULT_CANVAS);
    const placed = rectFor(result, NOTE_SIZE);
    expect(placed.minX).toBeGreaterThanOrEqual(visible.minX + PADDING);
    expect(placed.minY).toBeGreaterThanOrEqual(visible.minY + PADDING);
    expect(placed.maxX).toBeLessThanOrEqual(visible.maxX - PADDING);
    expect(placed.maxY).toBeLessThanOrEqual(visible.maxY - PADDING);
  });

  it("returns a position that does not overlap any existing node", () => {
    // Pack a cluster of nodes tightly around the viewport center so the
    // fan-out has to do real work.
    const centerX = DEFAULT_CANVAS.width / 2 - NOTE_SIZE.width / 2;
    const centerY = DEFAULT_CANVAS.height / 2 - NOTE_SIZE.height / 2;
    const existing = [
      { position: { x: centerX, y: centerY }, width: NOTE_SIZE.width, height: NOTE_SIZE.height },
      { position: { x: centerX - 80, y: centerY }, width: NOTE_SIZE.width, height: NOTE_SIZE.height },
      { position: { x: centerX + 80, y: centerY }, width: NOTE_SIZE.width, height: NOTE_SIZE.height },
      { position: { x: centerX, y: centerY - 60 }, width: NOTE_SIZE.width, height: NOTE_SIZE.height },
      { position: { x: centerX, y: centerY + 60 }, width: NOTE_SIZE.width, height: NOTE_SIZE.height },
    ];

    const result = findFreePositionInViewport({
      viewport: DEFAULT_VIEWPORT,
      canvasRect: DEFAULT_CANVAS,
      nodes: existing,
      nodeSize: NOTE_SIZE,
    });

    const placed = rectFor(result, NOTE_SIZE);
    for (const node of existing) {
      expect(overlaps(placed, rectFor(node.position, { width: node.width, height: node.height }))).toBe(false);
    }
  });

  it("stays inside the visible viewport after pan and zoom", () => {
    const viewport = { x: -500, y: -200, zoom: 0.5 };

    const result = findFreePositionInViewport({
      viewport,
      canvasRect: DEFAULT_CANVAS,
      nodes: [],
      nodeSize: NOTE_SIZE,
    });

    const visible = visibleBoundsFor(viewport, DEFAULT_CANVAS);
    const placed = rectFor(result, NOTE_SIZE);
    expect(placed.minX).toBeGreaterThanOrEqual(visible.minX + PADDING);
    expect(placed.minY).toBeGreaterThanOrEqual(visible.minY + PADDING);
    expect(placed.maxX).toBeLessThanOrEqual(visible.maxX - PADDING);
    expect(placed.maxY).toBeLessThanOrEqual(visible.maxY - PADDING);
  });

  it("falls back to fallbackCanvasSize when canvasRect is missing", () => {
    // Covers the real-world case where the canvas element hasn't been measured
    // yet (no ResizeObserver tick) — the helper must still return a usable
    // position inside the fallback bounds instead of NaN / 0-size.
    const fallback = { width: 400, height: 300 };

    const result = findFreePositionInViewport({
      viewport: DEFAULT_VIEWPORT,
      canvasRect: null,
      nodes: [],
      nodeSize: NOTE_SIZE,
      fallbackCanvasSize: fallback,
    });

    const visible = visibleBoundsFor(DEFAULT_VIEWPORT, fallback);
    const placed = rectFor(result, NOTE_SIZE);
    expect(Number.isFinite(placed.minX)).toBe(true);
    expect(Number.isFinite(placed.minY)).toBe(true);
    expect(placed.minX).toBeGreaterThanOrEqual(visible.minX + PADDING);
    expect(placed.minY).toBeGreaterThanOrEqual(visible.minY + PADDING);
  });
});
