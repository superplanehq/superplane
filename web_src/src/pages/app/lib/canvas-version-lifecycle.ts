export type ProcessCanvasLifecycleEventInput = {
  payload: { canvasId: string };
  eventName: string;
  canvasId?: string;
  activeCanvasVersionId: string;
  editSessionActive: boolean;
  hasLocalSaveActivity: boolean;
  consumeIgnoredCanvasUpdatedEcho: () => boolean;
  invalidateCanvasVersionData: (targetCanvasId: string, targetVersionId?: string) => void;
  resyncDraftToCommitted: (versionId: string) => void;
  setCanvasDeletedRemotely: (value: boolean) => void;
  setRemoteCanvasUpdatePending: (value: boolean) => void;
};

function handleCanvasUpdatedLifecycle({
  canvasId,
  activeCanvasVersionId,
  editSessionActive,
  hasLocalSaveActivity,
  invalidateCanvasVersionData,
  resyncDraftToCommitted,
  setRemoteCanvasUpdatePending,
}: Pick<
  ProcessCanvasLifecycleEventInput,
  | "canvasId"
  | "activeCanvasVersionId"
  | "editSessionActive"
  | "hasLocalSaveActivity"
  | "invalidateCanvasVersionData"
  | "resyncDraftToCommitted"
  | "setRemoteCanvasUpdatePending"
>): boolean {
  if (hasLocalSaveActivity) {
    setRemoteCanvasUpdatePending(true);
    if (canvasId && activeCanvasVersionId) {
      invalidateCanvasVersionData(canvasId, activeCanvasVersionId);
    }
    return true;
  }

  if (canvasId) {
    invalidateCanvasVersionData(canvasId, activeCanvasVersionId || undefined);
    if (editSessionActive && activeCanvasVersionId) {
      resyncDraftToCommitted(activeCanvasVersionId);
    }
  }

  return true;
}

export function processCanvasLifecycleEvent(input: ProcessCanvasLifecycleEventInput): boolean {
  const { eventName, canvasId } = input;

  if (eventName === "canvas_deleted") {
    input.setCanvasDeletedRemotely(true);
    return true;
  }

  if (eventName === "canvas_updated" && input.consumeIgnoredCanvasUpdatedEcho()) {
    return false;
  }

  if (!canvasId || eventName !== "canvas_updated") {
    return true;
  }

  return handleCanvasUpdatedLifecycle(input);
}
