import { useCallback, useEffect, useRef, useState } from "react";
import type { SetURLSearchParams } from "react-router-dom";
import { isWorkflowCanvasViewParam } from "./viewState";

/**
 * The user-facing console feature is keyed by `?view=console` in the URL.
 * The legacy value `?view=dashboard` is still accepted (for old bookmarks /
 * in-flight links) and silently rewritten to the canonical `console` value.
 */
const CONSOLE_VIEW = "console";
const LEGACY_CONSOLE_VIEW = "dashboard";
const LEGACY_RUNS_VIEW = "runs";
const LEGACY_VERSIONS_VIEW = "versions";

function isConsoleView(view: string): boolean {
  return view === CONSOLE_VIEW || view === LEGACY_CONSOLE_VIEW;
}

function migrateLegacyViewParams(view: string, params: URLSearchParams): URLSearchParams | null {
  if (view === LEGACY_CONSOLE_VIEW) {
    const next = new URLSearchParams(params);
    next.set("view", CONSOLE_VIEW);
    return next;
  }

  if (view === LEGACY_RUNS_VIEW) {
    const next = new URLSearchParams(params);
    next.delete("view");
    return next;
  }

  if (view === LEGACY_VERSIONS_VIEW) {
    const next = new URLSearchParams(params);
    next.delete("view");
    return next;
  }

  return null;
}

function migrateConflictingRunParam(view: string, params: URLSearchParams): URLSearchParams | null {
  if (!params.get("run")) {
    return null;
  }

  if (isWorkflowCanvasViewParam(view)) {
    return null;
  }

  const next = new URLSearchParams(params);
  next.delete("view");
  return next;
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

  const isMemoryMode = viewParam === "memory";
  const isFilesMode = viewParam === "files";
  const isConsoleMode = consoleViewActive;
  const isRunInspectionMode = Boolean(runParam) && isWorkflowCanvasViewParam(viewParam);
  const selectedRunId = isRunInspectionMode ? runParam : null;

  const [isConsoleAddPanelOpen, setIsConsoleAddPanelOpen] = useState(false);
  const [isConsoleYamlOpen, setIsConsoleYamlOpen] = useState(false);

  const setSearchParamsRef = useRef(setSearchParams);
  setSearchParamsRef.current = setSearchParams;

  useEffect(() => {
    const migrated = migrateLegacyViewParams(viewParam, searchParams);
    if (migrated) {
      setSearchParamsRef.current(migrated, { replace: true });
      return;
    }

    const runMigrated = migrateConflictingRunParam(viewParam, searchParams);
    if (runMigrated) {
      setSearchParamsRef.current(runMigrated, { replace: true });
      return;
    }

    if (consoleViewActive) {
      return;
    }

    setIsConsoleAddPanelOpen(false);
    setIsConsoleYamlOpen(false);
  }, [viewParam, consoleViewActive, searchParams]);

  const noopSetBoolean = useCallback((_value: boolean) => {}, []);
  const noopSetSelectedRunId = useCallback((_value: string | null) => {}, []);

  return {
    isRunInspectionMode,
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
