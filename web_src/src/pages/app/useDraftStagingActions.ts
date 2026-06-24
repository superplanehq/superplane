import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState, type Dispatch, type MutableRefObject, type SetStateAction } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { canvasKeys } from "@/hooks/useCanvasData";

import { syncCommittedCanvasDraftState } from "./lib/sync-committed-canvas-draft";

type CommitMutation = { mutateAsync: () => Promise<unknown> };
type DiscardMutation = { mutateAsync: (input: undefined) => Promise<unknown> };

type UseDraftStagingActionsOptions = {
  organizationId?: string;
  canvasId?: string;
  activeCanvasVersionId: string;
  hasEditableVersion: boolean;
  ensureVersionActionDraftReady: (errorMessage: string) => Promise<boolean>;
  commitCanvasStagingMutation: CommitMutation;
  discardCanvasStagingMutation: DiscardMutation;
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasesCanvas["spec"] | null>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<CanvasesCanvas["spec"] | null>>;
  setActiveCanvasVersion?: Dispatch<SetStateAction<CanvasesCanvasVersion | null>>;
  setStagingResetNonce: Dispatch<SetStateAction<number>>;
  consoleMutationGenerationRef: MutableRefObject<number>;
  setIsPreparingVersionAction: Dispatch<SetStateAction<boolean>>;
  flushRepositoryFileStaging?: () => Promise<void>;
  cancelPendingCanvasSaves?: () => void;
  onCanvasDraftRestoredToCommitted?: (version: CanvasesCanvasVersion) => void;
  // Recovers from a deleted draft when a mutation fails; returns whether it
  // handled the error (only true when the draft is confirmed gone).
  recoverIfDraftMissing?: (error: unknown, versionId: string) => Promise<boolean>;
};

async function restoreCommittedCanvasDraftState({
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
  queryClient: ReturnType<typeof useQueryClient>;
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasesCanvas["spec"] | null>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<CanvasesCanvas["spec"] | null>>;
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

export function useDraftStagingActions({
  organizationId,
  canvasId,
  activeCanvasVersionId,
  hasEditableVersion,
  ensureVersionActionDraftReady,
  commitCanvasStagingMutation,
  discardCanvasStagingMutation,
  draftCanvasSpecsRef,
  setDraftCanvasSpec,
  setActiveCanvasVersion,
  setStagingResetNonce,
  consoleMutationGenerationRef,
  setIsPreparingVersionAction,
  flushRepositoryFileStaging,
  cancelPendingCanvasSaves,
  onCanvasDraftRestoredToCommitted,
  recoverIfDraftMissing,
}: UseDraftStagingActionsOptions) {
  const queryClient = useQueryClient();
  const [resetStagingPending, setResetStagingPending] = useState(false);

  const handleCommitStaging = useCallback(async () => {
    if (!hasEditableVersion || !activeCanvasVersionId) {
      return;
    }

    try {
      await flushRepositoryFileStaging?.();
      const isReady = await ensureVersionActionDraftReady("Unable to prepare staged changes for commit");
      if (!isReady) {
        return;
      }

      setIsPreparingVersionAction(true);
      consoleMutationGenerationRef.current += 1;
      await commitCanvasStagingMutation.mutateAsync();
      await queryClient.invalidateQueries({ queryKey: canvasKeys.repository(canvasId!) });
      draftCanvasSpecsRef.current.delete(activeCanvasVersionId);
      setDraftCanvasSpec(null);
      setStagingResetNonce((nonce) => nonce + 1);
      showSuccessToast("Changes committed");
    } catch (error) {
      if (await recoverIfDraftMissing?.(error, activeCanvasVersionId)) {
        return;
      }
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
    flushRepositoryFileStaging,
    hasEditableVersion,
    recoverIfDraftMissing,
    queryClient,
    setDraftCanvasSpec,
    setIsPreparingVersionAction,
    setStagingResetNonce,
  ]);

  const handleResetStaging = useCallback(async () => {
    if (!hasEditableVersion || !activeCanvasVersionId) {
      return;
    }

    setResetStagingPending(true);
    setIsPreparingVersionAction(true);
    try {
      cancelPendingCanvasSaves?.();
      consoleMutationGenerationRef.current += 1;
      await discardCanvasStagingMutation.mutateAsync(undefined);
      await restoreCommittedCanvasDraftState({
        organizationId,
        canvasId,
        activeCanvasVersionId,
        queryClient,
        draftCanvasSpecsRef,
        setDraftCanvasSpec,
        setActiveCanvasVersion,
        onCanvasDraftRestoredToCommitted,
      });
      await queryClient.refetchQueries({ queryKey: canvasKeys.repository(canvasId!) });
      setStagingResetNonce((nonce) => nonce + 1);
      showSuccessToast("Reverted to last commit");
    } catch (error) {
      if (await recoverIfDraftMissing?.(error, activeCanvasVersionId)) {
        return;
      }
      showErrorToast(getApiErrorMessage(error, "Failed to reset staged changes"));
    } finally {
      setResetStagingPending(false);
      setIsPreparingVersionAction(false);
    }
  }, [
    activeCanvasVersionId,
    cancelPendingCanvasSaves,
    canvasId,
    consoleMutationGenerationRef,
    discardCanvasStagingMutation,
    draftCanvasSpecsRef,
    hasEditableVersion,
    onCanvasDraftRestoredToCommitted,
    recoverIfDraftMissing,
    organizationId,
    queryClient,
    setActiveCanvasVersion,
    setDraftCanvasSpec,
    setIsPreparingVersionAction,
    setStagingResetNonce,
  ]);

  return { handleCommitStaging, handleResetStaging, resetStagingPending };
}
