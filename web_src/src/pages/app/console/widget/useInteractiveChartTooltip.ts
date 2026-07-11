import { useCallback, useEffect, useRef, useState, type CSSProperties } from "react";

/** Grace period to move from a chart point onto an interactive tooltip. */
export const TOOLTIP_INTERACT_GRACE_MS = 200;

const TOOLTIP_WRAPPER_STYLE: CSSProperties = { transition: "none" };

// Timestamp tooltips host a CopyButton. Recharts defaults the wrapper to
// `pointer-events: none`, which makes that control unreachable — override it
// and keep the tooltip mounted briefly so the pointer can move onto it.
const INTERACTIVE_TOOLTIP_WRAPPER_STYLE: CSSProperties = {
  transition: "none",
  pointerEvents: "auto",
};

/**
 * Keeps a Recharts tooltip interactive: enables pointer events and holds the
 * tooltip open long enough to move from the chart point onto CopyButton / etc.
 *
 * Forced `active={true}` echoes back through Recharts. We ignore that echo
 * (same point key while forced) so the grace timer can dismiss — but a different
 * point key during grace means the pointer moved onto another bar, so we
 * re-engage natural hover tracking.
 */
export function useInteractiveChartTooltip(enabled: boolean) {
  const [forceActive, setForceActive] = useState(false);
  const forceActiveRef = useRef(false);
  const tooltipHoveredRef = useRef(false);
  const wasActiveRef = useRef(false);
  const activeKeyRef = useRef<string | null>(null);
  const frozenKeyRef = useRef<string | null>(null);
  const graceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearGraceTimer = useCallback(() => {
    if (graceTimerRef.current == null) return;
    clearTimeout(graceTimerRef.current);
    graceTimerRef.current = null;
  }, []);

  const setForced = useCallback((next: boolean) => {
    forceActiveRef.current = next;
    setForceActive(next);
  }, []);

  useEffect(() => () => clearGraceTimer(), [clearGraceTimer]);

  const syncRechartsActive = useCallback(
    (active: boolean, activeKey?: string) => {
      if (!enabled) return;
      const key = activeKey ?? null;

      if (active) {
        if (forceActiveRef.current) {
          // Echo of our forced active on the same point — keep grace running.
          if (key == null || key === frozenKeyRef.current) return;
          // Pointer moved onto a different point during grace: resume normal tracking.
          clearGraceTimer();
          setForced(false);
          wasActiveRef.current = true;
          activeKeyRef.current = key;
          frozenKeyRef.current = null;
          return;
        }
        clearGraceTimer();
        wasActiveRef.current = true;
        if (key != null) activeKeyRef.current = key;
        return;
      }

      if (!wasActiveRef.current) return;
      wasActiveRef.current = false;
      frozenKeyRef.current = activeKeyRef.current;
      clearGraceTimer();
      setForced(true);
      graceTimerRef.current = setTimeout(() => {
        graceTimerRef.current = null;
        if (!tooltipHoveredRef.current) setForced(false);
      }, TOOLTIP_INTERACT_GRACE_MS);
    },
    [enabled, clearGraceTimer, setForced],
  );

  const onTooltipEnter = useCallback(() => {
    if (!enabled) return;
    tooltipHoveredRef.current = true;
    clearGraceTimer();
    setForced(true);
  }, [enabled, clearGraceTimer, setForced]);

  const onTooltipLeave = useCallback(() => {
    if (!enabled) return;
    tooltipHoveredRef.current = false;
    clearGraceTimer();
    setForced(false);
    wasActiveRef.current = false;
  }, [enabled, clearGraceTimer, setForced]);

  return {
    // `true` forces the tooltip to stay visible; `undefined` defers to Recharts.
    activeProp: enabled && forceActive ? true : undefined,
    forceContentActive: enabled && forceActive,
    syncRechartsActive,
    onTooltipEnter,
    onTooltipLeave,
    wrapperStyle: enabled ? INTERACTIVE_TOOLTIP_WRAPPER_STYLE : TOOLTIP_WRAPPER_STYLE,
  };
}
