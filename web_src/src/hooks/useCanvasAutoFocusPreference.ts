import { useCallback, useState } from "react";

export const CANVAS_AUTO_FOCUS_STORAGE_KEY = "canvas-auto-focus-enabled";

const DEFAULT_ENABLED = true;

/**
 * Read the persisted auto-focus preference from `localStorage`. The default
 * is `true` so existing users keep the current framing behavior when the
 * value is missing, invalid, or unavailable (SSR, private mode, quota).
 */
export function readStoredCanvasAutoFocusEnabled(): boolean {
  if (typeof window === "undefined") {
    return DEFAULT_ENABLED;
  }

  try {
    const stored = window.localStorage.getItem(CANVAS_AUTO_FOCUS_STORAGE_KEY);
    if (stored === null) {
      return DEFAULT_ENABLED;
    }

    const parsed: unknown = JSON.parse(stored);
    return typeof parsed === "boolean" ? parsed : DEFAULT_ENABLED;
  } catch {
    return DEFAULT_ENABLED;
  }
}

function persistCanvasAutoFocusEnabled(value: boolean): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(CANVAS_AUTO_FOCUS_STORAGE_KEY, JSON.stringify(value));
  } catch {
    // Ignore storage failures (private mode, quota, etc.).
  }
}

/**
 * Owns the "auto-focus canvas viewport on run/step selection" preference.
 * Defaults to enabled, persists on toggle, and exposes a stable callback so
 * downstream memoized props do not thrash.
 */
export function useCanvasAutoFocusPreference(): {
  isAutoFocusEnabled: boolean;
  handleToggleAutoFocus: () => void;
} {
  const [isAutoFocusEnabled, setIsAutoFocusEnabled] = useState<boolean>(readStoredCanvasAutoFocusEnabled);

  const handleToggleAutoFocus = useCallback(() => {
    setIsAutoFocusEnabled((prev) => {
      const next = !prev;
      persistCanvasAutoFocusEnabled(next);
      return next;
    });
  }, []);

  return { isAutoFocusEnabled, handleToggleAutoFocus };
}
