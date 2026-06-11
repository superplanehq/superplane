import { useCallback, useEffect, useMemo, useState } from "react";

import type { CanvasesCanvasVersion } from "@/api-client";
import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

import { hasDraftVersusLiveGraphDiff } from "./draftNodeDiff";
import { draftEditTabToneFromStaging } from "./lib/draft-branch-edit-status";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "./lib/workflow-spec-paths";
import { hasLocalCanvasGraphDiff, hasLocalConsoleDiff } from "./lib/local-staging-indicators";
import { useDraftBranchesEditStatus } from "./useDraftBranchesEditStatus";
import type { useCommittedDraftBaselines } from "./useCommittedDraftBaselines";
import type { useCanvasConsoleVersionDiff } from "./useCanvasConsoleVersionDiff";
import type { useCanvasVersionStaging } from "@/hooks/useCanvasData";

type CommittedBaselines = ReturnType<typeof useCommittedDraftBaselines>;
type CanvasVersionStagingQuery = ReturnType<typeof useCanvasVersionStaging>;
type ConsoleVersionDiff = ReturnType<typeof useCanvasConsoleVersionDiff>;
type ConsoleQueryData = ConsoleVersionDiff["consoleQuery"]["data"];

function buildPublishableChangesByVersionId({
  activeCanvasVersionId,
  draftVersionForGraphDiff,
  draftVersionsFromBranches,
  liveVersionForGraphDiff,
}: {
  activeCanvasVersionId: string;
  draftVersionForGraphDiff: CanvasesCanvasVersion | undefined;
  draftVersionsFromBranches: CanvasesCanvasVersion[];
  liveVersionForGraphDiff: CanvasesCanvasVersion | undefined;
}) {
  const map = new Map<string, boolean>();
  for (const draft of draftVersionsFromBranches) {
    const versionId = draft.metadata?.id;
    if (!versionId) {
      continue;
    }
    const draftForDiff =
      versionId === activeCanvasVersionId && draftVersionForGraphDiff ? draftVersionForGraphDiff : draft;
    map.set(versionId, hasDraftVersusLiveGraphDiff(liveVersionForGraphDiff, draftForDiff));
  }
  if (activeCanvasVersionId && !map.has(activeCanvasVersionId) && draftVersionForGraphDiff) {
    map.set(activeCanvasVersionId, hasDraftVersusLiveGraphDiff(liveVersionForGraphDiff, draftVersionForGraphDiff));
  }
  return map;
}

function resolveEditingStagingFlags({
  isEditing,
  committedBaselinesReady,
  localHasCanvasStaging,
  localHasConsoleStaging,
  serverHasCanvasStaging,
  serverHasConsoleStaging,
  serverHasStagingChanges,
  filesLocalStagingActive,
  localHasFilesStaging,
  serverHasFilesStaging,
}: {
  isEditing: boolean;
  committedBaselinesReady: boolean;
  localHasCanvasStaging: boolean;
  localHasConsoleStaging: boolean;
  serverHasCanvasStaging: boolean;
  serverHasConsoleStaging: boolean;
  serverHasStagingChanges: boolean;
  filesLocalStagingActive: boolean;
  localHasFilesStaging: boolean;
  serverHasFilesStaging: boolean;
}) {
  if (!isEditing) {
    return {
      hasCanvasStagingChanges: false,
      hasConsoleStagingChanges: false,
      hasFilesStagingChanges: false,
      hasStagingChanges: false,
    };
  }

  const hasCanvasStagingChanges = committedBaselinesReady ? localHasCanvasStaging : serverHasCanvasStaging;
  const hasConsoleStagingChanges = committedBaselinesReady ? localHasConsoleStaging : serverHasConsoleStaging;
  const hasFilesStagingChanges = filesLocalStagingActive ? localHasFilesStaging : serverHasFilesStaging;
  const hasStagingChanges = committedBaselinesReady
    ? localHasCanvasStaging || localHasConsoleStaging || hasFilesStagingChanges
    : serverHasStagingChanges;

  return { hasCanvasStagingChanges, hasConsoleStagingChanges, hasFilesStagingChanges, hasStagingChanges };
}

