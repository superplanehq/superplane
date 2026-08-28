import { useCallback, useEffect, useRef, useState, type PointerEvent } from "react";

export const DEFAULT_LOG_PERCENT = 65;
const MIN_LOG_PERCENT = 22;
const MAX_LOG_PERCENT = 72;

/**
 * Horizontal split for the automation run page. The left pane width is a
 * percent of the container. Drag the gutter to change it.
 */
export function useSplitRunPanePercent() {
  const containerRef = useRef<HTMLDivElement>(null);
  const [percent, setPercent] = useState(DEFAULT_LOG_PERCENT);
  const [isResizing, setIsResizing] = useState(false);
  const pointerIdRef = useRef<number | null>(null);

  const resizeToClientX = useCallback((clientX: number) => {
    const box = containerRef.current?.getBoundingClientRect();
    if (!box || box.width <= 0) {
      return;
    }
    const next = ((clientX - box.left) / box.width) * 100;
    setPercent(Math.max(MIN_LOG_PERCENT, Math.min(MAX_LOG_PERCENT, next)));
  }, []);

  const startResize = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      event.preventDefault();
      pointerIdRef.current = event.pointerId;
      resizeToClientX(event.clientX);
      setIsResizing(true);
    },
    [resizeToClientX],
  );

  useEffect(() => {
    if (!isResizing) {
      return;
    }

    const handlePointerMove = (event: globalThis.PointerEvent) => {
      if (pointerIdRef.current !== null && event.pointerId !== pointerIdRef.current) {
        return;
      }
      resizeToClientX(event.clientX);
    };

    const finishResize = (event: globalThis.PointerEvent) => {
      if (pointerIdRef.current !== null && event.pointerId !== pointerIdRef.current) {
        return;
      }
      pointerIdRef.current = null;
      setIsResizing(false);
    };

    window.addEventListener("pointermove", handlePointerMove);
    window.addEventListener("pointerup", finishResize);
    window.addEventListener("pointercancel", finishResize);
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";

    return () => {
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerup", finishResize);
      window.removeEventListener("pointercancel", finishResize);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizing, resizeToClientX]);

  return { containerRef, percent, isResizing, startResize } as const;
}
