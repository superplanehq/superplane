/** Wait for the popup pane to get a real size, then fit the graph. */
export const COMPACT_CANVAS_FIT_SETTLE_MS = 200;

/** Stable key so the compact canvas remounts and fits when live nodes arrive. */
export function compactCanvasFitKey(nodeIds: string[]): string {
  if (nodeIds.length === 0) {
    return "empty";
  }
  return [...nodeIds].sort().join("|");
}

export function shouldFitCompactCanvas(contentKey: string): boolean {
  return contentKey !== "empty";
}
