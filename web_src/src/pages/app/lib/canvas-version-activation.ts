import type { QueryClient } from "@tanstack/react-query";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import type { SetURLSearchParams } from "react-router-dom";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys, invalidateStagedCanvasCaches } from "@/hooks/useCanvasData";

import { clearComponentSidebarSearchParams } from "../viewState";

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

function isCurrentLiveVersion(
  versionId: string,
  effectiveLiveCanvasVersionId?: string,
  liveCanvasVersionId?: string,
): boolean {
  return (
    (!!effectiveLiveCanvasVersionId && versionId === effectiveLiveCanvasVersionId) ||
    (!!liveCanvasVersionId && versionId === liveCanvasVersionId)
  );
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
  effectiveLiveCanvasVersionId,
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
  effectiveLiveCanvasVersionId?: string;
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
  const isCurrentLive = isCurrentLiveVersion(versionId, effectiveLiveCanvasVersionId, liveCanvasVersionId);
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
    const next = new URLSearchParams(current);
    next.delete("branch");
    if (isCurrentLive) {
      next.delete("version");
    } else {
      next.set("version", versionID);
    }
    return clearComponentSidebarSearchParams(next);
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
