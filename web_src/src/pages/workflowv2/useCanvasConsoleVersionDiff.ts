import { createElement, lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";

import { useCanvasConsole, useUpdateCanvasConsole } from "@/hooks/useCanvasData";

import { getDraftConsoleDiffCounts, hasDraftVersusLiveConsoleDiff } from "./draftConsoleDiff";
import { dashboardToYaml } from "./dashboard/dashboardYaml";
import { getDraftChangeIndicators } from "./lib/version-action-state";
import type { CanvasesConsole } from "@/api-client";

const CanvasYamlDiffModal = lazy(() =>
  import("./CanvasYamlDiffModal").then((module) => ({ default: module.CanvasYamlDiffModal })),
);

function dashboardYamlText(canvasId: string, dashboard?: CanvasesConsole | null): string {
  return dashboardToYaml({
    panels: (dashboard?.panels ?? []).map((panel) => ({
      id: panel.id ?? "",
      type: panel.type ?? "markdown",
      content: (panel.content as Record<string, unknown>) ?? {},
    })),
    layout: (dashboard?.layout ?? []).map((item) => ({
      i: item.i ?? "",
      x: item.x ?? 0,
      y: item.y ?? 0,
      w: item.w ?? 0,
      h: item.h ?? 0,
      ...(item.minW !== undefined ? { minW: item.minW } : {}),
      ...(item.minH !== undefined ? { minH: item.minH } : {}),
    })),
    canvasId,
  });
}

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
  const draftConsoleDiff = useMemo(() => {
    if (!hasDraftConsoleDiffVersusLive) return undefined;
    return { diffCounts: getDraftConsoleDiffCounts(liveDashboardQuery.data, draftDashboardQuery.data) };
  }, [hasDraftConsoleDiffVersusLive, liveDashboardQuery.data, draftDashboardQuery.data]);
  const consoleYamlDiffPayload = useMemo(() => {
    if (!hasDraftConsoleDiffVersusLive || !draftDashboardQuery.data) return null;
    const liveYamlText = dashboardYamlText(canvasId, liveDashboardQuery.data);
    const draftYamlText = dashboardYamlText(canvasId, draftDashboardQuery.data);

    if (liveYamlText === draftYamlText) return null;
    return { liveYamlText, draftYamlText, filename: "console.yaml" };
  }, [canvasId, hasDraftConsoleDiffVersusLive, liveDashboardQuery.data, draftDashboardQuery.data]);
  const [consoleDiffOpen, setConsoleDiffOpen] = useState(false);
  const onShowConsoleDiff = useCallback(() => setConsoleDiffOpen(true), []);
  useEffect(() => {
    if (!consoleYamlDiffPayload && consoleDiffOpen) {
      setConsoleDiffOpen(false);
    }
  }, [consoleDiffOpen, consoleYamlDiffPayload]);
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
    consoleDiffHeaderProps: {
      draftConsoleDiff,
      onShowConsoleDiff: consoleYamlDiffPayload ? onShowConsoleDiff : undefined,
    },
    consoleYamlDiffModal: consoleYamlDiffPayload
      ? createElement(
          Suspense,
          { fallback: null },
          createElement(CanvasYamlDiffModal, {
            open: consoleDiffOpen,
            onOpenChange: setConsoleDiffOpen,
            liveYamlText: consoleYamlDiffPayload.liveYamlText,
            draftYamlText: consoleYamlDiffPayload.draftYamlText,
            filename: consoleYamlDiffPayload.filename,
            title: "Console YAML diff",
            dialogTitle: "Console YAML diff",
            description: "Side-by-side YAML comparison between live and draft console versions.",
          }),
        )
      : null,
    draftChangeIndicators,
    hasDraftDiffVersusLive,
  };
}

export type CanvasConsoleVersionDiffResult = ReturnType<typeof useCanvasConsoleVersionDiff>;
