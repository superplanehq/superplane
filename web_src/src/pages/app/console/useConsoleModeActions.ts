import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface ConsoleModeActionsConfig {
  setIsConsoleMode: (value: boolean) => void;
  setIsConsoleAddPanelOpen: (value: boolean) => void;
  setIsConsoleYamlOpen: (value: boolean) => void;
  setIsRunsMode: (value: boolean) => void;
  setIsMemoryMode: (value: boolean) => void;
  setIsFilesMode: (value: boolean) => void;
  setSelectedRunId: (value: string | null) => void;
  setSearchParams: SetURLSearchParams;
}

export function useConsoleModeActions({
  setIsConsoleMode,
  setIsConsoleAddPanelOpen,
  setIsConsoleYamlOpen,
  setIsRunsMode,
  setIsMemoryMode,
  setIsFilesMode,
  setSelectedRunId,
  setSearchParams,
}: ConsoleModeActionsConfig) {
  const handleSelectConsoleMode = useCallback(() => {
    setIsConsoleMode(true);
    setIsRunsMode(false);
    setIsMemoryMode(false);
    setIsFilesMode(false);
    setSelectedRunId(null);
    setSearchParams(toConsoleSearchParams, { replace: true });
  }, [setIsConsoleMode, setIsFilesMode, setIsMemoryMode, setIsRunsMode, setSearchParams, setSelectedRunId]);

  const handleExitConsoleMode = useCallback(() => {
    setIsConsoleMode(false);
    setIsConsoleAddPanelOpen(false);
    setIsConsoleYamlOpen(false);
    setSearchParams(removeConsoleSearchParam, { replace: true });
  }, [setIsConsoleAddPanelOpen, setIsConsoleYamlOpen, setIsConsoleMode, setSearchParams]);

  return { handleSelectConsoleMode, handleExitConsoleMode };
}

function toConsoleSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.set("view", "console");
  next.delete("run");
  next.delete("sidebar");
  next.delete("node");
  return next;
}

function removeConsoleSearchParam(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("view");
  return next;
}
