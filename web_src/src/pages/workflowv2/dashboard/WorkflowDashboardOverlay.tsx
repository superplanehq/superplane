import { DashboardOverlay, type DashboardOverlayProps } from "./DashboardOverlay";

type WorkflowDashboardOverlayProps = Omit<DashboardOverlayProps, "readOnly" | "canImportYaml" | "canRunNodes"> & {
  isDashboardMode: boolean;
  canActOnCanvas: boolean;
  // Hides authoring affordances (panel edit/delete, drag/resize, YAML import)
  // when the app is in read mode.
  editLocked: boolean;
};

export function WorkflowDashboardOverlay({
  isDashboardMode,
  canActOnCanvas,
  editLocked,
  ...dashboardProps
}: WorkflowDashboardOverlayProps) {
  if (!isDashboardMode) return null;

  // Runtime triggers (widget row actions, Node panel Run button) only depend
  // on permission/template state — not on edit mode. The in-flight run lock
  // handles disabling while a run is already executing.
  const runLocked = !canActOnCanvas;
  return (
    <DashboardOverlay readOnly={editLocked} canImportYaml={!editLocked} canRunNodes={!runLocked} {...dashboardProps} />
  );
}
