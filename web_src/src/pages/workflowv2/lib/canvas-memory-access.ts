/**
 * Resolves whether the canvas memory delete affordance should be exposed for
 * the current viewer + version. Memory belongs to the live canvas, so this
 * should not depend on draft edit mode.
 */
export function canEditCanvasMemory({
  canUpdateCanvas,
  isTemplate,
  canvasDeletedRemotely,
  isViewingLiveVersion,
  isViewingDraftVersion,
}: {
  canUpdateCanvas: boolean;
  isTemplate: boolean;
  canvasDeletedRemotely: boolean;
  isViewingLiveVersion: boolean;
  isViewingDraftVersion: boolean;
}): boolean {
  return canUpdateCanvas && !isTemplate && !canvasDeletedRemotely && isViewingLiveVersion && !isViewingDraftVersion;
}
