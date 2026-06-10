import { useCallback, useEffect, useRef, useState } from "react";
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
 * Keeps console UI state and selected run in sync with `view` and `run` search params.
 * View mode flags are derived directly from the URL so they match header chrome on the
 * same render (including browser back/forward), without waiting for a post-render effect.
 */
export function useWorkflowViewSearchParams(searchParams: URLSearchParams, setSearchParams: SetURLSearchParams) {
  const viewParam = searchParams.get("view") ?? "";
  const runParam = searchParams.get("run") ?? "";
  const consoleViewActive = isConsoleView(viewParam);

  const isRunsMode = viewParam === "runs";
  const isVersionsMode = viewParam === "versions";
  const isMemoryMode = viewParam === "memory";
  const isFilesMode = viewParam === "files";
  const isConsoleMode = consoleViewActive;
  const selectedRunId = runParam || null;

  const [isConsoleAddPanelOpen, setIsConsoleAddPanelOpen] = useState(false);
  const [isConsoleYamlOpen, setIsConsoleYamlOpen] = useState(false);

  const setSearchParamsRef = useRef(setSearchParams);
  setSearchParamsRef.current = setSearchParams;

  useEffect(() => {
    if (consoleViewActive) {
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
      setIsConsoleAddPanelOpen(false);
      setIsConsoleYamlOpen(false);
    }
  }, [viewParam, consoleViewActive]);

  const noopSetBoolean = useCallback((_value: boolean) => {}, []);
  const noopSetSelectedRunId = useCallback((_value: string | null) => {}, []);

  return {
    isRunsMode,
    setIsRunsMode: noopSetBoolean,
    isVersionsMode,
    setIsVersionsMode: noopSetBoolean,
    isConsoleMode,
    setIsConsoleMode: noopSetBoolean,
    isMemoryMode,
    setIsMemoryMode: noopSetBoolean,
    isFilesMode,
    setIsFilesMode: noopSetBoolean,
    isConsoleAddPanelOpen,
    setIsConsoleAddPanelOpen,
    isConsoleYamlOpen,
    setIsConsoleYamlOpen,
    selectedRunId,
    setSelectedRunId: noopSetSelectedRunId,
  };
}
