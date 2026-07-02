import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface ConsoleModeActionsConfig {
  setIsConsoleAddPanelOpen: (value: boolean) => void;
  setIsConsoleYamlOpen: (value: boolean) => void;
  setSearchParams: SetURLSearchParams;
}

export function useConsoleModeActions({
  setIsConsoleAddPanelOpen,
  setIsConsoleYamlOpen,
  setSearchParams,
}: ConsoleModeActionsConfig) {
  const handleSelectConsoleMode = useCallback(() => {
    setSearchParams(toConsoleSearchParams, { replace: true });
  }, [setSearchParams]);

  const handleExitConsoleMode = useCallback(() => {
    setIsConsoleAddPanelOpen(false);
    setIsConsoleYamlOpen(false);
    setSearchParams(removeConsoleSearchParam, { replace: true });
  }, [setIsConsoleAddPanelOpen, setIsConsoleYamlOpen, setSearchParams]);

  return { handleSelectConsoleMode, handleExitConsoleMode };
}

function toConsoleSearchParams(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.set("view", "console");
  next.delete("run");
  next.delete("sidebar");
  next.delete("node");
  next.delete("file");
  return next;
}

function removeConsoleSearchParam(current: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(current);
  next.delete("view");
  return next;
}
