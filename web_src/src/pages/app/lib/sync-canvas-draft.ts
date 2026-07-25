import type { QueryClient } from "@tanstack/react-query";
import type { Dispatch, SetStateAction } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys, syncCanvasStagingCache } from "@/hooks/useCanvasData";

import { fetchCanvasVersionWithSpec } from "./repository-spec-files";

export async function syncCanvasDraftState({
  queryClient,
  organizationId,
  canvasId,
  versionId,
}: {
  queryClient: QueryClient;
  organizationId: string;
  canvasId: string;
  versionId: string;
}): Promise<CanvasesCanvasVersion | undefined> {
  const version = await fetchCanvasVersionWithSpec(canvasId, versionId);
  if (!version) {
    return undefined;
  }

  const cacheVersionId = version.metadata?.id ?? versionId;

  syncCanvasStagingCache(queryClient, canvasId, {
    stagingSummary: { hasStaging: false, stagedPaths: [] },
    spec: version.spec,
  });
  queryClient.setQueryData(canvasKeys.version(canvasId, cacheVersionId), version);

  if (version.spec) {
    queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
      if (!current) {
        return current;
      }

      return {
        ...current,
        spec: { ...current.spec, ...version.spec },
      };
    });
  }

  return version;
}

type DraftSpec = CanvasesCanvas["spec"] | null;

export async function restoreCanvasDraftState({
  organizationId,
  canvasId,
  activeCanvasVersionId,
  queryClient,
  draftCanvasSpecsRef,
  setDraftCanvasSpec,
  setActiveCanvasVersion,
  onCanvasDraftRestoredToCommitted,
}: {
  organizationId?: string;
  canvasId?: string;
  activeCanvasVersionId: string;
  queryClient: QueryClient;
  draftCanvasSpecsRef: { current: Map<string, DraftSpec> };
  setDraftCanvasSpec: Dispatch<SetStateAction<DraftSpec>>;
  setActiveCanvasVersion?: Dispatch<SetStateAction<CanvasesCanvasVersion | null>>;
  onCanvasDraftRestoredToCommitted?: (version: CanvasesCanvasVersion) => void;
}) {
  if (!organizationId || !canvasId) {
    draftCanvasSpecsRef.current.delete(activeCanvasVersionId);
    setDraftCanvasSpec(null);
    return;
  }

  const version = await syncCanvasDraftState({
    queryClient,
    organizationId,
    canvasId,
    versionId: activeCanvasVersionId,
  });

  if (!version?.spec) {
    draftCanvasSpecsRef.current.delete(activeCanvasVersionId);
    setDraftCanvasSpec(null);
    return;
  }

  draftCanvasSpecsRef.current.set(activeCanvasVersionId, version.spec);
  setDraftCanvasSpec(version.spec);
  setActiveCanvasVersion?.((current) =>
    current?.metadata?.id === activeCanvasVersionId ? { ...current, spec: version.spec } : current,
  );
  onCanvasDraftRestoredToCommitted?.(version);
}
