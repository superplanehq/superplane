import { useCallback } from "react";

import { getDashboardHeaderActions } from "./dashboard/dashboardHeaderActions";

interface WorkflowViewModeActionsConfig {
  isDashboardMode: boolean;
  isMemoryMode: boolean;
  isRunsMode: boolean;
  hasEditableVersion: boolean;
  dashboardsFeatureEnabled: boolean;
  isTemplate: boolean;
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  handleExitDashboardMode: () => void;
  handleExitMemoryMode: () => void;
  handleExitRunsMode: () => void;
  handleToggleEditMode: () => Promise<void>;
  setIsDashboardAddPanelOpen: (value: boolean) => void;
  setIsDashboardYamlOpen: (value: boolean) => void;
}

export function useWorkflowViewModeActions({
  isDashboardMode,
  isMemoryMode,
  isRunsMode,
  hasEditableVersion,
  dashboardsFeatureEnabled,
  isTemplate,
  canUpdateCanvas,
  canvasDeletedRemotely,
  handleExitDashboardMode,
  handleExitMemoryMode,
  handleExitRunsMode,
  handleToggleEditMode,
  setIsDashboardAddPanelOpen,
  setIsDashboardYamlOpen,
}: WorkflowViewModeActionsConfig) {
  const handleSelectCanvasView = useCallback(() => {
    if (isDashboardMode) {
      handleExitDashboardMode();
      return;
    }
    if (isMemoryMode) {
      handleExitMemoryMode();
      return;
    }
    if (isRunsMode) {
      handleExitRunsMode();
    }
  }, [handleExitDashboardMode, handleExitMemoryMode, handleExitRunsMode, isDashboardMode, isMemoryMode, isRunsMode]);

  const handleDashboardAddPanelRequest = useCallback(async () => {
    if (!hasEditableVersion) {
      await handleToggleEditMode();
    }
    setIsDashboardAddPanelOpen(true);
  }, [hasEditableVersion, handleToggleEditMode, setIsDashboardAddPanelOpen]);

  const handleDashboardAddPanelDialogOpenChange = useCallback(
    (open: boolean) => {
      if (open) {
        void handleDashboardAddPanelRequest();
        return;
      }
      setIsDashboardAddPanelOpen(false);
    },
    [handleDashboardAddPanelRequest, setIsDashboardAddPanelOpen],
  );

  return {
    handleSelectCanvasView,
    handleDashboardAddPanelRequest,
    handleDashboardAddPanelDialogOpenChange,
    ...getDashboardHeaderActions({
      isEditing: hasEditableVersion,
      isDashboardMode,
      dashboardsFeatureEnabled,
      isTemplate,
      canUpdateCanvas,
      canvasDeletedRemotely,
      openAddPanel: () => void handleDashboardAddPanelRequest(),
      openYaml: () => setIsDashboardYamlOpen(true),
    }),
  };
}
