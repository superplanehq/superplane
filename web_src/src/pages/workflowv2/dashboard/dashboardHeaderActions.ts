interface DashboardHeaderActionsConfig {
  isEditing: boolean;
  isDashboardMode: boolean;
  isTemplate: boolean;
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  openAddPanel: () => void;
  openYaml: () => void;
}

export function getDashboardHeaderActions({
  isEditing,
  isDashboardMode,
  isTemplate,
  canUpdateCanvas,
  canvasDeletedRemotely,
  openAddPanel,
  openYaml,
}: DashboardHeaderActionsConfig) {
  const dashboardVisible = isDashboardMode;
  const canEditDashboard = isEditing && dashboardVisible && !isTemplate && canUpdateCanvas && !canvasDeletedRemotely;

  return {
    onDashboardAddPanel: canEditDashboard ? openAddPanel : undefined,
    onDashboardOpenYaml: isEditing && dashboardVisible ? openYaml : undefined,
    dashboardYamlReadOnly: !canUpdateCanvas || isTemplate || canvasDeletedRemotely,
  };
}
