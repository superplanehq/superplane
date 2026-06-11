import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Width below which the rail auto-collapses. Chosen so a typical 2-column
 * panel card on the console grid (which gives the editor roughly 480-560px
 * of inner width on the narrower configurations) collapses the rail, while
 * a 3-column-wide card keeps it expanded.
 */
const NARROW_BREAKPOINT_PX = 520;

/**
 * Drive a rail's "collapsed" state from the observed width of a container
 * element while still letting the user override the auto choice manually.
 *
 * Behavior:
 *  - Observes the element attached via `containerRef` with a `ResizeObserver`.
 *  - `isNarrow` is `true` when the measured width is below the breakpoint.
 *    A width of `0` (e.g. before mount, or in jsdom where `ResizeObserver`
 *    is a no-op stub from `test/setup.ts`) is treated as "not narrow" so the
 *    rail stays expanded by default in tests.
 *  - The manual override is cleared whenever `isNarrow` flips, so dragging
 *    the panel back to a wider size re-applies the auto behavior (and vice
 *    versa) without leaving the user stuck in a stale forced state.
 */
export function useResponsiveRailCollapse(): {
  containerRef: React.RefObject<HTMLDivElement | null>;
  collapsed: boolean;
  toggle: () => void;
} {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [isNarrow, setIsNarrow] = useState(false);
  const [override, setOverride] = useState<boolean | null>(null);

  useEffect(() => {
    const element = containerRef.current;
    if (!element) return;

    const update = () => {
      const width = element.getBoundingClientRect().width;
      setIsNarrow(width > 0 && width < NARROW_BREAKPOINT_PX);
    };

    update();
    const observer = new ResizeObserver(update);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  // Reset the manual override whenever the breakpoint is crossed so resizing
  // back to a different size honors the auto choice again.
  useEffect(() => {
    setOverride(null);
  }, [isNarrow]);

  const collapsed = override ?? isNarrow;
  const toggle = useCallback(() => {
    setOverride((prev) => !(prev ?? isNarrow));
  }, [isNarrow]);

  return { containerRef, collapsed, toggle };
}
