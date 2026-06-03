import { createElement, lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";

import {
  useCanvasConsole,
  type CanvasConsoleQueryResult,
  type UpdateCanvasConsoleMutationResult,
} from "@/hooks/useCanvasData";

import { getDraftConsoleDiffCounts, hasDraftVersusLiveConsoleDiff } from "./draftConsoleDiff";
import { dashboardToYaml } from "./dashboard/dashboardYaml";
import { getDraftChangeIndicators } from "./lib/version-action-state";
import type { CanvasesCanvasDashboard } from "@/api-client";

const CanvasYamlDiffModal = lazy(() =>
  import("./CanvasYamlDiffModal").then((module) => ({ default: module.CanvasYamlDiffModal })),
);

function dashboardYamlText(canvasId: string, dashboard?: CanvasesCanvasDashboard | null): string {
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
  branchDashboard?: CanvasesCanvasDashboard | null;
  branchDashboardLoading?: boolean;
  updateDashboardMutation?: UpdateCanvasConsoleMutationResult;
  hasCanvasStagingChanges?: boolean;
  hasConsoleStagingChanges?: boolean;
  isEditingDraftBranch?: boolean;
};

export function useCanvasConsoleVersionDiff({
  canvasId,
  versionIds,
  hasDraftGraphDiffVersusLive,
  suppressUnpublishedDraftDiscard,
  enabled,
  branchDashboard,
  branchDashboardLoading,
  updateDashboardMutation,
  hasCanvasStagingChanges,
  hasConsoleStagingChanges,
  isEditingDraftBranch,
}: UseCanvasConsoleVersionDiffArgs) {
  const liveDashboardQuery = useCanvasConsole(canvasId, versionIds.live || undefined, enabled && !!versionIds.live);
  const fallbackDashboardQuery = useCanvasConsole(
    canvasId,
    versionIds.active || undefined,
    enabled && !isEditingDraftBranch,
  );

  const dashboardData = isEditingDraftBranch ? branchDashboard : fallbackDashboardQuery.data;
  const dashboardQuery = {
    data: dashboardData,
    isLoading: isEditingDraftBranch ? !!branchDashboardLoading : fallbackDashboardQuery.isLoading,
    error: isEditingDraftBranch ? null : fallbackDashboardQuery.error,
    isFetching: isEditingDraftBranch ? !!branchDashboardLoading : fallbackDashboardQuery.isFetching,
    isError: isEditingDraftBranch ? false : fallbackDashboardQuery.isError,
    refetch: fallbackDashboardQuery.refetch,
  } as CanvasConsoleQueryResult;

  const draftDashboardData = isEditingDraftBranch ? branchDashboard : fallbackDashboardQuery.data;
  const hasDraftConsoleDiffVersusLive = useMemo(
    () => !!draftDashboardData && hasDraftVersusLiveConsoleDiff(liveDashboardQuery.data, draftDashboardData),
    [draftDashboardData, liveDashboardQuery.data],
  );
  const draftConsoleDiff = useMemo(() => {
    if (!hasDraftConsoleDiffVersusLive) return undefined;
    return { diffCounts: getDraftConsoleDiffCounts(liveDashboardQuery.data, draftDashboardData) };
  }, [draftDashboardData, hasDraftConsoleDiffVersusLive, liveDashboardQuery.data]);
  const consoleYamlDiffPayload = useMemo(() => {
    if (!hasDraftConsoleDiffVersusLive || !draftDashboardData) return null;
    const liveYamlText = dashboardYamlText(canvasId, liveDashboardQuery.data);
    const draftYamlText = dashboardYamlText(canvasId, draftDashboardData);

    if (liveYamlText === draftYamlText) return null;
    return { liveYamlText, draftYamlText, filename: "console.yaml" };
  }, [canvasId, draftDashboardData, hasDraftConsoleDiffVersusLive, liveDashboardQuery.data]);
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
    hasLatestDraftVersion: isEditingDraftBranch || !!versionIds.draft,
    hasDraftGraphDiffVersusLive,
    hasDraftConsoleDiffVersusLive,
    hasDraftDiffVersusLive,
    hasCanvasStagingChanges,
    hasConsoleStagingChanges,
  });

  const noopUpdateDashboardMutation: UpdateCanvasConsoleMutationResult = useMemo(
    () => ({
      mutate: () => undefined,
      mutateAsync: async () => undefined,
      isPending: false,
    }),
    [],
  );

  return {
    dashboardQuery,
    updateDashboardMutation: updateDashboardMutation ?? noopUpdateDashboardMutation,
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
