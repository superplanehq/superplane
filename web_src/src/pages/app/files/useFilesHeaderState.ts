import { useCallback, useState } from "react";

import type { FilesHeaderActionsState } from "./types";

export function useFilesHeaderState(canvasId?: string) {
  const [filesHeaderActions, setFilesHeaderActions] = useState<FilesHeaderActionsState | null>(null);
  const onFilesHeaderActionsChange = useCallback((actions: FilesHeaderActionsState | null) => {
    setFilesHeaderActions((current) => {
      if (!current && !actions) {
        return current;
      }

      if (!current || !actions) {
        return actions;
      }

      if (
        current.hasPendingChanges === actions.hasPendingChanges &&
        current.publishDisabled === actions.publishDisabled &&
        current.publishDisabledTooltip === actions.publishDisabledTooltip &&
        current.discardDisabled === actions.discardDisabled &&
        current.publishPending === actions.publishPending &&
        current.onPublish === actions.onPublish &&
        current.onDiscardAll === actions.onDiscardAll
      ) {
        return current;
      }

      return actions;
    });
  }, []);

  const filesHeaderActionsSlotId = canvasId ? `canvas-files-header-actions-${canvasId}` : "canvas-files-header-actions";

  return {
    filesHeaderActions,
    onFilesHeaderActionsChange,
    filesHeaderActionsSlotId,
  };
}

type ResolveFilesHeaderVersionActionsArgs = {
  useFilesHeaderActions: boolean;
  filesHeaderActions: FilesHeaderActionsState | null;
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

export function resolveFilesHeaderVersionActions({
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
}: ResolveFilesHeaderVersionActionsArgs) {
  if (useFilesHeaderActions) {
    return {
      onPublishVersion: filesHeaderActions?.onPublish,
      onDiscardVersion: filesHeaderActions?.onDiscardAll,
      publishVersionDisabled: !filesHeaderActions || filesHeaderActions.publishDisabled,
      publishVersionDisabledTooltip: filesHeaderActions?.publishDisabledTooltip,
      hasUnpublishedDraftChanges: !!filesHeaderActions?.hasPendingChanges,
      discardVersionDisabled: !filesHeaderActions || filesHeaderActions.discardDisabled,
      discardVersionDisabledTooltip: undefined,
      publishVersionLabel: "Save",
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