function getServerStagingFlags(stagedPaths: string[] | undefined) {
  const paths = stagedPaths ?? [];
  return {
    serverHasCanvasStaging: paths.includes(CANVAS_YAML_PATH),
    serverHasConsoleStaging: paths.includes(CONSOLE_YAML_PATH),
    serverHasFilesStaging: paths.some((path) => path !== CANVAS_YAML_PATH && path !== CONSOLE_YAML_PATH),
  };
}

function buildDraftChangeFlags({
  isEditing,
  hasStagingChanges,
  hasCanvasStagingChanges,
  hasConsoleStagingChanges,
  hasFilesStagingChanges,
  draftChangeIndicators,
}: {
  isEditing: boolean;
  hasStagingChanges: boolean;
  hasCanvasStagingChanges: boolean;
  hasConsoleStagingChanges: boolean;
  hasFilesStagingChanges: boolean;
  draftChangeIndicators: ConsoleVersionDiff["draftChangeIndicators"];
}) {
  return {
    editTabTone: draftEditTabToneFromStaging(hasStagingChanges, isEditing),
    hasUncommittedCanvasDraftChanges: isEditing && hasCanvasStagingChanges,
    hasUncommittedConsoleDraftChanges: isEditing && hasConsoleStagingChanges,
    hasUncommittedFilesDraftChanges: isEditing && hasFilesStagingChanges,
    hasCommittedCanvasDraftChanges:
      isEditing && !hasCanvasStagingChanges && draftChangeIndicators.hasUnpublishedCanvasDraftChanges,
    hasCommittedConsoleDraftChanges:
      isEditing && !hasConsoleStagingChanges && draftChangeIndicators.hasUnpublishedConsoleDraftChanges,
  };
}

type UseDraftStagingIndicatorsOptions = {
  isEditing: boolean;
  canvasId?: string;
  activeCanvasVersionId: string;
  liveCanvasVersionId: string;
  stagingResetNonce: number;
  canvasVersionStagingQuery: CanvasVersionStagingQuery;
  committedBaselines: CommittedBaselines;
  effectiveCanvasSpec: CanvasesCanvasVersion["spec"] | undefined;
  consoleQueryData: ConsoleQueryData;
  draftVersionsFromBranches: CanvasesCanvasVersion[];
  draftVersionForGraphDiff: CanvasesCanvasVersion | undefined;
  liveVersionForGraphDiff: CanvasesCanvasVersion | undefined;
  draftBranches: CanvasesCanvasVersion[];
  hasDraftDiffVersusLive: boolean;
  draftChangeIndicators: ConsoleVersionDiff["draftChangeIndicators"];
};

