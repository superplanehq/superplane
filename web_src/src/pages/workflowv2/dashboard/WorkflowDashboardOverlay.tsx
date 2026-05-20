import { DashboardOverlay, type DashboardOverlayProps } from "./DashboardOverlay";

type WorkflowDashboardOverlayProps = Omit<DashboardOverlayProps, "readOnly" | "canImportYaml" | "canRunNodes"> & {
  isDashboardMode: boolean;
  dashboardsFeatureEnabled: boolean;
  canUpdateCanvas: boolean;
  isTemplate: boolean;
  canvasDeletedRemotely: boolean;
};

export function WorkflowDashboardOverlay({
  isDashboardMode,
  dashboardsFeatureEnabled,
  canUpdateCanvas,
  isTemplate,
  canvasDeletedRemotely,
  ...dashboardProps
}: WorkflowDashboardOverlayProps) {
  if (!isDashboardMode || !dashboardsFeatureEnabled) return null;

  const dashboardLocked = !canUpdateCanvas || isTemplate || canvasDeletedRemotely;
  return (
    <DashboardOverlay
      readOnly={dashboardLocked}
      canImportYaml={!dashboardLocked}
      canRunNodes={!dashboardLocked}
      {...dashboardProps}
    />
  );
}
