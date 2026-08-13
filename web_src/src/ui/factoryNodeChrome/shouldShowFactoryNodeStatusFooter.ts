/** Status footer is for run-scoped / live views only — not edit, compact, or factory topology Live. */
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
  if (!showRuntimeStatus) return false;
  if (isCompactView) return false;
  if (canvasMode === "edit") return false;
  return true;
}
