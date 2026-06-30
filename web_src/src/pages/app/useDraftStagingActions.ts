import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState, type Dispatch, type MutableRefObject, type SetStateAction } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";

import { executeCommitStaging } from "./lib/commit-staging-flow";
import { executeResetStaging } from "./lib/reset-staging-flow";

type CommitMutation = {
  mutateAsync: (input?: {
    commitMessage?: string;
    newBranchName?: string;
  }) => Promise<{ version?: CanvasesCanvasVersion } | undefined>;
};
type CommitStagingInput = { commitMessage: string; newBranchName?: string };
type DiscardMutation = { mutateAsync: (input: undefined) => Promise<unknown> };

async function runDraftStagingAction(
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
  recoverIfDraftMissing?: (error: unknown, versionId: string) => Promise<boolean>;
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
    recoverIfDraftMissing: recoverIfDraftMissingOption,
    registerIgnoredCanvasVersionUpdatedEcho,
  } = options;
  const queryClient = useQueryClient();
  const [commitStagingPending, setCommitStagingPending] = useState(false);
  const [resetStagingPending, setResetStagingPending] = useState(false);

  const handleCommitStaging = useCallback(
    async (input: CommitStagingInput): Promise<CanvasesCanvasVersion | null> => {
      if (!hasEditableVersion || !activeCanvasVersionId) {
        return null;
      }

      try {
        let committedVersion: CanvasesCanvasVersion | null = null;
        await runDraftStagingAction(setCommitStagingPending, setIsPreparingVersionAction, async () => {
          committedVersion = await executeCommitStaging({
            organizationId,
            canvasId,
            activeCanvasVersionId,
            queryClient,
            commitCanvasStagingMutation,
            commitMessage: input.commitMessage,
            newBranchName: input.newBranchName,
            consoleMutationGenerationRef,
            draftCanvasSpecsRef,
            setDraftCanvasSpec,
            setStagingResetNonce,
            ensureVersionActionDraftReady,
            flushRepositoryFileStaging,
            registerIgnoredCanvasVersionUpdatedEcho,
          });
          if (committedVersion) {
            showSuccessToast("Changes committed");
          }
        });
        return committedVersion;
      } catch (error) {
        if (!(await recoverIfDraftMissingOption?.(error, activeCanvasVersionId))) {
          showErrorToast(getApiErrorMessage(error, "Failed to commit changes"));
        }
        return null;
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
      organizationId,
      queryClient,
      recoverIfDraftMissingOption,
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
      await runDraftStagingAction(setResetStagingPending, setIsPreparingVersionAction, async () => {
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
      if (!(await recoverIfDraftMissingOption?.(error, activeCanvasVersionId))) {
        showErrorToast(getApiErrorMessage(error, "Failed to reset staged changes"));
      }
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
    recoverIfDraftMissingOption,
    setActiveCanvasVersion,
    setDraftCanvasSpec,
    setIsPreparingVersionAction,
    setStagingResetNonce,
  ]);

  return { handleCommitStaging, handleResetStaging, commitStagingPending, resetStagingPending };
}
