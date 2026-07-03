import { useCallback, useEffect, useRef, useState } from "react";

const CANVAS_VERSIONS_SIDEBAR_OPEN_STORAGE_KEY = "canvasVersionsSidebarOpen";

function canvasVersionsSidebarOpenStorageKey(canvasId?: string): string {
  return canvasId
    ? `${CANVAS_VERSIONS_SIDEBAR_OPEN_STORAGE_KEY}:${canvasId}`
    : CANVAS_VERSIONS_SIDEBAR_OPEN_STORAGE_KEY;
}

function readInitialVersionsSidebarOpen(canvasId?: string): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(canvasVersionsSidebarOpenStorageKey(canvasId)) === "true";
  } catch {
    return false;
  }
}

export function useCanvasVersionsSidebarState(canvasId?: string) {
  const [isVersionsSidebarOpen, setIsVersionsSidebarOpen] = useState(() => readInitialVersionsSidebarOpen(canvasId));

  const previousCanvasIdRef = useRef(canvasId);
  useEffect(() => {
    if (previousCanvasIdRef.current === canvasId) return;
    previousCanvasIdRef.current = canvasId;
    setIsVersionsSidebarOpen(readInitialVersionsSidebarOpen(canvasId));
  }, [canvasId]);

  const persistOpen = useCallback(
    (open: boolean) => {
      if (typeof window === "undefined") return;
      try {
        window.localStorage.setItem(canvasVersionsSidebarOpenStorageKey(canvasId), open ? "true" : "false");
      } catch {
        // Preference persistence is best-effort.
      }
    },
    [canvasId],
  );

  const handleVersionsSidebarToggle = useCallback(() => {
    setIsVersionsSidebarOpen((current) => {
      const next = !current;
      persistOpen(next);
      return next;
    });
  }, [persistOpen]);

  const openVersionsSidebar = useCallback(() => {
    setIsVersionsSidebarOpen(true);
    persistOpen(true);
  }, [persistOpen]);

  const closeVersionsSidebar = useCallback(() => {
    setIsVersionsSidebarOpen(false);
    persistOpen(false);
  }, [persistOpen]);

  return {
    isVersionsSidebarOpen,
    handleVersionsSidebarToggle,
    openVersionsSidebar,
    closeVersionsSidebar,
  };
}

export type CanvasVersionsSidebarState = ReturnType<typeof useCanvasVersionsSidebarState> & {
  showVersionsSidebarToggle: boolean;
};
