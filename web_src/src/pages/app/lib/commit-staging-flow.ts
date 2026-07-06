import type { QueryClient } from "@tanstack/react-query";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";

import type { CanvasesCanvas } from "@/api-client";
import { cancelCanvasVersionQueries, canvasKeys, removeCanvasVersionScopedQueries } from "@/hooks/useCanvasData";

type CommitMutation = { mutateAsync: (commitMessage: string) => Promise<{ version?: { metadata?: { id?: string } } }> };
type DraftSpec = CanvasesCanvas["spec"] | null;

async function invalidatePostCommitCaches(
  queryClient: QueryClient,
  organizationId: string,
  canvasId: string,
): Promise<void> {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId), refetchType: "all" }),
    queryClient.invalidateQueries({ queryKey: canvasKeys.versionList(canvasId), refetchType: "all" }),
    queryClient.invalidateQueries({ queryKey: canvasKeys.versionHistory(canvasId), refetchType: "all" }),
    queryClient.invalidateQueries({ queryKey: canvasKeys.canvasStaging(canvasId), refetchType: "all" }),
    queryClient.invalidateQueries({ queryKey: canvasKeys.console(canvasId, undefined), refetchType: "all" }),
    queryClient.invalidateQueries({ queryKey: canvasKeys.repositoryFiles(canvasId), refetchType: "all" }),
  ]);
}

async function removeStaleVersionQueriesAfterCommit(
  queryClient: QueryClient,
  canvasId: string,
  previousVersionId: string,
  committedVersionId: string,
): Promise<void> {
  if (!previousVersionId || previousVersionId === committedVersionId) {
    return;
  }

  await cancelCanvasVersionQueries(queryClient, canvasId, previousVersionId);
  removeCanvasVersionScopedQueries(queryClient, canvasId, previousVersionId);
}

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
  registerIgnoredCanvasUpdatedEcho,
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
  registerIgnoredCanvasUpdatedEcho?: () => () => void;
  onCommittedVersionId?: (versionId: string) => void;
}): Promise<boolean> {
  await flushRepositoryFileStaging?.();
  const isReady = await ensureVersionActionDraftReady("Unable to prepare staged changes for commit");
  if (!isReady) {
    return false;
  }

  consoleMutationGenerationRef.current += 1;
  const releaseCanvasUpdatedEcho = registerIgnoredCanvasUpdatedEcho?.();
  const previousVersionId = activeCanvasVersionId;
  let committedVersionId = activeCanvasVersionId;
  try {
    const response = await commitCanvasStagingMutation.mutateAsync(commitMessage);
    committedVersionId = response?.version?.metadata?.id || activeCanvasVersionId;
  } catch (error) {
    releaseCanvasUpdatedEcho?.();
    throw error;
  }

  // Leave edit mode before touching caches so version-scoped hooks (console,
  // files, baselines) stop querying the pre-commit live version id.
  onCommittedVersionId?.(committedVersionId);

  draftCanvasSpecsRef.current.delete(activeCanvasVersionId);
  if (committedVersionId !== activeCanvasVersionId) {
    draftCanvasSpecsRef.current.delete(committedVersionId);
  }
  setDraftCanvasSpec(null);

  if (organizationId && canvasId && committedVersionId) {
    await removeStaleVersionQueriesAfterCommit(queryClient, canvasId, previousVersionId, committedVersionId);
    await invalidatePostCommitCaches(queryClient, organizationId, canvasId);
  }

  releaseCanvasUpdatedEcho?.();
  setStagingResetNonce((nonce) => nonce + 1);
  return true;
}
