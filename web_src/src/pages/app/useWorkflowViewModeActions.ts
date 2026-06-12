import { useCallback } from "react";

import { getConsoleHeaderActions } from "./console/consoleHeaderActions";

interface WorkflowViewModeActionsConfig {
  isConsoleMode: boolean;
  isMemoryMode: boolean;
  isFilesMode: boolean;
  isRunsMode: boolean;
  isVersionsMode: boolean;
  hasEditableVersion: boolean;
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  handleExitConsoleMode: () => void;
  handleExitMemoryMode: () => void;
  handleExitFilesMode: () => void;
  handleExitRunsMode: () => void;
  handleExitVersionsMode: () => void;
  handleToggleEditMode: () => Promise<void>;
  setIsConsoleAddPanelOpen: (value: boolean) => void;
  setIsConsoleYamlOpen: (value: boolean) => void;
}

export function useWorkflowViewModeActions({
  isConsoleMode,
  isMemoryMode,
  isFilesMode,
  isRunsMode,
  isVersionsMode,
  hasEditableVersion,
  canUpdateCanvas,
  canvasDeletedRemotely,
  handleExitConsoleMode,
  handleExitMemoryMode,
  handleExitFilesMode,
  handleExitRunsMode,
  handleExitVersionsMode,
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
      return;
    }
    if (isVersionsMode) {
      handleExitVersionsMode();
    }
  }, [
    handleExitConsoleMode,
    handleExitFilesMode,
    handleExitMemoryMode,
    handleExitRunsMode,
    handleExitVersionsMode,
    isConsoleMode,
    isFilesMode,
    isMemoryMode,
    isRunsMode,
    isVersionsMode,
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
      canUpdateCanvas,
      canvasDeletedRemotely,
      openAddPanel: () => void handleConsoleAddPanelRequest(),
      openYaml: () => setIsConsoleYamlOpen(true),
    }),
  };
}
