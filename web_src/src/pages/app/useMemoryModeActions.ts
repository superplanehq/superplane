import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface MemoryModeActionsConfig {
  setIsConsoleAddPanelOpen: (value: boolean) => void;
  setIsConsoleYamlOpen: (value: boolean) => void;
  setSearchParams: SetURLSearchParams;
}

export function useMemoryModeActions({
  setIsConsoleAddPanelOpen,
  setIsConsoleYamlOpen,
  setSearchParams,
}: MemoryModeActionsConfig) {
  const handleSelectMemoryMode = useCallback(() => {
    setIsConsoleAddPanelOpen(false);
    setIsConsoleYamlOpen(false);
    setSearchParams(toMemorySearchParams, { replace: true });
  }, [setIsConsoleAddPanelOpen, setIsConsoleYamlOpen, setSearchParams]);

  const handleExitMemoryMode = useCallback(() => {
    setSearchParams(removeMemorySearchParam, { replace: true });
  }, [setSearchParams]);

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
