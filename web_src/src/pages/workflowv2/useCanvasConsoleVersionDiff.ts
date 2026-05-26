import { useMemo } from "react";

import { useCanvasConsole, useUpdateCanvasConsole } from "@/hooks/useCanvasData";

import { hasDraftVersusLiveConsoleDiff } from "./draftConsoleDiff";

type UseCanvasConsoleVersionDiffArgs = {
  canvasId: string;
  activeCanvasVersionId: string;
  liveCanvasVersionId: string | undefined;
  hasDraftGraphDiffVersusLive: boolean;
  enabled: boolean;
};

export function useCanvasConsoleVersionDiff({
  canvasId,
  activeCanvasVersionId,
  liveCanvasVersionId,
  hasDraftGraphDiffVersusLive,
  enabled,
}: UseCanvasConsoleVersionDiffArgs) {
  const dashboardQuery = useCanvasConsole(canvasId, activeCanvasVersionId || undefined, enabled);
  const liveDashboardQuery = useCanvasConsole(
    canvasId,
    liveCanvasVersionId || undefined,
    enabled && !!liveCanvasVersionId,
  );
  const hasDraftConsoleDiffVersusLive = useMemo(
    () => hasDraftVersusLiveConsoleDiff(liveDashboardQuery.data, dashboardQuery.data),
    [liveDashboardQuery.data, dashboardQuery.data],
  );
  const hasDraftDiffVersusLive = hasDraftGraphDiffVersusLive || hasDraftConsoleDiffVersusLive;
  const updateDashboardMutation = useUpdateCanvasConsole(canvasId, activeCanvasVersionId || undefined);

  return {
    dashboardQuery,
    updateDashboardMutation,
    hasDraftDiffVersusLive,
  };
}

export type CanvasConsoleVersionDiffResult = ReturnType<typeof useCanvasConsoleVersionDiff>;
