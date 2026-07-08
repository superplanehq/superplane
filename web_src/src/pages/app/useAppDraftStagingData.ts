import type { CanvasesCanvas } from "@/api-client";
import { useCommitCanvasStaging, useDiscardCanvasStaging, type useCanvasStaging } from "@/hooks/useCanvasData";

import { useCanvasConsoleVersionDiff } from "./useCanvasConsoleVersionDiff";
import type { useCommittedDraftBaselines } from "./useCommittedDraftBaselines";
import { useDraftStagingIndicators } from "./useDraftStagingIndicators";

type CommittedBaselines = ReturnType<typeof useCommittedDraftBaselines>;
type CanvasStagingQuery = ReturnType<typeof useCanvasStaging>;

type UseAppDraftStagingDataOptions = {
  canvasId: string;
  activeCanvasVersionId: string;
  liveCanvasVersionId: string | undefined;
  isEditing: boolean;
  hasEditableVersion: boolean;
  stagingResetNonce: number;
  draftSpecToRender: CanvasesCanvas["spec"] | null | undefined;
  canvas: CanvasesCanvas | null | undefined;
  getConsoleMutationGeneration: () => number;
  committedBaselines: CommittedBaselines;
  editBootstrapReady: boolean;
  canvasStagingQuery: CanvasStagingQuery;
};

export function useAppDraftStagingData({
  canvasId,
  activeCanvasVersionId,
  liveCanvasVersionId,
  isEditing,
  hasEditableVersion,
  stagingResetNonce,
  draftSpecToRender,
  canvas,
  getConsoleMutationGeneration,
  committedBaselines,
  editBootstrapReady,
  canvasStagingQuery,
}: UseAppDraftStagingDataOptions) {
  const commitCanvasStagingMutation = useCommitCanvasStaging(canvasId);
  const discardCanvasStagingMutation = useDiscardCanvasStaging(canvasId);

  const canvasConsoleVersionDiff = useCanvasConsoleVersionDiff({
    canvasId,
    versionIds: {
      active: activeCanvasVersionId,
      draft: activeCanvasVersionId,
      live: liveCanvasVersionId,
    },
    hasDraftGraphDiffVersusLive: false,
    suppressUnpublishedDraftDiscard: true,
    enabled: true,
    stageActiveConsole: hasEditableVersion,
    getConsoleMutationGeneration,
  });
  const { consoleQuery, updateConsoleMutation, draftChangeIndicators } = canvasConsoleVersionDiff;

  const effectiveCanvasSpec = editBootstrapReady ? (draftSpecToRender ?? canvas?.spec ?? undefined) : undefined;
  const stagingIndicators = useDraftStagingIndicators({
    isEditing,
    editBootstrapReady,
    canvasId,
    activeCanvasVersionId,
    stagingResetNonce,
    canvasStagingQuery,
    committedBaselines,
    effectiveCanvasSpec,
    consoleQueryData: consoleQuery.data,
    draftChangeIndicators,
  });

  return {
    stagingBaselinesReady: committedBaselines.ready,
    stagingStale: !!canvasStagingQuery.data?.stale,
    commitCanvasStagingMutation,
    discardCanvasStagingMutation,
    consoleQuery,
    updateConsoleMutation,
    draftChangeIndicators: {
      hasUnpublishedDraftChanges: false,
      hasUnpublishedCanvasDraftChanges: false,
      hasUnpublishedConsoleDraftChanges: false,
    },
    hasDraftDiffVersusLive: false,
    canvasConsoleVersionDiff,
    ...stagingIndicators,
  };
}
