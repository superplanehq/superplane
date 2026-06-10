import { useMemo } from "react";

export type WorkflowHeaderMode = "version-live" | "version-edit" | "runs" | "versions" | "console" | "memory" | "files";
export type CanvasPageHeaderMode = WorkflowHeaderMode | "default";
export type WorkflowCanvasStateMode = "default" | "editing" | "previewing-previous-version" | "awaiting-approval";

const PANEL_HEADER_MODES = new Set<WorkflowHeaderMode>(["runs", "versions", "memory", "files"]);

export function normalizeCanvasHeaderMode(headerMode: CanvasPageHeaderMode | undefined): WorkflowHeaderMode {
  if (!headerMode || headerMode === "default") {
    return "version-live";
  }

  return headerMode;
}

export function isPanelHeaderMode(headerMode: WorkflowHeaderMode): boolean {
  return PANEL_HEADER_MODES.has(headerMode);
}

export function blocksBuildingBlocksShortcut(headerMode: WorkflowHeaderMode): boolean {
  return headerMode === "console" || isPanelHeaderMode(headerMode);
}

export function allowsBuildingBlocksSidebar(headerMode: WorkflowHeaderMode): boolean {
  return headerMode !== "console" && !isPanelHeaderMode(headerMode);
}

export function isRunsOrVersionsHeaderMode(headerMode: WorkflowHeaderMode): boolean {
  return headerMode === "runs" || headerMode === "versions";
}

const CONSOLE_VIEW = "console";
const LEGACY_CONSOLE_VIEW = "dashboard";

function isConsoleViewParam(view: string): boolean {
  return view === CONSOLE_VIEW || view === LEGACY_CONSOLE_VIEW;
}

/** View flags read directly from the URL (source of truth for first paint and header tab selection). */
export function getWorkflowViewFlagsFromSearchParams(searchParams: URLSearchParams) {
  const view = searchParams.get("view") ?? "";
  return {
    isRunsMode: view === "runs",
    isVersionsMode: view === "versions",
    isMemoryMode: view === "memory",
    isFilesMode: view === "files",
    isConsoleMode: isConsoleViewParam(view),
  };
}

export function useWorkflowUrlViewFlags(searchParams: URLSearchParams) {
  return useMemo(() => getWorkflowViewFlagsFromSearchParams(searchParams), [searchParams]);
}

export function readStoredBoolean(key: string): boolean {
  if (typeof window === "undefined") {
    return false;
  }

  const stored = window.localStorage.getItem(key);
  if (stored === null) {
    return false;
  }

  try {
    return JSON.parse(stored) as boolean;
  } catch {
    return false;
  }
}

export function clearComponentSidebarSearchParams(params: URLSearchParams): URLSearchParams {
  params.delete("sidebar");
  params.delete("node");
  return params;
}

export function getWorkflowHeaderMode({
  isConsoleMode,
  isRunsMode,
  isVersionsMode,
  isMemoryMode,
  isFilesMode,
}: {
  isConsoleMode: boolean;
  isRunsMode: boolean;
  isVersionsMode: boolean;
  isMemoryMode: boolean;
  isFilesMode: boolean;
}): WorkflowHeaderMode {
  if (isConsoleMode) {
    return "console";
  }

  if (isMemoryMode) {
    return "memory";
  }

  if (isFilesMode) {
    return "files";
  }

  if (isRunsMode) {
    return "runs";
  }

  if (isVersionsMode) {
    return "versions";
  }

  return "version-live";
}

export function getWorkflowCanvasStateMode({
  hasEditableVersion,
  isViewingPendingApprovalVersion,
  isViewingCurrentLiveVersion,
}: {
  hasEditableVersion: boolean;
  isViewingPendingApprovalVersion: boolean;
  isViewingCurrentLiveVersion: boolean;
}): WorkflowCanvasStateMode {
  if (hasEditableVersion) {
    return "editing";
  }

  if (isViewingPendingApprovalVersion) {
    return "awaiting-approval";
  }

  if (!isViewingCurrentLiveVersion) {
    return "previewing-previous-version";
  }

  return "default";
}

export function getWorkflowViewPresentation({
  isConsoleMode,
  isRunsMode,
  isVersionsMode,
  isMemoryMode,
  isFilesMode,
  hasEditableVersion,
  isViewingPendingApprovalVersion,
  isViewingCurrentLiveVersion,
}: {
  isConsoleMode: boolean;
  isRunsMode: boolean;
  isVersionsMode: boolean;
  isMemoryMode: boolean;
  isFilesMode: boolean;
  hasEditableVersion: boolean;
  isViewingPendingApprovalVersion: boolean;
  isViewingCurrentLiveVersion: boolean;
}) {
  const hideNonCanvasChrome = isRunsMode || isVersionsMode || isMemoryMode || isFilesMode;

  return {
    headerMode: getWorkflowHeaderMode({ isConsoleMode, isRunsMode, isVersionsMode, isMemoryMode, isFilesMode }),
    canvasStateMode: getWorkflowCanvasStateMode({
      hasEditableVersion,
      isViewingPendingApprovalVersion,
      isViewingCurrentLiveVersion,
    }),
    showBottomStatusControls: !hideNonCanvasChrome,
    hideAddControls: hideNonCanvasChrome,
    readOnlyViewModes: isRunsMode || isVersionsMode || isFilesMode,
  };
}

export function getExitEditModeDisabledTooltip({
  canUpdateCanvas,
  canvasDeletedRemotely,
  hasEditableVersion,
}: {
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  hasEditableVersion: boolean;
}): string | undefined {
  if (!canUpdateCanvas) {
    return "You don't have permission to edit this canvas.";
  }

  if (canvasDeletedRemotely) {
    return "This canvas was deleted in another session.";
  }

  if (!hasEditableVersion) {
    return "Edit mode is not enabled.";
  }

  return undefined;
}

export function getRunActionState({
  hasRunBlockingChanges,
  canUpdateCanvas,
  canvasDeletedRemotely,
  isViewingDraftVersion,
  isViewingCurrentLiveVersion,
}: {
  hasRunBlockingChanges: boolean;
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  isViewingDraftVersion: boolean;
  isViewingCurrentLiveVersion: boolean;
}): { disabled: boolean; tooltip?: string } {
  const disabled =
    hasRunBlockingChanges ||
    !canUpdateCanvas ||
    canvasDeletedRemotely ||
    isViewingDraftVersion ||
    !isViewingCurrentLiveVersion;

  if (canvasDeletedRemotely) {
    return { disabled, tooltip: "This canvas was deleted in another session." };
  }

  if (isViewingDraftVersion) {
    return { disabled, tooltip: "Draft versions do not execute. Publish to run this canvas." };
  }

  if (!isViewingCurrentLiveVersion) {
    return { disabled, tooltip: "Only the current live version can execute." };
  }

  if (!canUpdateCanvas) {
    return { disabled, tooltip: "You don't have permission to emit events on this canvas." };
  }

  if (hasRunBlockingChanges) {
    return { disabled, tooltip: "Save canvas changes before running" };
  }

  return { disabled, tooltip: undefined };
}
