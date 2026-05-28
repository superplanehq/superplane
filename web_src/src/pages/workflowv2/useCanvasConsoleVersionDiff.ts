import { useMemo } from "react";

import { useCanvasConsole, useUpdateCanvasConsole } from "@/hooks/useCanvasData";

import { hasDraftVersusLiveConsoleDiff } from "./draftConsoleDiff";
import { getDraftChangeIndicators } from "./lib/version-action-state";

type UseCanvasConsoleVersionDiffArgs = {
  canvasId: string;
  versionIds: {
    active: string;
    draft: string | undefined;
    live: string | undefined;
  };
  hasDraftGraphDiffVersusLive: boolean;
  suppressUnpublishedDraftDiscard: boolean;
  enabled: boolean;
  registerIgnoredCanvasVersionUpdatedEcho?: (savingVersionId?: string) => () => void;
};

export function useCanvasConsoleVersionDiff({
  canvasId,
  versionIds,
  hasDraftGraphDiffVersusLive,
  suppressUnpublishedDraftDiscard,
  enabled,
  registerIgnoredCanvasVersionUpdatedEcho,
}: UseCanvasConsoleVersionDiffArgs) {
  const dashboardQuery = useCanvasConsole(canvasId, versionIds.active || undefined, enabled);
  const draftDiffVersionId = versionIds.active || versionIds.draft;
  const draftDashboardQuery = useCanvasConsole(
    canvasId,
    draftDiffVersionId || undefined,
    enabled && !!draftDiffVersionId,
  );
  const liveDashboardQuery = useCanvasConsole(canvasId, versionIds.live || undefined, enabled && !!versionIds.live);
  const hasDraftConsoleDiffVersusLive = useMemo(
    () => !!draftDiffVersionId && hasDraftVersusLiveConsoleDiff(liveDashboardQuery.data, draftDashboardQuery.data),
    [draftDiffVersionId, liveDashboardQuery.data, draftDashboardQuery.data],
  );
  const hasDraftDiffVersusLive = hasDraftGraphDiffVersusLive || hasDraftConsoleDiffVersusLive;
  const draftChangeIndicators = getDraftChangeIndicators({
    suppressUnpublishedDraftDiscard,
    hasLatestDraftVersion: !!versionIds.draft,
    hasDraftGraphDiffVersusLive,
    hasDraftConsoleDiffVersusLive,
    hasDraftDiffVersusLive,
  });
  const updateDashboardMutation = useUpdateCanvasConsole(canvasId, versionIds.active || undefined, {
    registerIgnoredCanvasVersionUpdatedEcho,
  });

  return {
    dashboardQuery,
    updateDashboardMutation,
    draftChangeIndicators,
    hasDraftDiffVersusLive,
  };
}

export type CanvasConsoleVersionDiffResult = ReturnType<typeof useCanvasConsoleVersionDiff>;
