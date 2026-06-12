import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface VersionsModeActionsConfig {
  setIsConsoleAddPanelOpen: (value: boolean) => void;
  setIsConsoleYamlOpen: (value: boolean) => void;
  setSearchParams: SetURLSearchParams;
}

export function useVersionsModeActions({
  setIsConsoleAddPanelOpen,
  setIsConsoleYamlOpen,
  setSearchParams,
}: VersionsModeActionsConfig) {
  const handleSelectVersionsMode = useCallback(() => {
    setIsConsoleAddPanelOpen(false);
    setIsConsoleYamlOpen(false);
    setSearchParams(toVersionsSearchParams, { replace: true });
  }, [setIsConsoleAddPanelOpen, setIsConsoleYamlOpen, setSearchParams]);

  const handleExitVersionsMode = useCallback(() => {
    setSearchParams(removeVersionsSearchParam, { replace: true });
  }, [setSearchParams]);

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
