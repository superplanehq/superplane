import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface WorkflowHeaderEditActionsConfig {
  isRunsMode: boolean;
  handleExitRunsMode: () => void;
  handleToggleEditMode: () => Promise<void>;
  setIsRunsMode: (value: boolean) => void;
  setSelectedRunId: (value: string | null) => void;
  setRunDetailNodeId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
}

export function useWorkflowHeaderEditActions({
  isRunsMode,
  handleExitRunsMode,
  handleToggleEditMode,
  setIsRunsMode,
  setSelectedRunId,
  setRunDetailNodeId,
  setSearchParams,
}: WorkflowHeaderEditActionsConfig) {
  const handleEnterEditModeFromHeader = useCallback(async () => {
    if (isRunsMode) {
      setIsRunsMode(false);
      setSelectedRunId(null);
      setRunDetailNodeId(null);
      setSearchParams(clearRunsViewSearchParams, { replace: true });
    }

    await handleToggleEditMode();
  }, [handleToggleEditMode, isRunsMode, setIsRunsMode, setRunDetailNodeId, setSearchParams, setSelectedRunId]);

  const handleExitEditModeFromHeader = useCallback(async () => {
    if (isRunsMode) {
      handleExitRunsMode();
      return;
    }
    await handleToggleEditMode();
  }, [handleExitRunsMode, handleToggleEditMode, isRunsMode]);

  return { handleEnterEditModeFromHeader, handleExitEditModeFromHeader };
}

function clearRunsViewSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("view");
  next.delete("run");
  return next;
}
