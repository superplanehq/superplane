import { useCallback, useState } from "react";

export type CanvasLayoutMode = "freeform" | "vertical";

export const CANVAS_LAYOUT_MODE_STORAGE_KEY = "canvas-layout-mode";

const DEFAULT_LAYOUT_MODE: CanvasLayoutMode = "freeform";

export function isCanvasLayoutMode(value: unknown): value is CanvasLayoutMode {
  return value === "freeform" || value === "vertical";
}

export function readStoredCanvasLayoutMode(): CanvasLayoutMode {
  if (typeof window === "undefined") {
    return DEFAULT_LAYOUT_MODE;
  }

  const stored = window.localStorage.getItem(CANVAS_LAYOUT_MODE_STORAGE_KEY);
  return isCanvasLayoutMode(stored) ? stored : DEFAULT_LAYOUT_MODE;
}

function persistCanvasLayoutMode(mode: CanvasLayoutMode): void {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(CANVAS_LAYOUT_MODE_STORAGE_KEY, mode);
}

/**
 * Persisted preference for how the canvas arranges nodes:
 * - "freeform": user-controlled positions (default).
 * - "vertical": engine-controlled top-to-bottom auto-layout view (render-only overlay).
 */
export function useCanvasLayoutMode() {
  const [layoutMode, setLayoutModeState] = useState<CanvasLayoutMode>(() => readStoredCanvasLayoutMode());

  const setLayoutMode = useCallback((mode: CanvasLayoutMode) => {
    setLayoutModeState(mode);
    persistCanvasLayoutMode(mode);
  }, []);

  const toggleLayoutMode = useCallback(() => {
    setLayoutModeState((previous) => {
      const next: CanvasLayoutMode = previous === "vertical" ? "freeform" : "vertical";
      persistCanvasLayoutMode(next);
      return next;
    });
  }, []);

  return { layoutMode, setLayoutMode, toggleLayoutMode };
}
