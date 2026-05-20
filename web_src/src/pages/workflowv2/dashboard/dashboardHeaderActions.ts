interface DashboardHeaderActionsConfig {
  isDashboardMode: boolean;
  dashboardsFeatureEnabled: boolean;
  isTemplate: boolean;
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  openAddPanel: () => void;
  openYaml: () => void;
}

export function getDashboardHeaderActions({
  isDashboardMode,
  dashboardsFeatureEnabled,
  isTemplate,
  canUpdateCanvas,
  canvasDeletedRemotely,
  openAddPanel,
  openYaml,
}: DashboardHeaderActionsConfig) {
  const dashboardVisible = isDashboardMode && dashboardsFeatureEnabled;
  const canEditDashboard = dashboardVisible && !isTemplate && canUpdateCanvas && !canvasDeletedRemotely;

  return {
    onDashboardAddPanel: canEditDashboard ? openAddPanel : undefined,
    onDashboardOpenYaml: dashboardVisible ? openYaml : undefined,
    dashboardYamlReadOnly: !canUpdateCanvas || isTemplate || canvasDeletedRemotely,
  };
}
