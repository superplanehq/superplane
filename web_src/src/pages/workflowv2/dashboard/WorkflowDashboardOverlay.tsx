import { DashboardOverlay, type DashboardOverlayProps } from "./DashboardOverlay";

type WorkflowDashboardOverlayProps = Omit<DashboardOverlayProps, "readOnly" | "canImportYaml" | "canRunNodes"> & {
  isDashboardMode: boolean;
  dashboardsFeatureEnabled: boolean;
  canUpdateCanvas: boolean;
  isTemplate: boolean;
  canvasDeletedRemotely: boolean;
  // Mirrors the workflow page's app-wide read-only state so hover/edit
  // affordances stay hidden when the canvas is not in edit mode.
  isReadOnly: boolean;
};

export function WorkflowDashboardOverlay({
  isDashboardMode,
  dashboardsFeatureEnabled,
  canUpdateCanvas,
  isTemplate,
  canvasDeletedRemotely,
  isReadOnly,
  ...dashboardProps
}: WorkflowDashboardOverlayProps) {
  if (!isDashboardMode || !dashboardsFeatureEnabled) return null;

  // Authoring affordances (panel edit/delete, drag/resize, YAML import) are
  // hidden whenever the app is in read mode. Runtime triggers (widget row
  // actions, Node panel Run button) only depend on permission/template state,
  // not on whether the canvas is in edit mode — the in-flight run lock
  // disables them when a run is already executing.
  const dashboardEditLocked = isReadOnly || !canUpdateCanvas || isTemplate || canvasDeletedRemotely;
  const dashboardRunLocked = !canUpdateCanvas || isTemplate || canvasDeletedRemotely;
  return (
    <DashboardOverlay
      readOnly={dashboardEditLocked}
      canImportYaml={!dashboardEditLocked}
      canRunNodes={!dashboardRunLocked}
      {...dashboardProps}
    />
  );
}
