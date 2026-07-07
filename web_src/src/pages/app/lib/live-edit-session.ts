import type { QueryClient } from "@tanstack/react-query";
import type { MutableRefObject } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

import { clearComponentSidebarSearchParams } from "../viewState";

export function isActiveCanvasVersionCurrentLive({
  activeCanvasVersionId,
  effectiveLiveCanvasVersionId,
  liveCanvasVersionId,
}: {
  activeCanvasVersionId: string;
  effectiveLiveCanvasVersionId?: string;
  liveCanvasVersionId?: string;
}): boolean {
  if (!activeCanvasVersionId) {
    return false;
  }

  return (
    (!!effectiveLiveCanvasVersionId && activeCanvasVersionId === effectiveLiveCanvasVersionId) ||
    (!!liveCanvasVersionId && activeCanvasVersionId === liveCanvasVersionId)
  );
}

export function isViewingCurrentLiveCanvasVersion({
  activeCanvasVersionId,
  selectedCanvasVersion,
  effectiveLiveCanvasVersionId,
  liveCanvasVersionId,
}: {
  activeCanvasVersionId: string;
  selectedCanvasVersion: CanvasesCanvasVersion | null;
  effectiveLiveCanvasVersionId?: string;
  liveCanvasVersionId?: string;
}): boolean {
  if (
    isActiveCanvasVersionCurrentLive({
      activeCanvasVersionId,
      effectiveLiveCanvasVersionId,
      liveCanvasVersionId,
    })
  ) {
    return true;
  }

  if (!selectedCanvasVersion) {
    return true;
  }

  const selectedVersionId = selectedCanvasVersion.metadata?.id;
  return (
    (!!effectiveLiveCanvasVersionId && selectedVersionId === effectiveLiveCanvasVersionId) ||
    selectedVersionId === liveCanvasVersionId
  );
}

function isStagedCanvasVersionForActiveVersion(
  loadedStagedCanvasVersion: CanvasesCanvasVersion | null | undefined,
  activeCanvasVersionId: string,
): loadedStagedCanvasVersion is CanvasesCanvasVersion {
  return !!loadedStagedCanvasVersion?.metadata?.id && loadedStagedCanvasVersion.metadata.id === activeCanvasVersionId;
}

export function isAwaitingStagedCanvasSpec({
  activeCanvasVersionId,
  shouldReadStagedCanvasVersion,
  loadedStagedCanvasVersion,
  loadedStagedCanvasVersionLoading,
  loadedStagedCanvasVersionFetching,
  isEnteringEditSession,
}: {
  activeCanvasVersionId: string;
  shouldReadStagedCanvasVersion: boolean;
  loadedStagedCanvasVersion: CanvasesCanvasVersion | null | undefined;
  loadedStagedCanvasVersionLoading: boolean;
  loadedStagedCanvasVersionFetching: boolean;
  isEnteringEditSession: boolean;
}): boolean {
  const matchedStagedCanvasVersion = isStagedCanvasVersionForActiveVersion(
    loadedStagedCanvasVersion,
    activeCanvasVersionId,
  )
    ? loadedStagedCanvasVersion
    : undefined;

  return (
    shouldReadStagedCanvasVersion &&
    !matchedStagedCanvasVersion &&
    (loadedStagedCanvasVersionLoading || loadedStagedCanvasVersionFetching || isEnteringEditSession)
  );
}

// While staged canvas.yaml is still loading, avoid falling back to the version-list
// shell spec (committed snapshot) which would overwrite resynced draft state.
export function resolveSelectedCanvasVersion({
  activeCanvasVersionId,
  shouldReadStagedCanvasVersion,
  loadedStagedCanvasVersion,
  loadedCommittedCanvasVersion,
  activeCanvasVersion,
  isAwaitingStagedSpec,
}: {
  activeCanvasVersionId: string;
  shouldReadStagedCanvasVersion: boolean;
  loadedStagedCanvasVersion: CanvasesCanvasVersion | null | undefined;
  loadedCommittedCanvasVersion: CanvasesCanvasVersion | null | undefined;
  activeCanvasVersion: CanvasesCanvasVersion | null;
  isAwaitingStagedSpec: boolean;
}): CanvasesCanvasVersion | null {
  if (!activeCanvasVersionId) {
    return null;
  }

  if (shouldReadStagedCanvasVersion) {
    if (isStagedCanvasVersionForActiveVersion(loadedStagedCanvasVersion, activeCanvasVersionId)) {
      return loadedStagedCanvasVersion;
    }

    if (isAwaitingStagedSpec && activeCanvasVersion) {
      return { ...activeCanvasVersion, spec: undefined };
    }

    return activeCanvasVersion;
  }

  return loadedCommittedCanvasVersion || activeCanvasVersion;
}

export function shouldReadStagedCanvasVersion({
  editSessionActive,
  activeCanvasVersionId,
  effectiveLiveCanvasVersionId,
  liveCanvasVersionId,
}: {
  editSessionActive: boolean;
  activeCanvasVersionId: string;
  effectiveLiveCanvasVersionId?: string;
  liveCanvasVersionId?: string;
}): boolean {
  return (
    editSessionActive &&
    !!activeCanvasVersionId &&
    ((!!effectiveLiveCanvasVersionId && activeCanvasVersionId === effectiveLiveCanvasVersionId) ||
      activeCanvasVersionId === liveCanvasVersionId)
  );
}

type LiveEditSessionDraftRefs = {
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasesCanvas["spec"] | null>>;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  previewingCurrentVersionRef: MutableRefObject<boolean>;
};

export function clearLiveEditSessionDraftState({
  setEditSessionActive,
  setActiveCanvasVersion,
  setDraftCanvasSpec,
  draftCanvasSpecsRef,
  activeCanvasVersionIdRef,
  previewingCurrentVersionRef,
}: LiveEditSessionDraftRefs & {
  setEditSessionActive: (value: boolean) => void;
  setActiveCanvasVersion: (value: null) => void;
  setDraftCanvasSpec: (value: null) => void;
}): void {
  setEditSessionActive(false);
  setActiveCanvasVersion(null);
  setDraftCanvasSpec(null);
  draftCanvasSpecsRef.current.clear();
  activeCanvasVersionIdRef.current = "";
  previewingCurrentVersionRef.current = false;
}

export function clearLiveEditSessionSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("version");
  next.delete("branch");
  return clearComponentSidebarSearchParams(next);
}

export function resetCommittedLiveCanvasDetail({
  queryClient,
  organizationId,
  canvasId,
  liveCanvasVersion,
}: {
  queryClient: QueryClient;
  organizationId: string;
  canvasId: string;
  liveCanvasVersion?: CanvasesCanvasVersion | null;
}): void {
  const committedSpec = liveCanvasVersion?.spec;
  if (!committedSpec) {
    return;
  }

  queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
    if (!current) {
      return current;
    }

    return { ...current, spec: committedSpec };
  });
}
