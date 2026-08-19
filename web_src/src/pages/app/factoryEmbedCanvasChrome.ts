import type { ReactNode, RefObject } from "react";
import type { CanvasesCanvas, CanvasesCanvasRun } from "@/api-client";

type ViewportLike = { x: number; y: number; zoom: number };

type FactoryEmbedCanvasChromeInput = {
  factoryEmbed: boolean;
  factoryAgentEnabled?: boolean;
  factoryViewOnly: boolean;
  factoryOwnedApp: boolean;
  routeFactoryId?: string;
  canvas?: CanvasesCanvas | null;
  headerBanner: ReactNode;
  canUpdateCanvas: boolean;
  showBottomStatusControls: boolean;
  hideAddControls: boolean;
  editSessionActive: boolean;
  runInspectionChromeActive: boolean;
  handleSelectMemoryMode: () => void;
  handleSelectConsoleMode: () => void;
  handleSelectFilesMode: () => void;
  handleEnterEditModeFromHeader: () => void | Promise<void>;
  handleExitEditSession: () => void;
  handleSelectLiveCanvas: () => void;
  handleBackToRunList: () => void;
  filesHeaderActionsSlotId: string;
  runsHasFitToViewRef: RefObject<boolean>;
  hasFitToViewRef: RefObject<boolean>;
  runsViewportRef: RefObject<ViewportLike | undefined>;
  viewportRef: RefObject<ViewportLike | undefined>;
  selectedRunId: string | null;
  selectedRunDetailDismissed: boolean;
  runCanvasLoading: boolean;
  selectedRun: CanvasesCanvasRun | null;
};

function resolveFactoryShellVisibility(input: FactoryEmbedCanvasChromeInput) {
  return {
    factoryId: input.canvas?.metadata?.factoryId ?? (input.factoryEmbed ? input.routeFactoryId : undefined),
    headerBanner: input.factoryEmbed ? null : input.headerBanner,
    showCanvasSettingsMenu: !input.factoryEmbed && input.canUpdateCanvas,
    showBottomStatusControls: !input.factoryEmbed && input.showBottomStatusControls,
    hideAddControls: input.hideAddControls || input.factoryViewOnly,
    hidePageChrome: input.factoryEmbed,
    hideCanvasToolSidebar: input.factoryEmbed && !input.factoryAgentEnabled,
    hideRightSideControls: input.factoryEmbed,
    factoryEmbed: input.factoryEmbed,
  };
}

function resolveFactoryShellModeActions(input: FactoryEmbedCanvasChromeInput) {
  const hideFactoryShellChrome = input.factoryOwnedApp || input.factoryEmbed;
  return {
    onSelectMemory: hideFactoryShellChrome ? undefined : input.handleSelectMemoryMode,
    onSelectConsole: hideFactoryShellChrome ? undefined : input.handleSelectConsoleMode,
    onSelectFiles: hideFactoryShellChrome ? undefined : input.handleSelectFilesMode,
    filesHeaderActionsSlotId: hideFactoryShellChrome ? undefined : input.filesHeaderActionsSlotId,
    isEditSessionActive: input.factoryViewOnly ? false : input.editSessionActive,
    onEnterEditMode: input.factoryViewOnly ? undefined : input.handleEnterEditModeFromHeader,
    onExitEditMode: input.factoryViewOnly ? undefined : input.handleExitEditSession,
  };
}

function resolveFactoryShellModeProps(input: FactoryEmbedCanvasChromeInput) {
  return {
    ...resolveFactoryShellVisibility(input),
    ...resolveFactoryShellModeActions(input),
  };
}

function resolveFactoryRunInspectionProps(input: FactoryEmbedCanvasChromeInput) {
  return {
    // Keep ?run= on close so factory canvas stays run-scoped (no stale Live lastExecutions).
    onRunNodeDetailClose: input.handleBackToRunList,
    onBackToLiveCanvas: input.factoryEmbed ? undefined : input.handleSelectLiveCanvas,
    hasFitToViewRef: input.runInspectionChromeActive ? input.runsHasFitToViewRef : input.hasFitToViewRef,
    isRunInspectionMode: input.runInspectionChromeActive,
    viewportRef: input.runInspectionChromeActive ? input.runsViewportRef : input.viewportRef,
    runCanvasLoading:
      input.runInspectionChromeActive &&
      input.selectedRunId !== null &&
      !input.selectedRunDetailDismissed &&
      input.runCanvasLoading,
    runNodeDetailRun:
      input.runInspectionChromeActive && input.selectedRunId && !input.selectedRunDetailDismissed
        ? input.selectedRun
        : null,
  };
}

/**
 * Factory-shell CanvasPage chrome flags. Kept outside AppPage so embed/configure
 * ternaries do not inflate AppPage complexity / line budget.
 */
export function resolveFactoryEmbedCanvasChrome(input: FactoryEmbedCanvasChromeInput) {
  return {
    ...resolveFactoryShellModeProps(input),
    ...resolveFactoryRunInspectionProps(input),
  };
}

export function resolveFactoryEmbedSidebars({
  factoryEmbed,
  headerModeAllowsRuns,
  editSessionActive,
  isMemoryMode,
  isFilesMode,
  runInspectionChromeActive,
}: {
  factoryEmbed: boolean;
  headerModeAllowsRuns: boolean;
  editSessionActive: boolean;
  isMemoryMode: boolean;
  isFilesMode: boolean;
  runInspectionChromeActive: boolean;
}) {
  return {
    showRunsSidebar: !factoryEmbed && headerModeAllowsRuns && !editSessionActive && !isMemoryMode && !isFilesMode,
    showVersionsSidebar: !factoryEmbed && editSessionActive && !runInspectionChromeActive && !isMemoryMode,
  };
}
