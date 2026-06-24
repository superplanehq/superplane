import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { consumeLocalStagingWrite } from "@/lib/canvasStagingEcho";
import { canvasKeys, pruneDeletedDraftBranchFromCache } from "@/hooks/useCanvasData";

import { processCanvasLifecycleEvent } from "./lib/canvas-version-lifecycle";

type UseCanvasLifecycleEventHandlersOptions = {
  canvasId?: string;
  activeCanvasVersionId: string;
  isEditing: boolean;
  editSessionActive: boolean;
  hasLocalSaveActivity: boolean;
  isViewingLiveVersion: boolean;
  canvasDeletedRemotely: boolean;
  consumeIgnoredCanvasUpdatedEcho: () => boolean;
  consumeIgnoredCreateDraftEcho: (targetCanvasId?: string, eventVersionId?: string) => boolean;
  consumeIgnoredCanvasVersionUpdatedEcho: (versionId?: string) => boolean;
  resyncDraftToCommitted: (versionId: string) => Promise<void>;
  resyncDraftToStaged: (versionId: string) => Promise<void>;
  setCanvasDeletedRemotely: (value: boolean) => void;
  setRemoteCanvasUpdatePending: (value: boolean) => void;
};

export function useCanvasLifecycleEventHandlers({
  canvasId,
  activeCanvasVersionId,
  isEditing,
  editSessionActive,
  hasLocalSaveActivity,
  isViewingLiveVersion,
  canvasDeletedRemotely,
  consumeIgnoredCanvasUpdatedEcho,
  consumeIgnoredCreateDraftEcho,
  consumeIgnoredCanvasVersionUpdatedEcho,
  resyncDraftToCommitted,
  resyncDraftToStaged,
  setCanvasDeletedRemotely,
  setRemoteCanvasUpdatePending,
}: UseCanvasLifecycleEventHandlersOptions) {
  const queryClient = useQueryClient();

  const invalidateCanvasVersionData = useCallback(
    (targetCanvasId: string, targetVersionId?: string) => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(targetCanvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.draftBranches(targetCanvasId) });
      if (targetVersionId) {
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionDetail(targetCanvasId, targetVersionId) });
      }
    },
    [queryClient],
  );

  const pruneDeletedCanvasVersion = useCallback(
    (targetVersionId: string) => {
      if (!canvasId) {
        return;
      }

      void pruneDeletedDraftBranchFromCache(queryClient, canvasId, targetVersionId);
    },
    [canvasId, queryClient],
  );

  const handleCanvasLifecycleEvent = useCallback(
    (payload: { canvasId: string; versionId?: string }, eventName: string) =>
      processCanvasLifecycleEvent({
        payload,
        eventName,
        canvasId,
        activeCanvasVersionId,
        isEditing,
        editSessionActive,
        hasLocalSaveActivity,
        consumeIgnoredCanvasUpdatedEcho,
        consumeIgnoredCreateDraftEcho,
        consumeIgnoredCanvasVersionUpdatedEcho,
        invalidateCanvasVersionData,
        pruneDeletedCanvasVersion,
        resyncDraftToCommitted: (versionId) => {
          void resyncDraftToCommitted(versionId);
        },
        setCanvasDeletedRemotely,
        setRemoteCanvasUpdatePending,
      }),
    [
      activeCanvasVersionId,
      canvasId,
      consumeIgnoredCanvasUpdatedEcho,
      consumeIgnoredCreateDraftEcho,
      consumeIgnoredCanvasVersionUpdatedEcho,
      editSessionActive,
      hasLocalSaveActivity,
      invalidateCanvasVersionData,
      isEditing,
      pruneDeletedCanvasVersion,
      resyncDraftToCommitted,
      setCanvasDeletedRemotely,
      setRemoteCanvasUpdatePending,
    ],
  );

  const shouldApplyCanvasUpdate = useCallback(
    () => isViewingLiveVersion && !hasLocalSaveActivity && !canvasDeletedRemotely,
    [isViewingLiveVersion, hasLocalSaveActivity, canvasDeletedRemotely],
  );

  const handleCanvasStagingEvent = useCallback(
    (payload: { canvasId: string; versionId?: string }) => {
      if (!payload.versionId) {
        return false;
      }

      if (consumeLocalStagingWrite(canvasId, payload.versionId)) {
        return false;
      }

      if (payload.versionId === activeCanvasVersionId && hasLocalSaveActivity) {
        setRemoteCanvasUpdatePending(true);
        return true;
      }

      void resyncDraftToStaged(payload.versionId);
      return true;
    },
    [activeCanvasVersionId, canvasId, hasLocalSaveActivity, resyncDraftToStaged, setRemoteCanvasUpdatePending],
  );

  return {
    handleCanvasLifecycleEvent,
    shouldApplyCanvasUpdate,
    handleCanvasStagingEvent,
  };
}
