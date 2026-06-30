import type { QueryClient } from "@tanstack/react-query";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

import { refreshCachesAfterCommit } from "./sync-committed-canvas-draft";

type CommitMutation = {
  mutateAsync: (input?: {
    commitMessage?: string;
    newBranchName?: string;
  }) => Promise<
    { version?: CanvasesCanvasVersion; stagingSummary?: { hasStaging?: boolean; stagedPaths?: string[] } } | undefined
  >;
};
type DraftSpec = CanvasesCanvas["spec"] | null;

export async function executeCommitStaging({
  organizationId,
  canvasId,
  activeCanvasVersionId,
  queryClient,
  commitCanvasStagingMutation,
  commitMessage,
  newBranchName,
  consoleMutationGenerationRef,
  draftCanvasSpecsRef,
  setDraftCanvasSpec,
  setStagingResetNonce,
  ensureVersionActionDraftReady,
  flushRepositoryFileStaging,
  registerIgnoredCanvasVersionUpdatedEcho,
}: {
  organizationId?: string;
  canvasId?: string;
  activeCanvasVersionId: string;
  queryClient: QueryClient;
  commitCanvasStagingMutation: CommitMutation;
  commitMessage: string;
  newBranchName?: string;
  consoleMutationGenerationRef: MutableRefObject<number>;
  draftCanvasSpecsRef: MutableRefObject<Map<string, DraftSpec>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<DraftSpec>>;
  setStagingResetNonce: Dispatch<SetStateAction<number>>;
  ensureVersionActionDraftReady: (errorMessage: string) => Promise<boolean>;
  flushRepositoryFileStaging?: () => Promise<void>;
  registerIgnoredCanvasVersionUpdatedEcho?: (versionId?: string) => () => void;
}): Promise<CanvasesCanvasVersion | null> {
  await flushRepositoryFileStaging?.();
  const isReady = await ensureVersionActionDraftReady("Unable to prepare staged changes for commit");
  if (!isReady) {
    return null;
  }

  consoleMutationGenerationRef.current += 1;
  const releaseCanvasVersionUpdatedEcho = registerIgnoredCanvasVersionUpdatedEcho?.(activeCanvasVersionId);
  let committedVersion: CanvasesCanvasVersion | undefined;
  try {
    const response = await commitCanvasStagingMutation.mutateAsync({ commitMessage, newBranchName });
    committedVersion = response?.version;
  } catch (error) {
    releaseCanvasVersionUpdatedEcho?.();
    throw error;
  }

  const committedVersionId = committedVersion?.metadata?.id ?? activeCanvasVersionId;

  // Commit already succeeded on the server; cache refresh and local cleanup must not fail the action.
  if (organizationId && canvasId && committedVersionId) {
    await refreshCachesAfterCommit({
      queryClient,
      organizationId,
      canvasId,
      versionId: committedVersionId,
    });
  }

  if (canvasId) {
    await queryClient.invalidateQueries({ queryKey: canvasKeys.repository(canvasId) });
  }
  draftCanvasSpecsRef.current.delete(activeCanvasVersionId);
  if (committedVersionId !== activeCanvasVersionId) {
    draftCanvasSpecsRef.current.delete(committedVersionId);
  }
  setDraftCanvasSpec(null);
  setStagingResetNonce((nonce) => nonce + 1);
  return committedVersion ?? null;
}
