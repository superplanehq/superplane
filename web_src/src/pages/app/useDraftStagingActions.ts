import { useQueryClient } from "@tanstack/react-query";
import { useCallback, type Dispatch, type MutableRefObject, type SetStateAction } from "react";

import type { CanvasesCanvas } from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { canvasKeys } from "@/hooks/useCanvasData";

type CommitMutation = { mutateAsync: () => Promise<unknown> };
type DiscardMutation = { mutateAsync: (input: undefined) => Promise<unknown> };

type UseDraftStagingActionsOptions = {
  canvasId?: string;
  activeCanvasVersionId: string;
  hasEditableVersion: boolean;
  ensureVersionActionDraftReady: (errorMessage: string) => Promise<boolean>;
  commitCanvasStagingMutation: CommitMutation;
  discardCanvasStagingMutation: DiscardMutation;
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasesCanvas["spec"] | null>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<CanvasesCanvas["spec"] | null>>;
  setStagingResetNonce: Dispatch<SetStateAction<number>>;
  consoleMutationGenerationRef: MutableRefObject<number>;
  setIsPreparingVersionAction: Dispatch<SetStateAction<boolean>>;
};

export function useDraftStagingActions({
  canvasId,
  activeCanvasVersionId,
  hasEditableVersion,
  ensureVersionActionDraftReady,
  commitCanvasStagingMutation,
  discardCanvasStagingMutation,
  draftCanvasSpecsRef,
  setDraftCanvasSpec,
  setStagingResetNonce,
  consoleMutationGenerationRef,
  setIsPreparingVersionAction,
}: UseDraftStagingActionsOptions) {
  const queryClient = useQueryClient();

  const handleCommitStaging = useCallback(async () => {
    if (!hasEditableVersion || !activeCanvasVersionId) {
      return;
    }

    setIsPreparingVersionAction(true);
    try {
      const isReady = await ensureVersionActionDraftReady("Unable to prepare staged changes for commit");
      if (!isReady) {
        return;
      }

      consoleMutationGenerationRef.current += 1;
      await commitCanvasStagingMutation.mutateAsync();
      await queryClient.invalidateQueries({ queryKey: canvasKeys.repository(canvasId!) });
      draftCanvasSpecsRef.current.delete(activeCanvasVersionId);
      setDraftCanvasSpec(null);
      setStagingResetNonce((nonce) => nonce + 1);
      showSuccessToast("Changes committed");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to commit changes"));
    } finally {
      setIsPreparingVersionAction(false);
    }
  }, [
    activeCanvasVersionId,
    canvasId,
    commitCanvasStagingMutation,
    consoleMutationGenerationRef,
    draftCanvasSpecsRef,
    ensureVersionActionDraftReady,
    hasEditableVersion,
    queryClient,
    setDraftCanvasSpec,
    setIsPreparingVersionAction,
    setStagingResetNonce,
  ]);

  const handleResetStaging = useCallback(async () => {
    if (!hasEditableVersion || !activeCanvasVersionId) {
      return;
    }

    setIsPreparingVersionAction(true);
    try {
      consoleMutationGenerationRef.current += 1;
      await discardCanvasStagingMutation.mutateAsync(undefined);
      await queryClient.refetchQueries({ queryKey: canvasKeys.repository(canvasId!) });
      draftCanvasSpecsRef.current.delete(activeCanvasVersionId);
      setDraftCanvasSpec(null);
      setStagingResetNonce((nonce) => nonce + 1);
      showSuccessToast("Reverted to last commit");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to reset staged changes"));
    } finally {
      setIsPreparingVersionAction(false);
    }
  }, [
    activeCanvasVersionId,
    canvasId,
    consoleMutationGenerationRef,
    discardCanvasStagingMutation,
    draftCanvasSpecsRef,
    hasEditableVersion,
    queryClient,
    setDraftCanvasSpec,
    setIsPreparingVersionAction,
    setStagingResetNonce,
  ]);

  return { handleCommitStaging, handleResetStaging };
}
