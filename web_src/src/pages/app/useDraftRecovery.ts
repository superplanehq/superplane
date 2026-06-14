import { useCallback, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import type { useSearchParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { ensureDraftVersionExists } from "@/hooks/useCanvasData";

import { clearPublishedDraftVersion } from "./lib/draft-spec-cache";
import { clearComponentSidebarSearchParams } from "./viewState";
import { isNotFoundError } from "./workflowPageHelpers";

type DraftSpec = CanvasesCanvas["spec"] | null;
type SetSearchParams = ReturnType<typeof useSearchParams>[1];
type CommitMutation = { mutateAsync: () => Promise<unknown> };
type PublishMutation = { mutateAsync: (versionId: string) => Promise<unknown> };

type UseDraftRecoveryOptions = {
  organizationId?: string;
  canvasId?: string;
  activeCanvasVersionId: string;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  draftCanvasSpecsRef: MutableRefObject<Map<string, DraftSpec>>;
  setActiveCanvasVersion: Dispatch<SetStateAction<CanvasesCanvasVersion | null>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<DraftSpec>>;
  exitToLive: () => void;
  setSearchParams: SetSearchParams;
  refreshLatestLiveCanvasData: () => Promise<void>;
  cancelPendingCanvasSaves?: () => void;
  ensureVersionActionDraftReady: (errorMessage: string) => Promise<boolean>;
  commitCanvasStagingMutation: CommitMutation;
  publishCanvasVersionMutation: PublishMutation;
  consoleMutationGenerationRef: MutableRefObject<number>;
  setIsPreparingVersionAction: Dispatch<SetStateAction<boolean>>;
};

// Owns the draft publish + exit-to-live + recovery lifecycle, guarding publish
// against a draft that was deleted out from under it so a stale id can't strand
// the UI on a "version not found" error.
export function useDraftRecovery({
  organizationId,
  canvasId,
  activeCanvasVersionId,
  activeCanvasVersionIdRef,
  draftCanvasSpecsRef,
  setActiveCanvasVersion,
  setDraftCanvasSpec,
  exitToLive,
  setSearchParams,
  refreshLatestLiveCanvasData,
  cancelPendingCanvasSaves,
  ensureVersionActionDraftReady,
  commitCanvasStagingMutation,
  publishCanvasVersionMutation,
  consoleMutationGenerationRef,
  setIsPreparingVersionAction,
}: UseDraftRecoveryOptions) {
  const queryClient = useQueryClient();

  // Shared by publish-success and missing-draft recovery so both end identically.
  const exitDraftToLive = useCallback(
    async (versionId: string) => {
      activeCanvasVersionIdRef.current = "";
      if (versionId) {
        clearPublishedDraftVersion(draftCanvasSpecsRef.current, setActiveCanvasVersion, setDraftCanvasSpec, versionId);
      }
      exitToLive();
      setSearchParams((current) => {
        const next = new URLSearchParams(current);
        next.delete("version");
        next.delete("branch");
        return clearComponentSidebarSearchParams(next);
      });
      await refreshLatestLiveCanvasData();
    },
    [
      activeCanvasVersionIdRef,
      draftCanvasSpecsRef,
      exitToLive,
      refreshLatestLiveCanvasData,
      setActiveCanvasVersion,
      setDraftCanvasSpec,
      setSearchParams,
    ],
  );

  // Recoverable state, not a fatal error: return to live with an info toast.
  const recoverFromMissingDraft = useCallback(
    async (versionId: string, message = "This draft no longer exists. Returned to the live canvas.") => {
      cancelPendingCanvasSaves?.();
      await exitDraftToLive(versionId);
      showErrorToast(message);
    },
    [cancelPendingCanvasSaves, exitDraftToLive],
  );

  const handlePublishVersion = useCallback(async () => {
    if (!organizationId || !canvasId || !activeCanvasVersionId) {
      return;
    }

    const versionIdToPublish = activeCanvasVersionIdRef.current;

    setIsPreparingVersionAction(true);
    try {
      const isReady = await ensureVersionActionDraftReady(
        "Unable to prepare the latest version changes for publishing",
      );
      if (!isReady || !versionIdToPublish) {
        return;
      }

      const draftExists = await ensureDraftVersionExists(queryClient, organizationId, canvasId, versionIdToPublish);
      if (!draftExists) {
        await recoverFromMissingDraft(versionIdToPublish);
        return;
      }

      // Flush staged edits into the committed row before promoting it to live.
      consoleMutationGenerationRef.current += 1;
      await commitCanvasStagingMutation.mutateAsync();
      await publishCanvasVersionMutation.mutateAsync(versionIdToPublish);
      await exitDraftToLive(versionIdToPublish);
      showSuccessToast("Version published");
    } catch (error) {
      // The draft can be deleted between the pre-check and the mutation.
      if (isNotFoundError(error)) {
        await recoverFromMissingDraft(versionIdToPublish);
        return;
      }
      showErrorToast(getUsageLimitToastMessage(error, getApiErrorMessage(error, "Failed to publish version")));
    } finally {
      setIsPreparingVersionAction(false);
    }
  }, [
    organizationId,
    canvasId,
    activeCanvasVersionId,
    activeCanvasVersionIdRef,
    ensureVersionActionDraftReady,
    queryClient,
    consoleMutationGenerationRef,
    commitCanvasStagingMutation,
    publishCanvasVersionMutation,
    setIsPreparingVersionAction,
    exitDraftToLive,
    recoverFromMissingDraft,
  ]);

  return { handlePublishVersion, recoverFromMissingDraft };
}
