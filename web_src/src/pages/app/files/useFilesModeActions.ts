import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface FilesModeActionsConfig {
  setIsFilesMode: (value: boolean) => void;
  setIsConsoleMode: (value: boolean) => void;
  setIsConsoleAddPanelOpen: (value: boolean) => void;
  setIsConsoleYamlOpen: (value: boolean) => void;
  setIsRunsMode: (value: boolean) => void;
  setIsMemoryMode: (value: boolean) => void;
  setSelectedRunId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
}

export function useFilesModeActions({
  setIsFilesMode,
  setIsConsoleMode,
  setIsConsoleAddPanelOpen,
  setIsConsoleYamlOpen,
  setIsRunsMode,
  setIsMemoryMode,
  setSelectedRunId,
  setSearchParams,
}: FilesModeActionsConfig) {
  const handleSelectFilesMode = useCallback(() => {
    setIsFilesMode(true);
    setIsConsoleMode(false);
    setIsConsoleAddPanelOpen(false);
    setIsConsoleYamlOpen(false);
    setIsRunsMode(false);
    setIsMemoryMode(false);
    setSelectedRunId(null);
    setSearchParams(toFilesSearchParams, { replace: true });
  }, [
    setIsConsoleAddPanelOpen,
    setIsConsoleMode,
    setIsConsoleYamlOpen,
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
