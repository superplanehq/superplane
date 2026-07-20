export function isEditBootstrapReady({
  isEditing,
  isEnteringEditSession,
  stagingBaselinesReady,
  draftCanvasSpec,
  shouldReadStagedCanvasVersion,
}: {
  isEditing: boolean;
  isEnteringEditSession: boolean;
  stagingBaselinesReady: boolean;
  draftCanvasSpec: unknown;
  shouldReadStagedCanvasVersion: boolean;
}): boolean {
  if (!isEditing) {
    return true;
  }

  if (isEnteringEditSession || !stagingBaselinesReady) {
    return false;
  }

  if (shouldReadStagedCanvasVersion && !draftCanvasSpec) {
    return false;
  }

  return true;
}

export function isEditStagingActionsReady(params: {
  isEditing: boolean;
  isEnteringEditSession: boolean;
  stagingBaselinesReady: boolean;
  draftCanvasSpec?: unknown;
  shouldReadStagedCanvasVersion?: boolean;
}): boolean {
  return isEditBootstrapReady({
    isEditing: params.isEditing,
    isEnteringEditSession: params.isEnteringEditSession,
    stagingBaselinesReady: params.stagingBaselinesReady,
    draftCanvasSpec: params.draftCanvasSpec,
    shouldReadStagedCanvasVersion: params.shouldReadStagedCanvasVersion ?? false,
  });
}

export function isDraftCanvasLoadingWhileEditing({
  isEditing,
  shouldReadStagedCanvasVersion,
  isEnteringEditSession,
  isEditBootstrapReady,
  draftCanvasSpec,
  loadedCanvasVersionLoading,
  loadedCanvasVersionFetching,
}: {
  isEditing: boolean;
  shouldReadStagedCanvasVersion: boolean;
  isEnteringEditSession: boolean;
  isEditBootstrapReady: boolean;
  draftCanvasSpec: unknown;
  loadedCanvasVersionLoading: boolean;
  loadedCanvasVersionFetching: boolean;
}): boolean {
  if (isEnteringEditSession) {
    return true;
  }

  if (isEditing && !isEditBootstrapReady) {
    return true;
  }

  if (!isEditing || !shouldReadStagedCanvasVersion) {
    return false;
  }

  if (draftCanvasSpec) {
    return false;
  }

  return loadedCanvasVersionLoading || loadedCanvasVersionFetching;
}
