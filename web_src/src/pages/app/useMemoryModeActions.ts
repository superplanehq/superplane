import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface MemoryModeActionsConfig {
  setIsMemoryMode: (value: boolean) => void;
  setIsConsoleMode: (value: boolean) => void;
  setIsConsoleAddPanelOpen: (value: boolean) => void;
  setIsConsoleYamlOpen: (value: boolean) => void;
  setIsRunsMode: (value: boolean) => void;
  setIsVersionsMode: (value: boolean) => void;
  setIsFilesMode: (value: boolean) => void;
  setSelectedRunId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
}

export function useMemoryModeActions({
  setIsMemoryMode,
  setIsConsoleMode,
  setIsConsoleAddPanelOpen,
  setIsConsoleYamlOpen,
  setIsRunsMode,
  setIsVersionsMode,
  setIsFilesMode,
  setSelectedRunId,
  setSearchParams,
}: MemoryModeActionsConfig) {
  const handleSelectMemoryMode = useCallback(() => {
    setIsMemoryMode(true);
    setIsConsoleMode(false);
    setIsConsoleAddPanelOpen(false);
    setIsConsoleYamlOpen(false);
    setIsRunsMode(false);
    setIsVersionsMode(false);
    setIsFilesMode(false);
    setSelectedRunId(null);
    setSearchParams(toMemorySearchParams, { replace: true });
  }, [
    setIsConsoleAddPanelOpen,
    setIsConsoleMode,
    setIsConsoleYamlOpen,
    setIsFilesMode,
    setIsMemoryMode,
    setIsRunsMode,
    setIsVersionsMode,
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
  next.delete("file");
  return next;
}

function removeMemorySearchParam(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("view");
  return next;
}
