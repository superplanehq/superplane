import { useLayoutEffect, useRef, useState } from "react";

/**
 * Suppress react-grid-layout's default 200ms transitions until the grid has
 * settled on its real width. `WidthProvider` mounts with a hardcoded 1280px
 * width and only learns the actual container size via a ResizeObserver callback
 * that fires asynchronously after layout — by which time a fixed
 * `requestAnimationFrame` delay has already re-enabled transitions and every
 * tile animates from the 1280px layout to the real one. We instead observe the
 * wrapper directly and arm transitions only after the first non-zero width
 * measurement has been painted, so drag / resize still feel responsive without
 * the tab-switch stretch animation.
 *
 * The effect must not arm while the grid is unmounted (loading / error / empty
 * early returns leave `gridWrapperRef` null).
 */
export function useConsoleGridTransitionArming(gridVisible: boolean) {
  const [transitionsArmed, setTransitionsArmed] = useState(false);
  const [gridWidth, setGridWidth] = useState(0);
  const gridWrapperRef = useRef<HTMLDivElement>(null);
  const armFrameRef = useRef<number | null>(null);

  useLayoutEffect(() => {
    if (!gridVisible) {
      setTransitionsArmed(false);
      setGridWidth(0);
      return undefined;
    }
    const el = gridWrapperRef.current;
    if (!el) return undefined;
    if (typeof ResizeObserver === "undefined") {
      armFrameRef.current = requestAnimationFrame(() => setTransitionsArmed(true));
      return () => {
        if (armFrameRef.current != null) cancelAnimationFrame(armFrameRef.current);
      };
    }
    const observer = new ResizeObserver((entries) => {
      const width = entries[0]?.contentRect.width ?? 0;
      setGridWidth(width);
      if (width <= 0) return;
      if (!transitionsArmed) {
        armFrameRef.current = requestAnimationFrame(() => setTransitionsArmed(true));
      }
    });
    observer.observe(el);
    return () => {
      observer.disconnect();
      if (armFrameRef.current != null) cancelAnimationFrame(armFrameRef.current);
    };
  }, [gridVisible, transitionsArmed]);

  return { transitionsArmed, gridWrapperRef, gridWidth };
}
