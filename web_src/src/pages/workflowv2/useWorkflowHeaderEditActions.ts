import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface WorkflowHeaderEditActionsConfig {
  isDashboardMode: boolean;
  isMemoryMode: boolean;
  isRunsMode: boolean;
  handleExitDashboardMode: () => void;
  handleExitMemoryMode: () => void;
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
  isRunsMode,
  handleExitDashboardMode,
  handleExitMemoryMode,
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
    if (isRunsMode) {
      setIsRunsMode(false);
      setSelectedRunId(null);
      setRunDetailNodeId(null);
      await handleToggleEditMode();
      setSearchParams(clearRunsViewSearchParams, { replace: true });
      return;
    }
    await handleToggleEditMode();
  }, [
    handleExitDashboardMode,
    handleExitMemoryMode,
    handleToggleEditMode,
    isDashboardMode,
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
    if (isRunsMode) {
      handleExitRunsMode();
      return;
    }
    await handleToggleEditMode();
  }, [
    handleExitDashboardMode,
    handleExitMemoryMode,
    handleExitRunsMode,
    handleToggleEditMode,
    isDashboardMode,
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
