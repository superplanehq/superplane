import { describe, expect, it } from "vitest";
import { findFreePositionInViewport } from "./find-free-position-in-viewport";

const DEFAULT_VIEWPORT = { x: 0, y: 0, zoom: 1 };
const DEFAULT_CANVAS = { width: 1000, height: 800 };
const NOTE_SIZE = { width: 320, height: 160 };

describe("findFreePositionInViewport", () => {
  it("places node at viewport center on an empty canvas", () => {
    const result = findFreePositionInViewport({
      viewport: DEFAULT_VIEWPORT,
      canvasRect: DEFAULT_CANVAS,
      nodes: [],
      nodeSize: NOTE_SIZE,
    });

    expect(result).toEqual({
      x: DEFAULT_CANVAS.width / 2 - NOTE_SIZE.width / 2,
      y: DEFAULT_CANVAS.height / 2 - NOTE_SIZE.height / 2,
    });
  });

  it("fans out to find a free spot when the center is occupied", () => {
    const center = {
      x: DEFAULT_CANVAS.width / 2 - NOTE_SIZE.width / 2,
      y: DEFAULT_CANVAS.height / 2 - NOTE_SIZE.height / 2,
    };

    const result = findFreePositionInViewport({
      viewport: DEFAULT_VIEWPORT,
      canvasRect: DEFAULT_CANVAS,
      nodes: [{ position: center, width: NOTE_SIZE.width, height: NOTE_SIZE.height }],
      nodeSize: NOTE_SIZE,
    });

    expect(result).not.toEqual(center);
  });

  it("accounts for viewport pan when computing center", () => {
    const pannedViewport = { x: -500, y: -200, zoom: 1 };

    const result = findFreePositionInViewport({
      viewport: pannedViewport,
      canvasRect: DEFAULT_CANVAS,
      nodes: [],
      nodeSize: NOTE_SIZE,
    });

    expect(result).toEqual({
      x: (DEFAULT_CANVAS.width / 2 - pannedViewport.x) / 1 - NOTE_SIZE.width / 2,
      y: (DEFAULT_CANVAS.height / 2 - pannedViewport.y) / 1 - NOTE_SIZE.height / 2,
    });
  });

  it("accounts for zoom when computing center", () => {
    const zoomedViewport = { x: 0, y: 0, zoom: 0.5 };

    const result = findFreePositionInViewport({
      viewport: zoomedViewport,
      canvasRect: DEFAULT_CANVAS,
      nodes: [],
      nodeSize: NOTE_SIZE,
    });

    expect(result).toEqual({
      x: DEFAULT_CANVAS.width / (2 * 0.5) - NOTE_SIZE.width / 2,
      y: DEFAULT_CANVAS.height / (2 * 0.5) - NOTE_SIZE.height / 2,
    });
  });

  it("falls back to fallbackCanvasSize when canvasRect is missing", () => {
    const result = findFreePositionInViewport({
      viewport: DEFAULT_VIEWPORT,
      canvasRect: null,
      nodes: [],
      nodeSize: NOTE_SIZE,
      fallbackCanvasSize: { width: 400, height: 300 },
    });

    expect(result).toEqual({
      x: 400 / 2 - NOTE_SIZE.width / 2,
      y: 300 / 2 - NOTE_SIZE.height / 2,
    });
  });

  it("uses default node size for collision when a node has no dimensions", () => {
    const center = {
      x: DEFAULT_CANVAS.width / 2 - NOTE_SIZE.width / 2,
      y: DEFAULT_CANVAS.height / 2 - NOTE_SIZE.height / 2,
    };

    const result = findFreePositionInViewport({
      viewport: DEFAULT_VIEWPORT,
      canvasRect: DEFAULT_CANVAS,
      nodes: [{ position: center }],
      nodeSize: NOTE_SIZE,
      defaultNodeSize: { width: 400, height: 300 },
    });

    expect(result).not.toEqual(center);
  });
});
