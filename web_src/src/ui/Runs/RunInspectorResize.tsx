import { useCallback, useEffect, useRef, useState, type PointerEvent } from "react";
import { cn } from "@/lib/utils";

const INSPECTOR_WIDTH_STORAGE_KEY = "superplane.runInspector.width.v3";
const DEFAULT_INSPECTOR_WIDTH = 480;
const MIN_INSPECTOR_WIDTH = 360;
const MAX_INSPECTOR_WIDTH_RATIO = 0.48;
const CANVAS_MIN_WIDTH = 280;

export function ResizeHandle({
  isResizing,
  onPointerDown,
}: {
  isResizing: boolean;
  onPointerDown: (event: PointerEvent<HTMLDivElement>) => void;
}) {
  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize run inspector"
      data-testid="run-inspector-resize-handle"
      onPointerDown={onPointerDown}
      className={cn(
        "absolute left-0 top-0 z-30 h-full w-1 -translate-x-1/2 cursor-ew-resize transition-colors hover:bg-blue-300/60",
        isResizing && "bg-blue-400/70",
      )}
    />
  );
}

export function useResizableInspectorWidth() {
  const [width, setWidth] = useState(readInspectorWidth);
  const [isResizing, setIsResizing] = useState(false);
  const activePointerIdRef = useRef<number | null>(null);

  const resizeToClientX = useCallback((clientX: number) => {
    if (!Number.isFinite(clientX)) return;

    const nextWidth = clampInspectorWidth(window.innerWidth - clientX);
    setWidth(nextWidth);
    localStorage.setItem(INSPECTOR_WIDTH_STORAGE_KEY, String(nextWidth));
  }, []);

  const startResize = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      event.preventDefault();
      activePointerIdRef.current = event.pointerId;
      resizeToClientX(event.clientX);
      setIsResizing(true);
    },
    [resizeToClientX],
  );

  useEffect(() => {
    if (!isResizing) return;

    const handlePointerMove = (event: globalThis.PointerEvent) => {
      if (activePointerIdRef.current !== null && event.pointerId !== activePointerIdRef.current) return;
      resizeToClientX(event.clientX);
    };

    const finishResize = (event: globalThis.PointerEvent) => {
      if (activePointerIdRef.current !== null && event.pointerId !== activePointerIdRef.current) return;
      activePointerIdRef.current = null;
      setIsResizing(false);
    };

    window.addEventListener("pointermove", handlePointerMove);
    window.addEventListener("pointerup", finishResize);
    window.addEventListener("pointercancel", finishResize);
    document.body.style.cursor = "ew-resize";
    document.body.style.userSelect = "none";

    return () => {
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerup", finishResize);
      window.removeEventListener("pointercancel", finishResize);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizing, resizeToClientX]);

  return { width, isResizing, startResize } as const;
}

function readInspectorWidth(): number {
  if (typeof window === "undefined") return MIN_INSPECTOR_WIDTH;

  const storedWidth = Number.parseInt(localStorage.getItem(INSPECTOR_WIDTH_STORAGE_KEY) || "", 10);
  if (Number.isFinite(storedWidth)) return clampInspectorWidth(storedWidth);

  return clampInspectorWidth(DEFAULT_INSPECTOR_WIDTH);
}

function clampInspectorWidth(width: number): number {
  if (typeof window === "undefined") return Math.max(MIN_INSPECTOR_WIDTH, width);

  const maxByViewport = Math.max(MIN_INSPECTOR_WIDTH, window.innerWidth - CANVAS_MIN_WIDTH);
  const maxByRatio = Math.round(window.innerWidth * MAX_INSPECTOR_WIDTH_RATIO);
  const maxWidth = Math.min(maxByViewport, maxByRatio);

  return Math.max(MIN_INSPECTOR_WIDTH, Math.min(maxWidth, Math.round(width)));
}
