import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState, type Dispatch, type MutableRefObject, type SetStateAction } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";

import { executeCommitStaging } from "./lib/commit-staging-flow";
import { executeResetStaging } from "./lib/reset-staging-flow";

type CommitMutation = {
  mutateAsync: (commitMessage: string) => Promise<{ version?: CanvasesCanvasVersion }>;
};
type DiscardMutation = { mutateAsync: (input: undefined) => Promise<unknown> };

async function runStagingAction(
  setActionPending: Dispatch<SetStateAction<boolean>>,
  setIsPreparingVersionAction: Dispatch<SetStateAction<boolean>>,
  action: () => Promise<void>,
): Promise<void> {
  setActionPending(true);
  setIsPreparingVersionAction(true);
  try {
    await action();
  } finally {
    setActionPending(false);
    setIsPreparingVersionAction(false);
  }
}

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
  onCommittedVersionId?: (versionId: string) => void;
  registerIgnoredCanvasUpdatedEcho?: () => () => void;
};

export function useDraftStagingActions(options: UseDraftStagingActionsOptions) {
  const {
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
    onCommittedVersionId,
    registerIgnoredCanvasUpdatedEcho,
  } = options;
  const queryClient = useQueryClient();
  const [commitStagingPending, setCommitStagingPending] = useState(false);
  const [resetStagingPending, setResetStagingPending] = useState(false);

  const handleCommitStaging = useCallback(
    async (commitMessage: string, options?: { versionId?: string }): Promise<boolean> => {
      // Prefer explicit versionId (Factory Configure Save) — React state for
      // activeCanvasVersionId can lag one frame behind the sync ref write.
      const versionId = options?.versionId || activeCanvasVersionId;
      if (!versionId) {
        showErrorToast("Edit session is not ready to commit");
        return false;
      }

      const trimmedMessage = commitMessage.trim();
      if (!trimmedMessage) {
        showErrorToast("Commit message is required");
        return false;
      }

      try {
        let committed = false;
        await runStagingAction(setCommitStagingPending, setIsPreparingVersionAction, async () => {
          committed = await executeCommitStaging({
            organizationId,
            canvasId,
            activeCanvasVersionId: versionId,
            commitMessage: trimmedMessage,
            queryClient,
            commitCanvasStagingMutation,
            consoleMutationGenerationRef,
            draftCanvasSpecsRef,
            setDraftCanvasSpec,
            setStagingResetNonce,
            ensureVersionActionDraftReady,
            flushRepositoryFileStaging,
            registerIgnoredCanvasUpdatedEcho,
            onCommittedVersionId,
          });
          if (committed) {
            showSuccessToast("Changes committed");
          }
        });
        return committed;
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to commit changes"));
        return false;
      }
    },
    [
      activeCanvasVersionId,
      canvasId,
      commitCanvasStagingMutation,
      consoleMutationGenerationRef,
      draftCanvasSpecsRef,
      ensureVersionActionDraftReady,
      flushRepositoryFileStaging,
      onCommittedVersionId,
      organizationId,
      queryClient,
      registerIgnoredCanvasUpdatedEcho,
      setDraftCanvasSpec,
      setIsPreparingVersionAction,
      setStagingResetNonce,
    ],
  );

  const handleResetStaging = useCallback(async () => {
    if (!hasEditableVersion || !activeCanvasVersionId) {
      return;
    }

    try {
      await runStagingAction(setResetStagingPending, setIsPreparingVersionAction, async () => {
        await executeResetStaging({
          organizationId,
          canvasId,
          activeCanvasVersionId,
          queryClient,
          discardCanvasStagingMutation,
          consoleMutationGenerationRef,
          draftCanvasSpecsRef,
          setDraftCanvasSpec,
          setActiveCanvasVersion,
          setStagingResetNonce,
          cancelPendingCanvasSaves,
          onCanvasDraftRestoredToCommitted,
        });
        showSuccessToast("Reverted to last commit");
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to reset staged changes"));
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
    organizationId,
    queryClient,
    setActiveCanvasVersion,
    setDraftCanvasSpec,
    setIsPreparingVersionAction,
    setStagingResetNonce,
  ]);

  return { handleCommitStaging, handleResetStaging, commitStagingPending, resetStagingPending };
}
