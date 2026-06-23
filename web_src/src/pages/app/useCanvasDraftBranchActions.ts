import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { useCallback, useMemo, useState } from "react";
import type { SetURLSearchParams } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { showErrorToast } from "@/lib/toast";
import { draftBranchName, draftDisplayName, draftVersionId } from "@/lib/draftVersion";
import { confirmDeleteDraftBranch } from "./confirmDeleteDraftBranch";
import { resolveDraftVersionIdForBranch } from "./draftBranchVersionId";

type DeleteDraftBranchMutation = {
  mutateAsync: (versionId: string) => Promise<unknown>;
  isPending: boolean;
};

type UseCanvasDraftBranchActionsOptions = {
  canUpdateCanvas: boolean;
  draftBranches: CanvasesCanvasVersion[];
  activeCanvasVersionId: string;
  activeBranchMeta: CanvasesCanvasVersion | null;
  activeBranch: string | null;
  liveCanvasVersionId?: string;
  liveCanvasVersion?: CanvasesCanvasVersion;
  organizationId?: string;
  canvasId?: string;
  latestDraftVersion?: CanvasesCanvasVersion;
  clearPendingAutoSaveWork: () => void;
  deleteDraftBranchMutation: DeleteDraftBranchMutation;
  exitToLive: () => void;
  handleUseVersion: (versionId: string) => void;
  handleCreateVersion: () => Promise<void>;
  queryClient: QueryClient;
  setSearchParams: SetURLSearchParams;
  setActiveCanvasVersion: (value: CanvasesCanvasVersion | null) => void;
  setDraftCanvasSpec: (value: CanvasesCanvas["spec"] | null) => void;
  setLastSavedWorkflowSnapshot: (workflow: CanvasesCanvas | null) => void;
};

export function useCanvasDraftBranchActions({
  canUpdateCanvas,
  draftBranches,
  activeCanvasVersionId,
  activeBranchMeta,
  activeBranch,
  liveCanvasVersionId,
  liveCanvasVersion,
  organizationId,
  canvasId,
  latestDraftVersion,
  clearPendingAutoSaveWork,
  deleteDraftBranchMutation,
  exitToLive,
  handleUseVersion,
  handleCreateVersion,
  queryClient,
  setSearchParams,
  setActiveCanvasVersion,
  setDraftCanvasSpec,
  setLastSavedWorkflowSnapshot,
}: UseCanvasDraftBranchActionsOptions) {
  const [draftVersionToDelete, setDraftVersionToDelete] = useState<string | null>(null);
  const [startEditingMenuOpen, setStartEditingMenuOpen] = useState(false);

  const handleContinueDraftBranch = useCallback(
    (branchName: string) => {
      const branch = draftBranches.find((item) => draftBranchName(item) === branchName);
      const versionId = branch ? draftVersionId(branch) : "";
      if (!versionId) {
        showErrorToast("Draft branch not found");
        return;
      }
      handleUseVersion(versionId);
    },
    [draftBranches, handleUseVersion],
  );

  const handleCreateDraftBranch = useCallback(async () => {
    setStartEditingMenuOpen(false);
    await handleCreateVersion();
  }, [handleCreateVersion]);

  const handleDeleteDraftBranch = useCallback(
    (versionId: string) => {
      if (!canUpdateCanvas) {
        showErrorToast("You don't have permission to edit this canvas");
        return;
      }

      const branch = draftBranches.find((item) => draftVersionId(item) === versionId);
      if (!branch) {
        showErrorToast("Draft branch not found");
        return;
      }

      setDraftVersionToDelete(versionId);
    },
    [canUpdateCanvas, draftBranches],
  );

  const draftVersionToDeleteName = useMemo(() => {
    if (!draftVersionToDelete) {
      return "";
    }

    const branch = draftBranches.find((item) => draftVersionId(item) === draftVersionToDelete);
    return branch ? draftDisplayName(branch) : draftVersionToDelete;
  }, [draftVersionToDelete, draftBranches]);

  const confirmDeleteDraftVersion = useCallback(async () => {
    if (!draftVersionToDelete) {
      return;
    }

    try {
      await confirmDeleteDraftBranch({
        versionId: draftVersionToDelete,
        draftBranches,
        activeCanvasVersionId,
        activeBranchMeta,
        activeBranch,
        liveCanvasVersionId,
        liveCanvasVersion,
        organizationId,
        canvasId,
        clearPendingAutoSaveWork,
        deleteDraftBranch: deleteDraftBranchMutation.mutateAsync,
        exitToLive,
        handleUseVersion,
        queryClient,
        setSearchParams,
        setActiveCanvasVersion,
        setDraftCanvasSpec,
        setLastSavedWorkflowSnapshot,
      });
      setDraftVersionToDelete(null);
    } catch {
      // Error toast is handled in confirmDeleteDraftBranch.
    }
  }, [
    draftVersionToDelete,
    draftBranches,
    activeCanvasVersionId,
    activeBranchMeta,
    activeBranch,
    clearPendingAutoSaveWork,
    deleteDraftBranchMutation,
    exitToLive,
    liveCanvasVersionId,
    handleUseVersion,
    liveCanvasVersion,
    queryClient,
    organizationId,
    canvasId,
    setSearchParams,
    setLastSavedWorkflowSnapshot,
    setActiveCanvasVersion,
    setDraftCanvasSpec,
  ]);

  const requestDeleteActiveDraft = useCallback(() => {
    const versionIdToDelete = resolveDraftVersionIdForBranch({
      activeBranchMeta,
      activeBranch,
      draftBranches,
      fallbackVersionId: activeCanvasVersionId,
    });

    if (!versionIdToDelete) {
      showErrorToast("Draft branch not found");
      return;
    }

    setDraftVersionToDelete(versionIdToDelete);
  }, [activeBranchMeta, activeBranch, draftBranches, activeCanvasVersionId]);

  const discardDraftAndCreateNew = useCallback(async () => {
    const versionIdToDelete = resolveDraftVersionIdForBranch({
      activeBranchMeta,
      activeBranch,
      draftBranches,
      latestDraftVersionId: latestDraftVersion?.metadata?.id,
    });

    if (versionIdToDelete) {
      try {
        await deleteDraftBranchMutation.mutateAsync(versionIdToDelete);
      } catch (error) {
        const message =
          (error as { response?: { data?: { message?: string } } })?.response?.data?.message ||
          (error as { message?: string })?.message ||
          "Failed to discard draft";
        showErrorToast(message);
        return;
      }
    }

    exitToLive();
    await handleCreateVersion();
  }, [
    activeBranchMeta,
    activeBranch,
    draftBranches,
    latestDraftVersion,
    deleteDraftBranchMutation,
    handleCreateVersion,
    exitToLive,
  ]);

  return {
    draftVersionToDelete,
    setDraftVersionToDelete,
    draftVersionToDeleteName,
    startEditingMenuOpen,
    setStartEditingMenuOpen,
    handleContinueDraftBranch,
    handleCreateDraftBranch,
    handleDeleteDraftBranch,
    confirmDeleteDraftVersion,
    requestDeleteActiveDraft,
    discardDraftAndCreateNew,
  };
}
