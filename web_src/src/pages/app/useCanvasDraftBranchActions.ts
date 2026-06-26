import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { SetURLSearchParams } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { showErrorToast } from "@/lib/toast";
import { draftBranchName, draftDisplayName, draftVersionId } from "@/lib/draftVersion";
import { finalizeDraftBranchDeletion } from "@/hooks/useCanvasData";
import {
  confirmDeleteDraftBranch,
  isDeletingActiveDraft,
  switchToLiveAfterActiveDraftDelete,
} from "./confirmDeleteDraftBranch";
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
  liveCanvas?: CanvasesCanvas | null;
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
  liveCanvas,
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
  const [pendingFinalizeDeletedVersionId, setPendingFinalizeDeletedVersionId] = useState<string | null>(null);
  const [startEditingMenuOpen, setStartEditingMenuOpen] = useState(false);

  useEffect(() => {
    if (!pendingFinalizeDeletedVersionId || !organizationId || !canvasId) {
      return;
    }

    if (activeCanvasVersionId === pendingFinalizeDeletedVersionId) {
      return;
    }

    const versionId = pendingFinalizeDeletedVersionId;
    let cancelled = false;

    void finalizeDraftBranchDeletion(queryClient, organizationId, canvasId, versionId).finally(() => {
      if (!cancelled) {
        setPendingFinalizeDeletedVersionId(null);
      }
    });

    return () => {
      cancelled = true;
    };
  }, [activeCanvasVersionId, canvasId, organizationId, pendingFinalizeDeletedVersionId, queryClient]);

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
      const result = await confirmDeleteDraftBranch({
        versionId: draftVersionToDelete,
        draftBranches,
        activeCanvasVersionId,
        activeBranchMeta,
        activeBranch,
        liveCanvasVersionId,
        liveCanvas,
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

      if (result.deferFinalizeVersionId) {
        setPendingFinalizeDeletedVersionId(result.deferFinalizeVersionId);
      }

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
    liveCanvas,
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
      const branch = draftBranches.find((item) => draftVersionId(item) === versionIdToDelete);
      const branchName = branch ? draftBranchName(branch) : "";
      const isActiveDraft = isDeletingActiveDraft(
        versionIdToDelete,
        branchName,
        activeCanvasVersionId,
        activeBranchMeta,
        activeBranch,
      );

      clearPendingAutoSaveWork();

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

      if (organizationId && canvasId) {
        if (isActiveDraft) {
          switchToLiveAfterActiveDraftDelete({
            liveCanvasVersionId,
            liveCanvas,
            organizationId,
            canvasId,
            exitToLive,
            handleUseVersion,
            queryClient,
            setSearchParams,
            setActiveCanvasVersion,
            setDraftCanvasSpec,
            setLastSavedWorkflowSnapshot,
          });
          setPendingFinalizeDeletedVersionId(versionIdToDelete);
        } else {
          await finalizeDraftBranchDeletion(queryClient, organizationId, canvasId, versionIdToDelete);
        }
      }
    }

    await handleCreateVersion();
  }, [
    activeBranchMeta,
    activeBranch,
    activeCanvasVersionId,
    canvasId,
    clearPendingAutoSaveWork,
    draftBranches,
    latestDraftVersion,
    deleteDraftBranchMutation,
    exitToLive,
    handleCreateVersion,
    handleUseVersion,
    liveCanvas,
    liveCanvasVersionId,
    organizationId,
    queryClient,
    setActiveCanvasVersion,
    setDraftCanvasSpec,
    setLastSavedWorkflowSnapshot,
    setSearchParams,
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
