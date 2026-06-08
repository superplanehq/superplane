import { createElement, lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";

import { useCanvasConsole, useUpdateCanvasConsole } from "@/hooks/useCanvasData";

import { getDraftConsoleDiffCounts, hasDraftVersusLiveConsoleDiff } from "./draftConsoleDiff";
import { apiPanelTypeToPanelType } from "./console/apiPanelType";
import { consoleToYaml } from "./console/consoleYaml";
import { getDraftChangeIndicators } from "./lib/version-action-state";
import type { CanvasesConsole } from "@/api-client";

const CanvasYamlDiffModal = lazy(() =>
  import("./CanvasYamlDiffModal").then((module) => ({ default: module.CanvasYamlDiffModal })),
);

function consoleYamlText(canvasId: string, consoleData?: CanvasesConsole | null): string {
  return consoleToYaml({
    panels: (consoleData?.panels ?? []).map((panel) => ({
      id: panel.id ?? "",
      type: apiPanelTypeToPanelType(panel.type) ?? "markdown",
      content: (panel.content as Record<string, unknown>) ?? {},
    })),
    layout: (consoleData?.layout ?? []).map((item) => ({
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
  const consoleQuery = useCanvasConsole(canvasId, versionIds.active || undefined, enabled);
  const draftDiffVersionId = versionIds.active || versionIds.draft;
  const draftConsoleQuery = useCanvasConsole(
    canvasId,
    draftDiffVersionId || undefined,
    enabled && !!draftDiffVersionId,
  );
  const liveConsoleQuery = useCanvasConsole(canvasId, versionIds.live || undefined, enabled && !!versionIds.live);
  const hasDraftConsoleDiffVersusLive = useMemo(
    () => !!draftDiffVersionId && hasDraftVersusLiveConsoleDiff(liveConsoleQuery.data, draftConsoleQuery.data),
    [draftDiffVersionId, liveConsoleQuery.data, draftConsoleQuery.data],
  );
  const draftConsoleDiff = useMemo(() => {
    if (!hasDraftConsoleDiffVersusLive) return undefined;
    return { diffCounts: getDraftConsoleDiffCounts(liveConsoleQuery.data, draftConsoleQuery.data) };
  }, [hasDraftConsoleDiffVersusLive, liveConsoleQuery.data, draftConsoleQuery.data]);
  const consoleYamlDiffPayload = useMemo(() => {
    if (!hasDraftConsoleDiffVersusLive || !draftConsoleQuery.data) return null;
    const liveYamlText = consoleYamlText(canvasId, liveConsoleQuery.data);
    const draftYamlText = consoleYamlText(canvasId, draftConsoleQuery.data);

    if (liveYamlText === draftYamlText) return null;
    return { liveYamlText, draftYamlText, filename: "console.yaml" };
  }, [canvasId, hasDraftConsoleDiffVersusLive, liveConsoleQuery.data, draftConsoleQuery.data]);
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
  const updateConsoleMutation = useUpdateCanvasConsole(canvasId, versionIds.active || undefined, {
    registerIgnoredCanvasVersionUpdatedEcho,
  });

  return {
    consoleQuery,
    updateConsoleMutation,
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
