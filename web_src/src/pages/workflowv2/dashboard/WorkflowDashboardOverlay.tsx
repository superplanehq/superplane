import { DashboardOverlay, type DashboardOverlayProps } from "./DashboardOverlay";

type WorkflowDashboardOverlayProps = Omit<DashboardOverlayProps, "readOnly" | "canImportYaml" | "canRunNodes"> & {
  isDashboardMode: boolean;
  dashboardsFeatureEnabled: boolean;
  canUpdateCanvas: boolean;
  isTemplate: boolean;
  canvasDeletedRemotely: boolean;
  // Hides authoring affordances (panel edit/delete, drag/resize, YAML import)
  // when the app is in read mode.
  editLocked: boolean;
};

export function WorkflowDashboardOverlay({
  isDashboardMode,
  dashboardsFeatureEnabled,
  canUpdateCanvas,
  isTemplate,
  canvasDeletedRemotely,
  editLocked,
  ...dashboardProps
}: WorkflowDashboardOverlayProps) {
  if (!isDashboardMode || !dashboardsFeatureEnabled) return null;

  // Runtime triggers (widget row actions, Node panel Run button) only depend
  // on permission/template state — not on edit mode. The in-flight run lock
  // handles disabling while a run is already executing.
  const runLocked = !canUpdateCanvas || isTemplate || canvasDeletedRemotely;
  return (
    <DashboardOverlay readOnly={editLocked} canImportYaml={!editLocked} canRunNodes={!runLocked} {...dashboardProps} />
  );
}
