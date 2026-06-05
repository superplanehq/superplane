import type { WorkflowFilesHeaderActionsState } from "../workflow-files-types";

type ResolveWorkflowFilesHeaderVersionActionsArgs = {
  useFilesHeaderActions: boolean;
  filesHeaderActions: WorkflowFilesHeaderActionsState | null;
  isChangeManagementDisabled: boolean;
  handlePublishVersion: () => void;
  handleCreateChangeRequest: () => void;
  handleResetDraftChanges: () => void;
  publishVersionDisabled: boolean;
  publishVersionDisabledTooltip?: string;
  resetDraftDisabled: boolean;
  resetDraftDisabledTooltip?: string;
  hasUnpublishedDraftChanges: boolean;
};

export function resolveWorkflowFilesHeaderVersionActions({
  useFilesHeaderActions,
  filesHeaderActions,
  isChangeManagementDisabled,
  handlePublishVersion,
  handleCreateChangeRequest,
  handleResetDraftChanges,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  resetDraftDisabled,
  resetDraftDisabledTooltip,
  hasUnpublishedDraftChanges,
}: ResolveWorkflowFilesHeaderVersionActionsArgs) {
  if (useFilesHeaderActions) {
    return {
      onPublishVersion: filesHeaderActions?.onPublish,
      onDiscardVersion: filesHeaderActions?.onDiscardAll,
      publishVersionDisabled: !filesHeaderActions || filesHeaderActions.publishDisabled,
      publishVersionDisabledTooltip: filesHeaderActions?.publishDisabledTooltip,
      hasUnpublishedDraftChanges: !!filesHeaderActions?.hasPendingChanges,
      discardVersionDisabled: !filesHeaderActions || filesHeaderActions.discardDisabled,
      discardVersionDisabledTooltip: undefined,
      publishVersionLabel: "Publish",
    };
  }

  return {
    onPublishVersion: isChangeManagementDisabled ? handlePublishVersion : handleCreateChangeRequest,
    onDiscardVersion: handleResetDraftChanges,
    publishVersionDisabled,
    publishVersionDisabledTooltip,
    hasUnpublishedDraftChanges,
    discardVersionDisabled: resetDraftDisabled,
    discardVersionDisabledTooltip: resetDraftDisabledTooltip,
    publishVersionLabel: isChangeManagementDisabled ? "Publish" : "Propose Change",
  };
}
