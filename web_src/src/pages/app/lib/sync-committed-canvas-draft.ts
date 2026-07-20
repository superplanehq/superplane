import type { QueryClient } from "@tanstack/react-query";
import type { Dispatch, SetStateAction } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys, fetchCanvasConsoleData } from "@/hooks/useCanvasData";

import { fetchCommittedCanvasVersionWithSpec, fetchLiveCommittedCanvasVersionWithSpec } from "./repository-spec-files";

export async function syncCommittedConsoleCaches({
  queryClient,
  canvasId,
  versionId,
}: {
  queryClient: QueryClient;
  canvasId: string;
  versionId: string;
}): Promise<void> {
  const consoleData = await fetchCanvasConsoleData(canvasId, versionId, false);
  if (!consoleData) {
    await queryClient.invalidateQueries({ queryKey: canvasKeys.stagedConsole(canvasId) });
    return;
  }

  queryClient.setQueryData(canvasKeys.console(canvasId, versionId), consoleData);
  // After commit, staging is cleared — mirror the committed console in the staged cache.
  queryClient.setQueryData(canvasKeys.stagedConsole(canvasId), consoleData);
}

export async function syncCommittedCanvasDraftState({
  queryClient,
  organizationId,
  canvasId,
  versionId,
  resolveLiveVersion = false,
  skipVersionListUpdate = false,
}: {
  queryClient: QueryClient;
  organizationId: string;
  canvasId: string;
  versionId: string;
  resolveLiveVersion?: boolean;
  skipVersionListUpdate?: boolean;
}): Promise<CanvasesCanvasVersion | undefined> {
  const committedVersion = resolveLiveVersion
    ? await fetchLiveCommittedCanvasVersionWithSpec(canvasId)
    : await fetchCommittedCanvasVersionWithSpec(canvasId, versionId);
  if (!committedVersion) {
    return undefined;
  }

  const cacheVersionId = committedVersion.metadata?.id ?? versionId;

  queryClient.setQueryData(canvasKeys.stagedCanvasSpec(canvasId), committedVersion);
  queryClient.setQueryData(canvasKeys.versionDetail(canvasId, cacheVersionId), committedVersion);

  if (!skipVersionListUpdate) {
    queryClient.setQueryData(canvasKeys.versionList(canvasId), (current: CanvasesCanvasVersion[] | undefined) => {
      const existing = current ?? [];
      const index = existing.findIndex((item) => item.metadata?.id === cacheVersionId);
      if (index === -1) {
        return [committedVersion, ...existing];
      }

      return existing.map((item) =>
        item.metadata?.id === cacheVersionId ? { ...item, spec: committedVersion.spec } : item,
      );
    });
  }

  if (committedVersion.spec) {
    queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
      if (!current) {
        return current;
      }

      return {
        ...current,
        spec: { ...current.spec, ...committedVersion.spec },
      };
    });
  }

  return committedVersion;
}

type DraftSpec = CanvasesCanvas["spec"] | null;

export async function restoreCommittedCanvasDraftState({
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

  const committedVersion = await syncCommittedCanvasDraftState({
    queryClient,
    organizationId,
    canvasId,
    versionId: activeCanvasVersionId,
  });

  if (!committedVersion?.spec) {
    draftCanvasSpecsRef.current.delete(activeCanvasVersionId);
    setDraftCanvasSpec(null);
    return;
  }

  draftCanvasSpecsRef.current.set(activeCanvasVersionId, committedVersion.spec);
  setDraftCanvasSpec(committedVersion.spec);
  setActiveCanvasVersion?.((current) =>
    current?.metadata?.id === activeCanvasVersionId ? { ...current, spec: committedVersion.spec } : current,
  );
  onCanvasDraftRestoredToCommitted?.(committedVersion);
}
