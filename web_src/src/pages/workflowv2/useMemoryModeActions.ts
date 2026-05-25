import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface MemoryModeActionsConfig {
  setIsMemoryMode: (value: boolean) => void;
  setIsDashboardMode: (value: boolean) => void;
  setIsDashboardAddPanelOpen: (value: boolean) => void;
  setIsDashboardYamlOpen: (value: boolean) => void;
  setIsRunsMode: (value: boolean) => void;
  setSelectedRunId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
}

export function useMemoryModeActions({
  setIsMemoryMode,
  setIsDashboardMode,
  setIsDashboardAddPanelOpen,
  setIsDashboardYamlOpen,
  setIsRunsMode,
  setSelectedRunId,
  setSearchParams,
}: MemoryModeActionsConfig) {
  const handleSelectMemoryMode = useCallback(() => {
    setIsMemoryMode(true);
    setIsDashboardMode(false);
    setIsDashboardAddPanelOpen(false);
    setIsDashboardYamlOpen(false);
    setIsRunsMode(false);
    setSelectedRunId(null);
    setSearchParams(toMemorySearchParams, { replace: true });
  }, [
    setIsDashboardAddPanelOpen,
    setIsDashboardMode,
    setIsDashboardYamlOpen,
    setIsMemoryMode,
    setIsRunsMode,
    setSearchParams,
    setSelectedRunId,
  ]);

  const handleExitMemoryMode = useCallback(() => {
    setIsMemoryMode(false);
    setSearchParams(removeMemorySearchParam, { replace: true });
  }, [setIsMemoryMode, setSearchParams]);

  return { handleSelectMemoryMode, handleExitMemoryMode };
}

function toMemorySearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.set("view", "memory");
  next.delete("run");
  next.delete("sidebar");
  next.delete("node");
  return next;
}

function removeMemorySearchParam(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("view");
  return next;
}
