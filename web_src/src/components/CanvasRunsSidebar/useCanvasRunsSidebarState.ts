import { useCallback, useEffect, useRef, useState } from "react";

const CANVAS_RUNS_SIDEBAR_OPEN_STORAGE_KEY = "canvasRunsSidebarOpen";

function canvasRunsSidebarOpenStorageKey(canvasId?: string): string {
  return canvasId ? `${CANVAS_RUNS_SIDEBAR_OPEN_STORAGE_KEY}:${canvasId}` : CANVAS_RUNS_SIDEBAR_OPEN_STORAGE_KEY;
}

function readInitialRunsSidebarOpen(canvasId?: string): boolean {
  if (typeof window === "undefined") return false;
  try {
    const saved = window.localStorage.getItem(canvasRunsSidebarOpenStorageKey(canvasId));
    if (saved === null) return false;
    return saved === "true";
  } catch {
    return false;
  }
}

export function useCanvasRunsSidebarState(canvasId?: string) {
  const [isRunsSidebarOpen, setIsRunsSidebarOpen] = useState(() => readInitialRunsSidebarOpen(canvasId));

  const previousCanvasIdRef = useRef(canvasId);
  useEffect(() => {
    if (previousCanvasIdRef.current === canvasId) return;
    previousCanvasIdRef.current = canvasId;
    setIsRunsSidebarOpen(readInitialRunsSidebarOpen(canvasId));
  }, [canvasId]);

  const persistOpen = useCallback(
    (open: boolean) => {
      if (typeof window === "undefined") return;
      window.localStorage.setItem(canvasRunsSidebarOpenStorageKey(canvasId), open ? "true" : "false");
    },
    [canvasId],
  );

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
};
