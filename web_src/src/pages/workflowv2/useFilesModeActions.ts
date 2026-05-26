import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface FilesModeActionsConfig {
  setIsFilesMode: (value: boolean) => void;
  setIsDashboardMode: (value: boolean) => void;
  setIsDashboardAddPanelOpen: (value: boolean) => void;
  setIsDashboardYamlOpen: (value: boolean) => void;
  setIsRunsMode: (value: boolean) => void;
  setIsMemoryMode: (value: boolean) => void;
  setSelectedRunId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
}

export function useFilesModeActions({
  setIsFilesMode,
  setIsDashboardMode,
  setIsDashboardAddPanelOpen,
  setIsDashboardYamlOpen,
  setIsRunsMode,
  setIsMemoryMode,
  setSelectedRunId,
  setSearchParams,
}: FilesModeActionsConfig) {
  const handleSelectFilesMode = useCallback(() => {
    setIsFilesMode(true);
    setIsDashboardMode(false);
    setIsDashboardAddPanelOpen(false);
    setIsDashboardYamlOpen(false);
    setIsRunsMode(false);
    setIsMemoryMode(false);
    setSelectedRunId(null);
    setSearchParams(toFilesSearchParams, { replace: true });
  }, [
    setIsDashboardAddPanelOpen,
    setIsDashboardMode,
    setIsDashboardYamlOpen,
    setIsFilesMode,
    setIsMemoryMode,
    setIsRunsMode,
    setSearchParams,
    setSelectedRunId,
  ]);

  const handleExitFilesMode = useCallback(() => {
    setIsFilesMode(false);
    setSearchParams(removeFilesSearchParam, { replace: true });
  }, [setIsFilesMode, setSearchParams]);

  return { handleSelectFilesMode, handleExitFilesMode };
}

function toFilesSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.set("view", "files");
  next.delete("run");
  next.delete("sidebar");
  next.delete("node");
  return next;
}

function removeFilesSearchParam(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("view");
  return next;
}
