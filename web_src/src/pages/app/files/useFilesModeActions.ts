import { useCallback } from "react";
import type { SetURLSearchParams } from "react-router-dom";

interface FilesModeActionsConfig {
  setIsConsoleAddPanelOpen: (value: boolean) => void;
  setIsConsoleYamlOpen: (value: boolean) => void;
  setSearchParams: SetURLSearchParams;
}

export function useFilesModeActions({
  setIsConsoleAddPanelOpen,
  setIsConsoleYamlOpen,
  setSearchParams,
}: FilesModeActionsConfig) {
  const handleSelectFilesMode = useCallback(() => {
    setIsConsoleAddPanelOpen(false);
    setIsConsoleYamlOpen(false);
    setSearchParams(toFilesSearchParams, { replace: true });
  }, [setIsConsoleAddPanelOpen, setIsConsoleYamlOpen, setSearchParams]);

  const handleExitFilesMode = useCallback(() => {
    setSearchParams(removeFilesSearchParam, { replace: true });
  }, [setSearchParams]);

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
  next.delete("file");
  return next;
}
