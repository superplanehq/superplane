import { useEffect, useRef, useState } from "react";
import type { SetURLSearchParams } from "react-router-dom";

/**
 * The user-facing console feature is keyed by `?view=console` in the URL.
 * The legacy value `?view=dashboard` is still accepted (for old bookmarks /
 * in-flight links) and silently rewritten to the canonical `console` value.
 */
const CONSOLE_VIEW = "console";
const LEGACY_CONSOLE_VIEW = "console";

function isConsoleView(view: string): boolean {
  return view === CONSOLE_VIEW || view === LEGACY_CONSOLE_VIEW;
}

/**
 * Keeps runs/console/memory/files view mode and selected run in sync with `view` and `run` search params.
 */
export function useWorkflowViewSearchParams(searchParams: URLSearchParams, setSearchParams: SetURLSearchParams) {
  const [isRunsMode, setIsRunsMode] = useState(() => searchParams.get("view") === "runs");
  const [isConsoleMode, setIsConsoleMode] = useState(() => isConsoleView(searchParams.get("view") ?? ""));
  const [isMemoryMode, setIsMemoryMode] = useState(() => searchParams.get("view") === "memory");
  const [isFilesMode, setIsFilesMode] = useState(() => searchParams.get("view") === "files");
  const [isConsoleAddPanelOpen, setIsConsoleAddPanelOpen] = useState(false);
  const [isConsoleYamlOpen, setIsConsoleYamlOpen] = useState(false);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(() => searchParams.get("run"));

  const viewParam = searchParams.get("view") ?? "";
  const runParam = searchParams.get("run") ?? "";
  const consoleViewActive = isConsoleView(viewParam);

  const setSearchParamsRef = useRef(setSearchParams);
  setSearchParamsRef.current = setSearchParams;

  useEffect(() => {
    setIsRunsMode(viewParam === "runs");
    setIsMemoryMode(viewParam === "memory");
    setIsFilesMode(viewParam === "files");
    if (consoleViewActive) {
      setIsConsoleMode(true);
      // Migrate legacy `?view=dashboard` to the canonical `?view=console`
      // in-place so the address bar and any future link sharing reflect
      // the renamed feature without breaking existing bookmarks.
      if (viewParam === LEGACY_CONSOLE_VIEW) {
        setSearchParamsRef.current(
          (current) => {
            const next = new URLSearchParams(current);
            if (next.get("view") !== LEGACY_CONSOLE_VIEW) {
              return current;
            }
            next.set("view", CONSOLE_VIEW);
            return next;
          },
          { replace: true },
        );
      }
    } else {
      setIsConsoleMode(false);
    }
    setSelectedRunId(runParam || null);
    if (!consoleViewActive) {
      setIsConsoleAddPanelOpen(false);
      setIsConsoleYamlOpen(false);
    }
  }, [viewParam, runParam, consoleViewActive]);

  return {
    isRunsMode,
    setIsRunsMode,
    isConsoleMode,
    setIsConsoleMode,
    isMemoryMode,
    setIsMemoryMode,
    isFilesMode,
    setIsFilesMode,
    isConsoleAddPanelOpen,
    setIsConsoleAddPanelOpen,
    isConsoleYamlOpen,
    setIsConsoleYamlOpen,
    selectedRunId,
    setSelectedRunId,
  };
}
