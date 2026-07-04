import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { consumeLocalStagingWrite } from "@/lib/canvasStagingEcho";
import { canvasKeys } from "@/hooks/useCanvasData";

import { processCanvasLifecycleEvent } from "./lib/canvas-version-lifecycle";

type UseCanvasLifecycleEventHandlersOptions = {
  canvasId?: string;
  currentUserId?: string;
  activeCanvasVersionId: string;
  editSessionActive: boolean;
  hasLocalSaveActivity: boolean;
  isViewingLiveVersion: boolean;
  canvasDeletedRemotely: boolean;
  consumeIgnoredCanvasUpdatedEcho: () => boolean;
  resyncDraftToCommitted: (versionId: string) => Promise<void>;
  onRemoteStagingUpdated?: () => void;
  setCanvasDeletedRemotely: (value: boolean) => void;
  setRemoteCanvasUpdatePending: (value: boolean) => void;
};

export function useCanvasLifecycleEventHandlers({
  canvasId,
  currentUserId,
  activeCanvasVersionId,
  editSessionActive,
  hasLocalSaveActivity,
  isViewingLiveVersion,
  canvasDeletedRemotely,
  consumeIgnoredCanvasUpdatedEcho,
  resyncDraftToCommitted,
  onRemoteStagingUpdated,
  setCanvasDeletedRemotely,
  setRemoteCanvasUpdatePending,
}: UseCanvasLifecycleEventHandlersOptions) {
  const queryClient = useQueryClient();

  const invalidateCanvasVersionData = useCallback(
    (targetCanvasId: string, targetVersionId?: string) => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(targetCanvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(targetCanvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.canvasStaging(targetCanvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.consoleAll(targetCanvasId) });
      if (targetVersionId) {
        queryClient.invalidateQueries({ queryKey: canvasKeys.versionDetail(targetCanvasId, targetVersionId) });
      }
    },
    [queryClient],
  );

  const handleCanvasLifecycleEvent = useCallback(
    (payload: { canvasId: string }, eventName: string) =>
      processCanvasLifecycleEvent({
        payload,
        eventName,
        canvasId,
        activeCanvasVersionId,
        editSessionActive,
        hasLocalSaveActivity,
        consumeIgnoredCanvasUpdatedEcho,
        invalidateCanvasVersionData,
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
      editSessionActive,
      hasLocalSaveActivity,
      invalidateCanvasVersionData,
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
    (payload: { canvasId: string; userId?: string }) => {
      if (payload.userId && currentUserId && payload.userId !== currentUserId) {
        return false;
      }

      if (consumeLocalStagingWrite(canvasId, payload.userId)) {
        return false;
      }

      if (activeCanvasVersionId && hasLocalSaveActivity) {
        setRemoteCanvasUpdatePending(true);
        return true;
      }

      onRemoteStagingUpdated?.();
      return true;
    },
    [
      activeCanvasVersionId,
      canvasId,
      currentUserId,
      hasLocalSaveActivity,
      onRemoteStagingUpdated,
      setRemoteCanvasUpdatePending,
    ],
  );

  return {
    handleCanvasLifecycleEvent,
    shouldApplyCanvasUpdate,
    handleCanvasStagingEvent,
  };
}
