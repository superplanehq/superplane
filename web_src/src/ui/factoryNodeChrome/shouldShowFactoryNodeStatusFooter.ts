/** Status footer for factory cards: live/run views, or edit with "No run". */
export function shouldShowFactoryNodeStatusFooter({
  canvasMode,
  isCompactView,
  showRuntimeStatus = true,
}: {
  canvasMode?: "live" | "edit";
  isCompactView?: boolean;
  /** False on factory Live without a selected run (topology only). */
  showRuntimeStatus?: boolean;
}): boolean {
  if (isCompactView) return false;
  // Edit always shows the neutral "No run" footer (even when runtime maps are empty).
  if (canvasMode === "edit") return true;
  if (!showRuntimeStatus) return false;
  return true;
}
