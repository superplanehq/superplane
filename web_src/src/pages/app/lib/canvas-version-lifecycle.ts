export type ProcessCanvasLifecycleEventInput = {
  payload: { canvasId: string };
  eventName: string;
  canvasId?: string;
  editSessionActive: boolean;
  hasLocalSaveActivity: boolean;
  consumeIgnoredCanvasUpdatedEcho: () => boolean;
  invalidateCanvasStaging: (targetCanvasId: string) => void;
  invalidateLiveVersionData: (targetCanvasId: string) => void;
  setCanvasDeletedRemotely: (value: boolean) => void;
  setRemoteCanvasUpdatePending: (value: boolean) => void;
};

function handleCanvasUpdatedLifecycle({
  canvasId,
  editSessionActive,
  hasLocalSaveActivity,
  invalidateCanvasStaging,
  invalidateLiveVersionData,
  setRemoteCanvasUpdatePending,
}: Pick<
  ProcessCanvasLifecycleEventInput,
  | "canvasId"
  | "editSessionActive"
  | "hasLocalSaveActivity"
  | "invalidateCanvasStaging"
  | "invalidateLiveVersionData"
  | "setRemoteCanvasUpdatePending"
>): boolean {
  if (hasLocalSaveActivity) {
    setRemoteCanvasUpdatePending(true);
  }

  if (!canvasId) {
    return true;
  }

  if (editSessionActive) {
    // While editing, only refresh staging metadata. Version-scoped repository
    // and console queries may still reference a version id that is no longer live
    // (e.g. immediately after a commit in this tab).
    invalidateCanvasStaging(canvasId);
    return true;
  }

  invalidateLiveVersionData(canvasId);
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
