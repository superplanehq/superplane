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
  publishVersionLabel?: string;
  allowDiscard?: boolean;
  allowPublish?: boolean;
};

export function resolveFilesHeaderVersionActions({
  handlePublishVersion,
  handleResetDraftChanges,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  resetDraftDisabled,
  resetDraftDisabledTooltip,
  hasUnpublishedDraftChanges,
  publishVersionLabel = "Publish",
  allowDiscard = false,
  allowPublish = false,
}: ResolveFilesHeaderVersionActionsArgs) {
  return {
    onPublishVersion: allowPublish ? handlePublishVersion : undefined,
    onDiscardVersion: allowDiscard ? handleResetDraftChanges : undefined,
    publishVersionDisabled,
    publishVersionDisabledTooltip,
    hasUnpublishedDraftChanges,
    discardVersionDisabled: resetDraftDisabled,
    discardVersionDisabledTooltip: resetDraftDisabledTooltip,
    publishVersionLabel,
  };
}
