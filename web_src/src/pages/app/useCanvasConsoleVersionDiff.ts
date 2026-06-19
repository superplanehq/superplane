import { createElement, lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";

import { useCanvasConsole, useUpdateCanvasConsole } from "@/hooks/useCanvasData";

import {
  buildDraftConsoleDiffSummary,
  getDraftConsoleDiffCounts,
  hasDraftVersusLiveConsoleDiff,
} from "./draftConsoleDiff";
import { materializeConsoleSpec } from "./lib/workflow-spec-files";
import { getDraftChangeIndicators } from "./lib/version-action-state";
import type { CanvasConsoleData } from "@/hooks/useCanvasData";

const CanvasYamlDiffModal = lazy(() =>
  import("./CanvasYamlDiffModal").then((module) => ({ default: module.CanvasYamlDiffModal })),
);

function consoleYamlText(canvasId: string, consoleData?: CanvasConsoleData | null): string {
  return materializeConsoleSpec({
    panels: (consoleData?.panels ?? []).map((panel) => ({
      id: panel.id ?? "",
      type: panel.type ?? "markdown",
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
  stageActiveConsole: boolean;
  registerIgnoredCanvasVersionUpdatedEcho?: (savingVersionId?: string) => () => void;
  getConsoleMutationGeneration?: () => number;
};

export function useCanvasConsoleVersionDiff({
  canvasId,
  versionIds,
  hasDraftGraphDiffVersusLive,
  suppressUnpublishedDraftDiscard,
  enabled,
  stageActiveConsole,
  registerIgnoredCanvasVersionUpdatedEcho,
  getConsoleMutationGeneration,
}: UseCanvasConsoleVersionDiffArgs) {
  // The active console drives the editor. Staged reads (stage=true) apply only
  // while editing a draft; published version previews use committed content.
  const consoleQuery = useCanvasConsole(canvasId, versionIds.active || undefined, enabled, stageActiveConsole);
  const draftDiffVersionId = versionIds.active || versionIds.draft;
  // Committed draft console (stage=false) is the publish basis: only committed
  // changes get promoted to live, so the publishable indicators below diff this
  // against live.
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

  // The on-canvas diff surfaces (X-ray panel badges, the diff summary, and the
  // "Show diff" modal) mirror the canvas tab, which diffs the *effective* draft
  // against live. Prefer the staged console; fall back to the committed draft
  // while it loads or when no version is actively being edited.
  const effectiveConsoleData = consoleQuery.data ?? draftConsoleQuery.data;
  const hasEffectiveConsoleDiffVersusLive = useMemo(
    () => hasDraftVersusLiveConsoleDiff(liveConsoleQuery.data, effectiveConsoleData),
    [liveConsoleQuery.data, effectiveConsoleData],
  );
  const draftConsoleDiff = useMemo(() => {
    if (!hasEffectiveConsoleDiffVersusLive) return undefined;
    return {
      diffCounts: getDraftConsoleDiffCounts(liveConsoleQuery.data, effectiveConsoleData),
    };
  }, [hasEffectiveConsoleDiffVersusLive, liveConsoleQuery.data, effectiveConsoleData]);
  const draftConsoleDiffSummary = useMemo(() => {
    if (!hasEffectiveConsoleDiffVersusLive) return undefined;
    return buildDraftConsoleDiffSummary(liveConsoleQuery.data, effectiveConsoleData);
  }, [hasEffectiveConsoleDiffVersusLive, liveConsoleQuery.data, effectiveConsoleData]);
  const consoleYamlDiffPayload = useMemo(() => {
    if (!hasEffectiveConsoleDiffVersusLive || !effectiveConsoleData) return null;
    const liveYamlText = consoleYamlText(canvasId, liveConsoleQuery.data);
    const draftYamlText = consoleYamlText(canvasId, effectiveConsoleData);

    if (liveYamlText === draftYamlText) return null;
    return { liveYamlText, draftYamlText, filename: "console.yaml" };
  }, [canvasId, hasEffectiveConsoleDiffVersusLive, liveConsoleQuery.data, effectiveConsoleData]);
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
    hasLatestDraftVersion: !!(versionIds.active || versionIds.draft),
    hasDraftGraphDiffVersusLive,
    hasDraftConsoleDiffVersusLive,
    hasDraftDiffVersusLive,
  });
  const updateConsoleMutation = useUpdateCanvasConsole(canvasId, versionIds.active || undefined, {
    registerIgnoredCanvasVersionUpdatedEcho,
    getMutationGeneration: getConsoleMutationGeneration,
  });

  return {
    consoleQuery,
    updateConsoleMutation,
    consoleDiffHeaderProps: {
      draftConsoleDiff,
      onShowConsoleDiff: consoleYamlDiffPayload ? onShowConsoleDiff : undefined,
    },
    draftConsoleDiffSummary,
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