export function useDraftStagingIndicators({
  isEditing,
  canvasId,
  activeCanvasVersionId,
  liveCanvasVersionId,
  stagingResetNonce,
  canvasVersionStagingQuery,
  committedBaselines,
  effectiveCanvasSpec,
  consoleQueryData,
  draftVersionsFromBranches,
  draftVersionForGraphDiff,
  liveVersionForGraphDiff,
  draftBranches,
  hasDraftDiffVersusLive,
  draftChangeIndicators,
}: UseDraftStagingIndicatorsOptions) {
  const [effectiveConsole, setEffectiveConsole] = useState<
    { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] } | undefined
  >();
  const [localHasFilesStaging, setLocalHasFilesStaging] = useState(false);
  const [filesLocalStagingActive, setFilesLocalStagingActive] = useState(false);

  const handleEffectiveConsoleChange = useCallback((next: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] }) => {
    setEffectiveConsole(next);
  }, []);

  const handleLocalFilesStagingChange = useCallback((hasStaging: boolean) => {
    setLocalHasFilesStaging(hasStaging);
    // Local files detection only knows about in-session edits (pending changes),
    // so it must not override the authoritative server staging flag until it has
    // actually observed a staged edit. On a fresh mount (e.g. after refresh) it
    // reports `false` before any file is inspected; switching to local detection
    // then would incorrectly hide persisted server-side staging (orange -> blue).
    // Once a local edit is seen we keep trusting local detection for the session
    // so reverting an edit back to committed clears the indicator immediately.
    if (hasStaging) {
      setFilesLocalStagingActive(true);
    }
  }, []);

  useEffect(() => {
    setFilesLocalStagingActive(false);
    setLocalHasFilesStaging(false);
  }, [stagingResetNonce]);

  useEffect(() => {
    if (consoleQueryData) {
      setEffectiveConsole({
        panels: consoleQueryData.panels ?? [],
        layout: consoleQueryData.layout ?? [],
      });
    }
  }, [consoleQueryData, stagingResetNonce]);

  const localHasCanvasStaging = useMemo(
    () => hasLocalCanvasGraphDiff(committedBaselines.canvasSpec, effectiveCanvasSpec),
    [committedBaselines.canvasSpec, effectiveCanvasSpec],
  );
  const localHasConsoleStaging = useMemo(
    () => hasLocalConsoleDiff(committedBaselines.console, effectiveConsole),
    [committedBaselines.console, effectiveConsole],
  );

  const { serverHasCanvasStaging, serverHasConsoleStaging, serverHasFilesStaging } = getServerStagingFlags(
    canvasVersionStagingQuery.data?.stagedPaths,
  );
  const serverHasStagingChanges = !!canvasVersionStagingQuery.data?.hasStaging;

  const { hasCanvasStagingChanges, hasConsoleStagingChanges, hasFilesStagingChanges, hasStagingChanges } =
    resolveEditingStagingFlags({
      isEditing,
      committedBaselinesReady: committedBaselines.ready,
      localHasCanvasStaging,
      localHasConsoleStaging,
      serverHasCanvasStaging,
      serverHasConsoleStaging,
      serverHasStagingChanges,
      filesLocalStagingActive,
      localHasFilesStaging,
      serverHasFilesStaging,
    });

  const publishableChangesByVersionId = useMemo(
    () =>
      buildPublishableChangesByVersionId({
        activeCanvasVersionId,
        draftVersionForGraphDiff,
        draftVersionsFromBranches,
        liveVersionForGraphDiff,
      }),
    [activeCanvasVersionId, draftVersionForGraphDiff, draftVersionsFromBranches, liveVersionForGraphDiff],
  );

  const activeHasPublishableChanges = isEditing && hasDraftDiffVersusLive;

  const draftBranchEditStatusByVersionId = useDraftBranchesEditStatus({
    canvasId,
    draftBranches,
    activeVersionId: activeCanvasVersionId || undefined,
    liveVersionId: liveCanvasVersionId || undefined,
    useLocalActiveStatus: isEditing && committedBaselines.ready,
    activeHasUncommittedChanges: hasStagingChanges,
    activeServerHasUncommittedChanges: serverHasStagingChanges,
    activeHasPublishableChanges,
    publishableChangesByVersionId,
  });

  const {
    editTabTone,
    hasUncommittedCanvasDraftChanges,
    hasUncommittedConsoleDraftChanges,
    hasUncommittedFilesDraftChanges,
    hasCommittedCanvasDraftChanges,
    hasCommittedConsoleDraftChanges,
  } = buildDraftChangeFlags({
    isEditing,
    hasStagingChanges,
    hasCanvasStagingChanges,
    hasConsoleStagingChanges,
    hasFilesStagingChanges,
    draftChangeIndicators,
  });

  return {
    effectiveConsole,
    handleEffectiveConsoleChange,
    handleLocalFilesStagingChange,
    hasStagingChanges,
    hasCanvasStagingChanges,
    hasConsoleStagingChanges,
    hasFilesStagingChanges,
    serverHasStagingChanges,
    draftBranchEditStatusByVersionId,
    editTabTone,
    hasUncommittedCanvasDraftChanges,
    hasUncommittedConsoleDraftChanges,
    hasUncommittedFilesDraftChanges,
    hasCommittedCanvasDraftChanges,
    hasCommittedConsoleDraftChanges,
  };
}
