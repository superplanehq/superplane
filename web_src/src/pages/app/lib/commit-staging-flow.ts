import type { QueryClient } from "@tanstack/react-query";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";

import type { CanvasesCanvas } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

import { refreshCachesAfterCommit } from "./sync-committed-canvas-draft";

type CommitMutation = { mutateAsync: (commitMessage: string) => Promise<{ version?: { metadata?: { id?: string } } }> };
type DraftSpec = CanvasesCanvas["spec"] | null;

export async function executeCommitStaging({
  organizationId,
  canvasId,
  activeCanvasVersionId,
  commitMessage,
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
}: {
  organizationId?: string;
  canvasId?: string;
  activeCanvasVersionId: string;
  commitMessage: string;
  queryClient: QueryClient;
  commitCanvasStagingMutation: CommitMutation;
  consoleMutationGenerationRef: MutableRefObject<number>;
  draftCanvasSpecsRef: MutableRefObject<Map<string, DraftSpec>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<DraftSpec>>;
  setStagingResetNonce: Dispatch<SetStateAction<number>>;
  ensureVersionActionDraftReady: (errorMessage: string) => Promise<boolean>;
  flushRepositoryFileStaging?: () => Promise<void>;
  registerIgnoredCanvasVersionUpdatedEcho?: (versionId?: string) => () => void;
  onCommittedVersionId?: (versionId: string) => void;
}): Promise<boolean> {
  await flushRepositoryFileStaging?.();
  const isReady = await ensureVersionActionDraftReady("Unable to prepare staged changes for commit");
  if (!isReady) {
    return false;
  }

  consoleMutationGenerationRef.current += 1;
  const releaseCanvasVersionUpdatedEcho = registerIgnoredCanvasVersionUpdatedEcho?.(activeCanvasVersionId);
  let committedVersionId = activeCanvasVersionId;
  try {
    const response = await commitCanvasStagingMutation.mutateAsync(commitMessage);
    committedVersionId = response?.version?.metadata?.id || activeCanvasVersionId;
  } catch (error) {
    releaseCanvasVersionUpdatedEcho?.();
    throw error;
  }

  if (organizationId && canvasId && committedVersionId) {
    await refreshCachesAfterCommit({
      queryClient,
      organizationId,
      canvasId,
      versionId: committedVersionId,
    });
    queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
    onCommittedVersionId?.(committedVersionId);
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
  return true;
}
