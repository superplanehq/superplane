import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import type { QueryClient } from "@tanstack/react-query";
import type { SetURLSearchParams } from "react-router-dom";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { draftBranchName, draftVersionId } from "@/lib/draftVersion";
import { canvasKeys, finalizeDraftBranchDeletion } from "@/hooks/useCanvasData";
import { clearComponentSidebarSearchParams } from "./viewState";

type ConfirmDeleteDraftBranchOptions = {
  versionId: string;
  draftBranches: CanvasesCanvasVersion[];
  activeCanvasVersionId: string;
  activeBranchMeta: CanvasesCanvasVersion | null;
  activeBranch: string | null;
  liveCanvasVersionId?: string;
  liveCanvasVersion?: CanvasesCanvasVersion;
  organizationId?: string;
  canvasId?: string;
  clearPendingAutoSaveWork: () => void;
  deleteDraftBranch: (versionId: string) => Promise<unknown>;
  exitToLive: () => void;
  handleUseVersion: (versionId: string) => void;
  queryClient: QueryClient;
  setSearchParams: SetURLSearchParams;
  setActiveCanvasVersion: (value: CanvasesCanvasVersion | null) => void;
  setDraftCanvasSpec: (value: CanvasesCanvas["spec"] | null) => void;
  setLastSavedWorkflowSnapshot: (workflow: CanvasesCanvas | null) => void;
};

export type ConfirmDeleteDraftBranchResult = {
  /** When deleting the active draft, cache cleanup waits until live exit renders. */
  deferFinalizeVersionId?: string;
};

export function isDeletingActiveDraft(
  versionId: string,
  branchName: string,
  activeCanvasVersionId: string,
  activeBranchMeta: CanvasesCanvasVersion | null,
  activeBranch: string | null,
): boolean {
  return (
    versionId === activeCanvasVersionId ||
    (activeBranchMeta ? versionId === draftVersionId(activeBranchMeta) : false) ||
    (!!activeBranch && branchName === activeBranch)
  );
}

export function switchToLiveAfterActiveDraftDelete({
  liveCanvasVersionId,
  liveCanvasVersion,
  organizationId,
  canvasId,
  exitToLive,
  handleUseVersion,
  queryClient,
  setSearchParams,
  setActiveCanvasVersion,
  setDraftCanvasSpec,
  setLastSavedWorkflowSnapshot,
}: Pick<
  ConfirmDeleteDraftBranchOptions,
  | "liveCanvasVersionId"
  | "liveCanvasVersion"
  | "organizationId"
  | "canvasId"
  | "exitToLive"
  | "handleUseVersion"
  | "queryClient"
  | "setSearchParams"
  | "setActiveCanvasVersion"
  | "setDraftCanvasSpec"
  | "setLastSavedWorkflowSnapshot"
>): void {
  setLastSavedWorkflowSnapshot(null);
  exitToLive();

  if (liveCanvasVersionId) {
    handleUseVersion(liveCanvasVersionId);
  } else {
    setActiveCanvasVersion(null);
    setDraftCanvasSpec(null);
    setSearchParams((current) => {
      const next = new URLSearchParams(current);
      next.delete("version");
      next.delete("branch");
      return clearComponentSidebarSearchParams(next);
    });
  }

  if (liveCanvasVersion?.spec && organizationId && canvasId) {
    queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
      if (!current) {
        return current;
      }

      return {
        ...current,
        spec: { ...current.spec, ...liveCanvasVersion.spec },
      };
    });
  }
}

export async function confirmDeleteDraftBranch({
  versionId,
  draftBranches,
  activeCanvasVersionId,
  activeBranchMeta,
  activeBranch,
  liveCanvasVersionId,
  liveCanvasVersion,
  organizationId,
  canvasId,
  clearPendingAutoSaveWork,
  deleteDraftBranch,
  exitToLive,
  handleUseVersion,
  queryClient,
  setSearchParams,
  setActiveCanvasVersion,
  setDraftCanvasSpec,
  setLastSavedWorkflowSnapshot,
}: ConfirmDeleteDraftBranchOptions): Promise<ConfirmDeleteDraftBranchResult> {
  const branch = draftBranches.find((item) => draftVersionId(item) === versionId);
  const branchName = branch ? draftBranchName(branch) : "";
  const isActiveDraft = isDeletingActiveDraft(
    versionId,
    branchName,
    activeCanvasVersionId,
    activeBranchMeta,
    activeBranch,
  );

  try {
    if (isActiveDraft) {
      clearPendingAutoSaveWork();
    }

    await deleteDraftBranch(versionId);

    if (isActiveDraft) {
      switchToLiveAfterActiveDraftDelete({
        liveCanvasVersionId,
        liveCanvasVersion,
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

      showSuccessToast("Draft branch deleted");
      return { deferFinalizeVersionId: versionId };
    }

    if (organizationId && canvasId) {
      await finalizeDraftBranchDeletion(queryClient, organizationId, canvasId, versionId);
    }

    showSuccessToast("Draft branch deleted");
    return {};
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Failed to delete draft branch"));
    throw error;
  }
}
