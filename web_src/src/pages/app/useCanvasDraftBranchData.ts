import type { CanvasesCanvasVersion } from "@/api-client";
import { useQueries } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import type { SetURLSearchParams } from "react-router-dom";
import { canvasKeys, useListDraftBranches } from "@/hooks/useCanvasData";
import { useActiveDraftBranch } from "@/hooks/useActiveDraftBranch";
import { draftBranchName, draftVersionId } from "@/lib/draftVersion";
import { fetchCanvasVersionWithSpec } from "./lib/repository-spec-files";
import { isDraftVersion, sortDraftVersionsDesc } from "./lib/canvas-versions";

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
  const { data: draftBranchesRaw = [] } = useListDraftBranches(organizationId!, canvasId!, true);
  const draftBranches = useMemo(() => sortDraftVersionsDesc(draftBranchesRaw), [draftBranchesRaw]);
  const draftVersionQueries = useQueries({
    queries: draftBranches
      .map((branch) => draftVersionId(branch))
      .filter((versionId) => !!versionId)
      .map((versionId) => ({
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
  const startEditingDefaultDraft = useMemo(() => pickDefaultDraftBranchForCanvas(), [pickDefaultDraftBranchForCanvas]);

  return {
    draftBranches,
    draftVersionsFromBranches,
    activeBranch,
    activeBranchMeta,
    activateBranch,
    exitToLive,
    startEditingDefaultDraft,
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
