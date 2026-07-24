import { useMemo } from "react";

export type WorkflowHeaderMode = "version-live" | "console" | "memory" | "files";
export type CanvasPageHeaderMode = WorkflowHeaderMode | "default";
export type WorkflowCanvasStateMode = "default" | "editing" | "previewing-previous-version";

const PANEL_HEADER_MODES = new Set<WorkflowHeaderMode>(["memory", "files"]);

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

export function isCanvasWorkflowTab(headerMode: CanvasPageHeaderMode | undefined): boolean {
  if (!headerMode || headerMode === "default") {
    return true;
  }

  return headerMode === "version-live";
}

/**
 * True when the runs sidebar (and its toggle icon) may be shown for the given
 * tab. The runs sidebar is available on the main workflow Canvas tab and on the
 * Console tab, but not on the Memory or Files surfaces. The Console overlay is
 * laid out beside the left sidebars, so an open runs sidebar coexists with it.
 */
export function allowsRunsSidebar(headerMode: CanvasPageHeaderMode | undefined): boolean {
  return isCanvasWorkflowTab(headerMode) || headerMode === "console";
}

const CONSOLE_VIEW = "console";
const LEGACY_CONSOLE_VIEW = "dashboard";
const LEGACY_RUNS_VIEW = "runs";

function isConsoleViewParam(view: string): boolean {
  return view === CONSOLE_VIEW || view === LEGACY_CONSOLE_VIEW;
}

/** True when the URL points at the main workflow canvas tab (not Console, Memory, Files, or Versions). */
export function isWorkflowCanvasViewParam(view: string): boolean {
  return view === "" || view === LEGACY_RUNS_VIEW;
}

/** View flags read directly from the URL (source of truth for first paint and header tab selection). */
export function getWorkflowViewFlagsFromSearchParams(searchParams: URLSearchParams) {
  const view = searchParams.get("view") ?? "";
  const run = searchParams.get("run") ?? "";
  const isRunInspectionMode = Boolean(run) && isWorkflowCanvasViewParam(view);
  return {
    isRunInspectionMode,
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

export function clearRunInspectionSearchParams(params: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(params);
  next.delete("run");
  next.delete("sidebar");
  next.delete("node");
  return next;
}

/** Run inspection is only valid on the canvas tab; panel views must be cleared first. */
export function clearNonCanvasViewSearchParam(params: URLSearchParams): void {
  const view = params.get("view") ?? "";
  if (!isWorkflowCanvasViewParam(view)) {
    params.delete("view");
  }
}

export function applyRunInspectionNavigationSearchParams(
  params: URLSearchParams,
  options: {
    runId: string;
    nodeId?: string | null;
  },
): URLSearchParams {
  const next = new URLSearchParams(params);
  next.set("run", options.runId);

  if (options.nodeId) {
    next.set("sidebar", "1");
    next.set("node", options.nodeId);
  } else {
    next.delete("sidebar");
    next.delete("node");
  }

  clearNonCanvasViewSearchParam(next);
  next.delete("file");
  return next;
}

export function getWorkflowHeaderMode({
  isConsoleMode,
  isMemoryMode,
  isFilesMode,
}: {
  isConsoleMode: boolean;
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

  return "version-live";
}

export function getWorkflowCanvasStateMode({
  hasEditableVersion,
  isViewingCurrentLiveVersion,
}: {
  hasEditableVersion: boolean;
  isViewingCurrentLiveVersion: boolean;
}): WorkflowCanvasStateMode {
  if (hasEditableVersion) {
    return "editing";
  }

  if (!isViewingCurrentLiveVersion) {
    return "previewing-previous-version";
  }

  return "default";
}

export function getWorkflowViewPresentation({
  isConsoleMode,
  isRunInspectionMode,
  isMemoryMode,
  isFilesMode,
  hasEditableVersion,
  isViewingCurrentLiveVersion,
}: {
  isConsoleMode: boolean;
  isRunInspectionMode: boolean;
  isMemoryMode: boolean;
  isFilesMode: boolean;
  hasEditableVersion: boolean;
  isViewingCurrentLiveVersion: boolean;
}) {
  const hideNonCanvasChrome = isRunInspectionMode || isMemoryMode || isFilesMode;

  return {
    headerMode: getWorkflowHeaderMode({ isConsoleMode, isMemoryMode, isFilesMode }),
    canvasStateMode: getWorkflowCanvasStateMode({
      hasEditableVersion,
      isViewingCurrentLiveVersion,
    }),
    showBottomStatusControls: !hideNonCanvasChrome,
    hideAddControls: hideNonCanvasChrome,
    readOnlyViewModes: isRunInspectionMode || isFilesMode,
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
  if (canvasDeletedRemotely) {
    return "This canvas was deleted in another session.";
  }

  if (!canUpdateCanvas) {
    return "You don't have permission to edit this canvas.";
  }

  if (!hasEditableVersion) {
    return "Edit mode is not enabled.";
  }

  return undefined;
}

export function getRunActionState({
  canUpdateCanvas,
  canvasDeletedRemotely,
  isViewingDraftVersion,
  isViewingCurrentLiveVersion,
}: {
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  isViewingDraftVersion: boolean;
  isViewingCurrentLiveVersion: boolean;
}): { disabled: boolean; tooltip?: string } {
  const disabled = !canUpdateCanvas || canvasDeletedRemotely || isViewingDraftVersion || !isViewingCurrentLiveVersion;

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

  return { disabled, tooltip: undefined };
}
