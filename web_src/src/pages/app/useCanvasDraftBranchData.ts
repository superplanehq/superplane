import type { CanvasesCanvasVersion } from "@/api-client";
import { useQueries } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import type { SetURLSearchParams } from "react-router-dom";
import { canvasKeys, useListCanvasBranches } from "@/hooks/useCanvasData";
import { useActiveDraftBranch } from "@/hooks/useActiveDraftBranch";
import { draftBranchName, draftVersionId } from "@/lib/draftVersion";
import { branchHeadVersionId, branchName, pickDefaultCanvasBranch, sortCanvasBranches } from "@/lib/canvas-branches";
import { fetchCanvasVersionWithSpec } from "./lib/repository-spec-files";
import { isDraftVersion } from "./lib/canvas-versions";

type UseCanvasDraftBranchQueriesOptions = {
  organizationId?: string;
  canvasId?: string;
  currentUserId?: string;
  searchParams: URLSearchParams;
  setSearchParams: SetURLSearchParams;
};

export function useCanvasDraftBranchQueries({
  organizationId,
  canvasId,
  searchParams,
  setSearchParams,
}: UseCanvasDraftBranchQueriesOptions) {
  const { data: canvasBranchesRaw = [], isFetched: canvasBranchesFetched } = useListCanvasBranches(
    organizationId!,
    canvasId!,
    true,
  );
  const canvasBranches = useMemo(() => sortCanvasBranches(canvasBranchesRaw), [canvasBranchesRaw]);

  const branchHeadVersionIds = useMemo(() => {
    const ids = new Set<string>();
    for (const branch of canvasBranches) {
      const versionId = branchHeadVersionId(branch);
      if (versionId) {
        ids.add(versionId);
      }
    }
    return Array.from(ids);
  }, [canvasBranches]);

  const draftVersionQueries = useQueries({
    queries: branchHeadVersionIds.map((versionId) => ({
      queryKey: canvasKeys.versionDetail(canvasId!, versionId),
      queryFn: async () => fetchCanvasVersionWithSpec(canvasId!, versionId),
      enabled: !!organizationId && !!canvasId && !!versionId,
    })),
  });

  const draftVersionsFromBranches = useMemo(
    () =>
      draftVersionQueries.map((query) => query.data).filter((version): version is CanvasesCanvasVersion => !!version),
    [draftVersionQueries],
  );

  const draftBranches = useMemo(() => {
    const byBranchName = new Map<string, CanvasesCanvasVersion>();
    for (const version of draftVersionsFromBranches) {
      const name = draftBranchName(version);
      if (name) {
        byBranchName.set(name, version);
      }
    }

    return canvasBranches
      .map((branch) => byBranchName.get(branchName(branch)))
      .filter((version): version is CanvasesCanvasVersion => !!version);
  }, [canvasBranches, draftVersionsFromBranches]);

  const {
    activeBranch,
    activeBranchMeta,
    activateBranch,
    exitToLive,
    pickDefaultDraftBranch: pickDefaultDraftBranchForCanvas,
  } = useActiveDraftBranch({
    canvasId,
    searchParams,
    setSearchParams,
    draftBranches,
  });

  const startEditingDefaultBranch = useMemo(() => pickDefaultCanvasBranch(canvasBranches), [canvasBranches]);
  const startEditingDefaultDraft = useMemo(() => {
    const defaultBranch = startEditingDefaultBranch;
    if (!defaultBranch) {
      return pickDefaultDraftBranchForCanvas();
    }

    const branchVersion = draftBranches.find((version) => draftBranchName(version) === branchName(defaultBranch));
    return branchVersion ?? pickDefaultDraftBranchForCanvas();
  }, [draftBranches, pickDefaultDraftBranchForCanvas, startEditingDefaultBranch]);

  return {
    canvasBranches,
    canvasBranchesFetched,
    draftBranches,
    draftVersionsFromBranches,
    activeBranch,
    activeBranchMeta,
    activateBranch,
    exitToLive,
    startEditingDefaultDraft,
    startEditingDefaultBranch,
  };
}

type UseResolvedActiveDraftBranchOptions = {
  canvasId?: string;
  activeBranch: string | null;
  activeBranchMeta: CanvasesCanvasVersion | null;
  draftBranches: CanvasesCanvasVersion[];
  activeCanvasVersionId: string;
  selectedCanvasVersion?: CanvasesCanvasVersion | null;
  hasEditableVersion: boolean;
  activateBranch: (branchName: string) => void;
};

export function useResolvedActiveDraftBranch({
  canvasId,
  activeBranch,
  activeBranchMeta,
  draftBranches,
  activeCanvasVersionId,
  selectedCanvasVersion,
  hasEditableVersion,
  activateBranch,
}: UseResolvedActiveDraftBranchOptions) {
  const resolvedActiveBranchMeta = useMemo(() => {
    if (activeCanvasVersionId) {
      const fromDraftList = draftBranches.find((branch) => draftVersionId(branch) === activeCanvasVersionId);
      if (fromDraftList) {
        return fromDraftList;
      }
    }

    if (activeBranchMeta) {
      return activeBranchMeta;
    }

    if (selectedCanvasVersion && isDraftVersion(selectedCanvasVersion)) {
      return selectedCanvasVersion;
    }

    return null;
  }, [activeBranchMeta, activeCanvasVersionId, draftBranches, selectedCanvasVersion]);
  const resolvedActiveBranch = useMemo(() => {
    const branchNameFromMeta = resolvedActiveBranchMeta ? draftBranchName(resolvedActiveBranchMeta) : "";
    if (branchNameFromMeta) {
      return branchNameFromMeta;
    }

    if (activeBranch) {
      return activeBranch;
    }

    if (selectedCanvasVersion && isDraftVersion(selectedCanvasVersion)) {
      return draftBranchName(selectedCanvasVersion) || null;
    }

    return null;
  }, [activeBranch, resolvedActiveBranchMeta, selectedCanvasVersion]);

  useEffect(() => {
    if (!hasEditableVersion || !canvasId) {
      return;
    }

    const branchName = resolvedActiveBranch;
    if (!branchName || activeBranch === branchName) {
      return;
    }

    activateBranch(branchName);
  }, [hasEditableVersion, canvasId, resolvedActiveBranch, activeBranch, activateBranch]);

  return { resolvedActiveBranchMeta, resolvedActiveBranch };
}
