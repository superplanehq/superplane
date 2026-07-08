import { useCallback, useEffect, useMemo, useState } from "react";

import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

import { hasLocalCanvasGraphDiff, hasLocalConsoleDiff } from "./lib/local-staging-indicators";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "./lib/workflow-spec-paths";
import type { useCommittedDraftBaselines } from "./useCommittedDraftBaselines";
import type { useCanvasConsoleVersionDiff } from "./useCanvasConsoleVersionDiff";
import type { useCanvasStaging } from "@/hooks/useCanvasData";

type CommittedBaselines = ReturnType<typeof useCommittedDraftBaselines>;
type CanvasStagingQuery = ReturnType<typeof useCanvasStaging>;
type ConsoleVersionDiff = ReturnType<typeof useCanvasConsoleVersionDiff>;
type ConsoleQueryData = ConsoleVersionDiff["consoleQuery"]["data"];

function resolveEditingStagingFlags({
  isEditing,
  editBootstrapReady,
  committedBaselinesReady,
  localHasCanvasStaging,
  localHasConsoleStaging,
  filesLocalStagingActive,
  localHasFilesStaging,
  serverHasFilesStaging,
}: {
  isEditing: boolean;
  editBootstrapReady: boolean;
  committedBaselinesReady: boolean;
  localHasCanvasStaging: boolean;
  localHasConsoleStaging: boolean;
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

  if (!editBootstrapReady || !committedBaselinesReady) {
    return {
      hasCanvasStagingChanges: false,
      hasConsoleStagingChanges: false,
      hasFilesStagingChanges: false,
      hasStagingChanges: false,
    };
  }

  const hasCanvasStagingChanges = localHasCanvasStaging;
  const hasConsoleStagingChanges = localHasConsoleStaging;
  const hasFilesStagingChanges = filesLocalStagingActive ? localHasFilesStaging : serverHasFilesStaging;
  const hasStagingChanges = localHasCanvasStaging || localHasConsoleStaging || hasFilesStagingChanges;

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
  hasCanvasStagingChanges,
  hasConsoleStagingChanges,
  hasFilesStagingChanges,
}: {
  isEditing: boolean;
  hasCanvasStagingChanges: boolean;
  hasConsoleStagingChanges: boolean;
  hasFilesStagingChanges: boolean;
}) {
  return {
    hasUncommittedCanvasDraftChanges: isEditing && hasCanvasStagingChanges,
    hasUncommittedConsoleDraftChanges: isEditing && hasConsoleStagingChanges,
    hasUncommittedFilesDraftChanges: isEditing && hasFilesStagingChanges,
    hasCommittedCanvasDraftChanges: false,
    hasCommittedConsoleDraftChanges: false,
  };
}

type UseDraftStagingIndicatorsOptions = {
  isEditing: boolean;
  editBootstrapReady: boolean;
  canvasId?: string;
  activeCanvasVersionId: string;
  stagingResetNonce: number;
  canvasStagingQuery: CanvasStagingQuery;
  committedBaselines: CommittedBaselines;
  effectiveCanvasSpec: ReturnType<typeof useCommittedDraftBaselines>["canvasSpec"];
  consoleQueryData: ConsoleQueryData;
  draftChangeIndicators: ConsoleVersionDiff["draftChangeIndicators"];
};

export function useDraftStagingIndicators({
  isEditing,
  editBootstrapReady,
  canvasStagingQuery,
  committedBaselines,
  effectiveCanvasSpec,
  consoleQueryData,
  stagingResetNonce,
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

  const { serverHasFilesStaging } = getServerStagingFlags(canvasStagingQuery.data?.stagedPaths);
  const serverHasStagingChanges = !!canvasStagingQuery.data?.hasStaging;

  const { hasCanvasStagingChanges, hasConsoleStagingChanges, hasFilesStagingChanges, hasStagingChanges } =
    resolveEditingStagingFlags({
      isEditing,
      editBootstrapReady,
      committedBaselinesReady: committedBaselines.ready,
      localHasCanvasStaging,
      localHasConsoleStaging,
      filesLocalStagingActive,
      localHasFilesStaging,
      serverHasFilesStaging,
    });

  const {
    hasUncommittedCanvasDraftChanges,
    hasUncommittedConsoleDraftChanges,
    hasUncommittedFilesDraftChanges,
    hasCommittedCanvasDraftChanges,
    hasCommittedConsoleDraftChanges,
  } = buildDraftChangeFlags({
    isEditing,
    hasCanvasStagingChanges,
    hasConsoleStagingChanges,
    hasFilesStagingChanges,
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
    hasUncommittedCanvasDraftChanges,
    hasUncommittedConsoleDraftChanges,
    hasUncommittedFilesDraftChanges,
    hasCommittedCanvasDraftChanges,
    hasCommittedConsoleDraftChanges,
  };
}
