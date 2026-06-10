import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface VersionsModeActionsConfig {
  setIsVersionsMode: (value: boolean) => void;
  setIsConsoleMode: (value: boolean) => void;
  setIsConsoleAddPanelOpen: (value: boolean) => void;
  setIsConsoleYamlOpen: (value: boolean) => void;
  setIsRunsMode: (value: boolean) => void;
  setIsMemoryMode: (value: boolean) => void;
  setIsFilesMode: (value: boolean) => void;
  setSelectedRunId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
}

export function useVersionsModeActions({
  setIsVersionsMode,
  setIsConsoleMode,
  setIsConsoleAddPanelOpen,
  setIsConsoleYamlOpen,
  setIsRunsMode,
  setIsMemoryMode,
  setIsFilesMode,
  setSelectedRunId,
  setSearchParams,
}: VersionsModeActionsConfig) {
  const handleSelectVersionsMode = useCallback(() => {
    setIsVersionsMode(true);
    setIsConsoleMode(false);
    setIsConsoleAddPanelOpen(false);
    setIsConsoleYamlOpen(false);
    setIsRunsMode(false);
    setIsMemoryMode(false);
    setIsFilesMode(false);
    setSelectedRunId(null);
    setSearchParams(toVersionsSearchParams, { replace: true });
  }, [
    setIsConsoleAddPanelOpen,
    setIsConsoleMode,
    setIsConsoleYamlOpen,
    setIsFilesMode,
    setIsMemoryMode,
    setIsRunsMode,
    setIsVersionsMode,
    setSelectedRunId,
    setSearchParams,
  ]);

  const handleExitVersionsMode = useCallback(() => {
    setIsVersionsMode(false);
    setSearchParams(removeVersionsSearchParam, { replace: true });
  }, [setIsVersionsMode, setSearchParams]);

  return { handleSelectVersionsMode, handleExitVersionsMode };
}

function toVersionsSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.set("view", "versions");
  next.delete("run");
  next.delete("sidebar");
  next.delete("node");
  next.delete("file");
  return next;
}

function removeVersionsSearchParam(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("view");
  return next;
}
