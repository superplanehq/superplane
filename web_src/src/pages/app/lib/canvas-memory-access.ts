/**
 * Resolves whether the canvas memory delete affordance should be exposed for
 * the current viewer. Memory belongs to the live canvas, but the delete
 * action mutates live data, so we only expose it while the app is in edit
 * mode (the user has explicitly switched out of read mode by entering an
 * editable draft session) and the viewer is otherwise authorized to act on
 * the canvas.
 */
export function canEditCanvasMemory({
  canUpdateCanvas,
  canvasDeletedRemotely,
  hasEditableVersion,
}: {
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  hasEditableVersion: boolean;
}): boolean {
  return canUpdateCanvas && !canvasDeletedRemotely && hasEditableVersion;
}

/**
 * Decides whether the workflow page should fetch canvas memory entries: when
 * the memory overlay is open, or when the user is viewing the live canvas
 * (memory belongs to the live canvas).
 */
export function shouldLoadCanvasMemoryEntries(isMemoryMode: boolean, isViewingLiveVersion: boolean): boolean {
  return isMemoryMode || isViewingLiveVersion;
}
