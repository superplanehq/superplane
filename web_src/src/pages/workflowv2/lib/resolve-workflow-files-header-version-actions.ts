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
  resetDraftLabel?: string;
  hasUnpublishedDraftChanges: boolean;
  publishVersionLabel?: string;
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
  resetDraftLabel,
  hasUnpublishedDraftChanges,
  publishVersionLabel,
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
    // While editing a draft the reset/discard action is always available: it
    // resets staged changes ("Reset") or deletes the draft ("Discard").
    discardVersionVisible: true,
    discardVersionDisabled: resetDraftDisabled,
    discardVersionDisabledTooltip: resetDraftDisabledTooltip,
    discardVersionLabel: resetDraftLabel ?? "Discard",
    publishVersionLabel: publishVersionLabel ?? (isChangeManagementDisabled ? "Publish" : "Propose Change"),
  };
}
