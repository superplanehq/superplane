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
  registerIgnoredCanvasVersionUpdatedEcho?: (versionId?: string) => () => void;
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
    registerIgnoredCanvasVersionUpdatedEcho,
  } = options;
  const queryClient = useQueryClient();
  const [commitStagingPending, setCommitStagingPending] = useState(false);
  const [resetStagingPending, setResetStagingPending] = useState(false);

  const handleCommitStaging = useCallback(
    async (commitMessage: string) => {
      if (!hasEditableVersion || !activeCanvasVersionId) {
        return;
      }

      const trimmedMessage = commitMessage.trim();
      if (!trimmedMessage) {
        showErrorToast("Commit message is required");
        return;
      }

      try {
        await runStagingAction(setCommitStagingPending, setIsPreparingVersionAction, async () => {
          const committed = await executeCommitStaging({
            organizationId,
            canvasId,
            activeCanvasVersionId,
            commitMessage: trimmedMessage,
            queryClient,
            commitCanvasStagingMutation,
            consoleMutationGenerationRef,
            draftCanvasSpecsRef,
            setDraftCanvasSpec,
            setStagingResetNonce,
            ensureVersionActionDraftReady,
            flushRepositoryFileStaging,
            registerIgnoredCanvasVersionUpdatedEcho,
            onCommittedVersionId,
          });
          if (committed) {
            showSuccessToast("Changes committed");
          }
        });
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to commit changes"));
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
      hasEditableVersion,
      onCommittedVersionId,
      organizationId,
      queryClient,
      registerIgnoredCanvasVersionUpdatedEcho,
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
