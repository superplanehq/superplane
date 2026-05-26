/**
 * Resolves whether the canvas memory delete affordance should be exposed for
 * the current viewer + version. Centralized so the workflow page and overlay
 * stay aligned on a single gating rule.
 */
export function canEditCanvasMemory({
  isReadOnly,
  canUpdateCanvas,
  isViewingLiveVersion,
  isViewingDraftVersion,
}: {
  isReadOnly: boolean;
  canUpdateCanvas: boolean;
  isViewingLiveVersion: boolean;
  isViewingDraftVersion: boolean;
}): boolean {
  return !isReadOnly && canUpdateCanvas && isViewingLiveVersion && !isViewingDraftVersion;
}
