import type { QueryClient } from "@tanstack/react-query";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

import { restoreCommittedCanvasDraftState } from "./sync-committed-canvas-draft";

type DiscardMutation = { mutateAsync: (input: undefined) => Promise<unknown> };
type DraftSpec = CanvasesCanvas["spec"] | null;

export async function executeResetStaging({
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
}: {
  organizationId?: string;
  canvasId?: string;
  activeCanvasVersionId: string;
  queryClient: QueryClient;
  discardCanvasStagingMutation: DiscardMutation;
  consoleMutationGenerationRef: MutableRefObject<number>;
  draftCanvasSpecsRef: MutableRefObject<Map<string, DraftSpec>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<DraftSpec>>;
  setActiveCanvasVersion?: Dispatch<SetStateAction<CanvasesCanvasVersion | null>>;
  setStagingResetNonce: Dispatch<SetStateAction<number>>;
  cancelPendingCanvasSaves?: () => void;
  onCanvasDraftRestoredToCommitted?: (version: CanvasesCanvasVersion) => void;
}) {
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
}
