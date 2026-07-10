import { useCallback, useEffect, useRef, useState } from "react";

const CANVAS_RUNS_SIDEBAR_OPEN_STORAGE_KEY = "canvasRunsSidebarOpen";

function canvasRunsSidebarOpenStorageKey(canvasId?: string): string {
  return canvasId ? `${CANVAS_RUNS_SIDEBAR_OPEN_STORAGE_KEY}:${canvasId}` : CANVAS_RUNS_SIDEBAR_OPEN_STORAGE_KEY;
}

export function writeCanvasRunsSidebarOpen(canvasId: string, open: boolean): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(canvasRunsSidebarOpenStorageKey(canvasId), open ? "true" : "false");
  } catch {
    // Preference persistence is best-effort.
  }
}

function readInitialRunsSidebarOpen(canvasId?: string): boolean {
  if (typeof window === "undefined") return true;
  try {
    const saved = window.localStorage.getItem(canvasRunsSidebarOpenStorageKey(canvasId));
    if (saved === null && canvasId) {
      const legacySaved = window.localStorage.getItem(CANVAS_RUNS_SIDEBAR_OPEN_STORAGE_KEY);
      if (legacySaved === null) return true;
      return legacySaved === "true";
    }

    if (saved === null) return true;
    return saved === "true";
  } catch {
    return true;
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
      if (canvasId) {
        writeCanvasRunsSidebarOpen(canvasId, open);
        return;
      }

      if (typeof window === "undefined") return;
      try {
        window.localStorage.setItem(canvasRunsSidebarOpenStorageKey(canvasId), open ? "true" : "false");
      } catch {
        // Preference persistence is best-effort.
      }
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
  runningRunsCount?: number;
};
