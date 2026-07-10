import type { QueryClient } from "@tanstack/react-query";
import type { MutableRefObject } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion, CanvasesStaging } from "@/api-client";
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

export function isAwaitingCanvasStaging({
  shouldReadStagedCanvasVersion,
  stagingLoading,
  stagingFetching,
  isEnteringEditSession,
  staging,
}: {
  shouldReadStagedCanvasVersion: boolean;
  stagingLoading: boolean;
  stagingFetching: boolean;
  isEnteringEditSession: boolean;
  staging: CanvasesStaging | undefined;
}): boolean {
  if (!shouldReadStagedCanvasVersion) {
    return false;
  }

  if (isEnteringEditSession) {
    return true;
  }

  if (stagingLoading || stagingFetching) {
    return true;
  }

  return !staging;
}

// While staged canvas spec is still loading, avoid falling back to the version-list
// shell spec (committed snapshot) which would overwrite resynced draft state.
export function resolveSelectedCanvasVersion({
  activeCanvasVersionId,
  shouldReadStagedCanvasVersion,
  staging,
  loadedCommittedCanvasVersion,
  activeCanvasVersion,
  awaitingCanvasStaging,
}: {
  activeCanvasVersionId: string;
  shouldReadStagedCanvasVersion: boolean;
  staging: CanvasesStaging | undefined;
  loadedCommittedCanvasVersion: CanvasesCanvasVersion | null | undefined;
  activeCanvasVersion: CanvasesCanvasVersion | null;
  awaitingCanvasStaging: boolean;
}): CanvasesCanvasVersion | null {
  if (!activeCanvasVersionId) {
    return null;
  }

  if (shouldReadStagedCanvasVersion) {
    if (staging?.spec && activeCanvasVersion) {
      return { ...activeCanvasVersion, spec: staging.spec };
    }

    if (awaitingCanvasStaging && activeCanvasVersion) {
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
