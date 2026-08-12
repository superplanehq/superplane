/** Status footer is for live/run views only — not while editing the graph. */
export function shouldShowFactoryNodeStatusFooter({
  canvasMode,
  isCompactView,
}: {
  canvasMode?: "live" | "edit";
  isCompactView?: boolean;
}): boolean {
  if (isCompactView) return false;
  if (canvasMode === "edit") return false;
  return true;
}
