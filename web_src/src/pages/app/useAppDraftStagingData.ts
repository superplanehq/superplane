import type { CanvasesCanvas } from "@/api-client";
import { useCommitCanvasStaging, useDiscardCanvasStaging, type useCanvasStaging } from "@/hooks/useCanvasData";

import { useCanvasConsoleVersionDiff } from "./useCanvasConsoleVersionDiff";
import type { useCommittedDraftBaselines } from "./useCommittedDraftBaselines";
import { useDraftStagingIndicators } from "./useDraftStagingIndicators";

type CommittedBaselines = ReturnType<typeof useCommittedDraftBaselines>;

type CanvasStagingQuery = ReturnType<typeof useCanvasStaging>;

type UseAppDraftStagingDataOptions = {
  organizationId: string;
  canvasId: string;
  activeCanvasVersionId: string;
  liveCanvasVersionId: string | undefined;
  isEditing: boolean;
  canvasStagingQuery: CanvasStagingQuery;
  stagingResetNonce: number;
  draftSpecToRender: CanvasesCanvas["spec"] | null | undefined;
  canvas: CanvasesCanvas | null | undefined;
  getConsoleMutationGeneration: () => number;
  committedBaselines: CommittedBaselines;
  editBootstrapReady: boolean;
};

export function useAppDraftStagingData({
  organizationId,
  canvasId,
  activeCanvasVersionId,
  liveCanvasVersionId,
  isEditing,
  canvasStagingQuery,
  stagingResetNonce,
  draftSpecToRender,
  canvas,
  getConsoleMutationGeneration,
  committedBaselines,
  editBootstrapReady,
}: UseAppDraftStagingDataOptions) {
  const commitCanvasStagingMutation = useCommitCanvasStaging(canvasId);
  const discardCanvasStagingMutation = useDiscardCanvasStaging(canvasId);
  const hasEditableVersion = isEditing;

  const canvasConsoleVersionDiff = useCanvasConsoleVersionDiff({
    organizationId,
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
    stagingStale: canvasStagingQuery.data?.stagingSummary?.stale === true,
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
