import type { CanvasesCanvas } from "@/api-client";
import { useCanvasStaging, useCommitCanvasStaging, useDiscardCanvasStaging } from "@/hooks/useCanvasData";

import { useCanvasConsoleVersionDiff } from "./useCanvasConsoleVersionDiff";
import { useCommittedDraftBaselines } from "./useCommittedDraftBaselines";
import { useDraftStagingIndicators } from "./useDraftStagingIndicators";

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
}: UseAppDraftStagingDataOptions) {
  const canvasStagingQuery = useCanvasStaging(canvasId, hasEditableVersion);
  const committedBaselines = useCommittedDraftBaselines({
    canvasId,
    versionId: activeCanvasVersionId || undefined,
    enabled: isEditing,
    stagingResetNonce,
  });
  const commitCanvasStagingMutation = useCommitCanvasStaging(canvasId);
  const discardCanvasStagingMutation = useDiscardCanvasStaging(canvasId, activeCanvasVersionId);

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

  const effectiveCanvasSpec = draftSpecToRender ?? canvas?.spec ?? undefined;
  const stagingIndicators = useDraftStagingIndicators({
    isEditing,
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
