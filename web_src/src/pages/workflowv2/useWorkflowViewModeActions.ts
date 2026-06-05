import { useCallback } from "react";

import { getConsoleHeaderActions } from "./console/consoleHeaderActions";

interface WorkflowViewModeActionsConfig {
  isConsoleMode: boolean;
  isMemoryMode: boolean;
  isFilesMode: boolean;
  isRunsMode: boolean;
  hasEditableVersion: boolean;
  isTemplate: boolean;
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  handleExitConsoleMode: () => void;
  handleExitMemoryMode: () => void;
  handleExitFilesMode: () => void;
  handleExitRunsMode: () => void;
  handleToggleEditMode: () => Promise<void>;
  setIsConsoleAddPanelOpen: (value: boolean) => void;
  setIsConsoleYamlOpen: (value: boolean) => void;
}

export function useWorkflowViewModeActions({
  isConsoleMode,
  isMemoryMode,
  isFilesMode,
  isRunsMode,
  hasEditableVersion,
  isTemplate,
  canUpdateCanvas,
  canvasDeletedRemotely,
  handleExitConsoleMode,
  handleExitMemoryMode,
  handleExitFilesMode,
  handleExitRunsMode,
  handleToggleEditMode,
  setIsConsoleAddPanelOpen,
  setIsConsoleYamlOpen,
}: WorkflowViewModeActionsConfig) {
  const handleSelectCanvasView = useCallback(() => {
    if (isConsoleMode) {
      handleExitConsoleMode();
      return;
    }
    if (isMemoryMode) {
      handleExitMemoryMode();
      return;
    }
    if (isFilesMode) {
      handleExitFilesMode();
      return;
    }
    if (isRunsMode) {
      handleExitRunsMode();
    }
  }, [
    handleExitConsoleMode,
    handleExitFilesMode,
    handleExitMemoryMode,
    handleExitRunsMode,
    isConsoleMode,
    isFilesMode,
    isMemoryMode,
    isRunsMode,
  ]);

  const handleConsoleAddPanelRequest = useCallback(async () => {
    if (!hasEditableVersion) {
      await handleToggleEditMode();
    }
    setIsConsoleAddPanelOpen(true);
  }, [hasEditableVersion, handleToggleEditMode, setIsConsoleAddPanelOpen]);

  const handleConsoleAddPanelDialogOpenChange = useCallback(
    (open: boolean) => {
      if (open) {
        void handleConsoleAddPanelRequest();
        return;
      }
      setIsConsoleAddPanelOpen(false);
    },
    [handleConsoleAddPanelRequest, setIsConsoleAddPanelOpen],
  );

  return {
    handleSelectCanvasView,
    handleConsoleAddPanelRequest,
    handleConsoleAddPanelDialogOpenChange,
    ...getConsoleHeaderActions({
      isEditing: hasEditableVersion,
      isConsoleMode,
      isTemplate,
      canUpdateCanvas,
      canvasDeletedRemotely,
      openAddPanel: () => void handleConsoleAddPanelRequest(),
      openYaml: () => setIsConsoleYamlOpen(true),
    }),
  };
}
