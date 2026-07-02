import { useCallback, useState } from "react";

const CANVAS_RUNS_SIDEBAR_OPEN_STORAGE_KEY = "canvasRunsSidebarOpen";

function readInitialRunsSidebarOpen(): boolean {
  if (typeof window === "undefined") return true;
  try {
    const saved = window.localStorage.getItem(CANVAS_RUNS_SIDEBAR_OPEN_STORAGE_KEY);
    if (saved === null) return true;
    return saved === "true";
  } catch {
    return true;
  }
}

export function useCanvasRunsSidebarState() {
  const [isRunsSidebarOpen, setIsRunsSidebarOpen] = useState(readInitialRunsSidebarOpen);

  const persistOpen = useCallback((open: boolean) => {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(CANVAS_RUNS_SIDEBAR_OPEN_STORAGE_KEY, open ? "true" : "false");
  }, []);

  const handleRunsSidebarToggle = useCallback(() => {
    setIsRunsSidebarOpen((current) => {
      const next = !current;
      persistOpen(next);
      return next;
    });
  }, [persistOpen]);

  const openRunsSidebar = useCallback(() => {
    setIsRunsSidebarOpen(true);
    persistOpen(true);
  }, [persistOpen]);

  const closeRunsSidebar = useCallback(() => {
    setIsRunsSidebarOpen(false);
    persistOpen(false);
  }, [persistOpen]);

  return {
    isRunsSidebarOpen,
    handleRunsSidebarToggle,
    openRunsSidebar,
    closeRunsSidebar,
  };
}

export type CanvasRunsSidebarState = ReturnType<typeof useCanvasRunsSidebarState> & {
  showRunsSidebarToggle: boolean;
  /** Number of currently running runs, surfaced as an animated badge on the toggle. */
  runningRunsCount?: number;
};
