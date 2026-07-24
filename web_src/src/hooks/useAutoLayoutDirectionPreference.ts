import { useCallback, useState } from "react";

import { DEFAULT_LAYOUT_DIRECTION, type LayoutDirection } from "@/lib/layout";

export const CANVAS_LAYOUT_DIRECTION_STORAGE_KEY = "canvas-auto-layout-direction";

/**
 * Read the persisted auto-layout direction from `localStorage`. Falls back to
 * the freeform-friendly horizontal default when the value is missing, invalid,
 * or storage is unavailable (SSR, private mode, quota).
 */
export function readStoredLayoutDirection(): LayoutDirection {
  if (typeof window === "undefined") {
    return DEFAULT_LAYOUT_DIRECTION;
  }

  return window.localStorage.getItem(CANVAS_LAYOUT_DIRECTION_STORAGE_KEY) === "vertical"
    ? "vertical"
    : DEFAULT_LAYOUT_DIRECTION;
}

function persistLayoutDirection(direction: LayoutDirection): void {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(CANVAS_LAYOUT_DIRECTION_STORAGE_KEY, direction);
}

/**
 * Tracks whether auto-layout should arrange the canvas horizontally (freeform
 * default) or vertically (clean top-to-bottom pipeline view). The preference is
 * persisted so a user's chosen orientation sticks across canvases and sessions.
 */
export function useAutoLayoutDirectionPreference(): {
  layoutDirection: LayoutDirection;
  setLayoutDirection: (direction: LayoutDirection) => void;
} {
  const [layoutDirection, setLayoutDirection] = useState<LayoutDirection>(readStoredLayoutDirection);

  const setDirection = useCallback((direction: LayoutDirection) => {
    setLayoutDirection(direction);
    persistLayoutDirection(direction);
  }, []);

  return { layoutDirection, setLayoutDirection: setDirection };
}
