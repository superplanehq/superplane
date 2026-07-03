import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";

import type { CanvasesCanvasVersion } from "@/api-client";
import { consumeLocalStagingWrite } from "@/lib/canvasStagingEcho";
import { canvasKeys } from "@/hooks/useCanvasData";

import { processCanvasLifecycleEvent } from "./lib/canvas-version-lifecycle";

type UseCanvasLifecycleEventHandlersOptions = {
  canvasId?: string;
  activeCanvasVersionId: string;
  isEditing: boolean;
  editSessionActive: boolean;
  isCreatingDraftBranch?: boolean;
  hasLocalSaveActivity: boolean;
  isViewingLiveVersion: boolean;
  canvasDeletedRemotely: boolean;
  consumeIgnoredCanvasUpdatedEcho: () => boolean;
  consumeIgnoredCreateDraftEcho: (targetCanvasId?: string, eventVersionId?: string) => boolean;
  consumeIgnoredCanvasVersionUpdatedEcho: (versionId?: string) => boolean;
  resyncDraftToCommitted: (versionId: string) => Promise<void>;
  onRemoteStagingUpdated?: (versionId?: string) => void;
  setCanvasDeletedRemotely: (value: boolean) => void;
  setRemoteCanvasUpdatePending: (value: boolean) => void;
};

export function useCanvasLifecycleEventHandlers({
  canvasId,
  activeCanvasVersionId,
  isEditing,
  editSessionActive,
  isCreatingDraftBranch = false,
  hasLocalSaveActivity,
  isViewingLiveVersion,
  canvasDeletedRemotely,
  consumeIgnoredCanvasUpdatedEcho,
  consumeIgnoredCreateDraftEcho,
  consumeIgnoredCanvasVersionUpdatedEcho,
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

      queryClient.setQueryData<CanvasesCanvasVersion[]>(canvasKeys.versionList(canvasId), (current = []) =>
        current.filter((version) => version.metadata?.id !== targetVersionId),
      );
      queryClient.removeQueries({ queryKey: canvasKeys.versionDetail(canvasId, targetVersionId) });
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
        isCreatingDraftBranch,
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
      isCreatingDraftBranch,
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
      if (payload.versionId && consumeLocalStagingWrite(canvasId, payload.versionId)) {
        return false;
      }

      if (payload.versionId && payload.versionId === activeCanvasVersionId && hasLocalSaveActivity) {
        setRemoteCanvasUpdatePending(true);
        return true;
      }

      onRemoteStagingUpdated?.(payload.versionId);
      return true;
    },
    [activeCanvasVersionId, canvasId, hasLocalSaveActivity, onRemoteStagingUpdated, setRemoteCanvasUpdatePending],
  );

  return {
    handleCanvasLifecycleEvent,
    shouldApplyCanvasUpdate,
    handleCanvasStagingEvent,
  };
}
