export function useFilesHeaderState(canvasId?: string) {
  const filesHeaderActionsSlotId = canvasId ? `canvas-files-header-actions-${canvasId}` : "canvas-files-header-actions";

  return {
    filesHeaderActionsSlotId,
  };
}

type ResolveFilesHeaderVersionActionsArgs = {
  handlePublishVersion: () => void;
  handleResetDraftChanges: () => void;
  publishVersionDisabled: boolean;
  publishVersionDisabledTooltip?: string;
  resetDraftDisabled: boolean;
  resetDraftDisabledTooltip?: string;
  hasUnpublishedDraftChanges: boolean;
};

export function resolveFilesHeaderVersionActions({
  handlePublishVersion,
  handleResetDraftChanges,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  resetDraftDisabled,
  resetDraftDisabledTooltip,
  hasUnpublishedDraftChanges,
}: ResolveFilesHeaderVersionActionsArgs) {
  return {
    onPublishVersion: handlePublishVersion,
    onDiscardVersion: handleResetDraftChanges,
    publishVersionDisabled,
    publishVersionDisabledTooltip,
    hasUnpublishedDraftChanges,
    discardVersionDisabled: resetDraftDisabled,
    discardVersionDisabledTooltip: resetDraftDisabledTooltip,
    publishVersionLabel: "Publish",
  };
}
