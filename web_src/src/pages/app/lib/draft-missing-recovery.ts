import type { QueryClient } from "@tanstack/react-query";

import { ensureDraftVersionExists } from "@/hooks/useCanvasData";

import { isNotFoundError } from "../workflowPageHelpers";

export async function recoverIfDraftMissing({
  error,
  versionId,
  organizationId,
  canvasId,
  queryClient,
  recoverFromMissingDraft,
}: {
  error: unknown;
  versionId: string;
  organizationId?: string;
  canvasId?: string;
  queryClient: QueryClient;
  recoverFromMissingDraft: (versionId: string, message?: string) => Promise<void>;
}): Promise<boolean> {
  if (!isNotFoundError(error) || !organizationId || !canvasId || !versionId) {
    return false;
  }

  const draftExists = await ensureDraftVersionExists(queryClient, organizationId, canvasId, versionId).catch(
    () => true,
  );
  if (draftExists) {
    return false;
  }

  await recoverFromMissingDraft(versionId);
  return true;
}
