export type ShouldReactToCanvasVersionUpdatedInput = {
  versionId?: string;
  activeCanvasVersionId: string;
  isEditing: boolean;
  editSessionActive: boolean;
};

// Other tabs on the same canvas only need version-list/detail refreshes when
// they are editing, previewing that version, or have the versions UI open.
// Live-view tabs can ignore remote draft churn; publish still flows through
// canvas_updated, and the agent sidebar listens to canvas:version-updated.
export function shouldReactToCanvasVersionUpdated({
  versionId,
  activeCanvasVersionId,
  isEditing,
  editSessionActive,
}: ShouldReactToCanvasVersionUpdatedInput): boolean {
  if (!versionId) {
    return editSessionActive;
  }

  if (isEditing && activeCanvasVersionId === versionId) {
    return true;
  }

  if (activeCanvasVersionId === versionId) {
    return true;
  }

  return editSessionActive;
}

export type ProcessCanvasLifecycleEventInput = {
  payload: { canvasId: string; versionId?: string };
  eventName: string;
  canvasId?: string;
  activeCanvasVersionId: string;
  isEditing: boolean;
  editSessionActive: boolean;
  hasLocalSaveActivity: boolean;
  consumeIgnoredCanvasUpdatedEcho: () => boolean;
  consumeIgnoredCreateDraftEcho: (targetCanvasId?: string) => boolean;
  consumeIgnoredCanvasVersionUpdatedEcho: (versionId?: string) => boolean;
  invalidateCanvasVersionData: (targetCanvasId: string, targetVersionId?: string) => void;
  resyncDraftToCommitted: (versionId: string) => void;
  setCanvasDeletedRemotely: (value: boolean) => void;
  setRemoteCanvasUpdatePending: (value: boolean) => void;
};

export function processCanvasLifecycleEvent({
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
  resyncDraftToCommitted,
  setCanvasDeletedRemotely,
  setRemoteCanvasUpdatePending,
}: ProcessCanvasLifecycleEventInput): boolean {
  if (eventName === "canvas_deleted") {
    setCanvasDeletedRemotely(true);
    return true;
  }

  if (eventName === "canvas_updated" && consumeIgnoredCanvasUpdatedEcho()) {
    return false;
  }

  if (eventName === "canvas_version_updated") {
    const consumedCreateDraftEcho = consumeIgnoredCreateDraftEcho(payload.canvasId);
    const consumedVersionEcho = consumeIgnoredCanvasVersionUpdatedEcho(payload.versionId);
    if (consumedCreateDraftEcho || consumedVersionEcho) {
      return false;
    }
  }

  if (!canvasId) {
    return true;
  }

  if (eventName === "canvas_version_updated") {
    window.dispatchEvent(new CustomEvent("canvas:version-updated", { detail: { versionId: payload.versionId } }));

    if (
      !shouldReactToCanvasVersionUpdated({
        versionId: payload.versionId,
        activeCanvasVersionId,
        isEditing,
        editSessionActive,
      })
    ) {
      return false;
    }

    if (!payload.versionId) {
      invalidateCanvasVersionData(canvasId);
      return true;
    }

    if (payload.versionId === activeCanvasVersionId && hasLocalSaveActivity) {
      setRemoteCanvasUpdatePending(true);
      return true;
    }

    invalidateCanvasVersionData(canvasId, payload.versionId);
    resyncDraftToCommitted(payload.versionId);
    return true;
  }

  if (eventName !== "canvas_updated") {
    return true;
  }

  if (hasLocalSaveActivity) {
    setRemoteCanvasUpdatePending(true);
    return true;
  }

  invalidateCanvasVersionData(canvasId, activeCanvasVersionId);
  return true;
}
