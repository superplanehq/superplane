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
 * Important: when we force `active={true}` on Recharts' Tooltip, that value
 * echoes back through the content's `active` prop. Ignoring that echo while
 * `forceActive` is set keeps the grace timer alive so the tooltip can dismiss.
 */
export function useInteractiveChartTooltip(enabled: boolean) {
  const [forceActive, setForceActive] = useState(false);
  const forceActiveRef = useRef(false);
  const tooltipHoveredRef = useRef(false);
  const wasActiveRef = useRef(false);
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
    (active: boolean) => {
      if (!enabled) return;
      if (active) {
        // Forced `active={true}` echoes back through RechartsActiveBridge.
        // Ignoring that echo keeps the grace timer alive so the tooltip can dismiss.
        if (forceActiveRef.current) return;
        clearGraceTimer();
        wasActiveRef.current = true;
        return;
      }
      if (!wasActiveRef.current) return;
      wasActiveRef.current = false;
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
