export interface ViewportLike {
  x: number;
  y: number;
  zoom: number;
}

export interface CanvasRectLike {
  width: number;
  height: number;
}

export interface PositionedNodeLike {
  position: { x: number; y: number };
  width?: number | null;
  height?: number | null;
}

export interface FindFreePositionInViewportInput {
  viewport: ViewportLike;
  canvasRect: CanvasRectLike | null | undefined;
  nodes: PositionedNodeLike[];
  nodeSize: { width: number; height: number };
  fallbackCanvasSize?: { width: number; height: number };
  defaultNodeSize?: { width: number; height: number };
  padding?: number;
  step?: number;
  maxRings?: number;
}

/**
 * Places a new node inside the currently visible part of the canvas (flow coords),
 * starting from the viewport center and fanning out in rings until a non-overlapping
 * spot is found. Used by the "Add Note" button and the keyboard-drop shortcut so
 * both entry points behave identically.
 *
 * Purely functional — all inputs are passed in so this is easy to test and to
 * call from both mouse and keyboard code paths.
 */
export function findFreePositionInViewport(input: FindFreePositionInViewportInput): { x: number; y: number } {
  const {
    viewport,
    canvasRect,
    nodes,
    nodeSize,
    fallbackCanvasSize = { width: 0, height: 0 },
    defaultNodeSize = { width: 240, height: 120 },
    padding = 16,
    step = 40,
    maxRings = 8,
  } = input;

  const visibleWidth = canvasRect?.width ?? fallbackCanvasSize.width;
  const visibleHeight = canvasRect?.height ?? fallbackCanvasSize.height;
  const zoom = viewport.zoom || 1;

  const visibleBounds = {
    minX: (0 - viewport.x) / zoom,
    minY: (0 - viewport.y) / zoom,
    maxX: (visibleWidth - viewport.x) / zoom,
    maxY: (visibleHeight - viewport.y) / zoom,
  };

  const basePosition = {
    x: (visibleWidth / 2 - viewport.x) / zoom - nodeSize.width / 2,
    y: (visibleHeight / 2 - viewport.y) / zoom - nodeSize.height / 2,
  };

  const intersects = (pos: { x: number; y: number }) => {
    const bounds = {
      minX: pos.x - padding,
      minY: pos.y - padding,
      maxX: pos.x + nodeSize.width + padding,
      maxY: pos.y + nodeSize.height + padding,
    };
    return nodes.some((node) => {
      const width = node.width ?? defaultNodeSize.width;
      const height = node.height ?? defaultNodeSize.height;
      const nodeBounds = {
        minX: node.position.x,
        minY: node.position.y,
        maxX: node.position.x + width,
        maxY: node.position.y + height,
      };
      return !(
        bounds.maxX < nodeBounds.minX ||
        bounds.minX > nodeBounds.maxX ||
        bounds.maxY < nodeBounds.minY ||
        bounds.minY > nodeBounds.maxY
      );
    });
  };

  const clampToVisible = (pos: { x: number; y: number }) => {
    const minX = visibleBounds.minX + padding;
    const minY = visibleBounds.minY + padding;
    const maxX = visibleBounds.maxX - nodeSize.width - padding;
    const maxY = visibleBounds.maxY - nodeSize.height - padding;
    return {
      x: Math.min(Math.max(pos.x, minX), maxX),
      y: Math.min(Math.max(pos.y, minY), maxY),
    };
  };

  const basePositionClamped = clampToVisible(basePosition);
  if (!intersects(basePositionClamped)) {
    return basePositionClamped;
  }

  for (let ring = 1; ring <= maxRings; ring += 1) {
    for (let dx = -ring; dx <= ring; dx += 1) {
      for (let dy = -ring; dy <= ring; dy += 1) {
        // Only walk the perimeter of the current ring — interior cells were
        // already tested by smaller rings.
        if (Math.abs(dx) !== ring && Math.abs(dy) !== ring) continue;
        const candidate = clampToVisible({
          x: basePosition.x + dx * step,
          y: basePosition.y + dy * step,
        });
        if (!intersects(candidate)) {
          return candidate;
        }
      }
    }
  }

  return basePositionClamped;
}
