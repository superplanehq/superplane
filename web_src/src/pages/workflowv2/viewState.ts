export type WorkflowHeaderMode = "version-live" | "version-edit" | "runs" | "dashboard";
export type WorkflowCanvasStateMode = "default" | "editing" | "previewing-previous-version" | "awaiting-approval";

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

export function getWorkflowHeaderMode({
  isDashboardMode,
  dashboardsFeatureEnabled,
  isRunsMode,
  canvasMode,
}: {
  isDashboardMode: boolean;
  dashboardsFeatureEnabled: boolean;
  isRunsMode: boolean;
  canvasMode: "edit" | "live";
}): WorkflowHeaderMode {
  if (isDashboardMode) {
    return dashboardsFeatureEnabled ? "dashboard" : "version-live";
  }

  if (isRunsMode) {
    return "runs";
  }

  return canvasMode === "edit" ? "version-edit" : "version-live";
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
  isTemplate,
  canUpdateCanvas,
  canvasDeletedRemotely,
  isViewingDraftVersion,
  isViewingCurrentLiveVersion,
}: {
  hasRunBlockingChanges: boolean;
  isTemplate: boolean;
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  isViewingDraftVersion: boolean;
  isViewingCurrentLiveVersion: boolean;
}): { disabled: boolean; tooltip?: string } {
  const disabled =
    hasRunBlockingChanges ||
    isTemplate ||
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

  if (isTemplate) {
    return { disabled, tooltip: "Templates are read-only" };
  }

  if (hasRunBlockingChanges) {
    return { disabled, tooltip: "Save canvas changes before running" };
  }

  return { disabled, tooltip: undefined };
}
