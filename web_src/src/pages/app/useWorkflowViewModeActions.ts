import { useCallback } from "react";

interface WorkflowViewModeActionsConfig {
  isConsoleMode: boolean;
  isMemoryMode: boolean;
  isFilesMode: boolean;
  isRunInspectionMode: boolean;
  hasEditableVersion: boolean;
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  handleExitConsoleMode: () => void;
  handleExitMemoryMode: () => void;
  handleExitFilesMode: () => void;
  handleClearRunInspection: () => void;
  handleToggleEditMode: () => Promise<boolean>;
  setIsConsoleAddPanelOpen: (value: boolean) => void;
  setIsConsoleYamlOpen: (value: boolean) => void;
}

export function useWorkflowViewModeActions({
  isConsoleMode,
  isMemoryMode,
  isFilesMode,
  isRunInspectionMode,
  hasEditableVersion,
  canUpdateCanvas,
  canvasDeletedRemotely,
  handleExitConsoleMode,
  handleExitMemoryMode,
  handleExitFilesMode,
  handleClearRunInspection,
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
    if (isRunInspectionMode) {
      handleClearRunInspection();
    }
  }, [
    handleClearRunInspection,
    handleExitConsoleMode,
    handleExitFilesMode,
    handleExitMemoryMode,
    isConsoleMode,
    isFilesMode,
    isMemoryMode,
    isRunInspectionMode,
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

  const onConsoleAddPanel = useCallback(() => {
    void handleConsoleAddPanelRequest();
  }, [handleConsoleAddPanelRequest]);

  const onConsoleOpenYaml = useCallback(() => {
    setIsConsoleYamlOpen(true);
  }, [setIsConsoleYamlOpen]);

  const consoleYamlReadOnly = !canUpdateCanvas || canvasDeletedRemotely || !hasEditableVersion;

  return {
    handleSelectCanvasView,
    handleConsoleAddPanelDialogOpenChange,
    onConsoleAddPanel,
    onConsoleOpenYaml,
    consoleYamlReadOnly,
  };
}
