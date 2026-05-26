import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface WorkflowHeaderEditActionsConfig {
  isDashboardMode: boolean;
  isMemoryMode: boolean;
  isFilesMode: boolean;
  isRunsMode: boolean;
  handleExitDashboardMode: () => void;
  handleExitMemoryMode: () => void;
  handleExitFilesMode: () => void;
  handleExitRunsMode: () => void;
  handleToggleEditMode: () => Promise<void>;
  setIsRunsMode: (value: boolean) => void;
  setSelectedRunId: (value: string | null) => void;
  setRunDetailNodeId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
}

export function useWorkflowHeaderEditActions({
  isDashboardMode,
  isMemoryMode,
  isFilesMode,
  isRunsMode,
  handleExitDashboardMode,
  handleExitMemoryMode,
  handleExitFilesMode,
  handleExitRunsMode,
  handleToggleEditMode,
  setIsRunsMode,
  setSelectedRunId,
  setRunDetailNodeId,
  setSearchParams,
}: WorkflowHeaderEditActionsConfig) {
  const handleEnterEditModeFromHeader = useCallback(async () => {
    if (isDashboardMode) {
      handleExitDashboardMode();
      await handleToggleEditMode();
      return;
    }
    if (isMemoryMode) {
      handleExitMemoryMode();
      await handleToggleEditMode();
      return;
    }
    if (isFilesMode) {
      handleExitFilesMode();
      await handleToggleEditMode();
      return;
    }
    if (isRunsMode) {
      setIsRunsMode(false);
      setSelectedRunId(null);
      setRunDetailNodeId(null);
      setSearchParams(clearRunsViewSearchParams, { replace: true });
    }

    await handleToggleEditMode();
  }, [
    handleExitDashboardMode,
    handleExitFilesMode,
    handleExitMemoryMode,
    handleToggleEditMode,
    isDashboardMode,
    isFilesMode,
    isMemoryMode,
    isRunsMode,
    setIsRunsMode,
    setRunDetailNodeId,
    setSearchParams,
    setSelectedRunId,
  ]);

  const handleExitEditModeFromHeader = useCallback(async () => {
    if (isDashboardMode) {
      handleExitDashboardMode();
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
    await handleToggleEditMode();
  }, [
    handleExitDashboardMode,
    handleExitFilesMode,
    handleExitMemoryMode,
    handleExitRunsMode,
    handleToggleEditMode,
    isDashboardMode,
    isFilesMode,
    isMemoryMode,
    isRunsMode,
  ]);

  return { handleEnterEditModeFromHeader, handleExitEditModeFromHeader };
}

function clearRunsViewSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("view");
  next.delete("run");
  return next;
}
