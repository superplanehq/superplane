import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface DashboardModeActionsConfig {
  dashboardsFeatureEnabled: boolean;
  setIsDashboardMode: (value: boolean) => void;
  setIsDashboardAddPanelOpen: (value: boolean) => void;
  setIsDashboardYamlOpen: (value: boolean) => void;
  setIsRunsMode: (value: boolean) => void;
  setIsMemoryMode: (value: boolean) => void;
  setIsFilesMode: (value: boolean) => void;
  setSelectedRunId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
}

export function useDashboardModeActions({
  dashboardsFeatureEnabled,
  setIsDashboardMode,
  setIsDashboardAddPanelOpen,
  setIsDashboardYamlOpen,
  setIsRunsMode,
  setIsMemoryMode,
  setIsFilesMode,
  setSelectedRunId,
  setSearchParams,
}: DashboardModeActionsConfig) {
  const handleSelectDashboardMode = useCallback(() => {
    if (!dashboardsFeatureEnabled) return;

    setIsDashboardMode(true);
    setIsRunsMode(false);
    setIsMemoryMode(false);
    setIsFilesMode(false);
    setSelectedRunId(null);
    setSearchParams(toDashboardSearchParams, { replace: true });
  }, [
    dashboardsFeatureEnabled,
    setIsDashboardMode,
    setIsFilesMode,
    setIsMemoryMode,
    setIsRunsMode,
    setSearchParams,
    setSelectedRunId,
  ]);

  const handleExitDashboardMode = useCallback(() => {
    setIsDashboardMode(false);
    setIsDashboardAddPanelOpen(false);
    setIsDashboardYamlOpen(false);
    setSearchParams(removeDashboardSearchParam, { replace: true });
  }, [setIsDashboardAddPanelOpen, setIsDashboardYamlOpen, setIsDashboardMode, setSearchParams]);

  return { handleSelectDashboardMode, handleExitDashboardMode };
}

function toDashboardSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.set("view", "console");
  next.delete("run");
  next.delete("sidebar");
  next.delete("node");
  return next;
}

function removeDashboardSearchParam(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("view");
  return next;
}
