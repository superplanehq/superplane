import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { useCanvasVersionStaging, useCommitCanvasStaging, useDiscardCanvasStaging } from "@/hooks/useCanvasData";

import { useCanvasConsoleVersionDiff } from "./useCanvasConsoleVersionDiff";
import { useCommittedDraftBaselines } from "./useCommittedDraftBaselines";
import { useDraftStagingIndicators } from "./useDraftStagingIndicators";
import { useDraftVersionGraphDiff } from "./useDraftVersionGraphDiff";

type UseAppDraftStagingDataOptions = {
  organizationId: string;
  canvasId: string;
  activeCanvasVersionId: string;
  liveCanvasVersionId: string | undefined;
  liveCanvasVersion: CanvasesCanvasVersion | undefined;
  latestDraftVersion: CanvasesCanvasVersion | undefined;
  isEditing: boolean;
  hasEditableVersion: boolean;
  stagingResetNonce: number;
  draftSpecToRender: CanvasesCanvas["spec"] | null | undefined;
  canvas: CanvasesCanvas | null | undefined;
  draftVersionsFromBranches: CanvasesCanvasVersion[];
  selectedCanvasVersion: CanvasesCanvasVersion | null;
  draftBranches: CanvasesCanvasVersion[];
  suppressUnpublishedDraftDiscard: boolean;
  registerIgnoredCanvasVersionUpdatedEcho: (savingVersionId?: string) => () => void;
  getConsoleMutationGeneration: () => number;
};

export function useAppDraftStagingData({
  organizationId,
  canvasId,
  activeCanvasVersionId,
  liveCanvasVersionId,
  liveCanvasVersion,
  latestDraftVersion,
  isEditing,
  hasEditableVersion,
  stagingResetNonce,
  draftSpecToRender,
  canvas,
  draftVersionsFromBranches,
  selectedCanvasVersion,
  draftBranches,
  suppressUnpublishedDraftDiscard,
  registerIgnoredCanvasVersionUpdatedEcho,
  getConsoleMutationGeneration,
}: UseAppDraftStagingDataOptions) {
  const canvasVersionStagingQuery = useCanvasVersionStaging(
    canvasId,
    activeCanvasVersionId || undefined,
    hasEditableVersion,
  );
  const committedBaselines = useCommittedDraftBaselines({
    canvasId,
    versionId: activeCanvasVersionId || undefined,
    enabled: isEditing,
    stagingResetNonce,
  });
  const { draftVersionForGraphDiff, hasDraftGraphDiffVersusLive, liveVersionForGraphDiff } = useDraftVersionGraphDiff({
    organizationId,
    canvasId,
    isEditing,
    activeCanvasVersionId,
    liveCanvasVersionId: liveCanvasVersionId || "",
    liveCanvasVersion,
    draftVersionsFromBranches,
    selectedCanvasVersion,
    latestDraftVersion,
    committedBaselines,
  });
  const commitCanvasStagingMutation = useCommitCanvasStaging(canvasId, activeCanvasVersionId);
  const discardCanvasStagingMutation = useDiscardCanvasStaging(canvasId, activeCanvasVersionId);

  const canvasConsoleVersionDiff = useCanvasConsoleVersionDiff({
    canvasId,
    versionIds: {
      active: activeCanvasVersionId,
      draft: activeCanvasVersionId || latestDraftVersion?.metadata?.id,
      live: liveCanvasVersionId,
    },
    hasDraftGraphDiffVersusLive,
    suppressUnpublishedDraftDiscard,
    enabled: true,
    stageActiveConsole: hasEditableVersion,
    registerIgnoredCanvasVersionUpdatedEcho,
    getConsoleMutationGeneration,
  });
  const { consoleQuery, updateConsoleMutation, draftChangeIndicators, hasDraftDiffVersusLive } =
    canvasConsoleVersionDiff;

  const effectiveCanvasSpec = draftSpecToRender ?? canvas?.spec ?? undefined;
  const stagingIndicators = useDraftStagingIndicators({
    isEditing,
    canvasId,
    activeCanvasVersionId,
    liveCanvasVersionId: liveCanvasVersionId || "",
    stagingResetNonce,
    canvasVersionStagingQuery,
    committedBaselines,
    effectiveCanvasSpec,
    consoleQueryData: consoleQuery.data,
    draftVersionsFromBranches,
    draftVersionForGraphDiff,
    liveVersionForGraphDiff,
    draftBranches,
    hasDraftDiffVersusLive,
    draftChangeIndicators,
  });

  return {
    hasDraftGraphDiffVersusLive,
    commitCanvasStagingMutation,
    discardCanvasStagingMutation,
    consoleQuery,
    updateConsoleMutation,
    draftChangeIndicators,
    hasDraftDiffVersusLive,
    canvasConsoleVersionDiff,
    ...stagingIndicators,
  };
}
