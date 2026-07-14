import type { QueryClient } from "@tanstack/react-query";
import type { MutableRefObject } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

function buildEditingCanvasView(
  liveCanvas: CanvasesCanvas | null | undefined,
  draftSpecToRender: CanvasesCanvas["spec"],
  canvasId: string,
): CanvasesCanvas {
  return {
    ...(liveCanvas ?? {}),
    metadata: {
      ...(liveCanvas?.metadata ?? {}),
      id: liveCanvas?.metadata?.id ?? canvasId,
      name: liveCanvas?.metadata?.name || "Canvas",
      description: liveCanvas?.metadata?.description ?? "",
    },
    spec: draftSpecToRender,
  };
}

export function resolveCanvasForView({
  isEditing,
  isViewingCurrentLiveVersion,
  liveCanvas,
  draftSpecToRender,
  selectedCanvasVersion,
  canvasId,
}: {
  isEditing: boolean;
  isViewingCurrentLiveVersion: boolean;
  liveCanvas?: CanvasesCanvas | null;
  draftSpecToRender: CanvasesCanvas["spec"] | null;
  selectedCanvasVersion: CanvasesCanvasVersion | null;
  canvasId: string;
}): CanvasesCanvas | null | undefined {
  if (isEditing) {
    if (!draftSpecToRender) {
      // Keep the live graph visible under enter-edit loading until staged draft
      // state is applied. Returning null here clears prepared nodes and flashes empty.
      return liveCanvas ?? null;
    }

    return buildEditingCanvasView(liveCanvas, draftSpecToRender, canvasId);
  }

  if (!liveCanvas) {
    return liveCanvas;
  }

  const versionSpec = selectedCanvasVersion?.spec;
  if (!versionSpec || isViewingCurrentLiveVersion) {
    return liveCanvas;
  }

  return { ...liveCanvas, spec: versionSpec };
}

export function shouldSyncLoadedVersionToCanvasDetail({
  activeCanvasVersionId,
  loadedCanvasVersionId,
  hasLocalSaveActivity,
  isEditing,
  isViewingCurrentLiveVersion,
}: {
  activeCanvasVersionId: string;
  loadedCanvasVersionId?: string;
  hasLocalSaveActivity: boolean;
  isEditing: boolean;
  isViewingCurrentLiveVersion: boolean;
}): boolean {
  if (!activeCanvasVersionId || !loadedCanvasVersionId || hasLocalSaveActivity) {
    return false;
  }

  if (loadedCanvasVersionId !== activeCanvasVersionId) {
    return false;
  }

  if (isEditing || isViewingCurrentLiveVersion) {
    return false;
  }

  return true;
}

export function syncLoadedVersionToCanvasDetail({
  organizationId,
  canvasId,
  activeCanvasVersionId,
  loadedCanvasVersion,
  hasLocalSaveActivity,
  isEditing,
  isViewingCurrentLiveVersion,
  queryClient,
  lastAppliedVersionSnapshotRef,
}: {
  organizationId?: string;
  canvasId?: string;
  activeCanvasVersionId: string;
  loadedCanvasVersion?: CanvasesCanvasVersion | null;
  hasLocalSaveActivity: boolean;
  isEditing: boolean;
  isViewingCurrentLiveVersion: boolean;
  queryClient: QueryClient;
  lastAppliedVersionSnapshotRef: MutableRefObject<string>;
}): void {
  if (
    !organizationId ||
    !canvasId ||
    !shouldSyncLoadedVersionToCanvasDetail({
      activeCanvasVersionId,
      loadedCanvasVersionId: loadedCanvasVersion?.metadata?.id,
      hasLocalSaveActivity,
      isEditing,
      isViewingCurrentLiveVersion,
    }) ||
    !loadedCanvasVersion?.spec
  ) {
    return;
  }

  const loadedVersionID = loadedCanvasVersion.metadata?.id;
  if (!loadedVersionID) {
    return;
  }

  const snapshotKey = `${loadedVersionID}:${loadedCanvasVersion.metadata?.updatedAt || ""}`;
  if (lastAppliedVersionSnapshotRef.current === snapshotKey) {
    return;
  }

  queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
    if (!current) {
      return current;
    }

    return {
      ...current,
      spec: { ...current.spec, ...loadedCanvasVersion.spec },
    };
  });

  lastAppliedVersionSnapshotRef.current = snapshotKey;
}
