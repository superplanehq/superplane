import { useCallback, type MutableRefObject } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { consumeLocalStagingWrite } from "@/lib/canvasStagingEcho";
import { canvasKeys } from "@/hooks/useCanvasData";

import { processCanvasLifecycleEvent } from "./lib/canvas-version-lifecycle";

type UseCanvasLifecycleEventHandlersOptions = {
  canvasId?: string;
  currentUserId?: string;
  editSessionActiveRef: MutableRefObject<boolean>;
  hasLocalSaveActivity: boolean;
  isViewingLiveVersion: boolean;
  canvasDeletedRemotely: boolean;
  consumeIgnoredCanvasUpdatedEcho: () => boolean;
  onRemoteStagingUpdated?: () => void;
  setCanvasDeletedRemotely: (value: boolean) => void;
  setRemoteCanvasUpdatePending: (value: boolean) => void;
};

export function useCanvasLifecycleEventHandlers({
  canvasId,
  currentUserId,
  editSessionActiveRef,
  hasLocalSaveActivity,
  isViewingLiveVersion,
  canvasDeletedRemotely,
  consumeIgnoredCanvasUpdatedEcho,
  onRemoteStagingUpdated,
  setCanvasDeletedRemotely,
  setRemoteCanvasUpdatePending,
}: UseCanvasLifecycleEventHandlersOptions) {
  const queryClient = useQueryClient();

  const invalidateCanvasStaging = useCallback(
    (targetCanvasId: string) => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.canvasStaging(targetCanvasId) });
    },
    [queryClient],
  );

  const invalidateLiveVersionData = useCallback(
    (targetCanvasId: string) => {
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(targetCanvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(targetCanvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.canvasStaging(targetCanvasId) });
      queryClient.invalidateQueries({ queryKey: canvasKeys.console(targetCanvasId, undefined) });
    },
    [queryClient],
  );

  const handleCanvasLifecycleEvent = useCallback(
    (payload: { canvasId: string }, eventName: string) =>
      processCanvasLifecycleEvent({
        payload,
        eventName,
        canvasId,
        editSessionActive: editSessionActiveRef.current,
        hasLocalSaveActivity,
        consumeIgnoredCanvasUpdatedEcho,
        invalidateCanvasStaging,
        invalidateLiveVersionData,
        setCanvasDeletedRemotely,
        setRemoteCanvasUpdatePending,
      }),
    [
      editSessionActiveRef,
      canvasId,
      consumeIgnoredCanvasUpdatedEcho,
      hasLocalSaveActivity,
      invalidateCanvasStaging,
      invalidateLiveVersionData,
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

      if (editSessionActiveRef.current && hasLocalSaveActivity) {
        setRemoteCanvasUpdatePending(true);
        return false;
      }

      onRemoteStagingUpdated?.();
      return false;
    },
    [
      editSessionActiveRef,
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
