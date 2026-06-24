import type { QueryClient } from "@tanstack/react-query";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";

import type { CanvasesCanvas } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

import { refreshCachesAfterCommit } from "./sync-committed-canvas-draft";

type CommitMutation = { mutateAsync: () => Promise<unknown> };
type DraftSpec = CanvasesCanvas["spec"] | null;

export async function executeCommitStaging({
  organizationId,
  canvasId,
  activeCanvasVersionId,
  queryClient,
  commitCanvasStagingMutation,
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
  consoleMutationGenerationRef: MutableRefObject<number>;
  draftCanvasSpecsRef: MutableRefObject<Map<string, DraftSpec>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<DraftSpec>>;
  setStagingResetNonce: Dispatch<SetStateAction<number>>;
  ensureVersionActionDraftReady: (errorMessage: string) => Promise<boolean>;
  flushRepositoryFileStaging?: () => Promise<void>;
  registerIgnoredCanvasVersionUpdatedEcho?: (versionId?: string) => () => void;
}): Promise<boolean> {
  await flushRepositoryFileStaging?.();
  const isReady = await ensureVersionActionDraftReady("Unable to prepare staged changes for commit");
  if (!isReady) {
    return false;
  }

  consoleMutationGenerationRef.current += 1;
  const releaseCanvasVersionUpdatedEcho = registerIgnoredCanvasVersionUpdatedEcho?.(activeCanvasVersionId);
  try {
    await commitCanvasStagingMutation.mutateAsync();
  } catch (error) {
    releaseCanvasVersionUpdatedEcho?.();
    throw error;
  }

  // Commit already succeeded on the server; cache refresh and local cleanup must not fail the action.
  if (organizationId && canvasId) {
    await refreshCachesAfterCommit({
      queryClient,
      organizationId,
      canvasId,
      versionId: activeCanvasVersionId,
    });
  }

  if (canvasId) {
    await queryClient.invalidateQueries({ queryKey: canvasKeys.repository(canvasId) });
  }
  draftCanvasSpecsRef.current.delete(activeCanvasVersionId);
  setDraftCanvasSpec(null);
  setStagingResetNonce((nonce) => nonce + 1);
  return true;
}
