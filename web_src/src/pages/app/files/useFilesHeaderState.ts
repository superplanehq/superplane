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
