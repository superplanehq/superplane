import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import type { QueryClient } from "@tanstack/react-query";
import { flushSync } from "react-dom";
import type { SetURLSearchParams } from "react-router-dom";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { draftBranchName, draftVersionId } from "@/lib/draftVersion";
import { canvasKeys, cancelCanvasVersionQueries, finalizeDraftBranchDeletion } from "@/hooks/useCanvasData";
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
}: ConfirmDeleteDraftBranchOptions): Promise<void> {
  const branch = draftBranches.find((item) => draftVersionId(item) === versionId);
  const branchName = branch ? draftBranchName(branch) : "";
  const isActiveDraft =
    versionId === activeCanvasVersionId ||
    (activeBranchMeta ? versionId === draftVersionId(activeBranchMeta) : false) ||
    (!!activeBranch && branchName === activeBranch);

  try {
    if (organizationId && canvasId) {
      await cancelCanvasVersionQueries(queryClient, canvasId, versionId);
    }

    await deleteDraftBranch(versionId);

    if (isActiveDraft) {
      flushSync(() => {
        clearPendingAutoSaveWork();
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
          queryClient.setQueryData<CanvasesCanvas | undefined>(
            canvasKeys.detail(organizationId, canvasId),
            (current) => {
              if (!current) {
                return current;
              }

              return {
                ...current,
                spec: { ...current.spec, ...liveCanvasVersion.spec },
              };
            },
          );
        }
      });
    }

    if (organizationId && canvasId) {
      await finalizeDraftBranchDeletion(queryClient, organizationId, canvasId, versionId);
    }

    showSuccessToast("Draft branch deleted");
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Failed to delete draft branch"));
    throw error;
  }
}
