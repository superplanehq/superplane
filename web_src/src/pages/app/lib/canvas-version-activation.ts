import type { QueryClient } from "@tanstack/react-query";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import type { SetURLSearchParams } from "react-router-dom";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys, invalidateStagedCanvasCaches } from "@/hooks/useCanvasData";

import { clearRunInspectionSearchParams } from "../viewState";

export function updateCanvasDetailForSelectedVersion({
  queryClient,
  organizationId,
  canvasId,
  isCurrentLive,
  version,
  liveCanvasVersion,
  liveCanvas,
}: {
  queryClient: QueryClient;
  organizationId: string;
  canvasId: string;
  isCurrentLive: boolean;
  version: CanvasesCanvasVersion;
  liveCanvasVersion?: CanvasesCanvasVersion;
  liveCanvas?: CanvasesCanvas | null;
}) {
  queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
    if (!current) {
      return current;
    }

    if (isCurrentLive) {
      return { ...current, spec: liveCanvasVersion?.spec || liveCanvas?.spec };
    }

    if (!version.spec) {
      return current;
    }

    return { ...current, spec: { ...current.spec, ...version.spec } };
  });
}

export function refreshLiveCanvasAfterVersionSelection({
  queryClient,
  organizationId,
  canvasId,
  activeCanvasVersionIdRef,
  initializeFromWorkflow,
}: {
  queryClient: QueryClient;
  organizationId: string;
  canvasId: string;
  activeCanvasVersionIdRef: { current: string };
  initializeFromWorkflow: (canvas: CanvasesCanvas) => void;
}) {
  void Promise.all([
    queryClient.invalidateQueries({
      queryKey: canvasKeys.detail(organizationId, canvasId),
      refetchType: "all",
    }),
    queryClient.invalidateQueries({
      queryKey: canvasKeys.infiniteRuns(canvasId),
      refetchType: "all",
    }),
  ]).then(() => {
    const refreshedLiveCanvas = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId));
    if (!refreshedLiveCanvas || activeCanvasVersionIdRef.current !== "") {
      return;
    }

    initializeFromWorkflow(refreshedLiveCanvas);
  });
}

type DraftSpec = CanvasesCanvas["spec"] | null;

function isCurrentLiveVersion(versionId: string, liveCanvasVersionId?: string): boolean {
  return !!liveCanvasVersionId && versionId === liveCanvasVersionId;
}

function stashDraftSpecForPreviousVersion({
  previousVersionId,
  draftCanvasSpec,
  draftCanvasSpecsRef,
}: {
  previousVersionId: string;
  draftCanvasSpec: DraftSpec;
  draftCanvasSpecsRef: MutableRefObject<Map<string, DraftSpec>>;
}) {
  if (!previousVersionId || !draftCanvasSpec) {
    return;
  }

  draftCanvasSpecsRef.current.set(previousVersionId, draftCanvasSpec);
}

function applyDraftSpecForVersionSwitch({
  isCurrentLive,
  preserveStagedLayer,
  version,
  setDraftCanvasSpec,
}: {
  isCurrentLive: boolean;
  preserveStagedLayer: boolean;
  version: CanvasesCanvasVersion;
  setDraftCanvasSpec: Dispatch<SetStateAction<DraftSpec>>;
}) {
  if (!isCurrentLive) {
    setDraftCanvasSpec(version.spec ?? null);
    return;
  }

  if (!preserveStagedLayer) {
    setDraftCanvasSpec(null);
  }
}

export function activateCanvasVersionForEditing({
  organizationId,
  canvasId,
  versionID,
  version,
  options,
  liveCanvasVersionId,
  queryClient,
  draftCanvasSpec,
  draftCanvasSpecsRef,
  activeCanvasVersionIdRef,
  lastAppliedVersionSnapshotRef,
  liveCanvasVersion,
  liveCanvas,
  clearPendingAutoSaveWork,
  setDraftCanvasSpec,
  setActiveCanvasVersion,
  setLastSavedWorkflowSnapshot,
  setSearchParams,
  initializeFromWorkflow,
}: {
  organizationId?: string;
  canvasId?: string;
  versionID: string;
  version: CanvasesCanvasVersion;
  options?: { preserveStagedLayer?: boolean };
  liveCanvasVersionId?: string;
  queryClient: QueryClient;
  draftCanvasSpec: DraftSpec;
  draftCanvasSpecsRef: MutableRefObject<Map<string, DraftSpec>>;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  lastAppliedVersionSnapshotRef: MutableRefObject<string>;
  liveCanvasVersion?: CanvasesCanvasVersion;
  liveCanvas?: CanvasesCanvas | null;
  clearPendingAutoSaveWork: () => void;
  setDraftCanvasSpec: Dispatch<SetStateAction<DraftSpec>>;
  setActiveCanvasVersion: Dispatch<SetStateAction<CanvasesCanvasVersion | null>>;
  setLastSavedWorkflowSnapshot: (workflow: CanvasesCanvas | null) => void;
  setSearchParams: SetURLSearchParams;
  initializeFromWorkflow: (canvas: CanvasesCanvas) => void;
}): boolean {
  if (!organizationId || !canvasId) {
    return false;
  }

  const versionId = version.metadata?.id || "";
  const isCurrentLive = isCurrentLiveVersion(versionId, liveCanvasVersionId);
  const preserveStagedLayer = !!options?.preserveStagedLayer && isCurrentLive;

  clearPendingAutoSaveWork();
  stashDraftSpecForPreviousVersion({
    previousVersionId: activeCanvasVersionIdRef.current,
    draftCanvasSpec,
    draftCanvasSpecsRef,
  });

  if (isCurrentLive && !preserveStagedLayer) {
    draftCanvasSpecsRef.current.delete(versionId);
    invalidateStagedCanvasCaches(queryClient, canvasId);
  }

  if (!isCurrentLive) {
    void queryClient.cancelQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
  }

  activeCanvasVersionIdRef.current = versionID;
  applyDraftSpecForVersionSwitch({ isCurrentLive, preserveStagedLayer, version, setDraftCanvasSpec });
  setActiveCanvasVersion(version);
  lastAppliedVersionSnapshotRef.current = "";
  setLastSavedWorkflowSnapshot(null);

  setSearchParams((current) => {
    const next = clearRunInspectionSearchParams(new URLSearchParams(current));
    next.delete("branch");
    if (isCurrentLive) {
      next.delete("version");
    } else {
      next.set("version", versionID);
    }
    // Same query string → keep current instance so React Router skips a no-op navigation.
    if (next.toString() === current.toString()) {
      return current;
    }
    return next;
  });

  if (!preserveStagedLayer) {
    updateCanvasDetailForSelectedVersion({
      queryClient,
      organizationId,
      canvasId,
      isCurrentLive,
      version,
      liveCanvasVersion,
      liveCanvas,
    });
  }

  if (isCurrentLive && !preserveStagedLayer) {
    refreshLiveCanvasAfterVersionSelection({
      queryClient,
      organizationId,
      canvasId,
      activeCanvasVersionIdRef,
      initializeFromWorkflow,
    });
  }

  return true;
}
