import { useCallback, useEffect, useState } from "react";

export const VIEWPORT_LOCK_STORAGE_KEY = "canvasViewportLocked";

function readStoredLock(): boolean {
  if (typeof window === "undefined") {
    return false;
  }
  try {
    return window.localStorage.getItem(VIEWPORT_LOCK_STORAGE_KEY) === "true";
  } catch {
    return false;
  }
}

/**
 * Persists whether the canvas viewport (zoom + position) is locked.
 *
 * When locked, SuperPlane stops taking over the viewport with automatic
 * fit-to-view on navigation (editing, run inspection, version switches), so the
 * user's chosen zoom level and position stay put and they remain in control.
 * The preference is stored globally so it survives the canvas remounts that
 * happen when switching between modes.
 */
export function useViewportLock(): [boolean, () => void] {
  const [isLocked, setIsLocked] = useState<boolean>(readStoredLock);

  // Keep multiple canvases / tabs in sync when the preference changes elsewhere.
  useEffect(() => {
    const handleStorage = (event: StorageEvent) => {
      if (event.key === VIEWPORT_LOCK_STORAGE_KEY) {
        setIsLocked(event.newValue === "true");
      }
    };
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);

  const toggle = useCallback(() => {
    setIsLocked((prev) => {
      const next = !prev;
      try {
        window.localStorage.setItem(VIEWPORT_LOCK_STORAGE_KEY, String(next));
      } catch {
        // Ignore storage failures (private mode / disabled storage).
      }
      return next;
    });
  }, []);

  return [isLocked, toggle];
}
