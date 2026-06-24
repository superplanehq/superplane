export type ShouldReactToCanvasVersionUpdatedInput = {
  versionId?: string;
  activeCanvasVersionId: string;
  isEditing: boolean;
  editSessionActive: boolean;
  isCreatingDraftBranch: boolean;
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
  isCreatingDraftBranch,
}: ShouldReactToCanvasVersionUpdatedInput): boolean {
  // Same-tab draft creation already refreshes version caches from the mutation
  // response; ignore websocket echoes while that request is in flight (notably
  // when creating another draft from the versions sidebar during an edit session).
  if (isCreatingDraftBranch) {
    return false;
  }

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
  isCreatingDraftBranch: boolean;
  hasLocalSaveActivity: boolean;
  consumeIgnoredCanvasUpdatedEcho: () => boolean;
  consumeIgnoredCreateDraftEcho: (targetCanvasId?: string, eventVersionId?: string) => boolean;
  consumeIgnoredCanvasVersionUpdatedEcho: (versionId?: string) => boolean;
  invalidateCanvasVersionData: (targetCanvasId: string, targetVersionId?: string) => void;
  pruneDeletedCanvasVersion: (targetVersionId: string) => void;
  resyncDraftToCommitted: (versionId: string) => void;
  setCanvasDeletedRemotely: (value: boolean) => void;
  setRemoteCanvasUpdatePending: (value: boolean) => void;
};

function shouldConsumeCanvasVersionUpdatedEcho({
  payload,
  consumeIgnoredCreateDraftEcho,
  consumeIgnoredCanvasVersionUpdatedEcho,
}: Pick<
  ProcessCanvasLifecycleEventInput,
  "payload" | "consumeIgnoredCreateDraftEcho" | "consumeIgnoredCanvasVersionUpdatedEcho"
>): boolean {
  const consumedCreateDraftEcho = consumeIgnoredCreateDraftEcho(payload.canvasId, payload.versionId);
  const consumedVersionEcho = consumeIgnoredCanvasVersionUpdatedEcho(payload.versionId);
  return consumedCreateDraftEcho || consumedVersionEcho;
}

function handleCanvasVersionDeletedLifecycle({
  payload,
  activeCanvasVersionId,
  isEditing,
  editSessionActive,
  isCreatingDraftBranch,
  pruneDeletedCanvasVersion,
}: Pick<
  ProcessCanvasLifecycleEventInput,
  | "payload"
  | "activeCanvasVersionId"
  | "isEditing"
  | "editSessionActive"
  | "isCreatingDraftBranch"
  | "pruneDeletedCanvasVersion"
>): boolean {
  window.dispatchEvent(new CustomEvent("canvas:version-deleted", { detail: { versionId: payload.versionId } }));

  if (payload.versionId) {
    // Every tab drops the deleted draft from cache so background draft-branch
    // queries stop fetching a version that no longer exists, even on passive live view.
    pruneDeletedCanvasVersion(payload.versionId);
  }

  if (!payload.versionId) {
    return editSessionActive;
  }

  return shouldReactToCanvasVersionUpdated({
    versionId: payload.versionId,
    activeCanvasVersionId,
    isEditing,
    editSessionActive,
    isCreatingDraftBranch,
  });
}

function handleCanvasVersionUpdatedLifecycle({
  payload,
  canvasId,
  activeCanvasVersionId,
  isEditing,
  editSessionActive,
  isCreatingDraftBranch,
  hasLocalSaveActivity,
  invalidateCanvasVersionData,
  resyncDraftToCommitted,
  setRemoteCanvasUpdatePending,
}: Pick<
  ProcessCanvasLifecycleEventInput,
  | "payload"
  | "canvasId"
  | "activeCanvasVersionId"
  | "isEditing"
  | "editSessionActive"
  | "isCreatingDraftBranch"
  | "hasLocalSaveActivity"
  | "invalidateCanvasVersionData"
  | "resyncDraftToCommitted"
  | "setRemoteCanvasUpdatePending"
>): boolean {
  window.dispatchEvent(new CustomEvent("canvas:version-updated", { detail: { versionId: payload.versionId } }));

  if (
    !shouldReactToCanvasVersionUpdated({
      versionId: payload.versionId,
      activeCanvasVersionId,
      isEditing,
      editSessionActive,
      isCreatingDraftBranch,
    })
  ) {
    return false;
  }

  if (!payload.versionId) {
    invalidateCanvasVersionData(canvasId!);
    return true;
  }

  if (payload.versionId === activeCanvasVersionId && hasLocalSaveActivity) {
    setRemoteCanvasUpdatePending(true);
    return true;
  }

  invalidateCanvasVersionData(canvasId!, payload.versionId);
  resyncDraftToCommitted(payload.versionId);
  return true;
}

function handleCanvasUpdatedLifecycle({
  canvasId,
  activeCanvasVersionId,
  hasLocalSaveActivity,
  invalidateCanvasVersionData,
  setRemoteCanvasUpdatePending,
}: Pick<
  ProcessCanvasLifecycleEventInput,
  | "canvasId"
  | "activeCanvasVersionId"
  | "hasLocalSaveActivity"
  | "invalidateCanvasVersionData"
  | "setRemoteCanvasUpdatePending"
>): boolean {
  if (hasLocalSaveActivity) {
    setRemoteCanvasUpdatePending(true);
    return true;
  }

  invalidateCanvasVersionData(canvasId!, activeCanvasVersionId);
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

  if (eventName === "canvas_version_updated" && shouldConsumeCanvasVersionUpdatedEcho(input)) {
    return false;
  }

  if (!canvasId) {
    return true;
  }

  if (eventName === "canvas_version_deleted") {
    return handleCanvasVersionDeletedLifecycle(input);
  }

  if (eventName === "canvas_version_updated") {
    return handleCanvasVersionUpdatedLifecycle(input);
  }

  if (eventName !== "canvas_updated") {
    return true;
  }

  return handleCanvasUpdatedLifecycle(input);
}
