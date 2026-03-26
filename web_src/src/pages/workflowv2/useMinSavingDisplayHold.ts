import { useEffect, useRef, useState } from "react";

/** Default minimum time to keep "Saving…" visible after the save request finishes (ms). */
export const DEFAULT_MIN_SAVING_DISPLAY_MS = 1000;

/**
 * When `isPending` flips false quickly, keeps an extra "holding" state so the UI can
 * show "Saving…" for at least `minMs` from when the save started.
 */
export function useMinSavingDisplayHold(isPending: boolean, minMs = DEFAULT_MIN_SAVING_DISPLAY_MS): boolean {
  const saveStartedAtRef = useRef<number | null>(null);
  const holdTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [holdDisplay, setHoldDisplay] = useState(false);

  useEffect(() => {
    if (isPending) {
      if (holdTimerRef.current) {
        clearTimeout(holdTimerRef.current);
        holdTimerRef.current = null;
      }
      saveStartedAtRef.current = Date.now();
      setHoldDisplay(false);
      return;
    }

    if (saveStartedAtRef.current === null) {
      return;
    }

    const start = saveStartedAtRef.current;
    const elapsed = Date.now() - start;
    const remaining = Math.max(0, minMs - elapsed);

    if (remaining === 0) {
      saveStartedAtRef.current = null;
      setHoldDisplay(false);
      return;
    }

    setHoldDisplay(true);
    holdTimerRef.current = setTimeout(() => {
      holdTimerRef.current = null;
      saveStartedAtRef.current = null;
      setHoldDisplay(false);
    }, remaining);

    return () => {
      if (holdTimerRef.current) {
        clearTimeout(holdTimerRef.current);
        holdTimerRef.current = null;
      }
    };
  }, [isPending, minMs]);

  return holdDisplay;
}
